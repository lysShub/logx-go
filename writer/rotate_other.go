//go:build !linux
// +build !linux

package writer

func (r *rotate) copyToHead(b []byte, size int64) (int64, error) {
	pos, err := r.pos(b, size)
	if err != nil {
		return 0, err
	} else if pos < 0 {
		return size, nil
	}
	return r.copyToHeadRaw(b, pos)
}
