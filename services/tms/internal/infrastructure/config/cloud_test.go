package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestLoad_CloudModeDoesNotRequireTheControlPlane(t *testing.T) {
	cfg, err := loadConfigYAML(t, EnvTest, validConfigYAML()+`
platform:
  mode: cloud
`)
	require.NoError(t, err)
	assert.True(t, cfg.Platform.IsCloud())
	assert.False(t, cfg.Platform.ControlPlane.Enabled)
}

func TestLoad_CloudDefaults(t *testing.T) {
	cfg, err := loadConfigYAML(t, EnvTest, validConfigYAML()+`
platform:
  mode: cloud
`)
	require.NoError(t, err)

	cloud := cfg.Platform.Cloud
	assert.False(t, cloud.Signup.Enabled)
	assert.Equal(t, DefaultCloudSignupMaxActiveTenants, cloud.Signup.GetMaxActiveTenants())
	assert.Equal(t, DefaultCloudSignupMaxSignupsPerDay, cloud.Signup.GetMaxSignupsPerDay())
	assert.Equal(t, DefaultCloudSignupPerIPPerHour, cloud.Signup.GetPerIPPerHour())
	assert.Equal(t, 24*time.Hour, cloud.Signup.GetVerificationTokenTTL())
	assert.True(t, cloud.Signup.BlockDisposableEmail)
	assert.Empty(t, cloud.Signup.GetAllowedEmailDomains())
	assert.Equal(t, DefaultCloudSignupTermsURL, cloud.Signup.GetTermsURL())
	assert.Equal(t, DefaultCloudSignupPrivacyURL, cloud.Signup.GetPrivacyURL())
	assert.True(t, cloud.Turnstile.Enabled)
	assert.Equal(t, DefaultCloudTurnstileVerifyURL, cloud.Turnstile.GetVerifyURL())
	assert.Equal(t, DefaultCloudTurnstileTimeout, cloud.Turnstile.GetTimeout())
	assert.Equal(t, CloudSystemEmailProviderResend, cloud.SystemEmail.GetProvider())
	assert.Equal(t, DefaultCloudSystemEmailFromAddress, cloud.SystemEmail.GetFromAddress())
	assert.Equal(t, DefaultCloudSystemEmailFromName, cloud.SystemEmail.GetFromName())
	assert.False(t, cloud.SystemEmail.HasAPIKey())
	assert.Equal(t, 720*time.Hour, cloud.Trial.GetLifetime())
	assert.Equal(t, 336*time.Hour, cloud.Trial.GetReadOnlyGrace())
	assert.Empty(t, cloud.FreePlan.GetLimitOverrides())
}

func TestLoad_CloudSettingsAndFreePlanOverrides(t *testing.T) {
	cfg, err := loadConfigYAML(t, EnvTest, validConfigYAML()+`
platform:
  mode: cloud
  cloud:
    signup:
      enabled: true
      maxActiveTenants: 0
      maxSignupsPerDay: 25
      perIpPerHour: 5
      verificationTokenTtl: 2h
      blockDisposableEmail: false
      allowedEmailDomains: ["Example.com", "trenova.app"]
    turnstile:
      siteKey: 1x00000000000000000000AA
      secretKey: 1x0000000000000000000000000000000AA
    trial:
      lifetime: 48h
      readOnlyGrace: 24h
    freePlan:
      limits:
        shipments.total: 20
        documents:
          storage_bytes: 2048
`)
	require.NoError(t, err)

	cloud := cfg.Platform.Cloud
	assert.True(t, cloud.Signup.Enabled)
	assert.Equal(t, 0, cloud.Signup.GetMaxActiveTenants())
	assert.Equal(t, 25, cloud.Signup.GetMaxSignupsPerDay())
	assert.Equal(t, 5, cloud.Signup.GetPerIPPerHour())
	assert.Equal(t, 2*time.Hour, cloud.Signup.GetVerificationTokenTTL())
	assert.False(t, cloud.Signup.BlockDisposableEmail)
	assert.Equal(t, []string{"example.com", "trenova.app"}, cloud.Signup.GetAllowedEmailDomains())
	assert.Equal(t, 48*time.Hour, cloud.Trial.GetLifetime())
	assert.Equal(t, 24*time.Hour, cloud.Trial.GetReadOnlyGrace())
	assert.Equal(t, map[string]int64{
		"shipments.total":         20,
		"documents.storage_bytes": 2048,
	}, cloud.FreePlan.GetLimitOverrides())
}

