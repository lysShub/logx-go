package logx

import (
	"context"
	"os"
	"reflect"
	"syscall"
	"unsafe"

	"github.com/lysShub/logx-go/handler"
	"github.com/lysShub/logx-go/stack"
)

func Log(level Level, msg string, attrs ...Attr) { defaultLogger.Load().log(level, msg, nil, attrs...) }
func Debug(msg string, attrs ...Attr)            { defaultLogger.Load().Debug(msg, attrs...) }
func Info(msg string, attrs ...Attr)             { defaultLogger.Load().Info(msg, attrs...) }
func Warn[T str | err](msg T, attrs ...Attr)     { defaultLogger.Load().Warn(msg, attrs...) }
func Error[T str | err](msg T, attrs ...Attr)    { defaultLogger.Load().Error(msg, attrs...) }
func Fatal[T str | err](msg T, attrs ...Attr)    { defaultLogger.Load().Fatal(msg, attrs...) }

type Logger struct {
	h handler.Handler
	o option
}
type str = string
type err = any

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
	return &Logger{h: l.h.WithAttrs(attrs), o: l.o}
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
func (l *Logger) Warn[T str | err](msg T, attrs ...Attr) {
	l.logerr(LevelWarn, msg, attrs...)
}
func (l *Logger) Error[T str | err](msg T, attrs ...Attr) {
	l.logerr(LevelError, msg, attrs...)
}
func (l *Logger) Fatal[T str | err](msg T, attrs ...Attr) {
	l.logerr(LevelFatal, msg, attrs...)
}

func (l *Logger) logmsg(level Level, msg string, attrs ...Attr) {
	if !l.h.Enabled(context.Background(), level) {
		return
	}
	l.log(level, msg, nil, attrs...)
}

func (l *Logger) logerr[T str | err](level Level, err T, attrs ...Attr) {
	if !l.h.Enabled(context.Background(), level) {
		return
	}
	s, e := specialize(err)

	var st stack.Stack
	if e != nil && level >= l.o.StackLevel {
		// 错误现场 替代 日志现场, 通常是相近的
		st = l.o.ErrStack(e)
	}
	l.log(level, s, st, attrs...)
}
func specialize[T str | err](v T) (msg string, err error) {
	t := reflect.TypeFor[T]()
	if t.Kind() == reflect.String {
		msg, err = *(*string)(unsafe.Pointer(&v)), nil
	} else {
		if t == errorType {
			msg, err = "", *(*error)(unsafe.Pointer(&v))
		} else if e1, ok := any(v).(error); ok {
			msg, err = "", e1
		} else {
			panic("not support type")
		}
		msg = err.Error()
	}
	return msg, err
}

func (l *Logger) log(level Level, msg string, st stack.Stack, attrs ...Attr) {
	rec := NewRecord(l.o.Now(), level, msg, 0)
	rec.AddAttrs(attrs...)

	if level >= l.o.StackLevel {
		if len(st) == 0 {
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
