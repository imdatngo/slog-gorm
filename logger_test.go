package sloggorm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var sourceDir string

func init() {
	_, file, _, _ := runtime.Caller(0)
	sourceDir = filepath.Dir(file) + "/"
}

func TestNew(t *testing.T) {
	t.Run("default handler", func(t *testing.T) {
		l := New()
		assert.Equal(t, slog.Default().Handler(), l.slogHandler)
	})
}

func TestNewWithConfig(t *testing.T) {
	t.Run("custom config", func(t *testing.T) {
		h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})
		want := &config{
			slogHandler:               h,
			slowThreshold:             time.Second,
			ignoreRecordNotFoundError: true,
			parameterizedQueries:      true,
			silent:                    true,
			traceAll:                  true,
			contextKeys:               map[string]any{"req_id": "id"},
			groupKey:                  "db",
			errorKey:                  "err",
			slowThresholdKey:          "threshold",
			queryKey:                  "sql",
			durationKey:               "dur",
			rowsKey:                   "count",
			sourceKey:                 "src",
			fullSourcePath:            true,
			okMsg:                     "Yeah!",
			slowMsg:                   "Hmmm...",
			errorMsg:                  "Shit!!",
		}

		cfg := NewConfig(h).
			WithSlowThreshold(time.Second).
			WithIgnoreRecordNotFoundError(true).
			WithParameterizedQueries(true).
			WithSilent(true).
			WithTraceAll(true).
			WithContextKeys(map[string]any{"req_id": "id"}).
			WithGroupKey("db").
			WithErrorKey("err").
			WithSlowThresholdKey("threshold").
			WithQueryKey("sql").
			WithDurationKey("dur").
			WithRowsKey("count").
			WithSourceKey("src").
			WithFullSourcePath(true).
			WithOkMsg("Yeah!").
			WithSlowMsg("Hmmm...").
			WithErrorMsg("Shit!!")
		l := NewWithConfig(cfg)
		assert.Equal(t, want, l.config)
	})
}

func Test_logger_LogMode(t *testing.T) {
	t.Run("Silent", func(t *testing.T) {
		l := New()
		nl := l.LogMode(gormlogger.Silent).(*logger)
		assert.NotSame(t, l, nl)
		assert.Equal(t, false, l.traceAll)
		assert.Equal(t, false, l.silent)
		assert.Equal(t, false, nl.traceAll)
		assert.Equal(t, true, nl.silent)
	})
	t.Run("Info", func(t *testing.T) {
		l := New()
		nl := l.LogMode(gormlogger.Info).(*logger)
		assert.NotSame(t, l, nl)
		assert.Equal(t, false, l.traceAll)
		assert.Equal(t, false, l.silent)
		assert.Equal(t, true, nl.traceAll)
		assert.Equal(t, false, nl.silent)
	})
	t.Run("Warn", func(t *testing.T) {
		l := New()
		nl := l.LogMode(gormlogger.Warn).(*logger)
		assert.Same(t, l, nl)
		assert.Equal(t, false, nl.traceAll)
		assert.Equal(t, false, nl.silent)
	})
	t.Run("Error", func(t *testing.T) {
		l := New()
		nl := l.LogMode(gormlogger.Error).(*logger)
		assert.Same(t, l, nl)
		assert.Equal(t, false, nl.traceAll)
		assert.Equal(t, false, nl.silent)
	})
}

