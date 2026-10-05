package cloudconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/configtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func loadConfigYAML(t *testing.T, env, contents string) (*config.Config, error) {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(contents), 0o600))

	return config.NewLoader(
		config.WithConfigPath(dir),
		config.WithEnvironment(env),
		config.WithSections(Section()),
	).Load()
}

func validConfigYAML() string {
	return configtest.ValidYAML()
}

func newValidConfig(t *testing.T) *config.Config {
	t.Helper()

	cfg, err := loadConfigYAML(t, config.EnvTest, validConfigYAML())
	require.NoError(t, err)

	return cfg
}

func TestLoad_CloudModeDoesNotRequireTheControlPlane(t *testing.T) {
	cfg, err := loadConfigYAML(t, config.EnvTest, validConfigYAML()+`
platform:
  mode: cloud
`)
	require.NoError(t, err)
	assert.True(t, cfg.Platform.IsCloud())
	assert.False(t, From(cfg).ControlPlane.Enabled)
}

func TestLoad_CloudDefaults(t *testing.T) {
	cfg, err := loadConfigYAML(t, config.EnvTest, validConfigYAML()+`
platform:
  mode: cloud
`)
	require.NoError(t, err)

	cloud := From(cfg).Cloud
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
	assert.Equal(t, 7*24*time.Hour, cloud.Trial.GetLifetime())
	assert.Equal(t, 336*time.Hour, cloud.Trial.GetReadOnlyGrace())
	assert.Empty(t, cloud.FreePlan.GetLimitOverrides())
}

