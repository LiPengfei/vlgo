// Package logproxy
/*
 @File : log_proxy_writer.go
 @Description: 日志代理服务，用来将日志发送到指定协程
 @Author : yyh
 @Time : 2021/10/22 16:00
 @Update:
*/
package logproxy

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"time"
	"vlgo/timerpool"
)

const (
	maxLogChanSize       = 2048
	defaultLogBufferSize = 256
	logTempBufferSize    = 4096
)

var (
	errLogRoutineWriteFailed = errors.New("log routine write failed")
	logBlockWaitTime         = time.Millisecond * 200
)

type Writer struct {
	LogCh      chan *BufferWrap
	Writer     io.Writer
	tempBuffer []byte
	Cache      *LogBufferCache
}

func NewLogProxyWriter(w io.Writer) *Writer {
	r := &Writer{
		LogCh:      make(chan *BufferWrap, maxLogChanSize),
		Writer:     w,
		tempBuffer: make([]byte, logTempBufferSize),
		Cache:      NewLogBufferCache(),
	}

	go r.process()
	return r
}

func (lr *Writer) Write(p []byte) (n int, err error) {
	n = len(p)
	logBuffer := lr.Cache.GetBuffer(n)
	logBuffer.Copy(p)

	select {
	case lr.LogCh <- logBuffer:
		return
	default:
	}

	// logCh已满，阻塞等待一段时间
	t := timerpool.GetTimer(logBlockWaitTime)
	defer timerpool.PutTimer(t)
	select {
	case lr.LogCh <- logBuffer:
		return
	case <-t.C:
		n = 0
		_, _ = fmt.Fprintf(os.Stderr, "LogProxy ReceiveLog failed:log=%v\n", logBuffer)
		err = errLogRoutineWriteFailed
	}
	return
}

func (lr *Writer) writeLog() {
	defer func() {
		if r := recover(); r != nil {
			_, _ = fmt.Fprintf(os.Stderr, "LogProxy write crash, stack:%s\n", debug.Stack())
		}
	}()

	select {
	case logBuffer := <-lr.LogCh:
		tempBuffer := lr.tempBuffer[:0]
		firstLog := logBuffer
		var nextLog *BufferWrap
		for {
			select {
			case nextLog = <-lr.LogCh:
			default:
				break
			}
			if nextLog == nil {
				break
			}
			if firstLog != nil {
				if firstLog.Len() > logTempBufferSize-nextLog.Len() {
					break
				}
				curLen := len(tempBuffer)
				newLen := curLen + firstLog.Len()
				copy(tempBuffer[curLen:newLen], firstLog.Buffer())
				tempBuffer = tempBuffer[:newLen]
				lr.Cache.PutBuffer(firstLog)
				firstLog = nil
			} else {
				if len(tempBuffer) > logTempBufferSize-nextLog.Len() {
					break
				}
			}
			// copy nextLog
			curLen := len(tempBuffer)
			newLen := curLen + nextLog.Len()
			copy(tempBuffer[curLen:newLen], nextLog.Buffer())
			tempBuffer = tempBuffer[:newLen]
			lr.Cache.PutBuffer(nextLog)
			nextLog = nil
		}
		if firstLog != nil {
			_, err := lr.Writer.Write(firstLog.Buffer())
			lr.Cache.PutBuffer(firstLog)
			if err != nil {
				fmt.Fprintf(os.Stderr, "LogProxy write firstLog failed:log=%v,err=%v\n", firstLog, err)
			}
		}
		if len(tempBuffer) > 0 {
			_, err := lr.Writer.Write(tempBuffer)
			if err != nil {
				fmt.Fprintf(os.Stderr, "LogProxy write tempBuffer failed:log=%v,err=%v\n", tempBuffer, err)
			}
		}
		if nextLog != nil {
			_, err := lr.Writer.Write(nextLog.Buffer())
			lr.Cache.PutBuffer(nextLog)
			if err != nil {
				fmt.Fprintf(os.Stderr, "LogProxy write nextLog failed:log=%v,err=%v\n", nextLog, err)
			}
		}
	}
}

func (lr *Writer) process() {
	for {
		lr.writeLog()
	}
}
