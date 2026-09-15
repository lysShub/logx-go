package logx

import (
	"log/slog"
	"sync/atomic"

	"github.com/lysShub/logx-go/handler"
)

var (
	_             = SetDefault(New(handler.Discard{}))
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
