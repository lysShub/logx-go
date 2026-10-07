// github.com/lysShub/logx-go

package logx

import (
	"context"
	"os"
	"syscall"

	"github.com/lysShub/logx-go/handler"
	"github.com/lysShub/logx-go/stack"
)

func Log(level Level, msg string, attrs ...Attr) { defaultLogger.Load().log(level, msg, attrs...) }
func Debug(msg string, attrs ...Attr)            { defaultLogger.Load().Debug(msg, attrs...) }
func Info(msg string, attrs ...Attr)             { defaultLogger.Load().Info(msg, attrs...) }
func Warn(msg string, attrs ...Attr)             { defaultLogger.Load().Warn(msg, attrs...) }
func Error(msg string, attrs ...Attr)            { defaultLogger.Load().Error(msg, attrs...) }
func Fatal(msg string, attrs ...Attr)            { defaultLogger.Load().Fatal(msg, attrs...) }

type Logger struct {
	h handler.Handler
	o option
}

func New(h handler.Handler, opts ...Option) *Logger {
	l := &Logger{
		h: h,
		o: defaultOption,
	}
	for _, e := range opts {
		e(&l.o)
	}
	return l
}
func (l *Logger) Close() error {
	if err := l.h.Sync(); err != nil {
		return err
	}
	return l.h.Close()
}
func (l *Logger) Sync() error              { return l.h.Sync() }
func (l *Logger) Handler() handler.Handler { return l.h }
func (l *Logger) Enabled(level Level) bool {
	return l.h.Enabled(context.Background(), level)
}
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{h: l.h.WithGroup(name), o: l.o}
}
func (l *Logger) WithAttrs(attrs ...Attr) *Logger {
	return &Logger{h: l.h.WithAttrs(attrs...), o: l.o}
}

func (l *Logger) Log(level Level, msg string, attrs ...Attr) {
	l.log(level, msg, attrs...)
}
func (l *Logger) Debug(msg string, attrs ...Attr) {
	l.log(LevelDebug, msg, attrs...)
}
func (l *Logger) Info(msg string, attrs ...Attr) {
	l.log(LevelInfo, msg, attrs...)
}
func (l *Logger) Warn(msg string, attrs ...Attr) {
	l.log(LevelWarn, msg, attrs...)
}
func (l *Logger) Error(msg string, attrs ...Attr) {
	l.log(LevelError, msg, attrs...)
}
func (l *Logger) Fatal(msg string, attrs ...Attr) {
	l.log(LevelFatal, msg, attrs...)
}

func (l *Logger) log(level Level, msg string, attrs ...Attr) {
	if !l.h.Enabled(context.Background(), level) {
		return
	}
	rec := NewRecord(l.o.Now(), level, msg, 0)
	rec.AddAttrs(attrs...)

	if level >= l.o.StackLevel {
		var st stack.Stack
		if len(attrs) > 0 && attrs[0].Key == ErrorKey && attrs[0].Value.Kind() == KindAny {
			if e, ok := attrs[0].Value.Any().(error); ok {
				st = l.o.ErrStack(e)
			}
		}
		if st == nil {
			st = stack.New(4)
		}
		rec.AddAttrs(Attr{Key: stack.StackKey, Value: AnyValue(st)})
	}

	if err := l.h.Handle(context.Background(), rec); err != nil {
		l.o.HandlerErr(err, rec)
	}

	if level >= LevelFatal {
		os.Exit(int(syscall.SIGQUIT))
	}
}
