package writer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func Test_rotate(t *testing.T) {

	t.Run("file cannot peek", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		defer w.Close()

		wr, err := Rotate(w, 1024)
		if err != nil {
			t.Fatal(err)
		}
		defer wr.Close()

		data := []byte("pipe data\n")
		n, err := wr.Write(data)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(data) {
			t.Fatalf("want n=%d, got n=%d", len(data), n)
		}

		buf := make([]byte, 64)
		nn, err := r.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if string(buf[:nn]) != "pipe data\n" {
			t.Fatalf("want %q, got %q", "pipe data\n", string(buf[:nn]))
		}
	})

	t.Run("basic write", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "w.log")

		w, err := Rotate(p, 1024*1024)
		if err != nil {
			t.Fatal(err)
		}
		defer w.Close()

		for i := 0; i < 10; i++ {
			line := fmt.Sprintf("line-%03d\n", i)
			if _, err := w.Write([]byte(line)); err != nil {
				t.Fatal(err)
			}
		}

		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if len(b) == 0 {
			t.Fatal("file is empty")
		}
		if !strings.Contains(string(b), "line-000") {
			t.Fatalf("missing expected data: %q", string(b[:min(100, len(b))]))
		}
	})

	t.Run("rotation triggers", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "rotate.log")

		const limit = 100
		w, err := Rotate(p, limit)
		if err != nil {
			t.Fatal(err)
		}
		defer w.Close()

		// write enough to exceed 2*limit
		for i := 0; i < 25; i++ {
			line := fmt.Sprintf("line-%03d\n", i)
			if _, err := w.Write([]byte(line)); err != nil {
				t.Fatal(err)
			}
		}

		// file size should be <= 2*limit (rotation happened at least once)
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Size() > 2*limit+100 {
			t.Fatalf("file too large after rotation: %d (limit=%d)", fi.Size(), limit)
		}

		// file should not be empty
		if fi.Size() == 0 {
			t.Fatal("file is empty after rotation")
		}
	})

	t.Run("write after rotation", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "rotate.log")

		w, err := Rotate(p, 100)
		if err != nil {
			t.Fatal(err)
		}
		defer w.Close()

		// trigger rotation
		for i := 0; i < 25; i++ {
			w.Write([]byte(fmt.Sprintf("line-%03d\n", i)))
		}

		// write more after rotation
		post := []byte("after-rotate\n")
		if _, err := w.Write(post); err != nil {
			t.Fatal(err)
		}

		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "after-rotate") {
			t.Fatalf("missing post-rotation data, file: %q", string(b[:min(200, len(b))]))
		}
	})

	t.Run("concurrent writes", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "race.log")

		w, err := Rotate(p, 50)
		if err != nil {
			t.Fatal(err)
		}
		defer w.Close()

		var wg sync.WaitGroup
		for g := 0; g < 4; g++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for i := 0; i < 50; i++ {
					line := fmt.Sprintf("g%d-line-%03d\n", id, i)
					if _, err := w.Write([]byte(line)); err != nil {
						t.Errorf("goroutine %d: %v", id, err)
						return
					}
				}
			}(g)
		}
		wg.Wait()

		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if len(b) == 0 {
			t.Fatal("file is empty after concurrent writes")
		}
	})
}
