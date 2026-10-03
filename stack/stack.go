package stack

import (
	"log/slog"
	"runtime"
	"strconv"
	"unsafe"
)

type stack [32]uintptr
type Stack = *stack

var emptyValue = slog.GroupValue()
var _ slog.LogValuer = (Stack)(nil)

func (s Stack) LogValue() slog.Value {
	var attrs []slog.Attr

	fs := runtime.CallersFrames(s[:])
	for {
		f, more := fs.Next()

		// todo: 这种方式不好, 会分配内存

		var b = make([]byte, 0, len(f.File)+16)
		b = append(b, f.File...)
		b = append(b, ':')
		b = strconv.AppendInt(b, int64(f.Line), 10)

		attrs = append(attrs, slog.Attr{
			Key:   strconv.Itoa(len(attrs)),
			Value: slog.StringValue(unsafe.String(unsafe.SliceData(b), len(b))),
		})
		if !more {
			break
		}
	}

	switch len(attrs) {
	case 0:
		return emptyValue
	case 1:
		return attrs[0].Value
	default:
		return slog.GroupValue(attrs...)
	}
}

// New captures a [Stack], skip is passed to [runtime.Callers].
func New(skip int) Stack {
	var pcs [32]uintptr
	n := runtime.Callers(skip, pcs[:])
	return Stack(pcs[:n])
}

const StackKey = "stack"
