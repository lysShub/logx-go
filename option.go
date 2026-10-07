package logx

import (
	"fmt"
	"os"
	"time"

	"github.com/lysShub/debug-go"
	"github.com/lysShub/errorx-go"
	"github.com/lysShub/logx-go/stack"
)

type option struct {
	// StackLevel log with stack trace when level >= StackLevel
	StackLevel Level
	// HandlerErr process handler-error
	HandlerErr func(err error, rec Record)
	// Now time now
	Now func() time.Time
	// ErrStack get [stack.StackTrace] from error
	ErrStack func(err error) stack.Stack
}
type Option func(*option)

var defaultOption = option{
	StackLevel: func() Level {
		if debug.Debug() {
			return LevelWarn
		} else {
			return LevelError
		}
	}(),
	HandlerErr: func(err error, rec Record) {
		fmt.Fprintf(os.Stderr, "logx handle falie: %+v\n", err)
	},
	Now: time.Now,
	ErrStack: func(err error) stack.Stack {
		if s := errorx.T[errorx.Stack](err); s != nil {
			return stack.Stack(s)
		}
		return nil
	},
}

// WithStackLevel sets the minimum level to attach a stack trace, default [LevelWarn].
func WithStackLevel(level Level) Option {
	return func(o *option) { o.StackLevel = level }
}

// WithNow sets the time source, default [time.Now].
func WithNow(fn func() time.Time) Option {
	return func(o *option) { o.Now = fn }
}

// WithErrStack sets the function to extract a [stack.Stack] from an error, default reading [errors.StackTrace].
func WithErrStack(fn func(err error) stack.Stack) Option {
	return func(o *option) { o.ErrStack = fn }
}

// WithHandlerErr sets the handler-error callback, default writing to [os.Stderr].
func WithHandlerErr(fn func(err error, rec Record)) Option {
	return func(o *option) { o.HandlerErr = fn }
}
