package logx

import (
	"log/slog"
	"os"
	"testing"
)

func TestXxxx(t *testing.T) {

	slog.SetDefault(
		slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})),
	)

	slog.Error("xxx", slog.Attr{"", slog.GroupValue()})

}
