package sloggorm

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"path"
	"runtime"
	"strconv"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type gormUser struct {
	ID   uint
	Name string
}

// lastLog decodes the last JSON log line written to buf, then resets it.
func lastLog(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.NotEmpty(t, lines[len(lines)-1], "no log lines written")
	m := map[string]any{}
	require.NoError(t, json.Unmarshal(lines[len(lines)-1], &m))
	buf.Reset()
	return m
}

// Test_logger_withGorm wires the logger into a real gorm DB, so the whole chain is
// exercised: gorm's callbacks calling Trace, and utils.FileWithLineNum skipping
// gorm's own frames to land on this file. This guards the stack-depth invariant
// that unit tests calling Trace directly cannot fully cover.
func Test_logger_withGorm(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithConfig(NewConfig(slog.NewJSONHandler(&buf, nil)).WithTraceAll(true))

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: l})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&gormUser{}))
	buf.Reset()

	t.Run("successful query", func(t *testing.T) {
		_, file, line, _ := runtime.Caller(0)
		require.NoError(t, db.Create(&gormUser{Name: "gopher"}).Error) // must stay directly below the runtime.Caller line
		m := lastLog(t, &buf)
		assert.Equal(t, "INFO", m[slog.LevelKey])
		assert.Equal(t, "Query OK", m[slog.MessageKey])
		assert.Equal(t, path.Base(file)+":"+strconv.Itoa(line+1), m["file"])
		assert.Equal(t, float64(1), m["rows"])
		assert.Contains(t, m["query"], "INSERT INTO `gorm_users`")
	})

	t.Run("record not found", func(t *testing.T) {
		err := db.First(&gormUser{}, 12345).Error
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		m := lastLog(t, &buf)
		assert.Equal(t, "ERROR", m[slog.LevelKey])
		assert.Equal(t, "Query ERROR", m[slog.MessageKey])
		assert.Equal(t, "record not found", m["error"])
		assert.Contains(t, m["file"], path.Base("logger_gorm_test.go"))
	})

	t.Run("debug session traces with parent logger silent trace-all off", func(t *testing.T) {
		quiet := NewWithConfig(NewConfig(slog.NewJSONHandler(&buf, nil)))
		tx := db.Session(&gorm.Session{Logger: quiet})
		require.NoError(t, tx.Create(&gormUser{Name: "quiet"}).Error)
		assert.Zero(t, buf.Len(), "successful query should not be logged without trace-all")

		require.NoError(t, tx.Debug().Create(&gormUser{Name: "loud"}).Error)
		m := lastLog(t, &buf)
		assert.Equal(t, "Query OK", m[slog.MessageKey])
	})
}
