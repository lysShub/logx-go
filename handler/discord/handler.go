package discord

import "github.com/lysShub/logx-go/handler"

// New returns a handler that discards all records, it is [handler.Discard].
func New() handler.Handler { return handler.Discard{} }
