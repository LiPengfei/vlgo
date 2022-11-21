// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package encoder

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// MyCallerEncode serializes a caller in "file:line package/func()" format, trimming
// all but the final directory from the full path.
func MyCallerEncode(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	{
		if !caller.Defined {
			enc.AppendString("undefined")
		} else {
			idx := strings.LastIndexByte(caller.File, '/')
			if idx == -1 {
				enc.AppendString(caller.FullPath())
			} else {
				enc.AppendString(caller.File[idx+1:])
				enc.AppendString(":")
				enc.AppendInt64(int64(caller.Line))
				enc.AppendString(" ")
			}
		}
	}
	// TODO: consider using a byte-oriented API to save an allocation.
	{
		funcname := runtime.FuncForPC(caller.PC).Name()
		idx := strings.LastIndexByte(funcname, '/')
		if idx == -1 {
			enc.AppendString(funcname)
			enc.AppendString("()")
		} else {
			enc.AppendString(funcname[idx+1:])
			enc.AppendString("()")
		}
	}
}

// Foreground colors.
const (
	Black Color = iota + 30
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

// Color represents a text color.
type Color uint8

// Add adds the coloring to the given string.
func (c Color) Add(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", uint8(c), s)
}

var (
	_levelToColor = map[zapcore.Level]Color{
		zap.DebugLevel:  White,
		zap.InfoLevel:   Cyan,
		zap.WarnLevel:   Yellow,
		zap.ErrorLevel:  Red,
		zap.DPanicLevel: Red,
		zap.PanicLevel:  Red,
		zap.FatalLevel:  Red,
	}
	_unknownLevelColor = Red

	_levelToCapitalColorString = make(map[zapcore.Level]string, len(_levelToColor))
)

func init() {
	for level, color := range _levelToColor {
		_levelToCapitalColorString[level] = color.Add(MyLevelString(level))
	}
}

// MyLevelString serializes a Level to an all-caps string. For example,
// InfoLevel is serialized to "INFO".
func MyLevelString(l zapcore.Level) string {
	switch l {
	case zapcore.DebugLevel:
		return "DEBU"
	case zapcore.InfoLevel:
		return "INFO"
	case zapcore.WarnLevel:
		return "WARN"
	case zapcore.ErrorLevel:
		return "ERRO"
	case zapcore.DPanicLevel:
		return "DPAN"
	case zapcore.PanicLevel:
		return "PANI"
	case zapcore.FatalLevel:
		return "FATA"
	default:
		return fmt.Sprintf("Level(%d)", l)
	}
}

// MyLevelEncoder serializes a Level to an all-caps string. For example,
// InfoLevel is serialized to "INFO".
func MyLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(MyLevelString(l))
}

// MyColorLevelEncoder serializes a Level to an all-caps string with color. For example,
// InfoLevel is serialized to "\x1b[33mINFO\x1b[0m".
func MyColorLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	s, ok := _levelToCapitalColorString[l]
	if !ok {
		s = _unknownLevelColor.Add(MyLevelString(l))
	}
	enc.AppendString(s)
}

var (
	_bufferpool = buffer.NewPool()
	// GetBuffer retrieves a buffer from the pool, creating one if necessary.
	GetBuffer = _bufferpool.Get
)

// sliceArrayEncoder is an ArrayEncoder backed by a simple []interface{}. Like
// the MapObjectEncoder, it's not designed for production use.
type sliceArrayEncoder struct {
	elems []interface{}
}

func (s *sliceArrayEncoder) AppendArray(v zapcore.ArrayMarshaler) error {
	enc := &sliceArrayEncoder{}
	err := v.MarshalLogArray(enc)
	s.elems = append(s.elems, enc.elems)
	return err
}

func (s *sliceArrayEncoder) AppendObject(v zapcore.ObjectMarshaler) error {
	m := zapcore.NewMapObjectEncoder()
	err := v.MarshalLogObject(m)
	s.elems = append(s.elems, m.Fields)
	return err
}

func (s *sliceArrayEncoder) AppendReflected(v interface{}) error {
	s.elems = append(s.elems, v)
	return nil
}

