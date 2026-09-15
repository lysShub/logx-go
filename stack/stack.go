package stack

import (
	"fmt"
	"log/slog"
	"runtime"
	"strconv"
	"unsafe"
)

type StackTrace []uintptr //
var emptyValue = slog.GroupValue()
var _ slog.LogValuer = (StackTrace)(nil)

// todo: 这种方式不好, 会分配内存

func (s StackTrace) LogValue() slog.Value {
	switch len(s) {
	case 0:
		return emptyValue
	case 1:
		fs := runtime.CallersFrames(s)
		f, _ := fs.Next()
		return slog.StringValue(fmt.Sprintf("%s:%d", f.File, f.Line))
	default:
		var attrs []slog.Attr

		fs := runtime.CallersFrames(s)
		for {
			f, more := fs.Next()

			var b = make([]byte, 0, len(f.File)+8)
			b = append(b, f.File...)
			b = append(b, ':')
			b = strconv.AppendInt(b, int64(f.Line), 10)

			attrs = append(attrs, slog.Attr{
				Key:   strconv.Itoa(len(attrs)),
				Value: slog.StringValue(str(b)),
			})
			if !more {
				break
			}
		}
		return slog.GroupValue(attrs...)
	}
}
func str(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// Kind is the stack kind captured by [New].
type Kind uint8 //
const (
	Source Kind = 1
	Trace  Kind = 2
)

// New captures a [StackTrace], skip is passed to [runtime.Callers].
func New(kind Kind, skip int) StackTrace {
	switch kind {
	case Source:
		var pcs [1]uintptr
		n := runtime.Callers(skip, pcs[:])
		return StackTrace(pcs[:n])
	case Trace:
		var pcs [32]uintptr
		n := runtime.Callers(skip, pcs[:])
		return StackTrace(pcs[:n])
	default:
		panic(kind)
	}
}

const StackKey = "stack"
