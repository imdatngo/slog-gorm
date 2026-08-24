# slog-gorm

[![Release](https://img.shields.io/github/v/release/imdatngo/slog-gorm)](https://github.com/imdatngo/slog-gorm/releases)
![Go version](https://img.shields.io/github/go-mod/go-version/imdatngo/slog-gorm)
[![Go Reference](https://pkg.go.dev/badge/github.com/imdatngo/slog-gorm/v2.svg)](https://pkg.go.dev/github.com/imdatngo/slog-gorm/v2)
![Tests](https://github.com/imdatngo/slog-gorm/actions/workflows/tests.yml/badge.svg)
[![Go report](https://goreportcard.com/badge/github.com/imdatngo/slog-gorm)](https://goreportcard.com/report/github.com/imdatngo/slog-gorm)
[![codecov](https://codecov.io/gh/imdatngo/slog-gorm/graph/badge.svg?token=KM0Y198PUH)](https://codecov.io/gh/imdatngo/slog-gorm)
[![License](https://img.shields.io/github/license/imdatngo/slog-gorm)](./LICENSE)

[Gorm Logger](https://gorm.io/docs/logger.html) implemented with slog, built for logs that stay useful once they reach your log pipeline: structured-native attributes, no SQL rendering for queries you never log, and full control over every key and message.

Originally inspired by [orandin/slog-gorm](https://github.com/orandin/slog-gorm), with my own ideas to tailor it to my specific needs.

> **Already on gorm >= v1.31.0?** Gorm now ships its own `logger.NewSlogLogger`. If all you need is for gorm's output to reach slog, use it — it costs you no extra dependency. This library is for when those logs have to be queried, aggregated and correlated. See [the comparison below](#-vs-gorms-built-in-newsloglogger).

## 🚀 Install

```sh
go get github.com/imdatngo/slog-gorm/v2
```

**Compatibility**: go >= 1.25 — see [go.mod](./go.mod) for the gorm version this release is built and tested against.

## 🤔 vs gorm's built-in `NewSlogLogger`

Gorm added `logger.NewSlogLogger` in [v1.31.0](https://github.com/go-gorm/gorm/releases/tag/v1.31.0). It is a thin adapter: it gets gorm's output into slog and stops there. The differences that matter in production:

| | gorm `NewSlogLogger` | slog-gorm |
| --- | --- | --- |
| Source of truth for log level | gorm's `LogLevel` **and** the slog level, both must agree | the slog handler, alone |
| `duration` | pre-formatted string, `"3.000ms"` | `slog.Duration`, the handler decides |
| `error` | flattened via `err.Error()` | `slog.Any`, error value preserved |
| SQL built for logs that get dropped | on every query | never |
| Gorm's own `%v` messages | passed through unformatted | formatted |
| Attribute keys and messages | hardcoded | all renameable, any droppable |
| Grouping | always nested under `trace` | opt-in, `WithGroupKey` |
| Query source location | record PC only, needs `AddSource` | dedicated `file` attribute, always on |
| Context attributes | none | `WithContextKeys` / `WithContextExtractor` |

### Your log backend can actually read this

The same successful query through a `slog.NewJSONHandler` (timestamps elided):

```jsonc
// gorm's NewSlogLogger — duration is a string, so no thresholding or aggregating without a parse step
{"level":"INFO","msg":"SQL executed","trace":{"duration":"3.000ms","sql":"SELECT * FROM users WHERE id = 1","rows":1}}

// slog-gorm — duration is a number, fields are top-level and named by you
{"level":"INFO","msg":"Query OK","duration":3000194,"rows":1,"file":"main.go:45","query":"SELECT * FROM users WHERE id = 1"}
```

### It doesn't pay for logs you throw away

The usual production setup logs slow queries and errors only. Gorm's logger renders the SQL string before it decides whether to log anything, so healthy queries pay full SQL construction — parameter interpolation included — and the result is discarded. Over 100 fast, successful queries against a handler set to `slog.LevelWarn`, both emit nothing, but only one does the work:

```
SQL rendered  ->  gorm's NewSlogLogger: 100     slog-gorm: 0
bytes emitted ->  gorm's NewSlogLogger: 0       slog-gorm: 0
```

### Gorm's own messages stay readable

Gorm calls its logger with printf-style format strings. `NewSlogLogger` puts the arguments in a `data` attribute and never applies them:

```
# gorm's NewSlogLogger
level=ERROR msg="failed to initialize database, got error %v" data="[dial tcp: connection refused]"

# slog-gorm
level=ERROR msg="failed to initialize database, got error dial tcp: connection refused"
```

> Some of the above are design differences that follow from gorm's backward-compatibility constraints — the dual log level in particular is baked into `logger.Interface`. Others are simply how the current implementation behaves and may well be fixed upstream. Verified against gorm v1.31.2.
>
> One caveat in the other direction: slog-gorm reports the query's origin through its own `file` attribute, which is always present and needs no handler options. If you additionally enable `AddSource` on your handler, the built-in `source` field will point inside gorm rather than at your code — use `file`.

## 💡 Usage

### Minimal

See [config.go](./config.go) for default values.

```go
import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	sloggorm "github.com/imdatngo/slog-gorm/v2"
)

// Create new slog-gorm instance with slog.Default()
glogger := sloggorm.New()

// Global mode
db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{
	Logger: glogger,
})

// Continuous session mode
tx := db.Session(&gorm.Session{Logger: glogger})
tx.First(&user)
tx.Model(&user).Update("Age", 18)

// Sample output:
// 2024/04/16 07:30:00 ERROR Query ERROR duration=128.364µs rows=0 file=main.go:45 error="record not found" query="SELECT * FROM `users` ORDER BY `users`.`id` LIMIT 1"
// 2024/04/16 07:30:00 WARN Query SLOW duration=234.117ms rows=1 file=main.go:46 slow_threshold=200ms query="UPDATE `users` SET `age`=18 WHERE `id` = 1"
```

### With config

```go
// Your slog.Logger instance
slogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

// Create new slog-gorm instance with custom config
cfg := sloggorm.NewConfig(slogger.Handler()).
	WithGroupKey("db").
	WithSlowThreshold(time.Second).
	WithIgnoreRecordNotFoundError(true).
	WithTraceAll(true)
glogger := sloggorm.NewWithConfig(cfg)

// Sample output:
// time=2024-04-16T07:35:40.696Z level=INFO msg="Query OK" db.duration=130.659µs db.rows=1 db.file=main.go:45 db.query="SELECT * FROM `users` WHERE `users`.`id` = 1 ORDER BY `users`.`id` LIMIT 1"
// time=2024-04-16T07:35:40.697Z level=INFO msg="Query OK" db.duration=940.445µs db.rows=1 db.file=main.go:46 db.query="UPDATE `users` SET `age`=18 WHERE `id` = 1"
```

### With Context

When you got the context keys:

```go
cfg.WithContextKeys(map[string]any{
	"trace_id": pkg.TraceIdCtxKey,
	"span_id":  pkg.SpanIdCtxKey,
	"service": "serviceName", // built-in type is not recommended though
})
```

or when some packages have it own extractor:

```go
cfg.WithContextExtractor(func(ctx context.Context) []slog.Attr {
	tracer := pkg.FromContext(ctx)
	return []slog.Attr{
		// group the context attributes as needed
		slog.Group("ctx",
			slog.String("trace_id", tracer.TraceId),
			slog.String("span_id", tracer.SpanId),
			slog.String("service", mypkg.FromCtx(ctx)),
		),
	}
})

// Sample output:
// time=2024-05-05T22:23:24.345Z level=INFO msg="Query OK" ctx.trace_id=014KG56DC01GG4TEB01ZEX7WFJ ctx.span_id=014KG56DC01GG4TEB022Z17KKS ctx.service=users db.duration=139.007µs db.rows=1 db.file=main.go:69 db.query="SELECT * FROM `users` WHERE `users`.`id` = 1 ORDER BY `users`.`id` LIMIT 1"
// time=2024-05-05T22:23:24.678Z level=INFO msg="Query OK" ctx.trace_id=014KG56DC01GG4TEB01ZEX7WFJ ctx.span_id=014KG56DC01GG4TEB022Z17KKS ctx.service=users db.duration=915.688µs db.rows=1 db.file=main.go:70 db.query="UPDATE `users` SET `age`=18 WHERE `id` = 1"
```

### Silence!

The slow queries and errors are logged by default, to discard all logs:

```go
cfg := sloggorm.NewConfig(slogger.Handler()).WithSilent(true)
glogger := sloggorm.NewWithConfig(cfg)
```

To on/off in session mode:

```go
// Start gorm's debug mode which is equivalent to cfg.WithTraceAll(true)
db.Debug().First(&User{})
// similar to new session
newLogger := glogger.LogMode(gormlogger.Info)
tx := db.Session(&gorm.Session{Logger: newLogger})

// or discard all logs for a session
silentTx := db.Session(&gorm.Session{Logger: db.Logger.LogMode(gormlogger.Silent)})
```

> **`LogMode` only responds to `Info` and `Silent`.** Passing `gormlogger.Error` or `gormlogger.Warn` returns the logger unchanged, by design — which of those two levels actually gets written is your slog handler's decision, not gorm's. So `LogMode(gormlogger.Warn)` will *not* quiet a logger that is already tracing every query; use `LogMode(gormlogger.Silent)` or set the level on the handler.

### Keep parameter values out of your logs

By default the logged SQL has its parameters interpolated, which puts user data — emails, tokens, anything in a `WHERE` clause — into your log pipeline. To log the statement with its placeholders intact instead:

```go
cfg := sloggorm.NewConfig(slogger.Handler()).WithParameterizedQueries(true)
glogger := sloggorm.NewWithConfig(cfg)

// Sample output, values no longer inlined (placeholder syntax depends on your driver):
// time=2024-04-16T07:35:40.696Z level=WARN msg="Query SLOW" duration=234.117ms rows=1 file=main.go:46 slow_threshold=200ms query="UPDATE `users` SET `age`=? WHERE `id` = ?"
```