func TestLoad_CloudSettingsAndFreePlanOverrides(t *testing.T) {
	cfg, err := loadConfigYAML(t, config.EnvTest, validConfigYAML()+`
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

	cloud := From(cfg).Cloud
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

	cfg, err := loadConfigYAML(t, config.EnvTest, validConfigYAML()+`
platform:
  mode: cloud
`)
	require.NoError(t, err)
	assert.Equal(t, 7, From(cfg).Cloud.Signup.GetMaxSignupsPerDay())
	assert.True(t, From(cfg).Cloud.SystemEmail.HasAPIKey())
}

func TestLoad_CloudSignupRequiresTurnstileKeys(t *testing.T) {
	_, err := loadConfigYAML(t, config.EnvTest, validConfigYAML()+`
platform:
  mode: cloud
  cloud:
    signup:
      enabled: true
`)
	require.ErrorIs(t, err, ErrCloudTurnstileSiteKeyRequired)

	_, err = loadConfigYAML(t, config.EnvTest, validConfigYAML()+`
platform:
  mode: cloud
  cloud:
    signup:
      enabled: true
    turnstile:
      siteKey: site
`)
	require.ErrorIs(t, err, ErrCloudTurnstileSecretKeyRequired)

	_, err = loadConfigYAML(t, config.EnvTest, validConfigYAML()+`
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
	_, err := loadConfigYAML(t, config.EnvTest, validConfigYAML()+`
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
	cfg := newValidConfig(t)
	settings := From(cfg)
	cfg.Platform.Mode = config.PlatformModeCloud
	cfg.Database.Driver = "sqlite"
	require.ErrorIs(t, validateCloudConfig(cfg, settings), ErrCloudRequiresPostgres)

	cfg.Database.Driver = "postgres"
	require.NoError(t, validateCloudConfig(cfg, settings))
}

func TestLoad_CloudProductionRequiresTurnstileAndSystemEmail(t *testing.T) {
	cfg := newValidConfig(t)
	settings := From(cfg)
	cfg.Platform.Mode = config.PlatformModeCloud
	settings.Cloud.Signup.Enabled = true
	settings.Cloud.Turnstile.Enabled = false

	require.ErrorIs(t, validateCloudProductionSecurity(cfg, settings), ErrProductionCloudTurnstileRequired)

	settings.Cloud.Turnstile.Enabled = true
	require.ErrorIs(t, validateCloudProductionSecurity(cfg, settings), ErrProductionCloudSystemEmailRequired)

	settings.Cloud.SystemEmail.APIKey = "re_live_key"
	require.NoError(t, validateCloudProductionSecurity(cfg, settings))

	settings.Cloud.Turnstile.SecretKey = "1x0000000000000000000000000000000AA"
	require.ErrorIs(t, validateCloudProductionSecurity(cfg, settings), ErrProductionCloudTurnstileTestSecret)
	settings.Cloud.Turnstile.SecretKey = "0x4AAAAAAA-live-secret"
	require.NoError(t, validateCloudProductionSecurity(cfg, settings))

	cfg.Platform.Mode = config.PlatformModeSelfHosted
	settings.Cloud.SystemEmail.APIKey = ""
	require.NoError(t, validateCloudProductionSecurity(cfg, settings))
}

func TestExampleConfigPlatformSectionAcceptsTheCloudSections(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("../../../config", "config.example.yaml"))
	require.NoError(t, err)

	var example map[string]any
	require.NoError(t, yaml.Unmarshal(contents, &example))

	platform, ok := example["platform"].(map[string]any)
	require.True(t, ok, "config.example.yaml must have a platform section")
	platform["mode"] = string(config.PlatformModeCloud)
	platform["cloud"] = map[string]any{
		"turnstile": map[string]any{
			"siteKey":   "1x00000000000000000000AA",
			"secretKey": "1x0000000000000000000000000000000AA",
		},
		"trial": map[string]any{"lifetime": "168h"},
	}

	section, err := yaml.Marshal(map[string]any{"platform": platform})
	require.NoError(t, err)

	cfg, err := loadConfigYAML(t, config.EnvTest, validConfigYAML()+string(section))
	require.NoError(t, err)
	settings := From(cfg)
	assert.Equal(t, "1x00000000000000000000AA", settings.Cloud.Turnstile.SiteKey)
	assert.Equal(t, "1x0000000000000000000000000000000AA", settings.Cloud.Turnstile.SecretKey)
	assert.Equal(t, 7*24*time.Hour, settings.Cloud.Trial.GetLifetime())
}

func TestLoad_ControlPlaneRequiresItsEndpointAndKey(t *testing.T) {
	cfg, err := loadConfigYAML(t, config.EnvTest, validConfigYAML()+`
platform:
  mode: cloud
  controlPlane:
    enabled: true
    endpoint: ""
    apiKey: ""
`)
	require.Nil(t, cfg)
	require.ErrorIs(t, err, ErrControlPlaneEndpointRequired)
}

func TestLoad_ControlPlaneEnvAliases(t *testing.T) {
	t.Setenv("TRENOVA_DEPLOYMENT_MODE", "development")
	t.Setenv("TRENOVA_CONTROL_PLANE_ENABLED", "true")
	t.Setenv("TRENOVA_CONTROL_PLANE_ENDPOINT", "https://control.trenova.test")
	t.Setenv("TRENOVA_INSTANCE_ID", "inst_01")
	t.Setenv("TRENOVA_CONTROL_PLANE_API_KEY", "cp_test_key")
	t.Setenv("TRENOVA_CONTROL_PLANE_HEARTBEAT_INTERVAL", "30s")
	t.Setenv("TRENOVA_CONTROL_PLANE_FAIL_OPEN_ON_ERROR", "true")

	cfg, err := loadConfigYAML(t, config.EnvTest, validConfigYAML())
	require.NoError(t, err)

	controlPlane := From(cfg).ControlPlane
	assert.Equal(t, config.PlatformModeDevelopment, cfg.Platform.GetMode())
	assert.True(t, controlPlane.Enabled)
	assert.Equal(t, "https://control.trenova.test", controlPlane.Endpoint)
	assert.Equal(t, "inst_01", cfg.Platform.InstanceID)
	assert.Equal(t, "cp_test_key", controlPlane.APIKey)
	assert.Equal(t, 30*time.Second, controlPlane.GetHeartbeatInterval())
	assert.Equal(t, time.Hour, controlPlane.GetTenantSyncInterval())
	assert.True(t, controlPlane.FailOpenOnError)
}

func TestLoad_NetworkPulseAndRetrainingDecodeIntoTheCloudSettings(t *testing.T) {
	cfg, err := loadConfigYAML(t, config.EnvTest, strings.Replace(
		validConfigYAML(),
		"system:\n",
		"system:\n  networkPulse:\n    enabled: true\n    cacheTtl: 30s\n",
		1,
	)+`
aiRetraining:
  enabled: true
  lookbackDays: 90
`)
	require.NoError(t, err)

	settings := From(cfg)
	assert.True(t, settings.NetworkPulse.Enabled)
	assert.Equal(t, 30*time.Second, settings.NetworkPulse.GetCacheTTL())
	assert.True(t, settings.AIRetraining.IsEnabled())
	assert.Equal(t, 90, settings.AIRetraining.GetLookbackDays())
}

func TestLoad_CloudSectionsRejectUnknownKeys(t *testing.T) {
	_, err := loadConfigYAML(t, config.EnvTest, validConfigYAML()+`
platform:
  mode: cloud
  cloud:
    signup:
      enabeld: true
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "enabeld")
}

func TestFromWithoutTheSectionReturnsZeroSettings(t *testing.T) {
	t.Parallel()

	settings := From(&config.Config{})
	require.NotNil(t, settings)
	assert.False(t, settings.ControlPlane.Enabled)
	assert.Equal(t, DefaultNetworkPulseCacheTTL, settings.NetworkPulse.GetCacheTTL())

	cfg := &config.Config{}
	attached := &Settings{ControlPlane: PlatformControlPlaneConfig{Enabled: true}}
	Attach(cfg, attached)
	assert.Same(t, attached, From(cfg))
}
