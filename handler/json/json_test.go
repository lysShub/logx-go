package json_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/lysShub/logx-go/handler/json"
)

type mockWriter struct {
	mu     sync.Mutex
	writes int
	syncs  int
}

func (m *mockWriter) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writes++
	return len(p), nil
}
func (m *mockWriter) Sync() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.syncs++
	return nil
}
func (m *mockWriter) Close() error { return nil }
func (m *mockWriter) count() (writes, syncs int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writes, m.syncs
}

func rec(level slog.Level) slog.Record {
	return slog.NewRecord(time.Now(), level, "msg", 0)
}

func Test_SyncLevel(t *testing.T) {

	t.Run("default is error", func(t *testing.T) {
		m := &mockWriter{}
		h := json.NewJSON(m) // default SyncLevel = slog.LevelError

		if err := h.Handle(context.Background(), rec(slog.LevelError)); err != nil {
			t.Fatal(err)
		}
		if w, s := m.count(); w != 1 || s != 1 {
			t.Fatalf("error: want write=1 sync=1, got write=%d sync=%d", w, s)
		}

		if err := h.Handle(context.Background(), rec(slog.LevelWarn)); err != nil {
			t.Fatal(err)
		}
		if w, s := m.count(); w != 2 || s != 1 {
			t.Fatalf("warn: want write=2 sync=1, got write=%d sync=%d", w, s)
		}
	})

	t.Run("custom level", func(t *testing.T) {
		m := &mockWriter{}
		h := json.NewJSON(m, json.WithSyncLevel(slog.LevelWarn))

		if err := h.Handle(context.Background(), rec(slog.LevelInfo)); err != nil {
			t.Fatal(err)
		}
		if w, s := m.count(); w != 1 || s != 0 {
			t.Fatalf("info: want write=1 sync=0, got write=%d sync=%d", w, s)
		}

		if err := h.Handle(context.Background(), rec(slog.LevelWarn)); err != nil {
			t.Fatal(err)
		}
		if w, s := m.count(); w != 2 || s != 1 {
			t.Fatalf("warn: want write=2 sync=1, got write=%d sync=%d", w, s)
		}
	})
}
