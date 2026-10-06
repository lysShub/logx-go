package stack

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func marshalJSON(t *testing.T, s Stack) string {
	t.Helper()
	b, err := jsonv2.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal(%p): %v", s, err)
	}
	return string(b)
}

func Test_Marshal(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s := Stack(&stack{})
		if act := marshalJSON(t, s); act != "null" {
			t.Fatalf("got %s, want null", act)
		}
	})

	t.Run("single_frame", func(t *testing.T) {
		pc, file, line, ok := runtime.Caller(1)
		if !ok {
			t.Fatal("runtime.Caller failed")
		}
		s := &stack{}
		s[0] = pc
		if s[0] == 0 || s[1] != 0 {
			t.Fatalf("test setup: want exactly one frame, got %v", s)
		}

		exp := marshalJSON(t, s)
		act := `"` + file + ":" + strconv.Itoa(line) + `"`
		if exp != act {
			t.Fatalf("got %s, want %s", exp, act)
		}
	})

	t.Run("multi_frame", func(t *testing.T) {
		s := New()

		var exp []string
		fs := runtime.CallersFrames(s[:])
		for {
			f, more := fs.Next()
			exp = append(exp, fmt.Sprintf("%s:%d", f.File, f.Line))
			if !more {
				break
			}
		}
		if len(exp) < 2 {
			t.Fatalf("test setup: want at least 2 frames, got %d", len(exp))
		}

		act := marshalJSON(t, s)
		var arr []string
		if err := jsonv2.Unmarshal([]byte(act), &arr); err != nil {
			t.Fatalf("output %s is not a JSON array: %v", act, err)
		}
		if !reflect.DeepEqual(arr, exp) {
			t.Fatalf("got %v, want %v", arr, exp)
		}
	})
}

func Test_No_MemEscape(t *testing.T) {
	s := New()
	enc := jsontext.NewEncoder(io.Discard)

	var err error
	allocs := testing.AllocsPerRun(100, func() {
		err = s.MarshalJSONTo(enc)
	})
	if err != nil {
		t.Fatal(err)
	}
	if allocs > 1 {
		t.Fatalf("MarshalJSONTo allocated %v allocs/op, want <= 1", allocs)
	}
}

func Test_Source(t *testing.T) {
	s := Source()
	if s == nil || s[0] == 0 || s[1] != 0 {
		t.Fatalf("Source: want exactly one non-nil frame, got %v", s)
	}

	got := marshalJSON(t, s)
	var str string
	if err := jsonv2.Unmarshal([]byte(got), &str); err != nil {
		t.Fatalf("output %s is not a JSON string: %v", got, err)
	}
	if !strings.Contains(str, "_test.go:") {
		t.Fatalf("source %q does not look like file:line", str)
	}
}
