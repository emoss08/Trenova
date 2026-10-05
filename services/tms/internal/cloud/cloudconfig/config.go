package cloudconfig

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
)

const SectionName = "cloud"

type Settings struct {
	Cloud        PlatformCloudConfig
	ControlPlane PlatformControlPlaneConfig
	NetworkPulse NetworkPulseConfig
	AIRetraining AIRetrainingConfig
}

type fileShape struct {
	Platform struct {
		Cloud        PlatformCloudConfig        `mapstructure:"cloud"`
		ControlPlane PlatformControlPlaneConfig `mapstructure:"controlPlane"`
	} `mapstructure:"platform"`
	System struct {
		NetworkPulse NetworkPulseConfig `mapstructure:"networkPulse"`
	} `mapstructure:"system"`
	AIRetraining AIRetrainingConfig `mapstructure:"aiRetraining"`
}

func (f *fileShape) settings() *Settings {
	return &Settings{
		Cloud:        f.Platform.Cloud,
		ControlPlane: f.Platform.ControlPlane,
		NetworkPulse: f.System.NetworkPulse,
		AIRetraining: f.AIRetraining,
	}
}

func From(cfg *config.Config) *Settings {
	if value, ok := cfg.Extension(SectionName); ok {
		if settings, isSettings := value.(*Settings); isSettings && settings != nil {
			return settings
		}
	}

	return &Settings{}
}

func Attach(cfg *config.Config, settings *Settings) {
	cfg.SetExtension(SectionName, settings)
}

func WithSettings(cfg *config.Config, settings *Settings) *config.Config {
	Attach(cfg, settings)

	return cfg
}

const (
	DefaultCloudSignupMaxActiveTenants     = 500
	DefaultCloudSignupMaxSignupsPerDay     = 100
	DefaultCloudSignupPerIPPerHour         = 3
	DefaultCloudSignupVerificationTokenTTL = 24 * time.Hour
	DefaultCloudSignupTermsURL             = "https://trenova.app/terms"
	DefaultCloudSignupPrivacyURL           = "https://trenova.app/privacy"
	DefaultCloudTurnstileVerifyURL         = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	DefaultCloudTurnstileTimeout           = 5 * time.Second
	CloudSystemEmailProviderResend         = "resend"
	DefaultCloudSystemEmailFromAddress     = "noreply@trenova.app"
	DefaultCloudSystemEmailFromName        = "Trenova"
	DefaultCloudSystemEmailTimeout         = 10 * time.Second
	DefaultCloudTrialLifetime              = 7 * 24 * time.Hour
	DefaultCloudTrialReadOnlyGrace         = 336 * time.Hour
)

type PlatformCloudConfig struct {
	Signup      CloudSignupConfig      `mapstructure:"signup"`
	Turnstile   CloudTurnstileConfig   `mapstructure:"turnstile"`
	SystemEmail CloudSystemEmailConfig `mapstructure:"systemEmail"`
	Trial       CloudTrialConfig       `mapstructure:"trial"`
	FreePlan    CloudFreePlanConfig    `mapstructure:"freePlan"`
}

type CloudSignupConfig struct {
	Enabled              bool          `mapstructure:"enabled"`
	MaxActiveTenants     int           `mapstructure:"maxActiveTenants"     validate:"min=0"`
	MaxSignupsPerDay     int           `mapstructure:"maxSignupsPerDay"     validate:"min=0"`
	PerIPPerHour         int           `mapstructure:"perIpPerHour"         validate:"min=0"`
	VerificationTokenTTL time.Duration `mapstructure:"verificationTokenTtl" validate:"omitempty,min=5m,max=168h"`
	BlockDisposableEmail bool          `mapstructure:"blockDisposableEmail"`
	AllowedEmailDomains  []string      `mapstructure:"allowedEmailDomains"  validate:"omitempty,dive,required,fqdn"`
	TermsURL             string        `mapstructure:"termsUrl"             validate:"omitempty,url"`
	PrivacyURL           string        `mapstructure:"privacyUrl"           validate:"omitempty,url"`
}

func (c *CloudSignupConfig) GetMaxActiveTenants() int {
	return max(c.MaxActiveTenants, 0)
}

func (c *CloudSignupConfig) GetMaxSignupsPerDay() int {
	return max(c.MaxSignupsPerDay, 0)
}

func (c *CloudSignupConfig) GetPerIPPerHour() int {
	if c.PerIPPerHour <= 0 {
		return DefaultCloudSignupPerIPPerHour
	}

	return c.PerIPPerHour
}

