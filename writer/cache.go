package writer

import (
	"slices"
	"sync"
	"time"

	"github.com/lysShub/errorx-go"
)

type cache struct {
	bytesLimit  int
	periodLimit time.Duration
	w           Writer

	m     sync.Mutex
	bytes int
	s     [][]byte
	ch    chan struct{}

	fmu      sync.Mutex
	closeErr errorx.CloseErr
}

func WithByteLimit(bytes int) func(*cache) {
	return func(b *cache) { b.bytesLimit = bytes }
}
func WithPeriodLimit(dur time.Duration) func(*cache) {
	return func(b *cache) { b.periodLimit = dur }
}

func Cache(w Writer, opts ...func(*cache)) Writer {
	var b = &cache{
		bytesLimit:  1024 * 1024,
		periodLimit: time.Minute * 5,
		w:           w,
		ch:          make(chan struct{}),
	}
	for _, e := range opts {
		e(b)
	}
	if b.bytesLimit <= 0 && b.periodLimit <= 0 {
		panic("invalid argument")
	}

	go b.service()
	return b
}

func (b *cache) Sync() error { return b.flush() }

func (b *cache) close(cause error) error {
	return b.closeErr.Close(func() (errs []error) {
		errs = append(errs, cause)

		b.m.Lock()
		{
			for _, e := range b.s {
				Pooler.Put(e)
			}
			b.s = nil
		}
		b.m.Unlock()

		if b.w != nil {
			errs = append(errs, b.w.Close())
		}
		return errs
	})
}
func (b *cache) Close() error { return b.close(nil) }

func (b *cache) Write(p []byte) (int, error) {
	b.m.Lock()
	defer b.m.Unlock()

	if b.closeErr.Closed() {
		return 0, b.closeErr.Error()
	}

	i := len(b.s) - 1
	if i >= 0 && rem(b.s[i]) >= len(p) {
		b.s[i] = append(b.s[i], p...)
	} else {
		p1 := Pooler.Get(max(len(p), 4*1024))
		n := copy(p1, p)
		b.s = append(b.s, p1[:n])
	}
	b.bytes += len(p)

	if b.bytesLimit > 0 && b.bytes > b.bytesLimit {
		select {
		case b.ch <- struct{}{}:
		default:
		}
	}
	return len(p), nil
}
func rem(b []byte) int { return cap(b) - len(b) }

func (b *cache) service() (_ error) {
	dur := time.Hour * 24
	if b.periodLimit > 0 {
		dur = b.periodLimit
	}
	tick := time.NewTicker(dur)
	defer tick.Stop()

	done := b.closeErr.Done()
	for {
		select {
		case <-b.ch:
		case <-tick.C:
		case <-done:
			return nil
		}
		if err := b.flush(); err != nil {
			return b.close(err)
		}
	}
}
func (b *cache) flush() error {
	b.fmu.Lock()
	defer b.fmu.Unlock()

	b.m.Lock()
	s := slices.Clone(b.s)
	{
		clear(b.s)
		b.s = b.s[:0]
		b.bytes = 0
	}
	b.m.Unlock()

	for _, e := range s {
		if _, err := b.w.Write(e); err != nil {
			return err
		}
		Pooler.Put(e)
	}
	return b.w.Sync()
}
