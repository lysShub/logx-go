package json

import (
	"context"
	"log/slog"
	"time"
	"unsafe"

	"github.com/lysShub/logx-go/handler"
	"github.com/lysShub/logx-go/writer"
)

type json struct {
	c *config
	w writer.Writer
	h slog.Handler
} //
var _ handler.Handler = (*json)(nil)

type config = handler.Options
type Option = func(*handler.Options)

// WithTimeLocation sets the time zone of log time, default [time.Local].
func WithTimeLocation(loc *time.Location) Option {
	return func(o *handler.Options) { o.TimeLocation = loc }
}

// WithTimeFormat sets the format of log time, default [handler.RFC3339Millis] "2006-01-02T15:04:05.000Z07:00".
func WithTimeFormat(format string) Option {
	return func(o *handler.Options) { o.TimeFormat = format }
}

// WithLevelString sets the level to string mapping, default {debug, info, warn, error, fatal}.
func WithLevelString(m map[slog.Level]string) Option {
	return func(o *handler.Options) { o.LevelString = m }
}

// WithLeveler sets the minimum log level to output, default [slog.LevelInfo].
func WithLeveler(level slog.Leveler) Option {
	return func(o *handler.Options) { o.Level = level }
}

// WithSyncLevel sets the level that triggers auto Sync when a record.Level >= SyncLevel, default [slog.LevelError].
func WithSyncLevel(level slog.Level) Option {
	return func(o *handler.Options) { o.SyncLevel = level }
}

// WithReplace sets a custom function to replace log attrs, default none.
func WithReplace(fn func(groups []string, a slog.Attr) slog.Attr) Option {
	return func(o *handler.Options) { o.ReplaceAttr = fn }
}

func NewJSON(w writer.Writer, opts ...Option) *json {
	var h = &json{
		c: &config{},
		w: w,
	}
	for _, e := range opts {
		e(h.c)
	}
	h.c.Init()

	h.h = slog.NewJSONHandler(w, &slog.HandlerOptions{
		ReplaceAttr: h.replaceAttr,
	})
	return h
}
func (h *json) replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) == 0 {
		switch a.Key {
		case slog.TimeKey:
			b := handler.AppendTime(make([]byte, 0, 32), a.Value.Time().In(h.c.TimeLocation), h.c.TimeFormat)
			str := unsafe.String(unsafe.SliceData(b), len(b))
			a.Value = slog.StringValue(str)
		case slog.LevelKey:
			l, ok := a.Value.Any().(slog.Level)
			if ok {
				if s, exist := h.c.LevelString[l]; exist {
					a.Value = slog.StringValue(s)
				} else {
					a.Value = slog.StringValue(l.String())
				}
			}
		default:
		}
	}
	if h.c.ReplaceAttr != nil {
		return h.c.ReplaceAttr(groups, a)
	}
	return a
}

func (h *json) Slog() slog.Handler { return &handler.WrapHandler{Handler: h} }
func (h *json) Enabled(ctx context.Context, l slog.Level) bool {
	return h.h.Enabled(ctx, l)
}
func (h *json) WithAttrs(attrs []slog.Attr) handler.Handler {
	return &json{
		c: h.c,
		w: h.w,
		h: h.h.WithAttrs(attrs),
	}
}
func (h *json) WithGroup(name string) handler.Handler {
	return &json{
		c: h.c,
		w: h.w,
		h: h.h.WithGroup(name),
	}
}

func (h *json) Handle(ctx context.Context, r slog.Record) error {
	return h.h.Handle(ctx, r)
}

func (h *json) Sync(ctx context.Context) error {
	return h.w.Sync(ctx)
}

func (t *json) Close() error { return nil }
