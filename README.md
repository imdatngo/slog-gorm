# slog-gorm

[![Release](https://img.shields.io/github/v/release/imdatngo/slog-gorm)](https://github.com/imdatngo/slog-gorm/releases)
![Go version](https://img.shields.io/github/go-mod/go-version/imdatngo/slog-gorm)
[![Go Reference](https://pkg.go.dev/badge/github.com/imdatngo/slog-gorm.svg)](https://pkg.go.dev/github.com/imdatngo/slog-gorm)
![Tests](https://github.com/imdatngo/slog-gorm/actions/workflows/tests.yml/badge.svg)
[![Go report](https://goreportcard.com/badge/github.com/imdatngo/slog-gorm)](https://goreportcard.com/report/github.com/imdatngo/slog-gorm)
[![codecov](https://codecov.io/gh/imdatngo/slog-gorm/graph/badge.svg?token=KM0Y198PUH)](https://codecov.io/gh/imdatngo/slog-gorm)
[![License](https://img.shields.io/github/license/imdatngo/slog-gorm)](./LICENSE)

slog handler for [Gorm](https://github.com/go-gorm/gorm), inspired by [orandin/slog-gorm](https://github.com/orandin/slog-gorm) with my own ideas to tailor it to my specific needs.

## 🚀 Install

```sh
go get github.com/imdatngo/slog-gorm
```

**Compatibility**: go >= 1.21

## 💡 Usage

### Minimal

See [config.go](./config.go) for default values.

```go
import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	sloggorm "github.com/imdatngo/slog-gorm"
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
// 2024/04/16 07:30:00 WARN Query SLOW duration=133.448µs rows=0 file=main.go:46 slow_threshold=100ns query="UPDATE `users` SET `age`=18 WHERE `id` = 1"
```

### With config

```go
// Your slog.Logger instance
slogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
// adding group or field to distinguish with application logs
// slogger = slogger.WithGroup("db")
// slogger = slogger.With(slog.String("logger", "db"))

// Create new slog-gorm instance with custom config
cfg := sloggorm.NewConfig(slogger.Handler()).
	WithSlowThreshold(time.Second).
	WithIgnoreRecordNotFoundError(true).
	WithTraceAll(true)
glogger := sloggorm.NewWithConfig(cfg)

// Sample output:
// time=2024-04-16T07:35:40.696Z level=INFO msg="Query OK" duration=130.659µs rows=1 file=main.go:45 query="SELECT * FROM `users` WHERE `users`.`id` = 1 ORDER BY `users`.`id` LIMIT 1"
// time=2024-04-16T07:35:40.697Z level=INFO msg="Query OK" duration=940.445µs rows=1 file=main.go:46 query="UPDATE `users` SET `age`=18 WHERE `id` = 1"
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
		// group the context fields as needed
		slog.Group("ctx",
			slog.String("trace_id", tracer.TraceId),
			slog.String("span_id", tracer.SpanId),
			slog.String("service", mypkg.FromCtx(ctx)),
		),
	}
})

// Sample output:
// time=2024-05-05T22:23:24.345Z level=INFO msg="Query OK" ctx.trace_id=014KG56DC01GG4TEB01ZEX7WFJ ctx.span_id=014KG56DC01GG4TEB022Z17KKS ctx.service=users duration=139.007µs rows=1 file=main.go:69 query="SELECT * FROM `users` WHERE `users`.`id` = 1 ORDER BY `users`.`id` LIMIT 1"
// time=2024-05-05T22:23:24.678Z level=INFO msg="Query OK" ctx.trace_id=014KG56DC01GG4TEB01ZEX7WFJ ctx.span_id=014KG56DC01GG4TEB022Z17KKS ctx.service=users duration=915.688µs rows=1 file=main.go:70 query="UPDATE `users` SET `age`=18 WHERE `id` = 1"
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
tx := db.Session(&gorm.Session{Logger: db.Logger.LogMode(gormlogger.Silent)})
```
