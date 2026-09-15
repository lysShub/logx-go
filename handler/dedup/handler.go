package dedup

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/lysShub/bytespool-go"
	"github.com/lysShub/logx-go/handler"
	"github.com/lysShub/logx-go/stack"
	"github.com/zeebo/xxh3"
)

type dedup struct {
	handler.Handler
	o    option
	recs []record
} //
var _ handler.Handler = (*dedup)(nil)

type record struct{ v atomic.Uint64 }
type option struct {
	hash  func(slog.Record) uint32
	ttl   uint32 // seconds
	count int
}
type Option func(*option)

var defaultOption = option{
	hash:  defalutHash,
	ttl:   15,
	count: 512, // 512 * 8 = 4KiB
}

func WithHash(fn func(slog.Record) uint32) func(*option) {
	return func(c *option) { c.hash = fn }
}

func WithTTL(ttl time.Duration) func(*option) {
	return func(c *option) { c.ttl = uint32(ttl.Seconds()) }
}

func WithCacheSize(count int) func(*option) {
	return func(o *option) { o.count = count }
}

func New(h handler.Handler, opts ...Option) handler.Handler {
	d := &dedup{
		Handler: h,
		o:       defaultOption,
	}
	for _, e := range opts {
		e(&d.o)
	}
	d.recs = bytespool.Get[[]record, record](d.o.count)
	return d
}
func (d *dedup) Close() error {
	bytespool.Put[[]record, record](d.recs)
	return d.Handler.Close()
}

func (d *dedup) Handle(ctx context.Context, rec slog.Record) error {
	if sum := d.o.hash(rec); sum != 0 {
		now := uint32(time.Now().Unix())

		t := &d.recs[sum%uint32(len(d.recs))]
		if hash, stamp := t.get(); hash == sum {
			if now-stamp < d.o.ttl {
				return nil
			} else {
				t.set(sum, now)
			}
		} else {
			t.set(sum, now)
		}
	}
	return d.Handler.Handle(ctx, rec)
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
		hash, next = hashStacks(hash, a)
		return !next
	})
	return uint32(hash>>32) ^ uint32(hash)
}
func hashStacks(hash uint64, a slog.Attr) (uint64, bool) {
	var ok = false
	if a.Key == stack.StackKey {
		v := a.Value.Any()
		if st, is := v.(stack.StackTrace); is {
			const word = int(unsafe.Sizeof(st[0]))

			b := unsafe.Slice((*byte)(unsafe.Pointer(&st[0])), len(st)*word)
			hash = xxh3.HashSeed(b, hash)
			ok = true
		}
	}
	return hash, ok
}