func (c *CloudSignupConfig) GetVerificationTokenTTL() time.Duration {
	if c.VerificationTokenTTL <= 0 {
		return DefaultCloudSignupVerificationTokenTTL
	}

	return c.VerificationTokenTTL
}

func (c *CloudSignupConfig) GetAllowedEmailDomains() []string {
	domains := make([]string, 0, len(c.AllowedEmailDomains))
	for _, domain := range c.AllowedEmailDomains {
		normalized := strings.ToLower(strings.TrimSpace(domain))
		if normalized != "" {
			domains = append(domains, normalized)
		}
	}

	return domains
}

func (c *CloudSignupConfig) GetTermsURL() string {
	if trimmed := strings.TrimSpace(c.TermsURL); trimmed != "" {
		return trimmed
	}

	return DefaultCloudSignupTermsURL
}

func (c *CloudSignupConfig) GetPrivacyURL() string {
	if trimmed := strings.TrimSpace(c.PrivacyURL); trimmed != "" {
		return trimmed
	}

	return DefaultCloudSignupPrivacyURL
}

type CloudTurnstileConfig struct {
	Enabled   bool          `mapstructure:"enabled"`
	SiteKey   string        `mapstructure:"siteKey"`
	SecretKey string        `mapstructure:"secretKey"`
	VerifyURL string        `mapstructure:"verifyUrl" validate:"omitempty,url"`
	Timeout   time.Duration `mapstructure:"timeout"   validate:"omitempty,min=1s,max=30s"`
}

var turnstileTestSecrets = map[string]struct{}{
	"1x0000000000000000000000000000000AA": {},
	"2x0000000000000000000000000000000AA": {},
	"3x0000000000000000000000000000000AA": {},
}

func IsTurnstileTestSecret(secret string) bool {
	_, ok := turnstileTestSecrets[strings.TrimSpace(secret)]
	return ok
}

func (c *CloudTurnstileConfig) UsesTestSecret() bool {
	return IsTurnstileTestSecret(c.SecretKey)
}

func (c *CloudTurnstileConfig) GetVerifyURL() string {
	if trimmed := strings.TrimSpace(c.VerifyURL); trimmed != "" {
		return trimmed
	}

	return DefaultCloudTurnstileVerifyURL
}

func (c *CloudTurnstileConfig) GetTimeout() time.Duration {
	if c.Timeout <= 0 {
		return DefaultCloudTurnstileTimeout
	}

	return c.Timeout
}

type CloudSystemEmailConfig struct {
	Provider    string        `mapstructure:"provider"    validate:"omitempty,oneof=resend"`
	APIKey      string        `mapstructure:"apiKey"`
	FromAddress string        `mapstructure:"fromAddress" validate:"omitempty,email"`
	FromName    string        `mapstructure:"fromName"    validate:"omitempty,max=100"`
	ReplyTo     string        `mapstructure:"replyTo"     validate:"omitempty,email"`
	Timeout     time.Duration `mapstructure:"timeout"     validate:"omitempty,min=1s,max=60s"`
}

func (c *CloudSystemEmailConfig) GetProvider() string {
	if trimmed := strings.ToLower(strings.TrimSpace(c.Provider)); trimmed != "" {
		return trimmed
	}

	return CloudSystemEmailProviderResend
}

func (c *CloudSystemEmailConfig) GetFromAddress() string {
	if trimmed := strings.TrimSpace(c.FromAddress); trimmed != "" {
		return trimmed
	}

	return DefaultCloudSystemEmailFromAddress
}

func (c *CloudSystemEmailConfig) GetFromName() string {
	if trimmed := strings.TrimSpace(c.FromName); trimmed != "" {
		return trimmed
	}

	return DefaultCloudSystemEmailFromName
}

func (c *CloudSystemEmailConfig) GetReplyTo() string {
	return strings.TrimSpace(c.ReplyTo)
}

func (c *CloudSystemEmailConfig) GetTimeout() time.Duration {
	if c.Timeout <= 0 {
		return DefaultCloudSystemEmailTimeout
	}

	return c.Timeout
}

func (c *CloudSystemEmailConfig) HasAPIKey() bool {
	return strings.TrimSpace(c.APIKey) != ""
}

type CloudTrialConfig struct {
	Lifetime      time.Duration `mapstructure:"lifetime"      validate:"omitempty,min=1h"`
	ReadOnlyGrace time.Duration `mapstructure:"readOnlyGrace" validate:"omitempty,min=0"`
}

