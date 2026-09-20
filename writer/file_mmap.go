//go:build windows || darwin || dragonfly || freebsd || linux || openbsd || solaris || netbsd
// +build windows darwin dragonfly freebsd linux openbsd solaris netbsd

package writer

import (
	"bytes"

	"github.com/edsrzf/mmap-go"
)

func (r *rotate) copyToHead() (n int64, err error) {
	if e := r.raw.Write(func(_ uintptr) (done bool) {
		// locked by os.File write mutext
		n, err = r._copyToHead()
		return false
	}); e != nil {
		return 0, e
	}
	return n, err
}
func (r *rotate) _copyToHead() (int64, error) {
	m, err := mmap.Map(r.fh, mmap.RDWR, 0o644)
	if err != nil {
		return 0, err
	}
	defer m.Unmap()

	i := bytes.IndexByte(m[len(m)/2:], '\n')
	if i <= 0 {
		return int64(len(m)), nil
	} else {
		i += len(m) / 2
	}
	n := copy(m[i:], m)
	return int64(n), nil
}
