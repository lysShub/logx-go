package logx

import (
	"log/slog"
	"time"
)

type (
	Level          = slog.Level
	Leveler        = slog.Leveler
	LevelVar       = slog.LevelVar
	Attr           = slog.Attr
	Record         = slog.Record
	Value          = slog.Value
	Kind           = slog.Kind
	Source         = slog.Source
	LogValuer      = slog.LogValuer
	SlogHandler    = slog.Handler
	HandlerOptions = slog.HandlerOptions
	TextHandler    = slog.TextHandler
	JSONHandler    = slog.JSONHandler
	MultiHandler   = slog.MultiHandler
)

const (
	LevelDebug Level = slog.LevelDebug
	LevelInfo  Level = slog.LevelInfo
	LevelWarn  Level = slog.LevelWarn
	LevelError Level = slog.LevelError
	LevelFatal Level = slog.LevelError + 4

	KindAny       = slog.KindAny
	KindBool      = slog.KindBool
	KindDuration  = slog.KindDuration
	KindFloat64   = slog.KindFloat64
	KindGroup     = slog.KindGroup
	KindInt64     = slog.KindInt64
	KindLogValuer = slog.KindLogValuer
	KindString    = slog.KindString
	KindTime      = slog.KindTime
	KindUint64    = slog.KindUint64

	TimeKey    = slog.TimeKey
	LevelKey   = slog.LevelKey
	MessageKey = slog.MessageKey
	SourceKey  = slog.SourceKey
	ErrorKey   = "err"
)

func NewRecord(t time.Time, level Level, msg string, pc uintptr) Record {
	return slog.NewRecord(t, level, msg, pc)
}

func AnyValue(v any) Value                { return slog.AnyValue(v) }
func BoolValue(v bool) Value              { return slog.BoolValue(v) }
func DurationValue(v time.Duration) Value { return slog.DurationValue(v) }
func Float64Value(v float64) Value        { return slog.Float64Value(v) }
func GroupValue(as ...Attr) Value         { return slog.GroupValue(as...) }
func Int64Value(v int64) Value            { return slog.Int64Value(v) }
func IntValue(v int) Value                { return slog.IntValue(v) }
func StringValue(v string) Value          { return slog.StringValue(v) }
func TimeValue(v time.Time) Value         { return slog.TimeValue(v) }
func Uint64Value(v uint64) Value          { return slog.Uint64Value(v) }

func Any(key string, value any) Attr            { return slog.Any(key, value) }
func Bool(key string, v bool) Attr              { return slog.Bool(key, v) }
func Duration(key string, v time.Duration) Attr { return slog.Duration(key, v) }
func Float64(key string, v float64) Attr        { return slog.Float64(key, v) }
func Group(key string, args ...any) Attr        { return slog.Group(key, args...) }
func GroupAttrs(key string, attrs ...Attr) Attr { return slog.GroupAttrs(key, attrs...) }
func Int(key string, value int) Attr            { return slog.Int(key, value) }
func Int64(key string, value int64) Attr        { return slog.Int64(key, value) }
func String(key, value string) Attr             { return slog.String(key, value) }
func Time(key string, v time.Time) Attr         { return slog.Time(key, v) }
func Uint64(key string, value uint64) Attr      { return slog.Uint64(key, value) }
func Msg(msg string) Attr                       { return slog.String(MessageKey, msg) }
func Err(err error) Attr                        { return slog.String(ErrorKey, err.Error()) }
