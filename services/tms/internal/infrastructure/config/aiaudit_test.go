package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testChainSecret      = "0123456789abcdef0123456789abcdef-primary"
	testChainSecretOther = "fedcba9876543210fedcba9876543210-previous"
)

func TestTracingConfig_AISamplingRateDefaultsToEverything(t *testing.T) {
	t.Parallel()

	var unset TracingConfig
	assert.InDelta(t, 1.0, unset.GetAISamplingRate(), 0)

	none := 0.0
	assert.InDelta(t, 0.0, (&TracingConfig{AISamplingRate: &none}).GetAISamplingRate(), 0)
}

func TestTracingConfig_TraceURL(t *testing.T) {
	t.Parallel()

	cfg := &TracingConfig{TraceURLTemplate: "https://tempo.example.com/trace/{traceId}?view=1"}
	assert.Equal(t,
		"https://tempo.example.com/trace/4bf92f3577b34da6a3ce929d0e0e4736?view=1",
		cfg.TraceURL("4bf92f3577b34da6a3ce929d0e0e4736"))
	assert.Empty(t, cfg.TraceURL(""))
	assert.Equal(t, "https://tempo.example.com/trace/a%2Fb?view=1", cfg.TraceURL("a/b"))
	assert.Empty(t, (&TracingConfig{}).TraceURL("4bf92f3577b34da6a3ce929d0e0e4736"))
}

func TestValidateTracingConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		template string
		wantErr  error
	}{
		{name: "no template", template: ""},
		{name: "https template", template: "https://tempo.example.com/trace/{traceId}"},
		{name: "http template with query", template: "http://jaeger:16686/trace/{traceId}?ui=1"},
		{
			name:     "template without placeholder",
			template: "https://tempo.example.com/trace/",
			wantErr:  ErrTraceURLTemplateMissingPlaceholder,
		},
		{
			name:     "template without scheme",
			template: "tempo.example.com/trace/{traceId}",
			wantErr:  ErrTraceURLTemplateInvalid,
		},
		{
			name:     "template with a script scheme",
			template: "javascript://tempo/{traceId}",
			wantErr:  ErrTraceURLTemplateInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newValidConfig()
			cfg.Monitoring.Tracing.TraceURLTemplate = tt.template

			err := NewLoader().validateConfig(cfg)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestValidateConfig_RejectsAnAISamplingRateOutOfRange(t *testing.T) {
	t.Parallel()

	cfg := newValidConfig()
	rate := 1.5
	cfg.Monitoring.Tracing.AISamplingRate = &rate

	err := NewLoader().validateConfig(cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "aisamplingrate must be at most 1")
}

func TestValidateAIAuditConfig(t *testing.T) {
	t.Parallel()

	primary := AIAuditChainKey{ID: "k2026", Secret: testChainSecret}
	previous := AIAuditChainKey{ID: "k2025", Secret: testChainSecretOther}

	tests := []struct {
		name    string
		mutate  func(*AIAuditConfig)
		wantErr error
	}{
		{name: "nothing configured", mutate: func(*AIAuditConfig) {}},
		{
			name: "rotated keys with the new one active",
			mutate: func(c *AIAuditConfig) {
				c.Chain.Keys = []AIAuditChainKey{primary, previous}
				c.Chain.ActiveKeyID = primary.ID
			},
		},
		{
			name:    "an active key with no keys",
			mutate:  func(c *AIAuditConfig) { c.Chain.ActiveKeyID = "k2026" },
			wantErr: ErrAIAuditChainActiveKeyWithoutKeys,
		},
		{
			name:    "keys without an active one",
			mutate:  func(c *AIAuditConfig) { c.Chain.Keys = []AIAuditChainKey{primary} },
			wantErr: ErrAIAuditChainActiveKeyRequired,
		},
		{
			name: "an active key that is not configured",
			mutate: func(c *AIAuditConfig) {
				c.Chain.Keys = []AIAuditChainKey{primary}
				c.Chain.ActiveKeyID = "k2024"
			},
			wantErr: ErrAIAuditChainActiveKeyUnknown,
		},
		{
			name: "a secret that is too short",
			mutate: func(c *AIAuditConfig) {
				c.Chain.Keys = []AIAuditChainKey{{ID: "k1", Secret: "short-secret"}}
				c.Chain.ActiveKeyID = "k1"
			},
			wantErr: ErrAIAuditChainSecretTooShort,
		},
		{
			name: "a key id the hash row cannot hold",
			mutate: func(c *AIAuditConfig) {
				c.Chain.Keys = []AIAuditChainKey{{ID: "key one", Secret: testChainSecret}}
				c.Chain.ActiveKeyID = "key one"
			},
			wantErr: ErrAIAuditChainKeyIDInvalid,
		},
		{
			name: "a key id over forty characters",
			mutate: func(c *AIAuditConfig) {
				id := strings.Repeat("k", 41)
				c.Chain.Keys = []AIAuditChainKey{{ID: id, Secret: testChainSecret}}
				c.Chain.ActiveKeyID = id
			},
			wantErr: ErrAIAuditChainKeyIDInvalid,
		},
		{
			name: "the same key id twice",
			mutate: func(c *AIAuditConfig) {
				c.Chain.Keys = []AIAuditChainKey{
					primary,
					{ID: primary.ID, Secret: testChainSecretOther},
				}
				c.Chain.ActiveKeyID = primary.ID
			},
			wantErr: ErrAIAuditChainKeyIDDuplicate,
		},
		{
			name: "an inline export larger than any export",
			mutate: func(c *AIAuditConfig) {
				c.Export.SyncMaxRows = 10_000
				c.Export.MaxRows = 5_000
			},
			wantErr: ErrAIAuditExportRowsInvalid,
		},
		{
			name:    "a negative export lifetime",
			mutate:  func(c *AIAuditConfig) { c.Export.TTL = -time.Hour },
			wantErr: ErrAIAuditExportTTLInvalid,
		},
		{
			name:    "a projector that runs faster than its watermark trails",
			mutate:  func(c *AIAuditConfig) { c.Projector.Interval = time.Second },
			wantErr: ErrAIAuditProjectorInterval,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newValidConfig()
			tt.mutate(&cfg.AIAudit)

			err := NewLoader().validateConfig(cfg)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
			for _, key := range cfg.AIAudit.Chain.Keys {
				assert.NotContains(t, err.Error(), key.Secret, "a secret never reaches an error")
			}
		})
	}
}

func TestValidateConfig_BoundsTheProjectorBatch(t *testing.T) {
	t.Parallel()

	cfg := newValidConfig()
	cfg.AIAudit.Projector.BatchSize = 20_000

	err := NewLoader().validateConfig(cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "batchsize must be at most 10000")
}

func TestAIAuditConfig_Defaults(t *testing.T) {
	t.Parallel()

	var cfg AIAuditConfig

	assert.Equal(t, 5000, cfg.Export.GetSyncMaxRows())
	assert.Equal(t, 1_000_000, cfg.Export.GetMaxRows())
	assert.Equal(t, 7*24*time.Hour, cfg.Export.GetTTL())
	assert.Equal(t, time.Minute, cfg.Projector.GetInterval())
	assert.Equal(t, 1000, cfg.Projector.GetBatchSize())
	assert.False(t, cfg.Chain.Configured())
	_, ok := cfg.Chain.ActiveKey()
	assert.False(t, ok)
}

func TestAIAuditChainConfig_FindsKeysById(t *testing.T) {
	t.Parallel()

	chain := AIAuditChainConfig{
		Keys: []AIAuditChainKey{
			{ID: "k2026", Secret: testChainSecret},
			{ID: "k2025", Secret: testChainSecretOther},
		},
		ActiveKeyID: "k2026",
	}

	active, ok := chain.ActiveKey()
	require.True(t, ok)
	assert.Equal(t, testChainSecret, active.Secret)

	older, ok := chain.Key("k2025")
	require.True(t, ok)
	assert.Equal(t, testChainSecretOther, older.Secret)

	_, ok = chain.Key("k2024")
	assert.False(t, ok)
}

func TestAIAuditChainKey_NeverPrintsItsSecret(t *testing.T) {
	t.Parallel()

	key := AIAuditChainKey{ID: "k2026", Secret: testChainSecret}
	chain := AIAuditChainConfig{Keys: []AIAuditChainKey{key}, ActiveKeyID: key.ID}

	for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
		assert.NotContains(t, fmt.Sprintf(format, key), testChainSecret, format)
		assert.NotContains(t, fmt.Sprintf(format, chain), testChainSecret, format)
	}
	assert.Equal(t, "k2026:[redacted]", key.String())
}

