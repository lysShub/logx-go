package writer

import (
	"context"
	"io"
)

type Writer interface {
	io.Writer
	Syncer
	io.Closer
}
type Syncer interface {
	Sync(context.Context) error
}
