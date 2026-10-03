package handler

import (
	"context"
	"io"
	"log/slog"
	"math"
	"strconv"
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
	TimeLocation    *time.Location                               // default [time.Local]
	TimeFormat      string                                       // default [RFC3339Millis]
	LevelString     map[slog.Level]string                        // default [LevelString]
	SyncLevel       slog.Level                                   // record.Level >= SyncLevel 时自动 Sync, default [slog.LevelError]
	Level, MaxLevel slog.Leveler                                 // [Level, MaxLevel)
	ReplaceAttr     func(groups []string, a slog.Attr) slog.Attr //
}

func (o *Options) Init() {
	if o.TimeLocation == nil {
		o.TimeLocation = time.Local
	}
	if o.TimeFormat == "" {
		o.TimeFormat = RFC3339Millis
	}
	if o.LevelString == nil {
		o.LevelString = LevelString
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

var LevelString = map[slog.Level]string{
	slog.LevelDebug:     "debug",
	slog.LevelInfo:      "info",
	slog.LevelWarn:      "warn",
	slog.LevelError:     "error",
	slog.LevelError + 4: "fatal",
}

const (
	RFC3339Millis   = "2006-01-02T15:04:05.000Z07:00"
	UnixFormat      = "unix"
	UnixMilliFormat = "unix_milli"
	UnixMicroFormat = "unix_micro"
	UnixNanoFormat  = "unix_nano"
)

func AppendTime(b []byte, t time.Time, format string) []byte {
	switch format {
	case RFC3339Millis:
		return appendRFC3339Millis(b, t)
	case UnixFormat:
		return strconv.AppendInt(b, t.Unix(), 10)
	case UnixMilliFormat:
		return strconv.AppendInt(b, t.UnixMilli(), 10)
	case UnixMicroFormat:
		return strconv.AppendInt(b, t.UnixMicro(), 10)
	case UnixNanoFormat:
		return strconv.AppendInt(b, t.UnixNano(), 10)
	default:
		return t.AppendFormat(b, format)
	}
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
