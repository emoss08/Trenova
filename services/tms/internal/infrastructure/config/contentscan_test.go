package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_TakesTheClamAVAddressFromTheEnvironment(t *testing.T) {
	dir := writeConfig(t, "")
	t.Setenv("APP_ENV", "development")
	t.Setenv("TRENOVA_CONTENTSCAN_CLAMAV_ADDRESS", "clamav:3310")

	cfg, err := NewLoader(WithConfigPath(dir), WithEnvironment(EnvDevelopment)).Load()

	require.NoError(t, err)
	assert.True(t, cfg.ContentScan.Enabled())
	assert.Equal(t, "clamav:3310", cfg.ContentScan.ClamAV.Address)
	assert.Equal(t, 60*time.Second, cfg.ContentScan.ClamAV.GetTimeout())
}

func TestLoad_ContentScanIsOffWhenNotConfigured(t *testing.T) {
	dir := writeConfig(t, "")
	t.Setenv("APP_ENV", "development")

	cfg, err := NewLoader(WithConfigPath(dir), WithEnvironment(EnvDevelopment)).Load()

	require.NoError(t, err)
	assert.False(t, cfg.ContentScan.Enabled())
}
