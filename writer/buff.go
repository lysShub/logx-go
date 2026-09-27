package writer

import (
	"os"
	"slices"
	"sync"
	"time"
)

type buff struct {
	bytesLimit  int
	periodLimit time.Duration
	w           Writer
	ch          chan uint8

	m        sync.Mutex
	bytes    int
	s        [][]byte
	flushErr error
	close    bool
}

const closed uint8 = 0xff

func WithByteLimit(bytes int) func(*buff) {
	return func(b *buff) { b.bytesLimit = bytes }
}
func WithPeriodLimit(dur time.Duration) func(*buff) {
	return func(b *buff) { b.periodLimit = dur }
}

func Buff(w Writer, opts ...func(*buff)) Writer {
	var b = &buff{
		bytesLimit:  1024 * 1024,
		periodLimit: time.Minute * 5,
		w:           w,
		ch:          make(chan uint8),
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

func (b *buff) Sync() error { return b.flush() }

func (b *buff) Close() error {
	if b == nil {
		return nil
	}
	b.m.Lock()
	if b.close {
		b.m.Unlock()
		return nil
	}
	b.close = true
	b.m.Unlock()

	b.ch <- closed

	b.m.Lock()
	for _, e := range b.s {
		Pooler.Put(e)
	}
	b.s = nil
	b.m.Unlock()

	return b.w.Close()
}

func (b *buff) Write(p []byte) (int, error) {
	b.m.Lock()
	defer b.m.Unlock()
	if b.flushErr != nil {
		return 0, b.flushErr
	} else if b.close {
		return 0, os.ErrClosed
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
		case b.ch <- 0:
		default:
		}
	}
	return len(p), nil
}
func rem(b []byte) int { return cap(b) - len(b) }

func (b *buff) service() {
	dur := time.Hour * 24
	if b.periodLimit > 0 {
		dur = b.periodLimit
	}
	tick := time.NewTicker(dur)
	defer tick.Stop()

	for {
		select {
		case v := <-b.ch:
			if v == closed {
				return
			}
		case <-tick.C:
		}
		if err := b.flush(); err != nil {
			b.m.Lock()
			b.flushErr = err
			b.m.Unlock()
		}
	}
}

func (b *buff) flush() error {
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
