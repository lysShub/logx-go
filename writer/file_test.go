package writer

import (
	"os"
	"path/filepath"
	"testing"
)

func Test_std(t *testing.T) {
	t.Run("close is isolated", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "std")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		w := std{f}
		if _, err := w.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
		if err := w.Sync(); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}

		// std.Close must be a no-op; the underlying file stays writable.
		if _, err := f.Write([]byte("y")); err != nil {
			t.Fatalf("underlying file was closed: %v", err)
		}
	})

	t.Run("std streams", func(t *testing.T) {
		for _, w := range []Writer{Stdout(), Stderr()} {
			if err := w.Sync(); err != nil {
				t.Fatalf("Sync: %v", err)
			}
			if err := w.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}
		}
	})
}

func Test_file(t *testing.T) {

	t.Run("path exist", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "test.log")
		if err := os.WriteFile(p, []byte("hello\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		w, err := File(p)
		if err != nil {
			t.Fatal(err)
		}
		defer w.Close()

		if _, err := w.Write([]byte("world\n")); err != nil {
			t.Fatal(err)
		}

		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if s := string(b); s != "hello\nworld\n" {
			t.Fatalf("want %q, got %q", "hello\nworld\n", s)
		}
	})

	t.Run("path not exist", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "new.log")

		w, err := File(p)
		if err != nil {
			t.Fatal(err)
		}
		defer w.Close()

		if _, err := w.Write([]byte("data")); err != nil {
			t.Fatal(err)
		}

		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if s := string(b); s != "data" {
			t.Fatalf("want %q, got %q", "data", s)
		}
	})

	t.Run("dir not exist", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "sub", "deep", "test.log")

		w, err := File(p)
		if err != nil {
			t.Fatal(err)
		}
		defer w.Close()

		if _, err := w.Write([]byte("nested")); err != nil {
			t.Fatal(err)
		}

		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if s := string(b); s != "nested" {
			t.Fatalf("want %q, got %q", "nested", s)
		}
	})

	t.Run("file input", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "f.log")
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		f.WriteString("a")

		w, err := File(f)
		if err != nil {
			t.Fatal(err)
		}
		defer w.Close()

		if _, err := w.Write([]byte("b")); err != nil {
			t.Fatal(err)
		}

		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if s := string(b); s != "ab" {
			t.Fatalf("want %q, got %q", "ab", s)
		}
	})

	t.Run("stderr no panic", func(t *testing.T) {
		w := MustFile(os.Stderr)
		if w == nil {
			t.Fatal("MustFile(os.Stderr) returned nil")
		}
	})
}