func TestParseAIAuditChainKeys(t *testing.T) {
	t.Parallel()

	keys, err := parseAIAuditChainKeys(
		"k2026:" + testChainSecret + " , k2025:" + testChainSecretOther + ":with:colons",
	)
	require.NoError(t, err)
	assert.Equal(t, []AIAuditChainKey{
		{ID: "k2026", Secret: testChainSecret},
		{ID: "k2025", Secret: testChainSecretOther + ":with:colons"},
	}, keys)

	for _, raw := range []string{"k2026", "k2026:", ":" + testChainSecret, "k1:" + testChainSecret + ","} {
		_, err = parseAIAuditChainKeys(raw)
		require.ErrorIs(t, err, ErrAIAuditChainKeysEnvMalformed, raw)
		assert.NotContains(t, err.Error(), testChainSecret)
	}
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

	cfg, err := NewLoader(WithConfigPath(dir), WithEnvironment(EnvDevelopment)).Load()

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

	cfg, err := NewLoader(WithConfigPath(dir), WithEnvironment(EnvDevelopment)).Load()

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

	_, err := NewLoader(WithConfigPath(dir), WithEnvironment(EnvDevelopment)).Load()

	require.ErrorIs(t, err, ErrAIAuditChainActiveKeyRequired)
	assert.NotContains(t, err.Error(), testChainSecret)
}

func TestLoad_ProductionRefusesAPlaceholderChainKey(t *testing.T) {
	configDir := t.TempDir()
	copyTrackedConfig(t, "config.test.yaml", filepath.Join(configDir, "config.yaml"))
	copyTrackedConfig(t, "config.prod.yaml", filepath.Join(configDir, "config.prod.yaml"))
	stageProductionBase(t, filepath.Join(configDir, "config.yaml"))
	placeholder := "change-me-local-ai-audit-chain-key-32chars"
	t.Setenv("TRENOVA_AI_AUDIT_CHAIN_KEYS", "k2026:"+placeholder)
	t.Setenv("TRENOVA_AI_AUDIT_CHAIN_ACTIVE_KEY_ID", "k2026")

	_, err := NewLoader(WithConfigPath(configDir), WithEnvironment(EnvProduction)).Load()

	require.ErrorIs(t, err, ErrAIAuditChainSecretInsecure)
	assert.NotContains(t, err.Error(), placeholder)
}

func TestValidateAIAuditChainSecrets_AcceptsARealKey(t *testing.T) {
	t.Parallel()

	cfg := newValidConfig()
	cfg.AIAudit.Chain.Keys = []AIAuditChainKey{
		{ID: "k2026", Secret: "q8F3vL0zT6nP1wR9yB4mK7cX2hJ5dG0s"},
	}
	cfg.AIAudit.Chain.ActiveKeyID = "k2026"

	require.NoError(t, validateAIAuditChainSecrets(cfg))
}
