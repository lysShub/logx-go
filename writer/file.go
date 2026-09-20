package writer

import (
	"bytes"
	"context"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/lysShub/bytespool-go"
)

type file struct {
	*os.File
} //
var _ Writer = (*file)(nil)
var Stderr Writer = MustFile(os.Stderr)
var Stdout Writer = MustFile(os.Stdout)

func (f *file) Sync(context.Context) error { return f.File.Sync() }

func File[T *os.File | string](f T) (w Writer, err error) {
	fh, err := openFile(f, os.O_CREATE|os.O_WRONLY|os.O_APPEND)
	if err != nil {
		return nil, err
	}
	fh.Seek(0, io.SeekEnd) // ensure append position; ignore for non-seekable (pipe, stderr)
	return &file{File: fh}, nil
}
func MustFile[T *os.File | string](f T) Writer {
	if w, err := File(f); err != nil {
		panic(err.Error())
	} else {
		return w
	}
}
func openFile[T *os.File | string](f T, flag int) (fh *os.File, err error) {
	if unsafe.Sizeof(new(int)) == unsafe.Sizeof(f) {
		fh, err = *(**os.File)(unsafe.Pointer(&f)), nil
	} else {
		fh, err = os.OpenFile(*(*string)(unsafe.Pointer(&f)), flag, 0o644)
	}
	if err != nil && fh != nil {
		fh.Close()
	}
	return fh, err
}

type rotate struct {
	limit int64
	fh    *os.File
	raw   syscall.RawConn
	size  atomic.Int64

	rotating atomic.Bool
	mu       sync.Mutex
	pending  [][]byte
} //
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

	r := &rotate{limit: int64(limit), fh: fh}
	r.raw, err = fh.SyscallConn()
	if err != nil {
		return nil, err
	}
	r.size.Store(st.Size())

	if r.size.Load()-r.limit > r.limit {
		if err := r.rotate(); err != nil {
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

func (r *rotate) Close() (err error) {
	if e := r.raw.Write(func(fd uintptr) (done bool) {
		if err = r.fh.Sync(); err != nil {
			return true
		}
		if err = r.fh.Close(); err != nil {
			return true
		}
		return true
	}); e != nil {
		return e
	}
	return err
}
func (r *rotate) Sync(context.Context) (err error) {
	if e := r.raw.Write(func(fd uintptr) (done bool) {
		err = r.fh.Sync()
		return true
	}); e != nil {
		return e
	}
	return err
}

func (r *rotate) Write(p []byte) (int, error) {
	if r.rotating.Load() {
		r.mu.Lock()
		if r.rotating.Load() {
			b := bytespool.Get[[]byte, byte](len(p))
			copy(b, p)
			r.pending = append(r.pending, b)
			r.mu.Unlock()
			return len(p), nil
		}
		r.mu.Unlock()
	}

	n, err := r.fh.Write(p)
	if err != nil {
		return n, err
	}
	size := r.size.Add(int64(n))

	if size-r.limit > r.limit {
		if err := r.rotate(); err != nil {
			return 0, err
		}
	}
	return n, err
}

func (r *rotate) rotate() (err error) {
	if !r.rotating.CompareAndSwap(false, true) {
		return nil
	}
	defer r.rotating.CompareAndSwap(true, false)

	if e := r.raw.Write(func(_ uintptr) (done bool) {
		err = r.moveToHead()
		return true
	}); e != nil {
		return e
	}
	if err != nil {
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

	var b = bytespool.Get[[]byte, byte](1024 * 32)
	defer bytespool.Put[[]byte, byte](b)

	n, err := r.copyToHead(b, size)
	if err != nil {
		return err
	}
	r.size.Store(n)

	if err := r.fh.Truncate(n); err != nil {
		return err
	}
	if _, err := r.fh.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	return nil
}
func (r *rotate) flush() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rotating.Store(false)

	for _, p := range r.pending {
		n, err := r.fh.Write(p)
		if err != nil {
			return err
		}
		r.size.Add(int64(n))
		bytespool.Put[[]byte, byte](p)
	}
	r.pending = r.pending[:0]
	return nil
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
