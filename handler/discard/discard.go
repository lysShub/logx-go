package discard

import (
	"context"
	"log/slog"

	"github.com/lysShub/logx-go/handler"
)

func New() handler.Handler { return Discard{} }

type Discard struct{} //
var _ handler.Handler = Discard{}

func (Discard) Enabled(context.Context, slog.Level) bool  { return false }
func (Discard) Handle(context.Context, slog.Record) error { return nil }
func (Discard) WithAttrs([]slog.Attr) handler.Handler     { return Discard{} }
func (Discard) WithGroup(string) handler.Handler          { return Discard{} }
func (Discard) Slog() slog.Handler                        { return &handler.WrapHandler{Handler: Discard{}} }
func (Discard) Sync() error                               { return nil }
func (Discard) Close() error                              { return nil }
