package dedup_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/lysShub/logx-go/handler"
	"github.com/lysShub/logx-go/handler/dedup"
	"github.com/lysShub/logx-go/stack"
)

type counter struct {
	n      int
	closed bool
}

func (c *counter) Enabled(context.Context, slog.Level) bool  { return true }
func (c *counter) Handle(context.Context, slog.Record) error { c.n++; return nil }
func (c *counter) WithAttrs([]slog.Attr) handler.Handler     { return c }
func (c *counter) WithGroup(string) handler.Handler          { return c }
func (c *counter) Slog() slog.Handler                        { return &handler.WrapHandler{Handler: c} }
func (c *counter) Sync() error                               { return nil }
func (c *counter) Close() error                              { c.closed = true; return nil }

func warn(msg string) slog.Record {
	return slog.NewRecord(time.Now(), slog.LevelWarn, msg, 0)
}

func Test_Duplicates(t *testing.T) {
	c := &counter{}
	d := dedup.New(c)
	ctx := context.Background()

	if err := d.Handle(ctx, warn("same")); err != nil {
		t.Fatal(err)
	}
	if err := d.Handle(ctx, warn("same")); err != nil {
		t.Fatal(err)
	}
	if c.n != 1 {
		t.Fatalf("duplicate warn: wrapped called %d times, want 1", c.n)
	}

	if err := d.Handle(ctx, warn("other")); err != nil {
		t.Fatal(err)
	}
	if c.n != 2 {
		t.Fatalf("different msg: wrapped called %d times, want 2", c.n)
	}

	if err := d.Handle(ctx, slog.NewRecord(time.Now(), slog.LevelInfo, "same", 0)); err != nil {
		t.Fatal(err)
	}
	if c.n != 3 {
		t.Fatalf("non-warn: wrapped called %d times, want 3", c.n)
	}
}

func Test_CustomHash(t *testing.T) {
	c := &counter{}
	d := dedup.New(c, dedup.WithHash(func(slog.Record) uint32 { return 1 }))
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := d.Handle(ctx, warn("anything")); err != nil {
			t.Fatal(err)
		}
	}
	if c.n != 1 {
		t.Fatalf("custom hash: wrapped called %d times, want 1", c.n)
	}
}

func Test_TTL(t *testing.T) {
	c := &counter{}
	d := dedup.New(c,
		dedup.WithHash(func(slog.Record) uint32 { return 1 }),
		dedup.WithTTL(0),
	)
	ctx := context.Background()

	if err := d.Handle(ctx, warn("m")); err != nil {
		t.Fatal(err)
	}
	if err := d.Handle(ctx, warn("m")); err != nil {
		t.Fatal(err)
	}
	if c.n != 2 {
		t.Fatalf("ttl=0: wrapped called %d times, want 2", c.n)
	}
}

func Test_Stack(t *testing.T) {
	c := &counter{}
	d := dedup.New(c)
	ctx := context.Background()

	var s1, s2 [32]uintptr
	s1[0] = 1
	s2[0] = 2

	r1 := warn("m")
	r1.AddAttrs(slog.Any(stack.StackKey, stack.Stack(&s1)))
	if err := d.Handle(ctx, r1); err != nil {
		t.Fatal(err)
	}
	if err := d.Handle(ctx, r1); err != nil {
		t.Fatal(err)
	}
	if c.n != 1 {
		t.Fatalf("same stack: wrapped called %d times, want 1", c.n)
	}

	r2 := warn("m")
	r2.AddAttrs(slog.Any(stack.StackKey, stack.Stack(&s2)))
	if err := d.Handle(ctx, r2); err != nil {
		t.Fatal(err)
	}
	if c.n != 2 {
		t.Fatalf("different stack: wrapped called %d times, want 2", c.n)
	}
}

func Test_Close(t *testing.T) {
	c := &counter{}
	d := dedup.New(c)
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if !c.closed {
		t.Fatal("wrapped handler not closed")
	}
}

func Test_Reuse(t *testing.T) {
	tests := []struct {
		name  string
		reuse bool
		want  int
	}{
		{"reuse", true, 1},
		{"no_reuse", false, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &counter{}
			d := dedup.New(c, dedup.WithReuse(tt.reuse))
			ctx := context.Background()

			if err := d.Handle(ctx, warn("x")); err != nil {
				t.Fatal(err)
			}
			if err := d.WithAttrs([]slog.Attr{slog.String("k", "v")}).Handle(ctx, warn("x")); err != nil {
				t.Fatal(err)
			}
			if c.n != tt.want {
				t.Fatalf("wrapped called %d times, want %d", c.n, tt.want)
			}
		})
	}
}

func Test_Derived(t *testing.T) {
	t.Run("attrs", func(t *testing.T) {
		c := &counter{}
		d := dedup.New(c).WithAttrs([]slog.Attr{slog.String("k", "v")})
		if err := d.Handle(context.Background(), warn("x")); err != nil {
			t.Fatal(err)
		}
		if err := d.Handle(context.Background(), warn("x")); err != nil {
			t.Fatal(err)
		}
		if c.n != 1 {
			t.Fatalf("wrapped called %d times, want 1", c.n)
		}
	})

	t.Run("group", func(t *testing.T) {
		c := &counter{}
		d := dedup.New(c).WithGroup("g")
		if err := d.Handle(context.Background(), warn("x")); err != nil {
			t.Fatal(err)
		}
		if err := d.Handle(context.Background(), warn("x")); err != nil {
			t.Fatal(err)
		}
		if c.n != 1 {
			t.Fatalf("wrapped called %d times, want 1", c.n)
		}
	})
}

func Test_Slog(t *testing.T) {
	c := &counter{}
	l := slog.New(dedup.New(c).Slog())

	l.Warn("x")
	l.Warn("x")
	if c.n != 1 {
		t.Fatalf("wrapped called %d times, want 1", c.n)
	}
}

func Test_RefCount(t *testing.T) {
	c := &counter{}
	d := dedup.New(c)
	d2 := d.WithAttrs([]slog.Attr{slog.String("k", "v")})
	if err := d2.Close(); err != nil {
		t.Fatal(err)
	}

	if err := d.Handle(context.Background(), warn("x")); err != nil {
		t.Fatal(err)
	}
	if err := d.Handle(context.Background(), warn("x")); err != nil {
		t.Fatal(err)
	}
	if c.n != 1 {
		t.Fatalf("after derived close: wrapped called %d times, want 1", c.n)
	}
}

type fakePooler struct {
	gets int
	puts int
}

func (p *fakePooler) Get(n int) []byte {
	p.gets++
	return make([]byte, n)
}
func (p *fakePooler) Put(b []byte) {
	p.puts++
}

func Test_Pooler(t *testing.T) {
	p := &fakePooler{}
	d := dedup.New(&counter{}, dedup.WithPooler(p))
	if p.gets != 1 {
		t.Fatalf("gets: got %d, want 1", p.gets)
	}

	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if p.puts != 1 {
		t.Fatalf("puts: got %d, want 1", p.puts)
	}
}
