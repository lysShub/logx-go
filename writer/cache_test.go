package writer

import (
	"bytes"
	"sync"
	"testing"
	"time"
)

type memWriter struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	syncs  int
	closed bool
}

func (m *memWriter) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.buf.Write(p)
}
func (m *memWriter) Sync() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.syncs++
	return nil
}
func (m *memWriter) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}
func (m *memWriter) String() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.buf.String()
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met in time")
}

func Test_cache(t *testing.T) {

	t.Run("sync flushes pending", func(t *testing.T) {
		w := &memWriter{}
		b := Cache(w, WithByteLimit(1<<20), WithPeriodLimit(time.Hour))
		defer b.Close()

		if _, err := b.Write([]byte("hello ")); err != nil {
			t.Fatal(err)
		}
		if _, err := b.Write([]byte("world")); err != nil {
			t.Fatal(err)
		}
		if w.String() != "" {
			t.Fatalf("unexpected early flush: %q", w.String())
		}

		if err := b.Sync(); err != nil {
			t.Fatal(err)
		}
		if w.String() != "hello world" {
			t.Fatalf("want %q, got %q", "hello world", w.String())
		}
		if w.syncs == 0 {
			t.Fatal("underlying Sync not called")
		}
	})

	t.Run("byte limit triggers flush", func(t *testing.T) {
		w := &memWriter{}
		b := Cache(w, WithByteLimit(16), WithPeriodLimit(time.Hour))
		defer b.Close()

		// let service enter its select loop, so the trigger isn't dropped
		time.Sleep(50 * time.Millisecond)

		if _, err := b.Write([]byte("0123456789")); err != nil {
			t.Fatal(err)
		}
		if _, err := b.Write([]byte("abcdefghij")); err != nil {
			t.Fatal(err)
		}
		waitFor(t, func() bool { return w.String() != "" })
		if w.String() != "0123456789abcdefghij" {
			t.Fatalf("got %q", w.String())
		}
	})

	t.Run("period limit triggers flush", func(t *testing.T) {
		w := &memWriter{}
		b := Cache(w, WithByteLimit(1<<20), WithPeriodLimit(10*time.Millisecond))
		defer b.Close()

		if _, err := b.Write([]byte("tick")); err != nil {
			t.Fatal(err)
		}
		waitFor(t, func() bool { return w.String() == "tick" })
	})

	t.Run("close drops pending", func(t *testing.T) {
		w := &memWriter{}
		b := Cache(w, WithByteLimit(1<<20), WithPeriodLimit(time.Hour))

		if _, err := b.Write([]byte("pending")); err != nil {
			t.Fatal(err)
		}
		if err := b.Close(); err != nil {
			t.Fatal(err)
		}
		if w.String() != "" {
			t.Fatalf("close must not flush, got %q", w.String())
		}
		if !w.closed {
			t.Fatal("underlying writer not closed")
		}
	})

	t.Run("write after close", func(t *testing.T) {
		w := &memWriter{}
		b := Cache(w, WithByteLimit(1<<20), WithPeriodLimit(time.Hour))
		if err := b.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := b.Write([]byte("x")); err == nil {
			t.Fatal("want error after close")
		}
	})

	t.Run("concurrent writes", func(t *testing.T) {
		w := &memWriter{}
		b := Cache(w, WithByteLimit(1<<20), WithPeriodLimit(time.Hour))
		defer b.Close()

		const per, n = 4, 100
		var wg sync.WaitGroup
		for g := 0; g < 4; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < n; i++ {
					if _, err := b.Write([]byte("abcd")); err != nil {
						t.Errorf("write: %v", err)
						return
					}
				}
			}()
		}
		wg.Wait()

		if err := b.Sync(); err != nil {
			t.Fatal(err)
		}
		if got, want := len(w.String()), per*n*4; got != want {
			t.Fatalf("want %d bytes, got %d", want, got)
		}
	})

	t.Run("invalid options panic", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
		}()
		Cache(&memWriter{}, WithByteLimit(0), WithPeriodLimit(0))
	})
}
