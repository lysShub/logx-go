//go:build linux
// +build linux

package writer

import (
	"io"
	"syscall"
)

func (r *rotate) copyToHead(b []byte, size int64) (int64, error) {
	mid, err := r.mid(b, size)
	if err != nil {
		return 0, err
	} else if mid < 0 {
		return size, nil
	}

	if _, err := r.fh.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}

	fd := int(r.fh.Fd())
	mid0 := mid
	for mid < size {
		_, err := syscall.Sendfile(fd, fd, &mid, int(size-mid))
		if err == syscall.EINTR {
			continue
		}
		if err != nil {
			return 0, err
		}
	}
	return size - mid0, nil
}
