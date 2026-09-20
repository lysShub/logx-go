package writer

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/lysShub/bytespool-go"
)

var (
	errNoCollapse = errors.New("writer: collapse range unsupported")
	errNoMmap     = errors.New("writer: mmap unsupported")
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
	return &file{File: fh}, nil
}
func MustFile[T *os.File | string](f T) Writer {
	if w, err := File(f); err != nil {
		panic(err.Error())
	} else {
		return w
	}
}
func openFile[T *os.File | string](f T, flag int) (*os.File, error) {
	if unsafe.Sizeof(new(int)) == unsafe.Sizeof(f) {
		return *(**os.File)(unsafe.Pointer(&f)), nil
	} else {
		return os.OpenFile(*(*string)(unsafe.Pointer(&f)), flag, 0o644)
	}
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

func (r *rotate) Sync(context.Context) error { return r.fh.Sync() }

func Rotate[T *os.File | string](f T, limit int) (w Writer, err error) {
	fh, err := openFile(f, os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, err
	}
	r := &rotate{limit: int64(limit), fh: fh}

	r.raw, err = fh.SyscallConn()
	if err != nil {
		return nil, err
	}

	st, err := fh.Stat()
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

func (r *rotate) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.fh.Close()
}

func (r *rotate) Write(p []byte) (int, error) {
	if r.rotating.Load() {
		r.mu.Lock()
		defer r.mu.Unlock()
		b := bytespool.Get[[]byte, byte](len(p))
		copy(b, p)
		r.pending = append(r.pending, b)
		return len(p), nil
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

func (r *rotate) rotate() error {
	if !r.rotating.CompareAndSwap(false, true) {
		return nil
	}
	{
		n, err := r.copyToHead()
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
	}
	{
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
	}
	return nil
}
