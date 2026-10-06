package cloudconfig

import (
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/spf13/viper"
)

var (
	ErrCloudRequiresPostgres = errors.New(
		"platform.mode=cloud requires the postgres database driver",
	)
	ErrCloudTurnstileSiteKeyRequired = errors.New(
		"platform.cloud.turnstile.siteKey is required when cloud signup and turnstile are enabled",
	)
	ErrCloudTurnstileSecretKeyRequired = errors.New(
		"platform.cloud.turnstile.secretKey is required when cloud signup and turnstile are enabled",
	)
	ErrCloudFreePlanLimitNegative = errors.New(
		"platform.cloud.freePlan.limits values must not be negative",
	)
	ErrProductionCloudTurnstileRequired = errors.New(
		"production and staging require platform.cloud.turnstile.enabled when cloud signup is enabled",
	)
	ErrProductionCloudTurnstileTestSecret = errors.New(
		"production and staging refuse a Cloudflare Turnstile test secret in platform.cloud.turnstile.secretKey",
	)
	ErrProductionCloudSystemEmailRequired = errors.New(
		"production and staging require platform.cloud.systemEmail.apiKey when cloud signup is enabled",
	)
	ErrControlPlaneEndpointRequired = errors.New(
		"platform.controlplane.endpoint is required when control plane is enabled",
	)
	ErrControlPlaneAPIKeyRequired = errors.New(
		"platform.controlplane.apikey is required when control plane is enabled",
	)
	ErrControlPlaneInstanceIDRequired = errors.New(
		"platform.instanceid is required when control plane is enabled",
	)
)

type section struct{}

func Section() config.Section {
	return section{}
}

func (section) Name() string {
	return SectionName
}

func (section) Paths() []string {
	return []string{
		"platform.cloud",
		"platform.controlPlane",
		"system.networkPulse",
		"aiRetraining",
	}
}

func (section) PlatformModes() []config.PlatformMode {
	return []config.PlatformMode{config.PlatformModeCloud}
}

func (section) SetDefaults(v *viper.Viper, envPrefix string) {
	v.SetDefault("platform.controlPlane.enabled", false)
	v.SetDefault("platform.controlPlane.timeout", "5s")
	v.SetDefault("platform.controlPlane.heartbeatInterval", "5m")
	v.SetDefault("platform.controlPlane.tenantSyncInterval", "1h")
	v.SetDefault("platform.controlPlane.failOpenOnError", false)
	v.SetDefault("platform.cloud.signup.enabled", false)
	v.SetDefault("platform.cloud.signup.maxActiveTenants", DefaultCloudSignupMaxActiveTenants)
	v.SetDefault("platform.cloud.signup.maxSignupsPerDay", DefaultCloudSignupMaxSignupsPerDay)
	v.SetDefault("platform.cloud.signup.perIpPerHour", DefaultCloudSignupPerIPPerHour)
	v.SetDefault(
		"platform.cloud.signup.verificationTokenTtl",
		DefaultCloudSignupVerificationTokenTTL.String(),
	)
	v.SetDefault("platform.cloud.signup.blockDisposableEmail", true)
	v.SetDefault("platform.cloud.signup.allowedEmailDomains", []string{})
	v.SetDefault("platform.cloud.signup.termsUrl", DefaultCloudSignupTermsURL)
	v.SetDefault("platform.cloud.signup.privacyUrl", DefaultCloudSignupPrivacyURL)
	v.SetDefault("platform.cloud.turnstile.enabled", true)
	v.SetDefault("platform.cloud.turnstile.siteKey", "")
	v.SetDefault("platform.cloud.turnstile.secretKey", "")
	v.SetDefault("platform.cloud.turnstile.verifyUrl", DefaultCloudTurnstileVerifyURL)
	v.SetDefault("platform.cloud.turnstile.timeout", DefaultCloudTurnstileTimeout.String())
	v.SetDefault("platform.cloud.systemEmail.provider", CloudSystemEmailProviderResend)
	v.SetDefault("platform.cloud.systemEmail.apiKey", "")
	v.SetDefault("platform.cloud.systemEmail.fromAddress", DefaultCloudSystemEmailFromAddress)
	v.SetDefault("platform.cloud.systemEmail.fromName", DefaultCloudSystemEmailFromName)
	v.SetDefault("platform.cloud.systemEmail.replyTo", "")
	v.SetDefault(
		"platform.cloud.systemEmail.timeout",
		DefaultCloudSystemEmailTimeout.String(),
	)
	v.SetDefault("platform.cloud.trial.lifetime", DefaultCloudTrialLifetime.String())
	v.SetDefault("platform.cloud.trial.readOnlyGrace", DefaultCloudTrialReadOnlyGrace.String())
	v.SetDefault(
		"platform.cloud.supportAccess.maxGrantDuration",
		DefaultSupportAccessMaxGrantDuration.String(),
	)
	v.SetDefault(
		"platform.cloud.supportAccess.maxSessionDuration",
		DefaultSupportAccessMaxSessionDuration.String(),
	)
	v.SetDefault(
		"platform.cloud.supportAccess.elevationDuration",
		DefaultSupportAccessElevationDuration.String(),
	)
	v.SetDefault(
		"platform.cloud.supportAccess.sessionStartsPerHour",
		DefaultSupportAccessSessionStartsPerHour,
	)
	v.SetDefault("platform.cloud.supportAccess.cookieName", DefaultSupportAccessCookieName)
	v.SetDefault(
		"platform.cloud.supportAccess.principalEmailDomain",
		DefaultSupportAccessPrincipalEmailDomain,
	)
	v.SetDefault("aiRetraining.alerts.webhookUrl", "")
	v.SetDefault("aiRetraining.alerts.secret", "")

	_ = v.BindEnv("platform.controlPlane.enabled", envPrefix+"_CONTROL_PLANE_ENABLED")
	_ = v.BindEnv("platform.controlPlane.endpoint", envPrefix+"_CONTROL_PLANE_ENDPOINT")
	_ = v.BindEnv("platform.controlPlane.apiKey", envPrefix+"_CONTROL_PLANE_API_KEY")
	_ = v.BindEnv(
		"platform.controlPlane.heartbeatInterval",
		envPrefix+"_CONTROL_PLANE_HEARTBEAT_INTERVAL",
	)
	_ = v.BindEnv(
		"platform.controlPlane.tenantSyncInterval",
		envPrefix+"_CONTROL_PLANE_TENANT_SYNC_INTERVAL",
	)
	_ = v.BindEnv(
		"platform.controlPlane.failOpenOnError",
		envPrefix+"_CONTROL_PLANE_FAIL_OPEN_ON_ERROR",
	)
}

