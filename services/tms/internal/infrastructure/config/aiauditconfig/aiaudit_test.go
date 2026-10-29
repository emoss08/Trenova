package aiauditconfig

import (
	"fmt"
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

func TestValidateAIAuditConfig(t *testing.T) {
	t.Parallel()

	primary := ChainKey{ID: "k2026", Secret: testChainSecret}
	previous := ChainKey{ID: "k2025", Secret: testChainSecretOther}

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr error
	}{
		{name: "nothing configured", mutate: func(*Config) {}},
		{
			name: "rotated keys with the new one active",
			mutate: func(c *Config) {
				c.Chain.Keys = []ChainKey{primary, previous}
				c.Chain.ActiveKeyID = primary.ID
			},
		},
		{
			name:    "an active key with no keys",
			mutate:  func(c *Config) { c.Chain.ActiveKeyID = "k2026" },
			wantErr: ErrChainActiveKeyWithoutKeys,
		},
		{
			name:    "keys without an active one",
			mutate:  func(c *Config) { c.Chain.Keys = []ChainKey{primary} },
			wantErr: ErrChainActiveKeyRequired,
		},
		{
			name: "an active key that is not configured",
			mutate: func(c *Config) {
				c.Chain.Keys = []ChainKey{primary}
				c.Chain.ActiveKeyID = "k2024"
			},
			wantErr: ErrChainActiveKeyUnknown,
		},
		{
			name: "a secret that is too short",
			mutate: func(c *Config) {
				c.Chain.Keys = []ChainKey{{ID: "k1", Secret: "short-secret"}}
				c.Chain.ActiveKeyID = "k1"
			},
			wantErr: ErrChainSecretTooShort,
		},
		{
			name: "a key id the hash row cannot hold",
			mutate: func(c *Config) {
				c.Chain.Keys = []ChainKey{{ID: "key one", Secret: testChainSecret}}
				c.Chain.ActiveKeyID = "key one"
			},
			wantErr: ErrChainKeyIDInvalid,
		},
		{
			name: "a key id over forty characters",
			mutate: func(c *Config) {
				id := strings.Repeat("k", 41)
				c.Chain.Keys = []ChainKey{{ID: id, Secret: testChainSecret}}
				c.Chain.ActiveKeyID = id
			},
			wantErr: ErrChainKeyIDInvalid,
		},
		{
			name: "the same key id twice",
			mutate: func(c *Config) {
				c.Chain.Keys = []ChainKey{
					primary,
					{ID: primary.ID, Secret: testChainSecretOther},
				}
				c.Chain.ActiveKeyID = primary.ID
			},
			wantErr: ErrChainKeyIDDuplicate,
		},
		{
			name: "an inline export larger than any export",
			mutate: func(c *Config) {
				c.Export.SyncMaxRows = 10_000
				c.Export.MaxRows = 5_000
			},
			wantErr: ErrExportRowsInvalid,
		},
		{
			name:    "a negative export lifetime",
			mutate:  func(c *Config) { c.Export.TTL = -time.Hour },
			wantErr: ErrExportTTLInvalid,
		},
		{
			name:    "a projector that runs faster than its watermark trails",
			mutate:  func(c *Config) { c.Projector.Interval = time.Second },
			wantErr: ErrProjectorInterval,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var cfg Config
			tt.mutate(&cfg)

			err := cfg.Validate()
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
			for _, key := range cfg.Chain.Keys {
				assert.NotContains(t, err.Error(), key.Secret, "a secret never reaches an error")
			}
		})
	}
}

func TestAIAuditConfig_Defaults(t *testing.T) {
	t.Parallel()

	var cfg Config

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

	chain := ChainConfig{
		Keys: []ChainKey{
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

	key := ChainKey{ID: "k2026", Secret: testChainSecret}
	chain := ChainConfig{Keys: []ChainKey{key}, ActiveKeyID: key.ID}

	for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
		assert.NotContains(t, fmt.Sprintf(format, key), testChainSecret, format)
		assert.NotContains(t, fmt.Sprintf(format, chain), testChainSecret, format)
	}
	assert.Equal(t, "k2026:[redacted]", key.String())
}

func TestParseAIAuditChainKeys(t *testing.T) {
	t.Parallel()

	keys, err := ParseChainKeys(
		"k2026:" + testChainSecret + " , k2025:" + testChainSecretOther + ":with:colons",
	)
	require.NoError(t, err)
	assert.Equal(t, []ChainKey{
		{ID: "k2026", Secret: testChainSecret},
		{ID: "k2025", Secret: testChainSecretOther + ":with:colons"},
	}, keys)

	for _, raw := range []string{"k2026", "k2026:", ":" + testChainSecret, "k1:" + testChainSecret + ","} {
		_, err = ParseChainKeys(raw)
		require.ErrorIs(t, err, ErrChainKeysEnvMalformed, raw)
		assert.NotContains(t, err.Error(), testChainSecret)
	}
}

func TestValidateAIAuditChainSecrets_AcceptsARealKey(t *testing.T) {
	t.Parallel()

	cfg := Config{Chain: ChainConfig{
		Keys:        []ChainKey{{ID: "k2026", Secret: "q8F3vL0zT6nP1wR9yB4mK7cX2hJ5dG0s"}},
		ActiveKeyID: "k2026",
	}}

	require.NoError(t, cfg.ValidateProduction())
}
