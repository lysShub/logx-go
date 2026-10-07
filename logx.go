// github.com/lysShub/logx-go

package logx

import (
	"context"
	"log/slog"
	"os"
	"syscall"

	"github.com/lysShub/errorx-go"
	"github.com/lysShub/logx-go/handler"
	"github.com/lysShub/logx-go/stack"
)

func Log(level Level, msg string, attrs ...Attr) { defaultLogger.Load().log(level, msg, nil, attrs...) }
func Debug(msg string, attrs ...Attr)            { defaultLogger.Load().Debug(msg, attrs...) }
func Info(msg string, attrs ...Attr)             { defaultLogger.Load().Info(msg, attrs...) }
func Warn(err error, attrs ...Attr)              { defaultLogger.Load().Warn(err, attrs...) }
func Error(err error, attrs ...Attr)             { defaultLogger.Load().Error(err, attrs...) }
func Fatal(err error, attrs ...Attr)             { defaultLogger.Load().Fatal(err, attrs...) }

func WarnMsg(msg string, attrs ...Attr)  { defaultLogger.Load().Warn(errorx.StringErr(msg), attrs...) }
func ErrorMsg(msg string, attrs ...Attr) { defaultLogger.Load().Error(errorx.StringErr(msg), attrs...) }
func FatalMsg(msg string, attrs ...Attr) { defaultLogger.Load().Fatal(errorx.StringErr(msg), attrs...) }

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
	l.logmsg(level, msg, attrs...)
}
func (l *Logger) Debug(msg string, attrs ...Attr) {
	l.logmsg(LevelDebug, msg, attrs...)
}
func (l *Logger) Info(msg string, attrs ...Attr) {
	l.logmsg(LevelInfo, msg, attrs...)
}
func (l *Logger) Warn(err error, attrs ...Attr) {
	l.logerr(LevelWarn, err, attrs...)
}
func (l *Logger) Error(err error, attrs ...Attr) {
	l.logerr(LevelError, err, attrs...)
}
func (l *Logger) Fatal(err error, attrs ...Attr) {
	l.logerr(LevelFatal, err, attrs...)
}

func (l *Logger) logmsg(level Level, msg string, attrs ...Attr) {
	if !l.h.Enabled(context.Background(), level) {
		return
	}
	l.log(level, msg, nil, attrs...)
}

func (l *Logger) logerr(level Level, err error, attrs ...Attr) {
	if !l.h.Enabled(context.Background(), level) {
		return
	}
	msg := err.Error()

	var (
		bak  Attr
		baki int = -1
	)
	for i, attr := range attrs {
		if attr.Key == MessageKey && attr.Value.Kind() == KindString {
			attrs[i] = slog.String(ErrorKey, msg)
			msg = attr.Value.String()

			baki, bak = i, attr
			break
		}
	}

	var st stack.Stack
	if level >= l.o.StackLevel {
		// 错误现场 替代 日志现场, 通常是相近的
		st = l.o.ErrStack(err)
	}
	l.log(level, msg, st, attrs...)

	if baki >= 0 {
		attrs[baki] = bak
	}
}

func (l *Logger) log(level Level, msg string, st stack.Stack, attrs ...Attr) {
	rec := NewRecord(l.o.Now(), level, msg, 0)
	rec.AddAttrs(attrs...)

	if level >= l.o.StackLevel {
		if st == nil {
			st = stack.New(5)
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
