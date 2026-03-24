package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvideLogger_BuildsForStandardOutputs(t *testing.T) {
	t.Parallel()

	t.Run("stdout", func(t *testing.T) {
		t.Parallel()

		cfg := newValidConfig()
		cfg.Logging.Output = "stdout"

		logger, err := ProvideLogger(cfg)

		require.NoError(t, err)
		require.NotNil(t, logger)
	})

	t.Run("stderr", func(t *testing.T) {
		t.Parallel()

		cfg := newValidConfig()
		cfg.Logging.Output = "stderr"

		logger, err := ProvideLogger(cfg)

		require.NoError(t, err)
		require.NotNil(t, logger)
	})
}

func TestProvideLogger_FileOutputWritesLog(t *testing.T) {
	t.Parallel()

	cfg := newValidConfig()
	logPath := filepath.Join(t.TempDir(), "nested", "trenova.log")
	cfg.Logging.Output = "file"
	cfg.Logging.File = testLogFileConfig(logPath)

	logger, err := ProvideLogger(cfg)
	require.NoError(t, err)

	logger.Info("file log message")
	require.NoError(t, logger.Sync())

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "file log message")
}

func TestProvideLogger_UsesJSONFormat(t *testing.T) {
	t.Parallel()

	cfg := newValidConfig()
	logPath := filepath.Join(t.TempDir(), "json.log")
	cfg.Logging.Format = "json"
	cfg.Logging.Output = "file"
	cfg.Logging.File = testLogFileConfig(logPath)

	logger, err := ProvideLogger(cfg)
	require.NoError(t, err)

	logger.Info("json message")
	require.NoError(t, logger.Sync())

	line := readFirstLogLine(t, logPath)
	assert.True(t, strings.HasPrefix(line, "{"))
	assert.Contains(t, line, `"msg":"json message"`)
}

func TestProvideLogger_UsesTextFormat(t *testing.T) {
	t.Parallel()

	cfg := newValidConfig()
	logPath := filepath.Join(t.TempDir(), "text.log")
	cfg.Logging.Format = "text"
	cfg.Logging.Output = "file"
	cfg.Logging.File = testLogFileConfig(logPath)

	logger, err := ProvideLogger(cfg)
	require.NoError(t, err)

	logger.Info("text message")
	require.NoError(t, logger.Sync())

	line := readFirstLogLine(t, logPath)
	assert.False(t, strings.HasPrefix(line, "{"))
	assert.Contains(t, line, "text message")
}

func TestProvideLogger_RespectsLogLevel(t *testing.T) {
	t.Parallel()

	cfg := newValidConfig()
	logPath := filepath.Join(t.TempDir(), "level.log")
	cfg.Logging.Level = "error"
	cfg.Logging.Output = "file"
	cfg.Logging.File = testLogFileConfig(logPath)

	logger, err := ProvideLogger(cfg)
	require.NoError(t, err)

	logger.Info("info message")
	logger.Error("error message")
	require.NoError(t, logger.Sync())

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "info message")
	assert.Contains(t, string(data), "error message")
}

func TestProvideLogger_DisablesSamplingWhenConfigured(t *testing.T) {
	t.Parallel()

	cfg := newValidConfig()
	logPath := filepath.Join(t.TempDir(), "sampling.log")
	cfg.Logging.Output = "file"
	cfg.Logging.File = testLogFileConfig(logPath)
	cfg.Logging.Sampling = false

	logger, err := ProvideLogger(cfg)
	require.NoError(t, err)

	for range 5 {
		logger.Info("repeat message")
	}
	require.NoError(t, logger.Sync())

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Equal(t, 5, strings.Count(string(data), "repeat message"))
}

func testLogFileConfig(path string) *LogFileConfig {
	return &LogFileConfig{
		Path:       path,
		MaxSize:    10,
		MaxAge:     1,
		MaxBackups: 1,
	}
}

func readFirstLogLine(t *testing.T, path string) string {
	t.Helper()

	file, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = file.Close()
	})

	scanner := bufio.NewScanner(file)
	require.True(t, scanner.Scan())
	require.NoError(t, scanner.Err())

	return scanner.Text()
}