func Test_logger(t *testing.T) {
	var buf bytes.Buffer
	newHandler := func(_ *testing.T, lvl slog.Leveler) slog.Handler {
		buf.Reset()
		return slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: lvl})
	}
	result := func(t *testing.T) map[string]any {
		m := map[string]any{}
		if buf.Len() > 0 {
			if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
				t.Fatalf("there maybe more than one log line: %v\n%s", err, buf.Bytes())
			}
		}
		return m
	}

	tests := []struct {
		name   string
		logLvl slog.Leveler
		config func(slog.Handler) *config
		log    func(*logger)
		checks []check
	}{
		{
			name:   "info",
			config: NewConfig,
			log: func(l *logger) {
				l.Info(context.Background(), "info msg: %v", "hello world!")
			},
			checks: []check{
				hasKey(slog.TimeKey),
				hasAttr(slog.LevelKey, "INFO"),
				hasAttr(slog.MessageKey, "info msg: hello world!"),
			},
		},
		{
			name: "with context key",
			config: func(h slog.Handler) *config {
				return NewConfig(h).WithContextKeys(map[string]any{"req_id": ctxKey("id")})
			},
			log: func(l *logger) {
				ctx := context.Background()
				ctx = context.WithValue(ctx, ctxKey("id"), "123")
				l.Info(ctx, "hello world!")
			},
			checks: []check{
				hasKey(slog.TimeKey),
				hasAttr(slog.LevelKey, "INFO"),
				hasAttr(slog.MessageKey, "hello world!"),
				hasAttr("req_id", "123"),
			},
		},
		{
			name: "with context extractor",
			config: func(h slog.Handler) *config {
				return NewConfig(h).WithContextExtractor(func(ctx context.Context) []slog.Attr {
					return []slog.Attr{
						slog.String("trace_id", ctx.Value(ctxKey("trace_id")).(string)),
						slog.String("span_id", ctx.Value(ctxKey("span_id")).(string)),
					}
				})
			},
			log: func(l *logger) {
				ctx := context.Background()
				ctx = context.WithValue(ctx, ctxKey("trace_id"), "abc123")
				ctx = context.WithValue(ctx, ctxKey("span_id"), "112233")
				l.Info(ctx, "tracing")
			},
			checks: []check{
				hasKey(slog.TimeKey),
				hasAttr(slog.LevelKey, "INFO"),
				hasAttr(slog.MessageKey, "tracing"),
				hasAttr("trace_id", "abc123"),
				hasAttr("span_id", "112233"),
			},
		},
		{
			name:   "warn",
			logLvl: slog.LevelWarn,
			config: NewConfig,
			log: func(l *logger) {
				l.Info(context.Background(), "this should be %s", "ignored")
				l.Warn(context.Background(), "warn msg")
			},
			checks: []check{
				hasKey(slog.TimeKey),
				hasAttr(slog.LevelKey, "WARN"),
				hasAttr(slog.MessageKey, "warn msg"),
			},
		},
		{
			name:   "error",
			logLvl: slog.LevelError,
			config: NewConfig,
			log: func(l *logger) {
				l.Info(context.Background(), "this should be %s", "ignored")
				l.Warn(context.Background(), "warn msg is ignored as well")
				l.Error(nil, "no context")
			},
			checks: []check{
				hasKey(slog.TimeKey),
				hasAttr(slog.LevelKey, "ERROR"),
				hasAttr(slog.MessageKey, "no context"),
			},
		},
		{
			name:   "silent",
			logLvl: slog.LevelInfo,
			config: func(h slog.Handler) *config {
				return NewConfig(h).WithSilent(true)
			},
			log: func(l *logger) {
				l.Info(context.Background(), "this should be %s", "ignored")
				l.Warn(context.Background(), "warn msg is ignored as well")
				l.Error(context.Background(), "no error")
				l.Trace(context.TODO(), time.Now(), func() (string, int64) { return "", 0 }, fmt.Errorf("something"))
			},
			checks: []check{
				emptyLogs(),
			},
		},
		{
			name:   "trace error",
			config: NewConfig,
			log: func(l *logger) {
				fc := func() (string, int64) {
					return "SELECT * FROM users", 69
				}
				// this success query should be ignored by default
				l.Trace(context.TODO(), time.Now(), fc, nil)
				// error log
				l.Trace(context.Background(), time.Now().Add(-1*time.Second), fc, fmt.Errorf("connection error"))
			},
			checks: []check{
				hasKey(slog.TimeKey),
				hasAttr(slog.LevelKey, "ERROR"),
				hasAttr(slog.MessageKey, "Query ERROR"),
				hasAttr("error", "connection error"),
				elapsedApprox("duration", time.Second),
				hasAttr("rows", float64(69)), // json encoded to float64!
				hasSource("file", 9, false),
				hasAttr("query", "SELECT * FROM users"),
			},
		},
		{
			name: "trace slow query",
			config: func(h slog.Handler) *config {
				return NewConfig(h).WithSlowThreshold(2 * time.Second)
			},
			log: func(l *logger) {
				fc := func() (string, int64) {
					return "SELECT * FROM users WHERE true", 6969
				}
				l.Trace(context.Background(), time.Now().Add(-10*time.Second), fc, nil)
			},
			checks: []check{
				hasKey(slog.TimeKey),
				hasAttr(slog.LevelKey, "WARN"),
				hasAttr(slog.MessageKey, "Query SLOW"),
				hasAttr("slow_threshold", float64(2*time.Second)), // json encoded to float64!
				elapsedApprox("duration", 10*time.Second),
				hasAttr("rows", float64(6969)), // json encoded to float64!
				hasSource("file", 9, false),
				hasAttr("query", "SELECT * FROM users WHERE true"),
			},
		},
		{
			name: "trace all",
			config: func(h slog.Handler) *config {
				return NewConfig(h).WithTraceAll(true)
			},
			log: func(l *logger) {
				fc := func() (string, int64) {
					return "SELECT * FROM profiles", 0
				}
				l.Trace(context.Background(), time.Now().Add(-10*time.Millisecond), fc, nil)
			},
			checks: []check{
				hasKey(slog.TimeKey),
				hasAttr(slog.LevelKey, "INFO"),
				hasAttr(slog.MessageKey, "Query OK"),
				elapsedApprox("duration", 10*time.Millisecond),
				hasAttr("rows", float64(0)), // json encoded to float64!
				hasSource("file", 8, false),
				hasAttr("query", "SELECT * FROM profiles"),
			},
		},
		{
			name: "trace all with group",
			config: func(h slog.Handler) *config {
				return NewConfig(h).WithTraceAll(true).WithGroupKey("db")
			},
			log: func(l *logger) {
				fc := func() (string, int64) {
					return "SELECT * FROM profiles", 0
				}
				l.Trace(context.Background(), time.Now().Add(-10*time.Millisecond), fc, nil)
			},
			checks: []check{
				hasKey(slog.TimeKey),
				hasAttr(slog.LevelKey, "INFO"),
				hasAttr(slog.MessageKey, "Query OK"),
				hasGroupAttrs("db", []check{
					elapsedApprox("duration", 10*time.Millisecond),
					hasAttr("rows", float64(0)), // json encoded to float64!
					hasSource("file", 9, false),
					hasAttr("query", "SELECT * FROM profiles"),
				}),
			},
		},
		{
			name: "trace slow should not log error if not asked",
			config: func(h slog.Handler) *config {
				return NewConfig(h).WithIgnoreRecordNotFoundError(true)
			},
			log: func(l *logger) {
				fc := func() (string, int64) {
					return "SELECT * FROM users", 0
				}
				l.Trace(context.Background(), time.Now().Add(-1*time.Second), fc, gorm.ErrRecordNotFound)
			},
			checks: []check{
				hasAttr(slog.LevelKey, "WARN"),
				hasAttr(slog.MessageKey, "Query SLOW"),
				missingKey("error"),
				hasAttr("query", "SELECT * FROM users"),
			},
		},
		{
			name: "trace all should not log error if not asked",
			config: func(h slog.Handler) *config {
				return NewConfig(h).WithIgnoreRecordNotFoundError(true).WithTraceAll(true)
			},
			log: func(l *logger) {
				fc := func() (string, int64) {
					return "SELECT * FROM users", 0
				}
				l.Trace(context.Background(), time.Now().Add(-10*time.Millisecond), fc, gorm.ErrRecordNotFound)
			},
			checks: []check{
				hasAttr(slog.LevelKey, "INFO"),
				hasAttr(slog.MessageKey, "Query OK"),
				missingKey("error"),
				hasAttr("query", "SELECT * FROM users"),
			},
		},
		{
			name: "full source path",
			config: func(h slog.Handler) *config {
				return NewConfig(h).WithTraceAll(true).WithFullSourcePath(true)
			},
			log: func(l *logger) {
				fc := func() (string, int64) {
					return "SELECT * FROM sources", 1
				}
				l.Trace(context.Background(), time.Now().Add(-1*time.Millisecond), fc, nil)
			},
			checks: []check{
				hasSource("file", 3, true),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.logLvl == nil {
				tt.logLvl = slog.LevelInfo
			}
			h := newHandler(t, tt.logLvl)
			c := tt.config(h)
			l := NewWithConfig(c)

			tt.log(l)
			got := result(t)
			for _, check := range tt.checks {
				if err := check(got); err != "" {
					t.Errorf(err)
				}
			}
		})
	}
}

