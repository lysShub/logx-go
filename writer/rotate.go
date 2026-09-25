package writer

import (
	"bytes"
	"context"
	"io"
	"os"
	"sync"
	"sync/atomic"
)

type rotate struct {
	limit int64

	mu   sync.Mutex
	fh   *os.File
	size int64

	rotating atomic.Bool
	pmu      sync.Mutex
	pending  [][]byte
}

var _ Writer = (*rotate)(nil)

func Rotate[T *os.File | string](f T, limit int) (w Writer, err error) {
	fh, err := openFile(f, os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, err
	}
	fh.Seek(0, io.SeekEnd)

	st, err := fh.Stat()
	if err != nil {
		fh.Close()
		return nil, err
	}
	if !st.Mode().IsRegular() {
		// pipe/terminal/socket cannot rotate (no seek/truncate), degrade to plain file
		return &file{File: fh}, nil
	}

	r := &rotate{limit: int64(limit), fh: fh, size: st.Size()}
	if r.size-r.limit > r.limit {
		r.mu.Lock()
		err := r.rotate()
		r.mu.Unlock()
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}
func MustRotate[T *os.File | string](f T, bytes int) (w Writer) {
	if w, err := Rotate(f, bytes); err != nil {
		panic(err.Error())
	} else {
		return w
	}
}

func (r *rotate) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.fh.Sync(); err != nil {
		r.fh.Close()
		return err
	}
	return r.fh.Close()
}
func (r *rotate) Sync(context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.fh.Sync()
}

func (r *rotate) Write(p []byte) (int, error) {
	if r.rotating.Load() {
		if err := r.append(p); err != nil {
			return 0, err
		}
		return len(p), nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	n, err := r.fh.Write(p)
	if err != nil {
		return n, err
	}
	r.size += int64(n)
	if r.size-r.limit > r.limit {
		err = r.rotate()
	}
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (r *rotate) rotate() error {
	if !r.rotating.CompareAndSwap(false, true) {
		return nil
	}
	defer r.rotating.Store(false)

	if err := r.moveToHead(); err != nil {
		return err
	}
	return r.flush()
}
func (r *rotate) moveToHead() error {
	st, err := r.fh.Stat()
	if err != nil {
		return err
	}
	size := st.Size()

	var b = Pooler.Get(1024 * 32)
	defer Pooler.Put(b)

	n, err := r.copyToHead(b, size)
	if err != nil {
		return err
	}
	r.size = n

	if err := r.fh.Truncate(n); err != nil {
		return err
	}
	if _, err := r.fh.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	return nil
}
func (r *rotate) append(p []byte) (err error) {
	b := Pooler.Get(len(p))
	copy(b, p)
	r.pmu.Lock()
	if !r.rotating.Load() {
		// rotate finished while we were in the fast path, write directly
		r.pmu.Unlock()
		Pooler.Put(b)

		r.mu.Lock()
		defer r.mu.Unlock()
		_, err = r.fh.Write(p)
		if err != nil {
			return err
		}
		r.size += int64(len(p))
		return nil
	}
	r.pending = append(r.pending, b)
	r.pmu.Unlock()
	return nil
}
func (r *rotate) flush() error {
	for {
		r.pmu.Lock()
		if len(r.pending) == 0 {
			r.pmu.Unlock()
			return nil
		}
		batch := r.pending
		r.pending = nil
		r.pmu.Unlock()

		for _, p := range batch {
			n, err := r.fh.Write(p)
			if err != nil {
				return err
			}
			r.size += int64(n)
			Pooler.Put(p)
		}
	}
}

func (r *rotate) mid(b []byte, size int64) (mid int64, err error) {
	mid = size / 2
	for {
		m, err := r.fh.ReadAt(b, mid)
		if err != nil && err != io.EOF {
			return 0, err
		}
		i := bytes.IndexByte(b[:m], '\n')
		if i >= 0 {
			mid += int64(i) + 1
			return mid, nil
		}
		if err == io.EOF {
			return -1, nil
		}
		mid += int64(m)
	}
}
