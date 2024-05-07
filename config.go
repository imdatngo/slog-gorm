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
		errorField:                "error",
		slowThresholdField:        "slow_threshold",
		queryField:                "query",
		durationField:             "duration",
		rowsField:                 "rows",
		sourceField:               "file",
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

	groupKey           string
	errorField         string
	slowThresholdField string
	queryField         string
	durationField      string
	rowsField          string
	sourceField        string
	fullSourcePath     bool

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

// WithContextKeys to add custom log fields from context by given keys
//
// Map keys are the log fields, and map values are the context keys to extract with ctx.Value()
func (c *config) WithContextKeys(v map[string]any) *config {
	c.contextKeys = v
	return c
}

// WithContextExtractor to add custom log fields extracted from context by given function
func (c *config) WithContextExtractor(v func(ctx context.Context) []slog.Attr) *config {
	c.contextExtractor = v
	return c
}

// WithGroupKey set group name to group all the trace attributes, except the context attributes
func (c *config) WithGroupKey(v string) *config {
	c.groupKey = v
	return c
}

// WithErrorField set attribute name for error field. Default "error"
func (c *config) WithErrorField(v string) *config {
	c.errorField = v
	return c
}

// WithSlowThresholdField changes attribute name of slow threshold field. Default "slow_threshold"
func (c *config) WithSlowThresholdField(v string) *config {
	c.slowThresholdField = v
	return c
}

// WithQueryField changes attribute name of SQL query field. Default "query"
func (c *config) WithQueryField(v string) *config {
	c.queryField = v
	return c
}

// WithDurationField changes attribute name of duration field. Default "duration"
func (c *config) WithDurationField(v string) *config {
	c.durationField = v
	return c
}

// WithRowsField changes attribute name of rows affected field. Default "rows"
func (c *config) WithRowsField(v string) *config {
	c.rowsField = v
	return c
}

// WithSourceField changes attribute name of source field. Default "file"
func (c *config) WithSourceField(v string) *config {
	c.sourceField = v
	return c
}

// WithFullSourcePath whether to include full path in source field or just the file name. Default false
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