func Test_logger_ParamsFilter(t *testing.T) {
	type args struct {
		sql    string
		params []any
	}
	tests := []struct {
		name                 string
		parameterizedQueries bool
		args                 args
		wantSql              string
		wantVars             []any
	}{
		{
			name:                 "parameterized",
			parameterizedQueries: true,
			args: args{
				sql:    "SELECT * FROM users WHERE name = ?",
				params: []any{"imdatngo"},
			},
			wantSql:  "SELECT * FROM users WHERE name = ?",
			wantVars: nil,
		},
		{
			name:                 "not parameterized",
			parameterizedQueries: false,
			args: args{
				sql:    "SELECT * FROM users WHERE name = ?",
				params: []any{"imdatngo"},
			},
			wantSql:  "SELECT * FROM users WHERE name = ?",
			wantVars: []any{"imdatngo"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &logger{
				config: &config{parameterizedQueries: tt.parameterizedQueries},
			}
			gotSql, gotVars := l.ParamsFilter(context.Background(), tt.args.sql, tt.args.params...)
			assert.Equal(t, tt.wantSql, gotSql)
			assert.Equal(t, tt.wantVars, gotVars)
		})
	}
}

type ctxKey string

type check func(map[string]any) (err string)

func emptyLogs() check {
	return func(m map[string]any) string {
		if len(m) > 0 {
			return fmt.Sprintf("got %#v, want empty", m)
		}
		return ""
	}
}

