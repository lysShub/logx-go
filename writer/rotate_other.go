//go:build !linux
// +build !linux

package writer

import (
	"io"
)

func (r *rotate) copyToHead(b []byte, size int64) (int64, error) {
	mid, err := r.mid(b, size)
	if err != nil {
		return 0, err
	} else if mid < 0 {
		return size, nil
	}

	i := int64(0)
	for {
		n, err := r.fh.ReadAt(b, mid+i)
		if err != nil && err != io.EOF {
			return 0, err
		}

		if _, err := r.fh.WriteAt(b[:n], i); err != nil {
			return 0, err
		}
		i += int64(n)
		if err == io.EOF {
			break
		}
	}
	return i, nil
}
