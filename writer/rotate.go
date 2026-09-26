package writer

import (
	"bytes"
	"context"
	"io"
	"os"
	"sync"
)

type rotate struct {
	limit int64

	mu   sync.Mutex
	fh   *os.File
	size int64

	canReflink  bool
	canSendfile bool
} //
var _ Writer = (*rotate)(nil)

func Rotate[T *os.File | string](f T, limit int) (w Writer, err error) {
	fh, err := openFile(f, os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, err
	}
	w, err = rotateRaw(fh, limit)
	if err != nil {
		fh.Close()
		return nil, err
	}
	return w, nil
}
func rotateRaw(fh *os.File, limit int) (w Writer, err error) {
	st, err := fh.Stat()
	if err != nil {
		fh.Close()
		return nil, err
	}
	if !st.Mode().IsRegular() {
		// pipe/terminal/socket cannot rotate (no seek/truncate), degrade to plain file
		return &file{File: fh}, nil
	}

	r := &rotate{
		limit:       int64(limit),
		fh:          fh,
		size:        st.Size(),
		canReflink:  true,
		canSendfile: true,
	}
	if r.size-r.limit > r.limit {
		err := r.rotate()
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
	if err := r.fh.Sync(); err != nil {
		r.fh.Close()
		return err
	}
	return r.fh.Close()
}
func (r *rotate) Sync(context.Context) error {
	return r.fh.Sync()
}

func (r *rotate) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	n, err := r.fh.Write(p)
	if err != nil {
		return n, err
	}
	r.size += int64(n)
	if r.size-r.limit > r.limit {
		if err := r.rotate(); err != nil {
			return 0, err
		}
	}
	return n, nil
}

func (r *rotate) rotate() error {
	return r.moveToHead()
}
func (r *rotate) moveToHead() error {
	var b = Pooler.Get(1024 * 32)
	defer Pooler.Put(b)

	n, err := r.copyToHead(b, r.size)
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
func (r *rotate) pos(b []byte, size int64) (int64, error) {
	p := size - r.limit
	if p < 0 {
		p = 0
	}
	for {
		m, err := r.fh.ReadAt(b, p)
		if err != nil && err != io.EOF {
			return 0, err
		}
		if i := bytes.IndexByte(b[:m], '\n'); i >= 0 {
			return p + int64(i) + 1, nil
		}
		if err == io.EOF {
			return -1, nil
		}
		p += int64(m)
	}
}
func (r *rotate) copyToHeadRaw(b []byte, pos int64) (int64, error) {
	i := int64(0)
	for {
		m, err := r.fh.ReadAt(b, pos+i)
		if err != nil && err != io.EOF {
			return 0, err
		}
		if m > 0 {
			if _, err := r.fh.WriteAt(b[:m], i); err != nil {
				return 0, err
			}
		}
		i += int64(m)
		if err == io.EOF {
			break
		}
	}
	return i, nil
}
