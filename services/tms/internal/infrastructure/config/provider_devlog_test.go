//go:build devlog

package config

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/loggingconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestProvideLogger_PrettyFormatWritesOneKeyValueLinePerEntry(t *testing.T) {
	t.Parallel()

	cfg := newValidConfig()
	logPath := filepath.Join(t.TempDir(), "pretty.log")
	cfg.Logging.Format = "pretty"
	cfg.Logging.Color = "auto"
	cfg.Logging.Output = "file"
	cfg.Logging.File = testLogFileConfig(logPath)

	logger, sink, err := ProvideLogger(&cfg.Logging)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sink.Close() })

	logger.With(zap.String("org_id", "org_1")).Info("pretty message", zap.Int("attempt", 2))
	require.NoError(t, logger.Sync())

	assert.Regexp(
		t,
		`^\d{2}:\d{2}:\d{2}\.\d{3} INF config/provider_devlog_test\.go:\d+ > pretty message org_id=org_1 attempt=2$`,
		readFirstLogLine(t, logPath),
	)
}

func TestProvideLogger_PrettyFormatColorsAFileWhenColorIsAlways(t *testing.T) {
	t.Parallel()

	cfg := newValidConfig()
	logPath := filepath.Join(t.TempDir(), "pretty-color.log")
	cfg.Logging.Format = "pretty"
	cfg.Logging.Color = "always"
	cfg.Logging.Output = "file"
	cfg.Logging.File = testLogFileConfig(logPath)

	logger, sink, err := ProvideLogger(&cfg.Logging)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sink.Close() })

	logger.Warn("colored message")
	require.NoError(t, logger.Sync())

	line := readFirstLogLine(t, logPath)
	assert.True(t, strings.HasPrefix(line, "\x1b[2m"), line)
	assert.Contains(t, line, " \x1b[33mWRN\x1b[0m ")
}

func TestPrettyColor_ExplicitModesWinAndAutoNeedsATerminal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		color  string
		output string
		want   bool
	}{
		{name: "always colors a file", color: "always", output: "file", want: true},
		{name: "never leaves stdout plain", color: "never", output: "stdout", want: false},
		{name: "auto leaves a file plain", color: "auto", output: "file", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := prettyColor(loggingconfig.Config{Color: tt.color, Output: tt.output})

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPrettyColor_AutoHonoursNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	assert.False(t, prettyColor(loggingconfig.Config{Color: "auto", Output: "stdout"}))
	assert.False(t, prettyColor(loggingconfig.Config{Color: "auto", Output: "stderr"}))
}
