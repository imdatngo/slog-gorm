package sloggorm

import (
	"context"
	"log/slog"
	"time"
)

// NewConfig creates a new config with the given non-nil slog.Handler
func NewConfig(h slog.Handler) *config {
	if h == nil {
		panic("nil Handler")
	}
	return &config{
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

// logger config
type config struct {
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
func (c *config) clone() *config {
	nc := *c
	nc.contextKeys = map[string]any{}
	for k, v := range c.contextKeys {
		nc.contextKeys[k] = v
	}
	return &nc
}

// WithSlowThreshold sets slow SQL threshold. Default 200ms
func (c *config) WithSlowThreshold(v time.Duration) *config {
	c.slowThreshold = v
	return c
}

// WithIgnoreRecordNotFoundError whether to skip ErrRecordNotFound error
func (c *config) WithIgnoreRecordNotFoundError(v bool) *config {
	c.ignoreRecordNotFoundError = v
	return c
}

// WithParameterizedQueries whether to include params in the SQL log
func (c *config) WithParameterizedQueries(v bool) *config {
	c.parameterizedQueries = v
	return c
}

// WithSilent whether to discard all logs
func (c *config) WithSilent(v bool) *config {
	c.silent = v
	return c
}

// WithTraceAll whether to include OK queries in logs
func (c *config) WithTraceAll(v bool) *config {
	c.traceAll = v
	return c
}

// WithContextKeys to add custom log attributes from context by given keys
//
// Map keys are the attribute name, and map values are the context keys to extract with ctx.Value()
func (c *config) WithContextKeys(v map[string]any) *config {
	c.contextKeys = v
	return c
}

// WithContextExtractor to add custom log attributes extracted from context by given function
func (c *config) WithContextExtractor(v func(ctx context.Context) []slog.Attr) *config {
	c.contextExtractor = v
	return c
}

// WithGroupKey set group name to group all the trace attributes, except the context attributes. Default is empty, i.e. no grouping
func (c *config) WithGroupKey(v string) *config {
	c.groupKey = v
	return c
}

// WithErrorKey set different name for error attribute, set empty value to drop it. Default "error"
func (c *config) WithErrorKey(v string) *config {
	c.errorKey = v
	return c
}

// WithSlowThresholdKey set different name for slow threshold attribute, set empty value to drop it. Default "slow_threshold"
func (c *config) WithSlowThresholdKey(v string) *config {
	c.slowThresholdKey = v
	return c
}

// WithQueryKey set different name for SQL query attribute, set empty value to drop it. Default "query"
func (c *config) WithQueryKey(v string) *config {
	c.queryKey = v
	return c
}

// WithDurationKey set different name for duration attribute, set empty value to drop it. Default "duration"
func (c *config) WithDurationKey(v string) *config {
	c.durationKey = v
	return c
}

// WithRowsKey set different name for rows affected attribute, set empty value to drop it. Default "rows"
func (c *config) WithRowsKey(v string) *config {
	c.rowsKey = v
	return c
}

// WithSourceKey set different name for source attribute, set empty value to drop it. Default "file"
func (c *config) WithSourceKey(v string) *config {
	c.sourceKey = v
	return c
}

// WithFullSourcePath whether to include full path in source attribute or just the file name. Default false
func (c *config) WithFullSourcePath(v bool) *config {
	c.fullSourcePath = v
	return c
}

// WithOkMsg changes log message for successful query. Default "Query OK"
func (c *config) WithOkMsg(v string) *config {
	c.okMsg = v
	return c
}

// WithSlowMsg changes log message for slow query. Default "Query SLOW"
func (c *config) WithSlowMsg(v string) *config {
	c.slowMsg = v
	return c
}

// WithErrorMsg changes log message for failed query. Default "Query ERROR"
func (c *config) WithErrorMsg(v string) *config {
	c.errorMsg = v
	return c
}
