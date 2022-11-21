/*
@File : log_proxy_buffer
@Description:
@Author : yyh
@Time : 2021/10/29 10:11
@Update:
*/
package logproxy

import (
	"sync/atomic"
	"unsafe"
)

const (
	logBufferCacheCount = 8
)

type BufferWrap struct {
	Buf []byte
}

func NewBufferWrap(n int) *BufferWrap {
	return &BufferWrap{
		Buf: make([]byte, n),
	}
}
func (bw *BufferWrap) Len() int {
	return len(bw.Buf)
}
func (bw *BufferWrap) Copy(src []byte) {
	n := len(src)
	if n > cap(bw.Buf) {
		n = cap(bw.Buf)
	}
	bw.Buf = bw.Buf[:n]
	copy(bw.Buf, src)
}

func (bw *BufferWrap) Buffer() []byte {
	return bw.Buf
}

type LogBufferCache struct {
	BufferList []*BufferWrap
}

func NewLogBufferCache() *LogBufferCache {
	return &LogBufferCache{
		BufferList: make([]*BufferWrap, logBufferCacheCount),
	}
}

func (bc *LogBufferCache) GetBuffer(n int) *BufferWrap {
	if n > defaultLogBufferSize {
		return NewBufferWrap(n)
	}

	for i := 0; i < logBufferCacheCount; i++ {
		old := atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&bc.BufferList[i])))
		if old != nil && atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&bc.BufferList[i])), old, unsafe.Pointer(nil)) {
			return (*BufferWrap)(old)
		}
	}
	return NewBufferWrap(defaultLogBufferSize)
}

func (bc *LogBufferCache) PutBuffer(buffer *BufferWrap) {
	if cap(buffer.Buf) != defaultLogBufferSize {
		return
	}
	for i := 0; i < logBufferCacheCount; i++ {
		if atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&bc.BufferList[i])), nil, unsafe.Pointer(buffer)) {
			return
		}
	}
}
