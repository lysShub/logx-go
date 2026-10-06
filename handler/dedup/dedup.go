package dedup

import (
	"bytes"
	"context"
	"log/slog"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/lysShub/bytespool-go"
	"github.com/lysShub/debug-go"
	"github.com/lysShub/logx-go/handler"
	"github.com/lysShub/logx-go/stack"
	"github.com/zeebo/xxh3"
)

type dedup struct {
	handler.Handler
	o    option
	recs *records
} //
var _ handler.Handler = (*dedup)(nil)

type option struct {
	hash   func(slog.Record) uint32
	ttl    uint32 // seconds
	bytes  int
	reuse  bool
	pooler pooler
}
type Option func(*option)
type pooler interface {
	Get(n int) []byte
	Put(b []byte)
}

type bytePool struct{}

func (bytePool) Get(n int) []byte { return bytespool.Alloc[[]byte, byte](n) }
func (bytePool) Put(b []byte)     { bytespool.Put[[]byte, byte](b) }

var defaultOption = option{
	hash:   defalutHash,
	ttl:    15,
	bytes:  4096, // 4KiB = 512 slots
	reuse:  true,
	pooler: bytePool{},
}

// WithHash sets the record hash function, default [defalutHash].
func WithHash(fn func(slog.Record) uint32) func(*option) {
	return func(c *option) { c.hash = fn }
}

// WithTTL sets the dedup time window, default 15s.
func WithTTL(ttl time.Duration) func(*option) {
	return func(c *option) { c.ttl = uint32(ttl.Seconds()) }
}

// WithBytes sets the dedup table size in bytes, default 4096.
func WithBytes(bytes int) func(*option) {
	return func(o *option) { o.bytes = bytes }
}

// WithReuse shares one dedup table across derived handlers, default true.
func WithReuse(reuse bool) func(*option) {
	return func(o *option) { o.reuse = reuse }
}

// WithPooler sets the dedup table pooler, default [bytespool.Pool].
func WithPooler(p pooler) func(*option) {
	return func(o *option) { o.pooler = p }
}

func New(h handler.Handler, opts ...Option) handler.Handler {
	d := &dedup{
		Handler: h,
		o:       defaultOption,
	}
	for _, e := range opts {
		e(&d.o)
	}
	d.recs = newRecords(d.o.pooler, d.o.bytes)
	return d
}
func (d *dedup) Close() error {
	d.recs.close()
	return d.Handler.Close()
}

func (d *dedup) Handle(ctx context.Context, rec slog.Record) error {
	if sum := d.o.hash(rec); sum != 0 {
		if d.recs.dedup(sum, d.o.ttl) {
			return nil
		}
	}
	return d.Handler.Handle(ctx, rec)
}

func (d *dedup) WithAttrs(attrs []slog.Attr) handler.Handler {
	if len(attrs) == 0 {
		return d
	}
	return d.derive(d.Handler.WithAttrs(attrs))
}

func (d *dedup) WithGroup(name string) handler.Handler {
	if name == "" {
		return d
	}
	return d.derive(d.Handler.WithGroup(name))
}

func (d *dedup) Slog() slog.Handler { return &handler.WrapHandler{Handler: d} }

func (d *dedup) derive(h handler.Handler) *dedup {
	d2 := &dedup{Handler: h, o: d.o}
	if d.o.reuse {
		d.recs.ref()
		d2.recs = d.recs
	} else {
		d2.recs = newRecords(d.o.pooler, d.o.bytes)
	}
	return d2
}

type records struct {
	p    pooler
	raw  []byte
	l    []record
	refs atomic.Int64
}
type record struct{ v atomic.Uint64 }

func newRecords(p pooler, byteSize int) *records {
	b := p.Get(byteSize)
	if debug.Debug() {
		debug.True(bytes.Equal(b, make([]byte, len(b))))
	}
	ptr := (*record)(unsafe.Pointer(unsafe.SliceData(b)))
	len := len(b) / int(unsafe.Sizeof(record{}))

	r := &records{
		p:   p,
		raw: b,
		l:   unsafe.Slice(ptr, len),
	}
	r.refs.Store(1)
	return r
}
func (r *records) ref() {
	r.refs.Add(1)
}
func (r *records) close() {
	if r.refs.Add(-1) == 0 {
		r.p.Put(r.raw)
	}
}

func (r *records) dedup(sum uint32, ttl uint32) (duplicated bool) {
	now := uint32(time.Now().Unix())

	t := &r.l[sum%uint32(len(r.l))]
	if hash, stamp := t.get(); hash == sum {
		if now-stamp < ttl {
			return true
		} else {
			t.set(sum, now)
		}
	} else {
		t.set(sum, now)
	}
	return false
}
func (r *record) set(hash, stamp uint32) {
	r.v.Store(uint64(hash) | uint64(stamp)<<32)
}
func (r *record) get() (hash, stamp uint32) {
	v := r.v.Load()
	return uint32(v), uint32(v >> 32)
}

func defalutHash(r slog.Record) uint32 {
	if r.Level != slog.LevelWarn {
		return 0
	}
	var hash = xxh3.HashString(r.Message)
	r.Attrs(func(a slog.Attr) (next bool) {
		if a.Key == stack.StackKey {
			v := a.Value.Any()
			if s, is := v.(stack.Stack); is && s != nil {
				b := unsafe.Slice((*byte)(unsafe.Pointer(s)), unsafe.Sizeof(*s))

				hash = xxh3.HashSeed(b, hash)
				return false
			}
		}
		return true
	})
	return uint32(hash>>32) ^ uint32(hash)
}
