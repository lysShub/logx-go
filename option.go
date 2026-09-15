package logx

import (
	"fmt"
	"os"
	"time"
	"unsafe"

	"github.com/lysShub/logx-go/internal"
	"github.com/pkg/errors"
)

type option struct {
	// StackLevel log with stack trace when level >= StackLevel
	StackLevel Level
	// StackKind set stack kind
	StackKind stackKind
	// HandlerErr process handler-error
	HandlerErr func(err internal.Error, rec Record)
	// Now time now
	Now func() time.Time
	// ErrStack get [StackTrace] from error
	ErrStack func(err internal.Error) StackTrace
}
type stackKind uint8 //
const (
	Source stackKind = 1
	Trace  stackKind = 2
) //
type Option func(*option)

var defaultOption = option{
	StackLevel: LevelWarn,
	StackKind:  Trace,
	HandlerErr: func(err internal.Error, rec Record) {
		fmt.Fprintf(os.Stderr, "logx handle falie: %+v\n", err)
	},
	Now: time.Now,
	ErrStack: func(err internal.Error) StackTrace {
		type stack interface {
			StackTrace() errors.StackTrace
		}
		var st errors.StackTrace
		if s, ok := err.(stack); ok {
			st = s.StackTrace()
		} else if errors.As(err, &s) {
			st = s.StackTrace()
		}
		if len(st) > 0 {
			return *(*StackTrace)(unsafe.Pointer(&st))
		} else {
			return nil
		}
	},
}
