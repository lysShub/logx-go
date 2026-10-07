package tee

import (
	"context"
	"log/slog"

	"github.com/lysShub/logx-go/handler"
)

type tee struct {
	hs []handler.Handler
} //
var _ handler.Handler = (*tee)(nil)

func Tee(hs ...handler.Handler) *tee { return &tee{hs: hs} }

func (t *tee) Enabled(ctx context.Context, l slog.Level) bool {
	for _, h := range t.hs {
		if h.Enabled(ctx, l) {
			return true
		}
	}
	return false
}
func (t *tee) Handle(ctx context.Context, r slog.Record) error {
	var err error
	for _, h := range t.hs {
		if !h.Enabled(ctx, r.Level) {
			continue
		}
		if e := h.Handle(ctx, r.Clone()); e != nil && err == nil {
			err = e
		}
	}
	return err
}
func (t *tee) WithAttrs(attrs ...slog.Attr) handler.Handler {
	var hs = make([]handler.Handler, len(t.hs))
	for i, h := range t.hs {
		hs[i] = h.WithAttrs(attrs...)
	}
	return &tee{hs: hs}
}
func (t *tee) WithGroup(name string) handler.Handler {
	var hs = make([]handler.Handler, len(t.hs))
	for i, h := range t.hs {
		hs[i] = h.WithGroup(name)
	}
	return &tee{hs: hs}
}
func (t *tee) Slog() slog.Handler { return &handler.WrapHandler{Handler: t} }
func (t *tee) Sync() error {
	var err error
	for _, h := range t.hs {
		if e := h.Sync(); e != nil && err == nil {
			err = e
		}
	}
	return err
}
func (t *tee) Close() error {
	var err error
	for _, h := range t.hs {
		if e := h.Close(); e != nil && err == nil {
			err = e
		}
	}
	return err
}