func TestLoad_CloudEnvironmentOverrides(t *testing.T) {
	t.Setenv("TRENOVA_PLATFORM_CLOUD_SIGNUP_MAXSIGNUPSPERDAY", "7")
	t.Setenv("TRENOVA_PLATFORM_CLOUD_SYSTEMEMAIL_APIKEY", "re_test_key")

	cfg, err := loadConfigYAML(t, EnvTest, validConfigYAML()+`
platform:
  mode: cloud
`)
	require.NoError(t, err)
	assert.Equal(t, 7, cfg.Platform.Cloud.Signup.GetMaxSignupsPerDay())
	assert.True(t, cfg.Platform.Cloud.SystemEmail.HasAPIKey())
}

func TestLoad_CloudSignupRequiresTurnstileKeys(t *testing.T) {
	_, err := loadConfigYAML(t, EnvTest, validConfigYAML()+`
platform:
  mode: cloud
  cloud:
    signup:
      enabled: true
`)
	require.ErrorIs(t, err, ErrCloudTurnstileSiteKeyRequired)

	_, err = loadConfigYAML(t, EnvTest, validConfigYAML()+`
platform:
  mode: cloud
  cloud:
    signup:
      enabled: true
    turnstile:
      siteKey: site
`)
	require.ErrorIs(t, err, ErrCloudTurnstileSecretKeyRequired)

	_, err = loadConfigYAML(t, EnvTest, validConfigYAML()+`
platform:
  mode: cloud
  cloud:
    signup:
      enabled: true
    turnstile:
      enabled: false
`)
	require.NoError(t, err)
}

func TestLoad_CloudRejectsNegativeFreePlanLimit(t *testing.T) {
	_, err := loadConfigYAML(t, EnvTest, validConfigYAML()+`
platform:
  mode: cloud
  cloud:
    freePlan:
      limits:
        shipments.total: -1
`)
	require.ErrorIs(t, err, ErrCloudFreePlanLimitNegative)
}

func TestValidateCloudConfig_RequiresPostgres(t *testing.T) {
	t.Parallel()

	cfg := newValidConfig()
	cfg.Platform.Mode = PlatformModeCloud
	cfg.Database.Driver = "sqlite"
	require.ErrorIs(t, validateCloudConfig(cfg), ErrCloudRequiresPostgres)

	cfg.Database.Driver = "postgres"
	require.NoError(t, validateCloudConfig(cfg))
}

func TestLoad_CloudProductionRequiresTurnstileAndSystemEmail(t *testing.T) {
	cfg := newValidConfig()
	cfg.Platform.Mode = PlatformModeCloud
	cfg.Platform.Cloud.Signup.Enabled = true

	require.ErrorIs(t, validateCloudProductionSecurity(cfg), ErrProductionCloudTurnstileRequired)

	cfg.Platform.Cloud.Turnstile.Enabled = true
	require.ErrorIs(t, validateCloudProductionSecurity(cfg), ErrProductionCloudSystemEmailRequired)

	cfg.Platform.Cloud.SystemEmail.APIKey = "re_live_key"
	require.NoError(t, validateCloudProductionSecurity(cfg))

	cfg.Platform.Cloud.Turnstile.SecretKey = "1x0000000000000000000000000000000AA"
	require.ErrorIs(t, validateCloudProductionSecurity(cfg), ErrProductionCloudTurnstileTestSecret)
	cfg.Platform.Cloud.Turnstile.SecretKey = "0x4AAAAAAA-live-secret"
	require.NoError(t, validateCloudProductionSecurity(cfg))

	cfg.Platform.Mode = PlatformModeSelfHosted
	cfg.Platform.Cloud.SystemEmail.APIKey = ""
	require.NoError(t, validateCloudProductionSecurity(cfg))
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

func TestExampleConfigPlatformSectionDecodesStrictly(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("../../../config", "config.example.yaml"))
	require.NoError(t, err)

	var example map[string]any
	require.NoError(t, yaml.Unmarshal(contents, &example))

	platform, ok := example["platform"].(map[string]any)
	require.True(t, ok, "config.example.yaml must have a platform section")
	platform["mode"] = string(PlatformModeCloud)

	section, err := yaml.Marshal(map[string]any{"platform": platform})
	require.NoError(t, err)

	cfg, err := loadConfigYAML(t, EnvTest, validConfigYAML()+string(section))
	require.NoError(t, err)
	assert.Equal(t, "1x00000000000000000000AA", cfg.Platform.Cloud.Turnstile.SiteKey)
	assert.Equal(t, "1x0000000000000000000000000000000AA", cfg.Platform.Cloud.Turnstile.SecretKey)
	assert.Equal(t, 720*time.Hour, cfg.Platform.Cloud.Trial.GetLifetime())

	encryption, ok := example["security"].(map[string]any)["encryption"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, encryption, "allowLocalKeyManagerInProduction")
}
