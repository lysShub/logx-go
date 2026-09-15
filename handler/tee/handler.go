package tee

import (
	"context"
	"errors"
	"log/slog"

	"github.com/lysShub/logx-go"
	"github.com/lysShub/logx-go/handler"
)

type tee struct {
	hs []logx.Handler
} //
var _ logx.Handler = (*tee)(nil)

func New(hs ...logx.Handler) *tee { return &tee{hs: hs} }

func (t *tee) Enabled(ctx context.Context, l logx.Level) bool {
	for _, h := range t.hs {
		if h.Enabled(ctx, l) {
			return true
		}
	}
	return false
}
func (t *tee) Handle(ctx context.Context, r logx.Record) error {
	var errs []error
	for _, h := range t.hs {
		if e := h.Handle(ctx, r.Clone()); e != nil {
			errs = append(errs, e)
		}
	}
	return errors.Join(errs...)
}
func (t *tee) WithAttrs(attrs []logx.Attr) logx.Handler {
	var hs = make([]logx.Handler, len(t.hs))
	for i, h := range t.hs {
		hs[i] = h.WithAttrs(attrs)
	}
	return &tee{hs: hs}
}
func (t *tee) WithGroup(name string) logx.Handler {
	var hs = make([]logx.Handler, len(t.hs))
	for i, h := range t.hs {
		hs[i] = h.WithGroup(name)
	}
	return &tee{hs: hs}
}
func (t *tee) Slog() slog.Handler { return &handler.WrapHandler{Handler: t} }
func (t *tee) Sync(ctx context.Context) error {
	var errs []error
	for _, h := range t.hs {
		if e := h.Sync(ctx); e != nil {
			errs = append(errs, e)
		}
	}
	return errors.Join(errs...)
}
func (t *tee) Close() error { return nil }
