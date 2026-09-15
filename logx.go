package logx

import (
	"context"
	"io"
	"log/slog"
	"os"
	"reflect"
	"syscall"
	"unsafe"

	"github.com/lysShub/logx-go/internal"
	"github.com/lysShub/logx-go/writer"
	"github.com/pkg/errors"
)

func Log(level Level, msg string, attrs ...Attr)   { defaultLogger.Load().log(level, msg, nil, attrs...) }
func Debug(msg string, attrs ...Attr)              { defaultLogger.Load().Debug(msg, attrs...) }
func Info(msg string, attrs ...Attr)               { defaultLogger.Load().Info(msg, attrs...) }
func Warn[T string | error](msg T, attrs ...Attr)  { defaultLogger.Load().Warn(msg, attrs...) }
func Error[T string | error](msg T, attrs ...Attr) { defaultLogger.Load().Error(msg, attrs...) }
func Fatal[T string | error](msg T, attrs ...Attr) { defaultLogger.Load().Fatal(msg, attrs...) }

// reference [slog.Handler]
type Handler interface {
	Enabled(context.Context, Level) bool
	Handle(context.Context, Record) internal.Error
	WithAttrs(attrs []Attr) Handler
	WithGroup(name string) Handler
	Slog() slog.Handler
	writer.Syncer
	io.Closer
}

type Logger struct {
	h Handler
	o option
}
type error = any

func New(h Handler, opts ...Option) *Logger {
	l := &Logger{
		h: h,
		o: defaultOption,
	}
	for _, e := range opts {
		e(&l.o)
	}
	return l
}
func (l *Logger) Close() error                   { return l.h.Close() }
func (l *Logger) Sync(ctx context.Context) error { return l.h.Sync(ctx) }
func (l *Logger) Handler() Handler               { return l.h }
func (l *Logger) Enabled(level Level) bool {
	return l.h.Enabled(context.Background(), level)
}
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{h: l.h.WithGroup(name)}
}
func (l *Logger) WithAttrs(attrs ...slog.Attr) *Logger {
	return &Logger{h: l.h.WithAttrs(attrs)}
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
func (l *Logger) Warn[T string | error](msg T, attrs ...Attr) {
	l.logerr(LevelWarn, msg, attrs...)
}
func (l *Logger) Error[T string | error](msg T, attrs ...Attr) {
	l.logerr(LevelError, msg, attrs...)
}
func (l *Logger) Fatal[T string | error](msg T, attrs ...Attr) {
	l.logerr(LevelFatal, msg, attrs...)
}

var errIface = reflect.TypeOf(errors.New(""))

func (l *Logger) logmsg(level Level, msg string, attrs ...Attr) {
	l.log(level, msg, nil, attrs...)
}
func (l *Logger) logerr[T string | error](level Level, err T, attrs ...Attr) {
	t := reflect.TypeFor[T]()
	if k := t.Kind(); k == reflect.String {
		l.log(level, *(*string)(unsafe.Pointer(&err)), nil, attrs...)
	} else if k == reflect.Interface {
		var e internal.Error
		if t == errIface {
			e = *(*internal.Error)(unsafe.Pointer(&err))
		} else if t.Implements(errIface) {
			e = any(err).(internal.Error)
		} else {
			panic("not support type")
		}

		var st StackTrace
		if l.o.StackKind == Trace && level >= l.o.StackLevel {
			// 错误现场 替代 日志现场, 通常是相近的
			st = l.o.ErrStack(e)
		}
		l.log(level, e.Error(), st, attrs...)
	} else {
		panic("not support type")
	}
}
func (l *Logger) log(level Level, msg string, stack StackTrace, attrs ...Attr) {
	rec := NewRecord(l.o.Now(), level, msg, 0)
	rec.AddAttrs(attrs...)

	if level >= l.o.StackLevel {
		if len(stack) == 0 {
			switch l.o.StackKind {
			case Source:
				stack = newStack(Source)
			case Trace:
				stack = newStack(Trace)
			default:
				panic(l.o.StackKind)
			}
		}
		rec.AddAttrs(Attr{Key: StackKey, Value: slog.AnyValue(stack)})
	}

	h := Default().Handler()
	if err := h.Handle(context.Background(), rec); err != nil {
		l.o.HandlerErr(err, rec)
	}

	if level >= LevelFatal {
		os.Exit(int(syscall.SIGQUIT))
	}
}
