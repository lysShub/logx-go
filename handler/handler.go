package handler

import (
	"context"
	"io"
	"log/slog"
	"math"
	"time"

	"github.com/lysShub/logx-go/writer"
)

// reference [slog.Handler]
type Handler interface {
	Enabled(context.Context, slog.Level) bool
	Handle(context.Context, slog.Record) error
	WithAttrs(attrs []slog.Attr) Handler
	WithGroup(name string) Handler
	Slog() slog.Handler
	writer.Syncer
	io.Closer
}

// common Options
type Options struct {
	TimeLocation *time.Location                               // default [time.Local]
	TimeValue    TimeValue                                    // default [RFC3339Millis]
	LevelValue   LevelValue                                   // default [LevelString]
	SyncLevel    slog.Level                                   // default [slog.LevelError], while record.Level >= SyncLevel, auto call h.Sync()
	Level        slog.Leveler                                 // default [slog.LevelInfo]
	MaxLevel     slog.Leveler                                 // default [math.MaxInt], range in [Level, MaxLevel)
	ReplaceAttr  func(groups []string, a slog.Attr) slog.Attr //
}
type TimeValue func(t time.Time) slog.Value
type LevelValue func(l slog.Level) slog.Value

func (o *Options) Init() {
	if o.TimeLocation == nil {
		o.TimeLocation = time.Local
	}
	if o.TimeValue == nil {
		o.TimeValue = RFC3339Millis
	}
	if o.LevelValue == nil {
		o.LevelValue = LevelString
	}
	if o.SyncLevel == 0 {
		o.SyncLevel = slog.LevelError
	}
	if o.Level == nil {
		o.Level = slog.Level(0)
	}
	if o.MaxLevel == nil {
		o.MaxLevel = slog.Level(math.MaxInt)
	}
}

func LevelString(l slog.Level) slog.Value {
	i := int(l + (-slog.LevelDebug))
	if i >= 0 && int(i) < len(levelString) && levelString[i] != "" {
		return slog.StringValue(levelString[i])
	} else {
		return slog.StringValue(l.String())
	}
} //
var levelString = [-slog.LevelDebug + slog.LevelError + 5]string{
	-slog.LevelDebug + slog.LevelDebug:     "debug",
	-slog.LevelDebug + slog.LevelInfo:      "info",
	-slog.LevelDebug + slog.LevelWarn:      "warn",
	-slog.LevelDebug + slog.LevelError:     "error",
	-slog.LevelDebug + slog.LevelError + 4: "fatal",
}

// RFC3339Millis marshal to RFC3339Millis string
func RFC3339Millis(t time.Time) slog.Value {
	var b = make([]byte, 0, 24)
	b = appendRFC3339Millis(b, t)
	return slog.StringValue(string(b))
}
func appendRFC3339Millis(b []byte, t time.Time) []byte {
	// copy from [slog.appendRFC3339Millis]
	//
	// Format according to time.RFC3339Nano since it is highly optimized,
	// but truncate it to use millisecond resolution.
	// Unfortunately, that format trims trailing 0s, so add 1/10 millisecond
	// to guarantee that there are exactly 4 digits after the period.
	const prefixLen = len("2006-01-02T15:04:05.000")
	n := len(b)
	t = t.Truncate(time.Millisecond).Add(time.Millisecond / 10)
	b = t.AppendFormat(b, time.RFC3339Nano)
	b = append(b[:n+prefixLen], b[n+prefixLen+1:]...) // drop the 4th digit
	return b
}

type WrapHandler struct{ Handler }       //
var _ slog.Handler = (*WrapHandler)(nil) //
func (h *WrapHandler) WithGroup(name string) slog.Handler {
	return &WrapHandler{h.Handler.WithGroup(name)}
}
func (h *WrapHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &WrapHandler{h.Handler.WithAttrs(attrs)}
}