func (section) Decode(v *viper.Viper) (any, error) {
	shape, err := config.DecodeSection[fileShape](v)
	if err != nil {
		return nil, err
	}

	settings := shape.settings()
	if err = config.ValidateStruct(settings); err != nil {
		return nil, err
	}

	return settings, nil
}

func (section) Validate(cfg *config.Config, env string) error {
	settings := From(cfg)

	validators := []func(*config.Config, *Settings) error{
		validateCloudConfig,
		validateControlPlaneConfig,
		validateAIRetrainingConfig,
		validateSupportAccessConfig,
	}
	for _, validate := range validators {
		if err := validate(cfg, settings); err != nil {
			return err
		}
	}

	if !config.IsProductionLike(env) {
		return nil
	}

	if err := validateCloudProductionSecurity(cfg, settings); err != nil {
		return err
	}

	return validateAIRetrainingAlertsProduction(settings)
}

func validateControlPlaneConfig(cfg *config.Config, settings *Settings) error {
	controlPlane := &settings.ControlPlane
	if !controlPlane.Enabled {
		return nil
	}

	if controlPlane.Endpoint == "" {
		return ErrControlPlaneEndpointRequired
	}
	if controlPlane.APIKey == "" {
		return ErrControlPlaneAPIKeyRequired
	}
	if cfg.Platform.InstanceID == "" {
		return ErrControlPlaneInstanceIDRequired
	}

	return nil
}

func validateCloudConfig(cfg *config.Config, settings *Settings) error {
	if !cfg.Platform.IsCloud() {
		return nil
	}

	if !cfg.Database.GetDialect().IsPostgres() {
		return ErrCloudRequiresPostgres
	}

	cloud := &settings.Cloud
	if cloud.Signup.Enabled && cloud.Turnstile.Enabled {
		if strings.TrimSpace(cloud.Turnstile.SiteKey) == "" {
			return ErrCloudTurnstileSiteKeyRequired
		}
		if strings.TrimSpace(cloud.Turnstile.SecretKey) == "" {
			return ErrCloudTurnstileSecretKeyRequired
		}
	}

	for key, value := range cloud.FreePlan.GetLimitOverrides() {
		if value < 0 {
			return fmt.Errorf("%w: %s is %d", ErrCloudFreePlanLimitNegative, key, value)
		}
	}

	return nil
}

func validateCloudProductionSecurity(cfg *config.Config, settings *Settings) error {
	if !cfg.Platform.IsCloud() {
		return nil
	}

	cloud := &settings.Cloud
	if cloud.Signup.Enabled && !cloud.Turnstile.Enabled {
		return ErrProductionCloudTurnstileRequired
	}
	if cloud.Signup.Enabled && cloud.Turnstile.UsesTestSecret() {
		return ErrProductionCloudTurnstileTestSecret
	}
	if cloud.Signup.Enabled && !cloud.SystemEmail.HasAPIKey() {
		return ErrProductionCloudSystemEmailRequired
	}

	return nil
}
