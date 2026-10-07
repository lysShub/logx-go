package logx_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/lysShub/logx-go"
	"github.com/lysShub/logx-go/handler"
	"github.com/lysShub/logx-go/stack"
)

type counter struct {
	last   slog.Record
	n      int
	err    error
	synced bool
	closed bool
}

func (c *counter) Enabled(context.Context, slog.Level) bool      { return true }
func (c *counter) Handle(_ context.Context, r slog.Record) error { c.last = r; c.n++; return c.err }
func (c *counter) WithAttrs(...slog.Attr) handler.Handler        { return c }
func (c *counter) WithGroup(string) handler.Handler              { return c }
func (c *counter) Slog() slog.Handler                            { return &handler.WrapHandler{Handler: c} }
func (c *counter) Sync() error                                   { c.synced = true; return nil }
func (c *counter) Close() error                                  { c.closed = true; return nil }

func (c *counter) stack() stack.Stack {
	var s stack.Stack
	c.last.Attrs(func(a slog.Attr) bool {
		if a.Key == stack.StackKey {
			s, _ = a.Value.Any().(stack.Stack)
		}
		return true
	})
	return s
}

func Test_Stack(t *testing.T) {
	t.Run("warn", func(t *testing.T) {
		c := &counter{}
		logx.New(c, logx.WithStackLevel(logx.LevelWarn)).Warn("boom", logx.Err(errors.New("boom")))
		if c.stack() == nil {
			t.Fatal("warn should attach a stack when StackLevel <= warn, got nil")
		}
	})

	t.Run("error", func(t *testing.T) {
		c := &counter{}
		logx.New(c).Error("boom", logx.Err(errors.New("boom")))
		if c.stack() == nil {
			t.Fatal("error should attach a captured stack, got nil")
		}
	})

	t.Run("info", func(t *testing.T) {
		c := &counter{}
		logx.New(c).Info("hi")
		if c.stack() != nil {
			t.Fatal("info should not attach a stack by default")
		}
	})

	t.Run("stack_level", func(t *testing.T) {
		c := &counter{}
		logx.New(c, logx.WithStackLevel(logx.LevelDebug)).Info("hi")
		if c.stack() == nil {
			t.Fatal("info should attach a stack when StackLevel is debug")
		}
	})
}

func Test_Message(t *testing.T) {
	t.Run("info", func(t *testing.T) {
		c := &counter{}
		logx.New(c).Info("hi")
		if c.last.Message != "hi" {
			t.Fatalf("message got %q, want %q", c.last.Message, "hi")
		}
	})

	t.Run("error", func(t *testing.T) {
		c := &counter{}
		logx.New(c).Error("kaboom")
		if c.last.Message != "kaboom" {
			t.Fatalf("message got %q, want %q", c.last.Message, "kaboom")
		}
	})
}

func Test_ErrAttr(t *testing.T) {
	c := &counter{}
	logx.New(c).Error("boom", logx.Err(errors.New("kaboom")))

	var got error
	c.last.Attrs(func(a slog.Attr) bool {
		if a.Key == logx.ErrorKey {
			got, _ = a.Value.Any().(error)
		}
		return true
	})
	if got == nil || got.Error() != "kaboom" {
		t.Fatalf("err attr got %v, want kaboom", got)
	}
}

func Test_HandlerErr(t *testing.T) {
	c := &counter{err: errors.New("handler failed")}
	var got error
	l := logx.New(c, logx.WithHandlerErr(func(err error, _ logx.Record) { got = err }))

	l.Info("hi")
	if got == nil {
		t.Fatal("HandlerErr should be called when the handler returns an error")
	}
}

func Test_SyncClose(t *testing.T) {
	c := &counter{}
	l := logx.New(c)

	if err := l.Sync(); err != nil {
		t.Fatal(err)
	}
	if !c.synced {
		t.Fatal("Sync not forwarded to handler")
	}

	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if !c.closed {
		t.Fatal("Close not forwarded to handler")
	}
}

func Test_WithAttrsGroup(t *testing.T) {
	c := &counter{}
	logx.New(c).WithAttrs(logx.String("k", "v")).WithGroup("g").Info("hi")
	if c.n != 1 {
		t.Fatalf("handled %d times, want 1", c.n)
	}
}

func Test_DefaultLogger(t *testing.T) {
	c := &counter{}
	old := logx.SetDefault(logx.New(c))
	defer logx.SetDefault(old)

	logx.Warn("boom")
	if c.n != 1 {
		t.Fatalf("default logger handled %d times, want 1", c.n)
	}
}
