package writer

import (
	"io"
	"sync"
)

type buff struct {
	limit int

	m     sync.Mutex
	rntf  *sync.Cond
	bytes int
	s     [][]byte
}

func newBuff(limit int) *buff {
	b := &buff{limit: limit}
	b.rntf = sync.NewCond(&b.m)
	return b
}

func (b *buff) append(p []byte) int {
	b.m.Lock()
	defer b.m.Unlock()
	b.bytes += len(p)
	for b.bytes > b.limit && b.bytes > 0 {
		b.rntf.Wait()
	}
	p1 := Pooler.Get(len(p))
	copy(p1, p)
	b.s = append(b.s, p1)
	return len(p)
}

func (b *buff) write(to io.Writer) (err error) {
	defer b.rntf.Broadcast()
	b.m.Lock()
	defer b.m.Unlock()
	for i, e := range b.s {
		if _, err := to.Write(e); err != nil {
			return err
		}
		Pooler.Put(e)
		b.s[i] = nil
	}
	b.s = b.s[:0]
	return nil
}

func (b *buff) Close() (_ error) {
	if b != nil {
		b.m.Lock()
		defer b.m.Unlock()
		for _, e := range b.s {
			Pooler.Put(e)
		}
		b.s = nil
	}
	return nil
}
