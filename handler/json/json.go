package json

import (
	"context"
	"log/slog"
	"time"

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

// WithTimeValue sets the format of log time, default [handler.RFC3339Millis] "2006-01-02T15:04:05.000Z07:00".
func WithTimeValue(f handler.TimeValue) Option {
	return func(o *handler.Options) { o.TimeValue = f }
}

// WithLevelValue sets the level to string mapping, default {debug, info, warn, error, fatal}.
func WithLevelValue(f handler.LevelValue) Option {
	return func(o *handler.Options) { o.LevelValue = f }
}

// WithLeveler sets the minimum log level to output, default [slog.LevelInfo].
func WithLeveler(level slog.Leveler, max ...slog.Level) Option {
	return func(o *handler.Options) {
		o.Level = level
		if len(max) > 0 {
			o.MaxLevel = max[0]
		}
	}
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
			//todo: validate years are 4 digits
			t := a.Value.Time().In(h.c.TimeLocation)
			a.Value = h.c.TimeValue(t)
		case slog.LevelKey:
			l, ok := a.Value.Any().(slog.Level)
			if ok {
				a.Value = h.c.LevelValue(l)
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
	return h.c.Level.Level() <= l && l < h.c.MaxLevel.Level()
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
func (h *json) Sync() error  { return h.w.Sync() }
func (h *json) Close() error { return h.w.Close() }

func (h *json) Handle(ctx context.Context, r slog.Record) error {
	if err := h.h.Handle(ctx, r); err != nil {
		return err
	}
	if r.Level >= h.c.SyncLevel {
		return h.w.Sync()
	}
	return nil
}
