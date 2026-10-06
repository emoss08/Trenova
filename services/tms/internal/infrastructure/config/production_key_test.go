package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func loadConfigYAML(t *testing.T, env, contents string) (*Config, error) {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(contents), 0o600))

	return NewLoader(WithConfigPath(dir), WithEnvironment(env)).Load()
}

func productionLocalKeyConfig(t *testing.T, allow bool, key string) string {
	t.Helper()

	const gcpEncryption = `  encryption:
    mode: envelope
    keyManager: gcp-autokey
    key: "a-very-long-encryption-key-that-is-at-least-32-chars"
    gcpKms:
      cryptoKey: "projects/test/locations/us/keyRings/autokey/cryptoKeys/trenova"
`
	base := validConfigYAML()
	require.Contains(t, base, gcpEncryption)

	allowed := "false"
	if allow {
		allowed = "true"
	}

	return strings.Replace(base, gcpEncryption, `  encryption:
    mode: envelope
    keyManager: local
    key: "`+key+`"
    allowLocalKeyManagerInProduction: `+allowed+`
`, 1)
}

func TestLoad_ProductionLocalKeyManager(t *testing.T) {
	const strongKey = "3f9b1c7e5a2d48f6b0c9e1a7d4f2b8c6"

	t.Run("refused without the opt-in", func(t *testing.T) {
		_, err := loadConfigYAML(t, EnvProduction, productionLocalKeyConfig(t, false, strongKey))
		require.ErrorIs(t, err, ErrProductionKMSRequired)
	})

	t.Run("accepted with the opt-in and a strong key", func(t *testing.T) {
		cfg, err := loadConfigYAML(t, EnvProduction, productionLocalKeyConfig(t, true, strongKey))
		require.NoError(t, err)
		assert.True(t, cfg.Security.Encryption.AllowLocalKeyManagerInProduction)
		assert.Equal(t, EncryptionKeyManagerLocal, cfg.Security.Encryption.KeyManager)
	})

	t.Run("refuses a placeholder key", func(t *testing.T) {
		_, err := loadConfigYAML(
			t,
			EnvProduction,
			productionLocalKeyConfig(t, true, "change-me-change-me-change-me-change-me"),
		)
		require.ErrorIs(t, err, ErrEncryptionKeyIsInsecure)
	})
}

func TestValidateProductionLocalKey_RequiresLength(t *testing.T) {
	t.Parallel()

	require.ErrorIs(t, validateProductionLocalKey("short"), ErrProductionLocalEncryptionKeyRequired)
	require.NoError(t, validateProductionLocalKey("3f9b1c7e5a2d48f6b0c9e1a7d4f2b8c6"))
}

func TestExampleConfigDecodesStrictly(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("../../../config", "config.example.yaml"))
	require.NoError(t, err)

	var example map[string]any
	require.NoError(t, yaml.Unmarshal(contents, &example))

	platform, ok := example["platform"].(map[string]any)
	require.True(t, ok, "config.example.yaml must have a platform section")
	for _, path := range editionSectionPaths {
		segments := strings.Split(path, ".")
		if len(segments) == 1 {
			assert.NotContainsf(t, example, segments[0], "%s belongs to an edition", path)
			continue
		}
		if segments[0] == "platform" {
			assert.NotContainsf(t, platform, segments[1], "%s belongs to an edition", path)
		}
	}

	section, err := yaml.Marshal(map[string]any{"platform": platform})
	require.NoError(t, err)

	cfg, err := loadConfigYAML(t, EnvTest, validConfigYAML()+string(section))
	require.NoError(t, err)
	assert.Equal(t, PlatformModeSelfHosted, cfg.Platform.GetMode())

	encryption, ok := example["security"].(map[string]any)["encryption"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, encryption, "allowLocalKeyManagerInProduction")
}
