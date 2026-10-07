package tee_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/lysShub/logx-go/handler"
	"github.com/lysShub/logx-go/handler/tee"
)

type counter struct {
	n      int
	off    bool
	closed bool
	synced bool
}

func (c *counter) Enabled(context.Context, slog.Level) bool  { return !c.off }
func (c *counter) Handle(context.Context, slog.Record) error { c.n++; return nil }
func (c *counter) WithAttrs(...slog.Attr) handler.Handler    { return c }
func (c *counter) WithGroup(string) handler.Handler          { return c }
func (c *counter) Slog() slog.Handler                        { return &handler.WrapHandler{Handler: c} }
func (c *counter) Sync() error                               { c.synced = true; return nil }
func (c *counter) Close() error                              { c.closed = true; return nil }

func Test_Handle(t *testing.T) {
	a, b := &counter{}, &counter{off: true}
	d := tee.Tee(a, b)

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "m", 0)
	if err := d.Handle(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if a.n != 1 || b.n != 0 {
		t.Fatalf("handled a=%d b=%d, want 1,0", a.n, b.n)
	}
}

func Test_Enabled(t *testing.T) {
	if !tee.Tee(&counter{off: true}, &counter{}).Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("want enabled when any child is enabled")
	}
	if tee.Tee(&counter{off: true}, &counter{off: true}).Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("want disabled when all children are disabled")
	}
}

func Test_SyncClose(t *testing.T) {
	a, b := &counter{}, &counter{}
	d := tee.Tee(a, b)

	if err := d.Sync(); err != nil {
		t.Fatal(err)
	}
	if !a.synced || !b.synced {
		t.Fatalf("synced a=%v b=%v, want true,true", a.synced, b.synced)
	}

	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if !a.closed || !b.closed {
		t.Fatalf("closed a=%v b=%v, want true,true", a.closed, b.closed)
	}
}