func (s *sliceArrayEncoder) AppendBool(v bool)              { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendByteString(v []byte)      { s.elems = append(s.elems, string(v)) }
func (s *sliceArrayEncoder) AppendComplex128(v complex128)  { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendComplex64(v complex64)    { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendDuration(v time.Duration) { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendFloat64(v float64)        { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendFloat32(v float32)        { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendInt(v int)                { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendInt64(v int64)            { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendInt32(v int32)            { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendInt16(v int16)            { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendInt8(v int8)              { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendString(v string)          { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendTime(v time.Time)         { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUint(v uint)              { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUint64(v uint64)          { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUint32(v uint32)          { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUint16(v uint16)          { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUint8(v uint8)            { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUintptr(v uintptr)        { s.elems = append(s.elems, v) }

var _sliceEncoderPool = sync.Pool{
	New: func() interface{} {
		return &sliceArrayEncoder{elems: make([]interface{}, 0, 2)}
	},
}

func getSliceEncoder() *sliceArrayEncoder {
	return _sliceEncoderPool.Get().(*sliceArrayEncoder)
}

func putSliceEncoder(e *sliceArrayEncoder) {
	e.elems = e.elems[:0]
	_sliceEncoderPool.Put(e)
}

type consoleEncoder struct {
	*jsonEncoder
}

// NewConsoleEncoder creates an encoder whose output is designed for human -
// rather than machine - consumption. It serializes the core log entry data
// (message, level, timestamp, etc.) in a plain-text format and leaves the
// structured context as JSON.
//
// Note that although the console encoder doesn't use the keys specified in the
// encoder configuration, it will omit any element whose key is set to the empty
// string.
func NewConsoleEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	return consoleEncoder{newJSONEncoder(cfg, true)}
}

func (c consoleEncoder) Clone() zapcore.Encoder {
	return consoleEncoder{c.jsonEncoder.Clone().(*jsonEncoder)}
}

func (c consoleEncoder) EncodeEntry(ent zapcore.Entry, fields []zap.Field) (*buffer.Buffer, error) {
	line := GetBuffer()

	// We don't want the entry's metadata to be quoted and escaped (if it's
	// encoded as strings), which means that we can't use the JSON encoder. The
	// simplest option is to use the memory encoder and fmt.Fprint.
	//
	// If this ever becomes a performance bottleneck, we can implement
	// ArrayEncoder for our plain-text format.
	arr := getSliceEncoder()
	if c.LevelKey != "" && c.EncodeLevel != nil {
		c.EncodeLevel(ent.Level, arr)
	}
	if c.TimeKey != "" && c.EncodeTime != nil {
		c.EncodeTime(ent.Time, arr)
	}
	if ent.LoggerName != "" && c.NameKey != "" {
		nameEncoder := c.EncodeName

		if nameEncoder == nil {
			// Fall back to FullNameEncoder for backward compatibility.
			nameEncoder = zapcore.FullNameEncoder
		}

		nameEncoder(ent.LoggerName, arr)
	}
	if ent.Caller.Defined && c.CallerKey != "" && c.EncodeCaller != nil {
		c.EncodeCaller(ent.Caller, arr)
	}
	for i := range arr.elems {
		if i > 0 {
			// line.AppendByte('\t')
		}
		fmt.Fprint(line, arr.elems[i])
	}
	putSliceEncoder(arr)

	// Add the message itself.
	if c.MessageKey != "" {
		c.addTabIfNecessary(line)
		line.AppendString(ent.Message)
	}

	// Add any structured context.
	c.writeContext(line, fields)

	// If there's no stacktrace key, honor that; this allows users to force
	// single-line output.
	if ent.Stack != "" && c.StacktraceKey != "" {
		line.AppendByte('\n')
		line.AppendString(ent.Stack)
	}

	if c.LineEnding != "" {
		line.AppendString(c.LineEnding)
	} else {
		line.AppendString(zapcore.DefaultLineEnding)
	}
	return line, nil
}

func (c consoleEncoder) writeContext(line *buffer.Buffer, extra []zap.Field) {
	context := c.jsonEncoder.Clone().(*jsonEncoder)
	defer context.buf.Free()

	addFields(context, extra)
	context.closeOpenNamespaces()
	if context.buf.Len() == 0 {
		return
	}

	c.addTabIfNecessary(line)
	line.AppendByte('{')
	line.Write(context.buf.Bytes())
	line.AppendByte('}')
}

func (c consoleEncoder) addTabIfNecessary(line *buffer.Buffer) {
	if line.Len() > 0 {
		line.AppendByte(' ')
	}
}

func addFields(enc zapcore.ObjectEncoder, fields []zap.Field) {
	for i := range fields {
		fields[i].AddTo(enc)
	}
}
