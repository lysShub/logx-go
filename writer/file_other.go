//go:build !windows && !darwin && !dragonfly && !freebsd && !linux && !openbsd && !solaris && !netbsd
// +build !windows,!darwin,!dragonfly,!freebsd,!linux,!openbsd,!solaris,!netbsd

package writer

import (
	"bytes"
	"io"

	"github.com/lysShub/bytespool-go"
)

func (r *rotate) copyToHead() (int64, error) {
	var b = bytespool.Get[[]byte, byte](1024 * 32)
	defer bytespool.Put[[]byte, byte](b)

	i := r.size.Load() / 2
	for {
		m, err := r.fh.ReadAt(b, i)
		if err != nil && err != io.EOF {
			return 0, err
		}
		j := bytes.IndexByte(b[:m], '\n')
		if j > 0 {
			i += int64(j)
		}

		if j > 0 || err == io.EOF {
			break
		}
	}
	if i == r.size.Load()/2 {
		return r.size.Load(), nil
	}

	j := int64(0)
	for {
		n, err := r.fh.ReadAt(b, i+j)
		if err != nil && err != io.EOF {
			return 0, err
		}

		if _, err := r.fh.WriteAt(b[:n], j); err != nil {
			return 0, err
		}
		j += int64(n)
		if err == io.EOF {
			break
		}
	}
	return j, nil
}
