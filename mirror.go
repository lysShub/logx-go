package logx

import (
	"log/slog"
	"time"
)

type (
	Level  = slog.Level
	Attr   = slog.Attr
	Record = slog.Record
	Value  = slog.Value
)

const (
	LevelDebug Level = slog.LevelDebug
	LevelInfo  Level = slog.LevelInfo
	LevelWarn  Level = slog.LevelWarn
	LevelError Level = slog.LevelError
	LevelFatal Level = slog.LevelError + 4
)

func NewRecord(t time.Time, level Level, msg string, pc uintptr) Record {
	return slog.NewRecord(t, level, msg, pc)
}
