package discord

import (
	"context"
	"log/slog"

	"github.com/lysShub/logx-go"
	"github.com/lysShub/logx-go/handler"
	"github.com/lysShub/logx-go/internal"
)

type discard struct{} //
var _ logx.Handler = discard{}

func New() discard { return discard{} }

func (discard) Enabled(context.Context, logx.Level) bool           { return true }
func (discard) Handle(context.Context, logx.Record) internal.Error { return nil }
func (discard) WithAttrs([]logx.Attr) logx.Handler                 { return discard{} }
func (discard) WithGroup(string) logx.Handler                      { return discard{} }
func (discard) Slog() slog.Handler                                 { return &handler.WrapHandler{Handler: discard{}} }
func (discard) Sync(context.Context) error                         { return nil }
func (discard) Close() error                                       { return nil }
