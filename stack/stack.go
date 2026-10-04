package stack

import (
	"encoding/json/jsontext"
	"runtime"
	"strconv"
	"unsafe"
)

type stack [32]uintptr
type Stack = *stack

func (s Stack) MarshalJSONTo(enc *jsontext.Encoder) error {
	if s[0] == 0 {
		return enc.WriteToken(jsontext.Null)
	}
	fs := runtime.CallersFrames(s[:])

	var buf [256]byte
	if s[1] == 0 {
		f, _ := fs.Next()
		b := buf[:0]
		b = append(b, f.File...)
		b = append(b, ':')
		b = strconv.AppendInt(b, int64(f.Line), 10)
		return enc.WriteToken(jsontext.String(str(b)))
	} else {
		if err := enc.WriteToken(jsontext.BeginObject); err != nil {
			return err
		}

		for i := int64(0); ; i++ {
			f, more := fs.Next()

			key := strconv.AppendInt(buf[:0], i, 10)
			if err := enc.WriteToken(jsontext.String(str(key))); err != nil {
				return err
			}

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
		return enc.WriteToken(jsontext.EndObject)
	}
}

func str(b []byte) string { return unsafe.String(unsafe.SliceData(b), len(b)) }

// New captures a [Stack], skip is passed to [runtime.Callers].
func New(skip int) Stack {
	var pcs [32]uintptr
	runtime.Callers(skip, pcs[:])
	return Stack(&pcs)
}

const StackKey = "stack"
