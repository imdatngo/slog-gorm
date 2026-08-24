package sloggorm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"path"
	"runtime"
	"slices"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

// New creates a new logger with default config.
//
// It snapshots slog.Default().Handler() at construction, so call it after any
// slog.SetDefault, or use NewWithConfig with the handler of your choice.
func New() *Logger {
	return NewWithConfig(NewConfig(slog.Default().Handler()))
}

// NewWithConfig creates a new logger with the given non-nil config.
//
// The config is cloned, so changes made to it afterwards do not affect the logger.
func NewWithConfig(config *Config) *Logger {
	if config == nil {
		panic("nil config")
	}
	return &Logger{
		config.clone(),
	}
}

// Logger is a gorm logger.Interface implementation backed by a slog.Handler.
type Logger struct {
	*Config
}

// ensure our logger implements gormlogger.Interface
var _ gormlogger.Interface = (*Logger)(nil)

// LogMode log mode
func (l *Logger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	// This function will switch to logging all queries, whenever the level is set to Info.
	// It's to support the Debug() function of gorm which sets the log level to info for subsequent queries, see:
	//   https://gorm.io/docs/session.html#Debug

	// Note: Error and Warn levels don't change the verbosity as the log level is managed by slog already,
	// except that they clear silent mode: per gorm's contract they are more verbose than Silent.
	if (level == gormlogger.Error || level == gormlogger.Warn) && !l.silent {
		return l
	}

	// clone logger for session mode
	nc := l.clone().WithTraceAll(level == gormlogger.Info).WithSilent(level == gormlogger.Silent)
	return &Logger{nc}
}

// Info logs info message
func (l *Logger) Info(ctx context.Context, format string, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	if l.enabled(ctx, slog.LevelInfo) {
		l.log(ctx, slog.LevelInfo, fmt.Sprintf(format, args...), l.contextAttrs(ctx)...)
	}
}

// Warn logs warn message
func (l *Logger) Warn(ctx context.Context, format string, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	if l.enabled(ctx, slog.LevelWarn) {
		l.log(ctx, slog.LevelWarn, fmt.Sprintf(format, args...), l.contextAttrs(ctx)...)
	}
}

// Error logs error message
func (l *Logger) Error(ctx context.Context, format string, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}
	if l.enabled(ctx, slog.LevelError) {
		l.log(ctx, slog.LevelError, fmt.Sprintf(format, args...), l.contextAttrs(ctx)...)
	}
}

// Trace logs sql message
func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.silent {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	elapsed := time.Since(begin)
	var (
		level    slog.Level
		msg      string
		traceErr error
		slow     bool
	)
	switch {
	case err != nil && l.enabled(ctx, slog.LevelError) && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.ignoreRecordNotFoundError):
		level, msg, traceErr = slog.LevelError, l.errorMsg, err
	case l.slowThreshold != 0 && elapsed > l.slowThreshold && l.enabled(ctx, slog.LevelWarn):
		level, msg, slow = slog.LevelWarn, l.slowMsg, true
	case l.traceAll && l.enabled(ctx, slog.LevelInfo):
		level, msg = slog.LevelInfo, l.okMsg
	default:
		return
	}

	// utils.FileWithLineNum walks the stack relative to this frame, so it must be
	// called directly from Trace, and only when the source attribute is wanted.
	var file string
	if l.sourceKey != "" {
		file = utils.FileWithLineNum()
	}
	l.log(ctx, level, msg, l.traceAttrs(ctx, elapsed, fc, file, traceErr, slow)...)
}

// ParamsFilter filter params
func (l *Logger) ParamsFilter(_ context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.parameterizedQueries {
		return sql, nil
	}
	return sql, params
}

// log adds context attributes and logs a message with the given slog level
func (l *Logger) log(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(3, pcs[:])
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.AddAttrs(attrs...)

	_ = l.slogHandler.Handle(ctx, r)
}

func (l *Logger) traceAttrs(ctx context.Context, elapsed time.Duration, fc func() (string, int64), file string, err error, slow bool) []slog.Attr {
	sql, rows := fc()

	attrs := make([]slog.Attr, 0, 5)

	if l.durationKey != "" {
		attrs = append(attrs, slog.Duration(l.durationKey, elapsed))
	}
	if rows >= 0 && l.rowsKey != "" { // rows could be -1
		attrs = append(attrs, slog.Int64(l.rowsKey, rows))
	}
	if l.sourceKey != "" {
		if l.fullSourcePath {
			attrs = append(attrs, slog.String(l.sourceKey, file))
		} else {
			attrs = append(attrs, slog.String(l.sourceKey, path.Base(file)))
		}
	}
	if err != nil && l.errorKey != "" {
		attrs = append(attrs, slog.Any(l.errorKey, err))
	} else if slow && l.slowThresholdKey != "" {
		attrs = append(attrs, slog.Duration(l.slowThresholdKey, l.slowThreshold))
	}
	if l.queryKey != "" {
		attrs = append(attrs, slog.String(l.queryKey, sql))
	}

	if l.groupKey != "" {
		return append(l.contextAttrs(ctx), slog.Attr{Key: l.groupKey, Value: slog.GroupValue(attrs...)})
	}

	return append(l.contextAttrs(ctx), attrs...)
}

// contextAttrs extracts attributes from context
func (l *Logger) contextAttrs(ctx context.Context) []slog.Attr {
	var attrs []slog.Attr
	if l.contextExtractor != nil {
		attrs = l.contextExtractor(ctx)
	}
	// sorted for a deterministic attribute order across log lines
	for _, ak := range slices.Sorted(maps.Keys(l.contextKeys)) {
		if val := ctx.Value(l.contextKeys[ak]); val != nil {
			attrs = append(attrs, slog.Any(ak, val))
		}
	}
	return attrs
}

// enabled reports whether the logger is enabled at the given level
func (l *Logger) enabled(ctx context.Context, lvl slog.Level) bool {
	return !l.silent && l.slogHandler.Enabled(ctx, lvl)
}
