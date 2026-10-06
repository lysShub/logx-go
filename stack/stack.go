package stack

import (
	"encoding/json/jsontext"
	"runtime"
	"slices"
	"strconv"
	"unsafe"
)

type Stack []uintptr

func (s Stack) MarshalJSONTo(enc *jsontext.Encoder) error {
	fs := runtime.CallersFrames(s)

	var buf = make([]byte, 256)
	switch len(s) {
	case 0:
		return enc.WriteToken(jsontext.Null)
	case 1:
		f, _ := fs.Next()
		b := buf[:0]
		b = append(b, f.File...)
		b = append(b, ':')
		b = strconv.AppendInt(b, int64(f.Line), 10)
		return enc.WriteToken(jsontext.String(str(b)))
	default:
		if err := enc.WriteToken(jsontext.BeginArray); err != nil {
			return err
		}
		for {
			f, more := fs.Next()

			val := buf[:0]
			val = append(val, f.File...)
			val = append(val, ':')
			val = strconv.AppendInt(val, int64(f.Line), 10)
			if err := enc.WriteToken(jsontext.String(str(val))); err != nil {
				return err
			}
			if !more {
				break
			}
		}
		return enc.WriteToken(jsontext.EndArray)
	}
}
func str(b []byte) string { return unsafe.String(unsafe.SliceData(b), len(b)) }

// New captures a [Stack], skip is passed to [runtime.Callers].
func New(skip ...int) Stack {
	var pcs [64]uintptr

	var n = 2
	if len(skip) > 0 {
		n = skip[0]
	}
	m := runtime.Callers(n, pcs[:])
	return Stack(slices.Clone(pcs[:m]))
}

const StackKey = "stack"

func Source(skip ...int) Stack {
	var pc = []uintptr{0: 0}

	var n = 2
	if len(skip) > 0 {
		n = skip[0]
	}
	runtime.Callers(n, pc)
	return Stack(pc)
}
