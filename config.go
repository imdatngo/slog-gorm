package sloggorm

import (
	"context"
	"log/slog"
	"time"
)

// NewConfig creates a new config with the given non-nil slog.Handler
func NewConfig(h slog.Handler) *Config {
	if h == nil {
		panic("nil Handler")
	}
	return &Config{
		slogHandler:               h,
		slowThreshold:             200 * time.Millisecond,
		ignoreRecordNotFoundError: false,
		parameterizedQueries:      false,
		silent:                    false,
		traceAll:                  false,
		contextKeys:               map[string]any{},
		contextExtractor:          nil,
		groupKey:                  "",
		errorKey:                  "error",
		slowThresholdKey:          "slow_threshold",
		queryKey:                  "query",
		durationKey:               "duration",
		rowsKey:                   "rows",
		sourceKey:                 "file",
		fullSourcePath:            false,
		okMsg:                     "Query OK",
		slowMsg:                   "Query SLOW",
		errorMsg:                  "Query ERROR",
	}
}

// Config is the logger configuration, to be built with NewConfig and the With* methods.
//
// A Config is not safe for concurrent use: build it in one goroutine, then hand it to
// NewWithConfig, which snapshots it. Changes made after that do not affect the logger.
type Config struct {
	slogHandler slog.Handler

	slowThreshold             time.Duration
	ignoreRecordNotFoundError bool
	parameterizedQueries      bool
	silent                    bool
	traceAll                  bool

	contextKeys      map[string]any
	contextExtractor func(ctx context.Context) []slog.Attr

	groupKey         string
	errorKey         string
	slowThresholdKey string
	queryKey         string
	durationKey      string
	rowsKey          string
	sourceKey        string
	fullSourcePath   bool

	okMsg    string
	slowMsg  string
	errorMsg string
}

// clone returns a new config with same values
func (c *Config) clone() *Config {
	nc := *c
	nc.contextKeys = map[string]any{}
	for k, v := range c.contextKeys {
		nc.contextKeys[k] = v
	}
	return &nc
}

// WithSlowThreshold sets slow SQL threshold, 0 to disable slow query logging. Default 200ms
func (c *Config) WithSlowThreshold(v time.Duration) *Config {
	c.slowThreshold = v
	return c
}

// WithIgnoreRecordNotFoundError whether to skip ErrRecordNotFound error
func (c *Config) WithIgnoreRecordNotFoundError(v bool) *Config {
	c.ignoreRecordNotFoundError = v
	return c
}

// WithParameterizedQueries whether to include params in the SQL log
func (c *Config) WithParameterizedQueries(v bool) *Config {
	c.parameterizedQueries = v
	return c
}

// WithSilent whether to discard all logs
func (c *Config) WithSilent(v bool) *Config {
	c.silent = v
	return c
}

// WithTraceAll whether to include OK queries in logs
func (c *Config) WithTraceAll(v bool) *Config {
	c.traceAll = v
	return c
}

// WithContextKeys to add custom log attributes from context by given keys
//
// Map keys are the attribute name, and map values are the context keys to extract with ctx.Value()
func (c *Config) WithContextKeys(v map[string]any) *Config {
	c.contextKeys = v
	return c
}

// WithContextExtractor to add custom log attributes extracted from context by given function
func (c *Config) WithContextExtractor(v func(ctx context.Context) []slog.Attr) *Config {
	c.contextExtractor = v
	return c
}

// WithGroupKey set group name to group all the trace attributes, except the context attributes. Default is empty, i.e. no grouping
func (c *Config) WithGroupKey(v string) *Config {
	c.groupKey = v
	return c
}

// WithErrorKey set different name for error attribute, set empty value to drop it. Default "error"
func (c *Config) WithErrorKey(v string) *Config {
	c.errorKey = v
	return c
}

// WithSlowThresholdKey set different name for slow threshold attribute, set empty value to drop it. Default "slow_threshold"
func (c *Config) WithSlowThresholdKey(v string) *Config {
	c.slowThresholdKey = v
	return c
}

// WithQueryKey set different name for SQL query attribute, set empty value to drop it. Default "query"
func (c *Config) WithQueryKey(v string) *Config {
	c.queryKey = v
	return c
}

// WithDurationKey set different name for duration attribute, set empty value to drop it. Default "duration"
func (c *Config) WithDurationKey(v string) *Config {
	c.durationKey = v
	return c
}

// WithRowsKey set different name for rows affected attribute, set empty value to drop it. Default "rows"
func (c *Config) WithRowsKey(v string) *Config {
	c.rowsKey = v
	return c
}

// WithSourceKey set different name for source attribute, set empty value to drop it. Default "file"
func (c *Config) WithSourceKey(v string) *Config {
	c.sourceKey = v
	return c
}

// WithFullSourcePath whether to include full path in source attribute or just the file name. Default false
func (c *Config) WithFullSourcePath(v bool) *Config {
	c.fullSourcePath = v
	return c
}

// WithOkMsg changes log message for successful query. Default "Query OK"
func (c *Config) WithOkMsg(v string) *Config {
	c.okMsg = v
	return c
}

// WithSlowMsg changes log message for slow query. Default "Query SLOW"
func (c *Config) WithSlowMsg(v string) *Config {
	c.slowMsg = v
	return c
}

// WithErrorMsg changes log message for failed query. Default "Query ERROR"
func (c *Config) WithErrorMsg(v string) *Config {
	c.errorMsg = v
	return c
}