func (c *CloudTrialConfig) GetLifetime() time.Duration {
	if c.Lifetime <= 0 {
		return DefaultCloudTrialLifetime
	}

	return c.Lifetime
}

func (c *CloudTrialConfig) GetReadOnlyGrace() time.Duration {
	if c.ReadOnlyGrace <= 0 {
		return DefaultCloudTrialReadOnlyGrace
	}

	return c.ReadOnlyGrace
}

type CloudFreePlanConfig struct {
	Limits map[string]map[string]int64 `mapstructure:"limits"`
}

func (c *CloudFreePlanConfig) GetLimitOverrides() map[string]int64 {
	overrides := make(map[string]int64, len(c.Limits)*2)
	for group, entries := range c.Limits {
		for name, value := range entries {
			overrides[strings.ToLower(strings.TrimSpace(group))+"."+strings.ToLower(strings.TrimSpace(name))] = value
		}
	}

	return overrides
}

const defaultControlPlaneMaxProvisioningBodyBytes int64 = 1 << 20

type GraphQLAccessMode string

const (
	GraphQLAccessModeDisabled GraphQLAccessMode = "disabled"
	GraphQLAccessModeObserve  GraphQLAccessMode = "observe"
	GraphQLAccessModeEnforce  GraphQLAccessMode = "enforce"
)

type PlatformControlPlaneConfig struct {
	Enabled                  bool              `mapstructure:"enabled"`
	Endpoint                 string            `mapstructure:"endpoint"                 validate:"omitempty,url,no_trailing_slash"`
	APIKey                   string            `mapstructure:"apiKey"`
	Timeout                  time.Duration     `mapstructure:"timeout"`
	HeartbeatInterval        time.Duration     `mapstructure:"heartbeatInterval"`
	TenantSyncInterval       time.Duration     `mapstructure:"tenantSyncInterval"`
	FailOpenOnError          bool              `mapstructure:"failOpenOnError"`
	MaxProvisioningBodyBytes int64             `mapstructure:"maxProvisioningBodyBytes" validate:"omitempty,min=1024"`
	GraphQLAccessMode        GraphQLAccessMode `mapstructure:"graphqlAccessMode"        validate:"omitempty,oneof=disabled observe enforce"`
	DisableLegacyGrants      bool              `mapstructure:"disableLegacyGrants"`
}

func (c *PlatformControlPlaneConfig) HonorLegacyGrants() bool {
	return !c.DisableLegacyGrants
}

func (c *PlatformControlPlaneConfig) GetGraphQLAccessMode() GraphQLAccessMode {
	switch c.GraphQLAccessMode {
	case GraphQLAccessModeDisabled, GraphQLAccessModeObserve, GraphQLAccessModeEnforce:
		return c.GraphQLAccessMode
	default:
		return GraphQLAccessModeDisabled
	}
}

func (c *PlatformControlPlaneConfig) GetMaxProvisioningBodyBytes() int64 {
	if c.MaxProvisioningBodyBytes <= 0 {
		return defaultControlPlaneMaxProvisioningBodyBytes
	}

	return c.MaxProvisioningBodyBytes
}

func (c *PlatformControlPlaneConfig) GetTimeout() time.Duration {
	if c.Timeout <= 0 {
		return 5 * time.Second
	}

	return c.Timeout
}

func (c *PlatformControlPlaneConfig) GetHeartbeatInterval() time.Duration {
	if c.HeartbeatInterval <= 0 {
		return 5 * time.Minute
	}

	return c.HeartbeatInterval
}

func (c *PlatformControlPlaneConfig) GetTenantSyncInterval() time.Duration {
	if c.TenantSyncInterval <= 0 {
		return time.Hour
	}

	return c.TenantSyncInterval
}

// NetworkPulseConfig gates the instance-wide figures the sign-in screen shows beside
// the credential receipt. It is disabled by default and must be turned on deliberately:
// the endpoint answers before any session exists, so on an internet-facing deployment
// anyone who can load the login page can read the shipment volume and service level of
// every organization on the instance.
type NetworkPulseConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// CacheTTL bounds how often an anonymous caller can make the database aggregate.
	// Zero falls back to DefaultNetworkPulseCacheTTL rather than to no caching.
	CacheTTL time.Duration `mapstructure:"cacheTtl" validate:"omitempty,min=0"`
}

const DefaultNetworkPulseCacheTTL = time.Minute

func (c NetworkPulseConfig) GetCacheTTL() time.Duration {
	if c.CacheTTL <= 0 {
		return DefaultNetworkPulseCacheTTL
	}
	return c.CacheTTL
}
