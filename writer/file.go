package writer

import (
	"io"
	"os"
	"path/filepath"
	"unsafe"
)

type std struct{ *os.File } //
var _ Writer = std{}

func Stderr() Writer { return std{os.Stderr} }
func Stdout() Writer { return std{os.Stdout} }

func (std) Close() error { return nil }
func (s std) Sync() error {
	s.File.Sync()
	return nil
}

type file struct{ *os.File } //
var _ Writer = (*file)(nil)

func File[T *os.File | string](f T) (w Writer, err error) {
	fh, err := openFile(f, os.O_CREATE|os.O_WRONLY|os.O_APPEND)
	if err != nil {
		return nil, err
	}
	// ensure append position; ignore for non-seekable (pipe, stderr)
	fh.Seek(0, io.SeekEnd)

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
		p := *(*string)(unsafe.Pointer(&f))
		fh, err = os.OpenFile(p, flag, 0o644)
		if err != nil {
			os.MkdirAll(filepath.Dir(p), 0o755)
			fh, err = os.OpenFile(p, flag, 0o644)
		}
	}
	if err != nil && fh != nil {
		fh.Close()
	}
	return fh, err
}
