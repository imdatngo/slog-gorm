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
			slogHandler:        slog.Default().Handler(),
			slowThreshold:      200 * time.Millisecond,
			errorField:         "error",
			slowThresholdField: "slow_threshold",
			contextKeys:        map[string]string{},
			queryField:         "query",
			durationField:      "duration",
			rowsField:          "rows",
			sourceField:        "file",
		}, cfg)
	})
}

func Test_config_clone(t *testing.T) {
	t.Run("all", func(t *testing.T) {
		cfg := NewConfig(slog.Default().Handler()).WithContextKeys(map[string]string{"context": "key"})
		newCfg := cfg.clone()
		assert.NotSame(t, cfg, newCfg)
		assert.Same(t, cfg.slogHandler, newCfg.slogHandler)
		assert.NotSame(t, cfg.slowThreshold, newCfg.slowThreshold)
		assert.NotSame(t, cfg.errorField, newCfg.errorField)
		assert.NotSame(t, cfg.contextKeys, newCfg.contextKeys)
	})
}
