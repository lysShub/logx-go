package logx

import (
	"fmt"
	"os"
	"time"
	"unsafe"

	"github.com/lysShub/logx-go/internal"
	"github.com/lysShub/logx-go/stack"
	"github.com/pkg/errors"
)

type option struct {
	// StackLevel log with stack trace when level >= StackLevel
	StackLevel Level
	// StackKind set stack kind
	StackKind stack.Kind
	// HandlerErr process handler-error
	HandlerErr func(err internal.Error, rec Record)
	// Now time now
	Now func() time.Time
	// ErrStack get [stack.StackTrace] from error
	ErrStack func(err internal.Error) stack.StackTrace
}
type Option func(*option)

var defaultOption = option{
	StackLevel: LevelWarn,
	StackKind:  stack.Trace,
	HandlerErr: func(err internal.Error, rec Record) {
		fmt.Fprintf(os.Stderr, "logx handle falie: %+v\n", err)
	},
	Now: time.Now,
	ErrStack: func(err internal.Error) stack.StackTrace {
		type stacker interface {
			StackTrace() errors.StackTrace
		}
		var st errors.StackTrace
		if s, ok := err.(stacker); ok {
			st = s.StackTrace()
		} else if errors.As(err, &s) {
			st = s.StackTrace()
		}
		if len(st) > 0 {
			return *(*stack.StackTrace)(unsafe.Pointer(&st))
		} else {
			return nil
		}
	},
}

// WithStackLevel sets the minimum level to attach a stack trace, default [LevelWarn].
func WithStackLevel(level Level) Option {
	return func(o *option) { o.StackLevel = level }
}

// WithStackKind sets how the stack is rendered, [stack.Source] for the call site only, [stack.Trace] for the full call stack, default [stack.Trace].
func WithStackKind(kind stack.Kind) Option {
	return func(o *option) { o.StackKind = kind }
}

// WithNow sets the time source, default [time.Now].
func WithNow(fn func() time.Time) Option {
	return func(o *option) { o.Now = fn }
}

// WithErrStack sets the function to extract a [stack.StackTrace] from an error, default reading [errors.StackTrace].
func WithErrStack(fn func(err internal.Error) stack.StackTrace) Option {
	return func(o *option) { o.ErrStack = fn }
}

// WithHandlerErr sets the handler-error callback, default writing to [os.Stderr].
func WithHandlerErr(fn func(err internal.Error, rec Record)) Option {
	return func(o *option) { o.HandlerErr = fn }
}
