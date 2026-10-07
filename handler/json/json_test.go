package json_test

import (
	"bytes"
	"context"
	stdjson "encoding/json"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/lysShub/logx-go/handler"
	"github.com/lysShub/logx-go/handler/json"
)

type mockWriter struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	writes int
	syncs  int
}

func (m *mockWriter) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writes++
	return m.buf.Write(p)
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

func Test_TimeValue(t *testing.T) {
	tm := time.Unix(1700000000, 123456789).UTC()

	tests := []struct {
		name string
		f    handler.TimeValue
		want string
	}{
		{"rfc3339", handler.RFC3339Millis, `"2023-11-14T22:13:20.123Z"`},
		{"unix", func(t time.Time) slog.Value { return slog.Int64Value(t.Unix()) }, "1700000000"},
		{"unix_nano", func(t time.Time) slog.Value { return slog.Int64Value(t.UnixNano()) }, "1700000000123456789"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockWriter{}
			h := json.NewJSON(m, json.WithTimeLocation(time.UTC), json.WithTimeValue(tt.f))
			if err := h.Handle(context.Background(), slog.NewRecord(tm, slog.LevelInfo, "msg", 0)); err != nil {
				t.Fatal(err)
			}

			var rec map[string]stdjson.RawMessage
			if err := stdjson.Unmarshal(m.buf.Bytes(), &rec); err != nil {
				t.Fatal(err)
			}
			if got := string(rec[slog.TimeKey]); got != tt.want {
				t.Fatalf("time: got %s, want %s", got, tt.want)
			}
		})
	}
}

func Test_LevelValue(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		tests := []struct {
			level slog.Level
			want  string
		}{
			{slog.LevelDebug, `"debug"`},
			{slog.LevelInfo, `"info"`},
			{slog.LevelWarn, `"warn"`},
			{slog.LevelError, `"error"`},
			{slog.LevelError + 4, `"fatal"`},
		}
		for _, tt := range tests {
			m := &mockWriter{}
			h := json.NewJSON(m)
			if err := h.Handle(context.Background(), rec(tt.level)); err != nil {
				t.Fatal(err)
			}

			var got map[string]stdjson.RawMessage
			if err := stdjson.Unmarshal(m.buf.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if s := string(got[slog.LevelKey]); s != tt.want {
				t.Fatalf("level %v: got %s, want %s", tt.level, s, tt.want)
			}
		}
	})

	t.Run("custom", func(t *testing.T) {
		m := &mockWriter{}
		h := json.NewJSON(m, json.WithLevelValue(func(l slog.Level) slog.Value {
			return slog.Int64Value(int64(l))
		}))
		if err := h.Handle(context.Background(), rec(slog.LevelWarn)); err != nil {
			t.Fatal(err)
		}

		var got map[string]stdjson.RawMessage
		if err := stdjson.Unmarshal(m.buf.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if s := string(got[slog.LevelKey]); s != "4" {
			t.Fatalf("level: got %s, want 4", s)
		}
	})
}

func Test_Enabled(t *testing.T) {
	m := &mockWriter{}
	h := json.NewJSON(m, json.WithLeveler(slog.LevelWarn))

	for level, want := range map[slog.Level]bool{
		slog.LevelInfo:  false,
		slog.LevelWarn:  true,
		slog.LevelError: true,
	} {
		if got := h.Enabled(context.Background(), level); got != want {
			t.Fatalf("level %v: got %v, want %v", level, got, want)
		}
	}
}

func Test_Replace(t *testing.T) {
	m := &mockWriter{}
	h := json.NewJSON(m, json.WithReplace(func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.MessageKey {
			a.Value = slog.StringValue("replaced")
		}
		return a
	}))
	if err := h.Handle(context.Background(), rec(slog.LevelInfo)); err != nil {
		t.Fatal(err)
	}

	var got map[string]stdjson.RawMessage
	if err := stdjson.Unmarshal(m.buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if s := string(got[slog.MessageKey]); s != `"replaced"` {
		t.Fatalf("msg: got %s, want %q", s, "replaced")
	}
}

func Test_Attrs(t *testing.T) {
	t.Run("with_attrs", func(t *testing.T) {
		m := &mockWriter{}
		h := json.NewJSON(m).WithAttrs(slog.String("a", "b"))
		if err := h.Handle(context.Background(), rec(slog.LevelInfo)); err != nil {
			t.Fatal(err)
		}

		var got map[string]stdjson.RawMessage
		if err := stdjson.Unmarshal(m.buf.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if s := string(got["a"]); s != `"b"` {
			t.Fatalf("a: got %s, want %q", s, "b")
		}
	})

	t.Run("with_group", func(t *testing.T) {
		m := &mockWriter{}
		h := json.NewJSON(m).WithGroup("g")
		r := rec(slog.LevelInfo)
		r.AddAttrs(slog.String("a", "b"))
		if err := h.Handle(context.Background(), r); err != nil {
			t.Fatal(err)
		}

		var got map[string]stdjson.RawMessage
		if err := stdjson.Unmarshal(m.buf.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if s := string(got["g"]); s != `{"a":"b"}` {
			t.Fatalf("g: got %s, want %q", s, `{"a":"b"}`)
		}
	})
}
