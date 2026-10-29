package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/aiauditconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testChainSecret      = "0123456789abcdef0123456789abcdef-primary"
	testChainSecretOther = "fedcba9876543210fedcba9876543210-previous"
)

func TestValidateConfig_RejectsAnAISamplingRateOutOfRange(t *testing.T) {
	t.Parallel()

	cfg := newValidConfig()
	rate := 1.5
	cfg.Monitoring.Tracing.AISamplingRate = &rate

	err := NewLoader().validateConfig(cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "aisamplingrate must be at most 1")
}

func TestValidateConfig_BoundsTheProjectorBatch(t *testing.T) {
	t.Parallel()

	cfg := newValidConfig()
	cfg.AIAudit.Projector.BatchSize = 20_000

	err := NewLoader().validateConfig(cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "batchsize must be at most 10000")
}

func writeConfig(t *testing.T, extra string) string {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "config.yaml"),
		[]byte(validConfigYAML()+extra),
		0o600,
	))

	return dir
}

func TestLoad_ReadsTheAITracingAndAuditSettings(t *testing.T) {
	dir := writeConfig(t, `
aiAudit:
  chain:
    keys:
      - id: k2026
        secret: "`+testChainSecret+`"
      - id: k2025
        secret: "`+testChainSecretOther+`"
    activeKeyId: k2026
  export:
    syncMaxRows: 2000
    maxRows: 250000
    ttl: 48h
  projector:
    interval: 30s
    batchSize: 500
`)
	t.Setenv("APP_ENV", "development")
	t.Setenv("TRENOVA_MONITORING_TRACING_AISAMPLINGRATE", "0.25")
	t.Setenv(
		"TRENOVA_MONITORING_TRACING_TRACEURLTEMPLATE",
		"https://tempo.example.com/trace/{traceId}",
	)

	cfg, err := NewLoader(WithConfigPath(dir), WithEnvironment(appconfig.EnvDevelopment)).Load()

	require.NoError(t, err)
	assert.InDelta(t, 0.25, cfg.Monitoring.Tracing.GetAISamplingRate(), 0)
	assert.Equal(t,
		"https://tempo.example.com/trace/{traceId}",
		cfg.Monitoring.Tracing.TraceURLTemplate)
	active, ok := cfg.AIAudit.Chain.ActiveKey()
	require.True(t, ok)
	assert.Equal(t, testChainSecret, active.Secret)
	assert.Len(t, cfg.AIAudit.Chain.Keys, 2)
	assert.Equal(t, 2000, cfg.AIAudit.Export.GetSyncMaxRows())
	assert.Equal(t, 250000, cfg.AIAudit.Export.GetMaxRows())
	assert.Equal(t, 48*time.Hour, cfg.AIAudit.Export.GetTTL())
	assert.Equal(t, 30*time.Second, cfg.AIAudit.Projector.GetInterval())
	assert.Equal(t, 500, cfg.AIAudit.Projector.GetBatchSize())
}

func TestLoad_TakesChainKeysFromTheEnvironment(t *testing.T) {
	dir := writeConfig(t, "")
	t.Setenv("APP_ENV", "development")
	t.Setenv("TRENOVA_AI_AUDIT_CHAIN_KEYS", "k2026:"+testChainSecret+",k2025:"+testChainSecretOther)
	t.Setenv("TRENOVA_AI_AUDIT_CHAIN_ACTIVE_KEY_ID", "k2026")

	cfg, err := NewLoader(WithConfigPath(dir), WithEnvironment(appconfig.EnvDevelopment)).Load()

	require.NoError(t, err)
	active, ok := cfg.AIAudit.Chain.ActiveKey()
	require.True(t, ok)
	assert.Equal(t, "k2026", active.ID)
	assert.Equal(t, testChainSecret, active.Secret)
	previous, ok := cfg.AIAudit.Chain.Key("k2025")
	require.True(t, ok)
	assert.Equal(t, testChainSecretOther, previous.Secret)
}

func TestLoad_RefusesAChainWithoutItsActiveKey(t *testing.T) {
	dir := writeConfig(t, "")
	t.Setenv("APP_ENV", "development")
	t.Setenv("TRENOVA_AI_AUDIT_CHAIN_KEYS", "k2026:"+testChainSecret)

	_, err := NewLoader(WithConfigPath(dir), WithEnvironment(appconfig.EnvDevelopment)).Load()

	require.ErrorIs(t, err, aiauditconfig.ErrChainActiveKeyRequired)
	assert.NotContains(t, err.Error(), testChainSecret)
}

func TestLoad_ProductionRefusesAPlaceholderChainKey(t *testing.T) {
	configDir := t.TempDir()
	copyTrackedConfig(t, "config.test.yaml", filepath.Join(configDir, "config.yaml"))
	copyTrackedConfig(t, "config.prod.example.yaml", filepath.Join(configDir, "config.prod.yaml"))
	stageProductionBase(t, filepath.Join(configDir, "config.yaml"))
	placeholder := "change-me-local-ai-audit-chain-key-32chars"
	t.Setenv("TRENOVA_AI_AUDIT_CHAIN_KEYS", "k2026:"+placeholder)
	t.Setenv("TRENOVA_AI_AUDIT_CHAIN_ACTIVE_KEY_ID", "k2026")

	_, err := NewLoader(WithConfigPath(configDir), WithEnvironment(appconfig.EnvProduction)).Load()

	require.ErrorIs(t, err, aiauditconfig.ErrChainSecretInsecure)
	assert.NotContains(t, err.Error(), placeholder)
}
