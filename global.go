package logx

import (
	"log/slog"
	"sync/atomic"

	"github.com/lysShub/logx-go/handler/json"
	"github.com/lysShub/logx-go/stack"
	"github.com/lysShub/logx-go/writer"
)

var (
	_             = SetDefault(New(json.NewJSON(writer.Stdout())))
	defaultLogger atomic.Pointer[Logger]
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
