package writer

import (
	"context"
	"io"

	"github.com/lysShub/bytespool-go"
)

type Writer interface {
	io.Writer
	Syncer
	io.Closer
}
type Syncer interface {
	Sync(context.Context) error
}

type pooler bytespool.Pooler[[]byte, byte] //
var Pooler pooler = bytespool.Pool[[]byte, byte]{}
