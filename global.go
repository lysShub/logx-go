package logx

import (
	"log/slog"
	"reflect"
	"sync/atomic"

	"github.com/lysShub/logx-go/handler/discard"
	"github.com/lysShub/logx-go/stack"
)

var (
	_             = SetDefault(New(discard.Discard{}))
	defaultLogger atomic.Pointer[Logger]
	errorType     = reflect.TypeFor[error]()
)

func Default() *Logger {
	return defaultLogger.Load()
}
func SetDefault(l *Logger) (old *Logger) {
	old = defaultLogger.Swap(l)
	slog.SetDefault(slog.New(l.h.Slog()))
	return old
}

func Stack() Attr {
	return Attr{Key: stack.StackKey, Value: AnyValue(stack.New(3))}
}
