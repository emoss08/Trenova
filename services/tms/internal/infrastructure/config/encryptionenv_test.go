package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_TakesRetiredEncryptionKeysFromTheEnvironment(t *testing.T) {
	dir := writeConfig(t, "")
	t.Setenv("APP_ENV", "development")
	t.Setenv(
		"TRENOVA_SECURITY_ENCRYPTION_RETIREDKEYS",
		"retired-key-one-with-at-least-32-bytes,retired-key-two-with-at-least-32-bytes",
	)

	cfg, err := NewLoader(WithConfigPath(dir), WithEnvironment(EnvDevelopment)).Load()

	require.NoError(t, err)
	assert.Equal(t, []string{
		"retired-key-one-with-at-least-32-bytes",
		"retired-key-two-with-at-least-32-bytes",
	}, cfg.Security.Encryption.RetiredKeys)
}