func hasKey(key string) check {
	return func(m map[string]any) string {
		if _, ok := m[key]; !ok {
			return fmt.Sprintf("missing key %q", key)
		}
		return ""
	}
}

func missingKey(key string) check {
	return func(m map[string]any) string {
		if _, ok := m[key]; ok {
			return fmt.Sprintf("unexpected key %q", key)
		}
		return ""
	}
}

func elapsedApprox(key string, d time.Duration) check {
	return func(m map[string]any) string {
		if s := hasKey(key)(m); s != "" {
			return s
		}
		gotVal := time.Duration(m[key].(float64)).Truncate(time.Millisecond)
		if gotVal != d {
			return fmt.Sprintf("%q: got %#v, want %#v", key, gotVal, d)
		}
		return ""
	}
}

func hasAttr(key string, wantVal any) check {
	return func(m map[string]any) string {
		if s := hasKey(key)(m); s != "" {
			return s
		}
		gotVal := m[key]
		if !reflect.DeepEqual(gotVal, wantVal) {
			return fmt.Sprintf("%q: got %#v, want %#v", key, gotVal, wantVal)
		}
		return ""
	}
}

func hasGroupAttrs(key string, checks []check) check {
	return func(m map[string]any) string {
		if s := hasKey(key)(m); s != "" {
			return s
		}
		gotGroup := m[key].(map[string]any)
		for _, check := range checks {
			if err := check(gotGroup); err != "" {
				return err
			}
		}
		return ""
	}
}

func hasSource(key string, offset int, full bool) check {
	_, file, line, _ := runtime.Caller(1)
	if !full {
		file = path.Base(file)
	}
	return func(m map[string]any) string {
		if s := hasKey(key)(m); s != "" {
			return s
		}
		gotVal := m[key]
		wantVal := file + ":" + strconv.FormatInt(int64(line-offset), 10)
		if gotVal != wantVal {
			return fmt.Sprintf("%q: got %#v, want %#v", key, gotVal, wantVal)
		}

		return ""
	}
}
