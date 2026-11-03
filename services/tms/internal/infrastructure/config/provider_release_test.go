//go:build !devlog

package config

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/loggingconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvideLogger_PrettyFormatIsRefusedWithoutDevlogBuild(t *testing.T) {
	t.Parallel()

	cfg := newValidConfig()
	cfg.Logging.Format = "pretty"
	cfg.Logging.Output = "stdout"

	logger, sink, err := ProvideLogger(&cfg.Logging)

	require.ErrorIs(t, err, loggingconfig.ErrPrettyFormatNotBuilt)
	assert.Nil(t, logger)
	assert.Nil(t, sink)
}
