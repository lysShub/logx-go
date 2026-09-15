package logx

import (
	"time"

	"github.com/lysShub/logx-go/handler"
)

type (
	Level  = handler.Level
	Attr   = handler.Attr
	Record = handler.Record
	Value  = handler.Value
)

const (
	LevelDebug = handler.LevelDebug
	LevelInfo  = handler.LevelInfo
	LevelWarn  = handler.LevelWarn
	LevelError = handler.LevelError
	LevelFatal = handler.LevelFatal
)

func NewRecord(t time.Time, level Level, msg string, pc uintptr) Record {
	return handler.NewRecord(t, level, msg, pc)
}
