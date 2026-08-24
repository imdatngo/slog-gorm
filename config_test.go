package sloggorm

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	t.Run("panic with nil handler", func(t *testing.T) {
		assert.PanicsWithValue(t, "nil Handler", func() { NewConfig(nil) })
	})

	t.Run("default config", func(t *testing.T) {
		var cfg *config
		assert.NotPanics(t, func() { cfg = NewConfig(slog.Default().Handler()) })
		assert.Equal(t, &config{
			slogHandler:      slog.Default().Handler(),
			slowThreshold:    200 * time.Millisecond,
			errorKey:         "error",
			slowThresholdKey: "slow_threshold",
			contextKeys:      map[string]any{},
			queryKey:         "query",
			durationKey:      "duration",
			rowsKey:          "rows",
			sourceKey:        "file",
			okMsg:            "Query OK",
			slowMsg:          "Query SLOW",
			errorMsg:         "Query ERROR",
		}, cfg)
	})
}

func Test_config_clone(t *testing.T) {
	t.Run("all", func(t *testing.T) {
		cfg := NewConfig(slog.Default().Handler()).WithContextKeys(map[string]any{"context": "key"})
		newCfg := cfg.clone()
		assert.NotSame(t, cfg, newCfg)
		assert.Same(t, cfg.slogHandler, newCfg.slogHandler)
		assert.Equal(t, cfg.slowThreshold, newCfg.slowThreshold)
		assert.Equal(t, cfg.errorKey, newCfg.errorKey)
		// contextKeys is deep copied: same contents, independent storage
		assert.Equal(t, cfg.contextKeys, newCfg.contextKeys)
		newCfg.contextKeys["added"] = "value"
		assert.NotContains(t, cfg.contextKeys, "added")
	})
}
