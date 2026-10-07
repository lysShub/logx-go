package writer

import (
	"errors"
	"testing"
)

type errWriter struct {
	werr   error
	serr   error
	cerr   error
	closed bool
}

func (w *errWriter) Write(p []byte) (int, error) { return 0, w.werr }
func (w *errWriter) Sync() error                 { return w.serr }
func (w *errWriter) Close() error                { w.closed = true; return w.cerr }

func Test_tee(t *testing.T) {
	t.Run("fan out", func(t *testing.T) {
		a, b := &memWriter{}, &memWriter{}
		w := Tee(a, b)

		n, err := w.Write([]byte("hello"))
		if err != nil || n != len("hello") {
			t.Fatalf("Write: n=%d err=%v", n, err)
		}
		if a.String() != "hello" || b.String() != "hello" {
			t.Fatalf("got a=%q b=%q", a.String(), b.String())
		}
	})

	t.Run("write error does not stop others", func(t *testing.T) {
		e, a, b := &errWriter{werr: errors.New("boom")}, &memWriter{}, &memWriter{}
		w := Tee(e, a, b)

		if _, err := w.Write([]byte("x")); err == nil {
			t.Fatal("want error")
		}
		if a.String() != "x" || b.String() != "x" {
			t.Fatalf("others not written: a=%q b=%q", a.String(), b.String())
		}
	})

	t.Run("sync and close", func(t *testing.T) {
		a, b := &memWriter{}, &memWriter{}
		w := Tee(a, b)

		if err := w.Sync(); err != nil {
			t.Fatal(err)
		}
		if a.syncs != 1 || b.syncs != 1 {
			t.Fatalf("syncs a=%d b=%d, want 1,1", a.syncs, b.syncs)
		}

		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		if !a.closed || !b.closed {
			t.Fatalf("closed a=%v b=%v, want true,true", a.closed, b.closed)
		}
	})
}
