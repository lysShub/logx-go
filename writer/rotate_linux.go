//go:build linux
// +build linux

package writer

import (
	"io"
	"sync/atomic"

	"golang.org/x/sys/unix"
)

var (
	reflink  atomic.Bool
	sendfile atomic.Bool
	_        = reflink.Swap(true)
	_        = sendfile.Swap(true)
)

func (r *rotate) copyToHead(b []byte, size int64) (int64, error) {
	pos, err := r.pos(b, size)
	if err != nil {
		return 0, err
	} else if pos < 0 {
		return size, nil
	}
	n := size - pos

	if reflink.Load() {
		err := unix.IoctlFileCloneRange(int(r.fh.Fd()), &unix.FileCloneRange{
			Src_fd:      int64(r.fh.Fd()),
			Src_offset:  uint64(pos),
			Src_length:  uint64(n),
			Dest_offset: 0,
		})
		if err == nil {
			return n, nil
		}
		reflink.Store(false)
	}

	if sendfile.Load() {
		if err := r.copyToHeadSendfile(pos, n); err == nil {
			return n, nil
		}
		sendfile.Store(false)
	}

	return r.copyToHeadRaw(b, pos)
}

func (r *rotate) copyToHeadSendfile(pos, n int64) error {
	fd := int(r.fh.Fd())
	if _, err := r.fh.Seek(0, io.SeekStart); err != nil {
		return err
	}
	end := pos + n
	for pos < end {
		m, err := unix.Sendfile(fd, fd, &pos, int(end-pos))
		if err != nil {
			return err
		}
		if m == 0 {
			break
		}
	}
	return nil
}
