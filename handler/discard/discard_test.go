package discard_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/lysShub/logx-go/handler/discard"
)

func Test_Discard(t *testing.T) {
	d := discard.New()
	if d.Enabled(context.Background(), slog.LevelError) {
		t.Fatal("discard should be disabled")
	}
	if err := d.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "m", 0)); err != nil {
		t.Fatal(err)
	}
	if d.WithAttrs(slog.String("k", "v")) == nil {
		t.Fatal("WithAttrs returned nil")
	}
	if d.WithGroup("g") == nil {
		t.Fatal("WithGroup returned nil")
	}
	if d.Slog() == nil {
		t.Fatal("Slog returned nil")
	}
	if err := d.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
}
