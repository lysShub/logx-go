package logx

import (
	"log/slog"

	"github.com/lysShub/logx-go/stack"
)

func Stack() Attr {
	return Attr{Key: stack.StackKey, Value: slog.AnyValue(stack.New(stack.Trace, 3))}
}
