package writer

import (
	"io"

	"github.com/lysShub/bytespool-go"
)

// Writer is the write/sync/close contract, Close does not Sync
// buffered data silent.
type Writer interface {
	io.Writer
	Syncer
	io.Closer
}
type Syncer interface{ Sync() error }

type pooler bytespool.Pooler[[]byte, byte] //
var Pooler pooler = bytespool.Pool[[]byte, byte]{}
