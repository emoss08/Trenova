package securityconfig

import (
	"crypto/hkdf"
	"crypto/sha256"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

var (
	ErrInvalidHostPrefixCookie = errors.New(
		"__Host- session cookies require secure=true, httpOnly=true, sameSite=strict, path=/, and an empty domain",
	)
	ErrSessionIdleTimeoutExceedsMaxAge = errors.New(
		"security.session.idleTimeout cannot be longer than security.session.maxAge",
	)
	ErrSessionIdleWarningOutOfRange = errors.New(
		"security.session.idleWarning must be set and shorter than security.session.idleTimeout",
	)
)

type Config struct {
	Session    SessionConfig    `mapstructure:"session"    validate:"required"`
	APIToken   APITokenConfig   `mapstructure:"apiToken"`
	RateLimit  RateLimitConfig  `mapstructure:"rateLimit"`
	CSRF       CSRFConfig       `mapstructure:"csrf"`
	Encryption EncryptionConfig `mapstructure:"encryption"`
	GraphQL    GraphQLConfig    `mapstructure:"graphql"`

	Idempotency IdempotencyConfig `mapstructure:"idempotency"`

	PasswordReset PasswordResetConfig `mapstructure:"passwordReset"`

	SSO SSOConfig `mapstructure:"sso"`
	MFA MFAConfig `mapstructure:"mfa"`
}

type MFAConfig struct {
	RecoveryCodePepper string `mapstructure:"recoveryCodePepper" validate:"omitempty,min=32"`
}

const recoveryCodePepperInfo = "trenova mfa recovery code pepper"

func (c *Config) RecoveryCodePepperKey() ([]byte, error) {
	if c == nil {
		return hkdf.Key(sha256.New, nil, nil, recoveryCodePepperInfo, sha256.Size)
	}
	secret := strings.TrimSpace(c.MFA.RecoveryCodePepper)
	if secret == "" {
		secret = c.Session.Secret
	}
	return hkdf.Key(sha256.New, []byte(secret), nil, recoveryCodePepperInfo, sha256.Size)
}

const defaultSSOProviderCacheTTL = time.Hour

type SSOConfig struct {
	AllowPrivateNetworkIssuers bool          `mapstructure:"allowPrivateNetworkIssuers"`
	ProviderCacheTTL           time.Duration `mapstructure:"providerCacheTtl"           validate:"omitempty,min=0"`
}

func (c *Config) SSOIssuerPolicy() httpsafe.Policy {
	if c == nil {
		return httpsafe.Policy{}
	}
	return c.SSO.IssuerPolicy()
}

func (c *SSOConfig) IssuerPolicy() httpsafe.Policy {
	return httpsafe.Policy{AllowPrivateNetworks: c != nil && c.AllowPrivateNetworkIssuers}
}

func (c *SSOConfig) GetProviderCacheTTL() time.Duration {
	if c == nil || c.ProviderCacheTTL <= 0 {
		return defaultSSOProviderCacheTTL
	}
	return c.ProviderCacheTTL
}

const (
	defaultPasswordResetTokenTTL           = 30 * time.Minute
	defaultPasswordResetInviteTokenTTL     = 7 * 24 * time.Hour
	defaultPasswordResetMaxRequestsPerHour = 5
)

// PasswordResetConfig governs the unauthenticated "forgot password" flow.
//
// There is no enable switch: being able to recover an account is not optional. What
// is configurable is where the link points and how hard the endpoint can be leaned
// on, because both depend on the deployment.
type PasswordResetConfig struct {
	// BaseURL is the origin the reset link is built against, e.g.
	// https://app.example.com. The server cannot infer it from the request without
	// trusting a Host header an attacker controls, which is how reset links end up
	// pointing at somebody else's domain.
	BaseURL string `mapstructure:"baseUrl"            validate:"omitempty,url"`
	// TokenTTL is how long a link stays redeemable. Short by default: a reset link is
	// a bearer credential sitting in a mailbox.
	TokenTTL       time.Duration `mapstructure:"tokenTtl"           validate:"omitempty,min=0"`
	InviteTokenTTL time.Duration `mapstructure:"inviteTokenTtl"     validate:"omitempty,min=0"`
	// MaxRequestsPerHour caps how many links one account can be sent in an hour, so
	// the endpoint cannot be used to flood somebody's inbox.
	MaxRequestsPerHour int `mapstructure:"maxRequestsPerHour" validate:"omitempty,min=1"`
}

func (c PasswordResetConfig) GetBaseURL() string {
	return strings.TrimSuffix(c.BaseURL, "/")
}

func (c PasswordResetConfig) GetTokenTTL() time.Duration {
	if c.TokenTTL <= 0 {
		return defaultPasswordResetTokenTTL
	}
	return c.TokenTTL
}

func (c PasswordResetConfig) GetInviteTokenTTL() time.Duration {
	if c.InviteTokenTTL <= 0 {
		return defaultPasswordResetInviteTokenTTL
	}
	return c.InviteTokenTTL
}

func (c PasswordResetConfig) GetMaxRequestsPerHour() int {
	if c.MaxRequestsPerHour <= 0 {
		return defaultPasswordResetMaxRequestsPerHour
	}
	return c.MaxRequestsPerHour
}

const (
	defaultGraphQLMaxRequestBodyBytes       int64 = 1 << 20
	defaultGraphQLCostBudgetPointsPerMin          = 6_000_000
	defaultGraphQLCostBudgetBurst                 = 3_000_000
	defaultGraphQLCostBudgetCleanupInterval       = 5 * time.Minute
)

const (
	defaultIdempotencyRecordTTL        = 24 * time.Hour
	defaultIdempotencyLockTTL          = 2 * time.Minute
	defaultIdempotencyStoreTimeout     = 500 * time.Millisecond
	defaultIdempotencyMaxRequestBytes  = 4 << 20
	defaultIdempotencyMaxResponseBytes = 1 << 20
)

type IdempotencyConfig struct {
	Disabled         bool          `mapstructure:"disabled"`
	RecordTTL        time.Duration `mapstructure:"recordTTL"`
	LockTTL          time.Duration `mapstructure:"lockTTL"`
	StoreTimeout     time.Duration `mapstructure:"storeTimeout"`
	MaxRequestBytes  int64         `mapstructure:"maxRequestBytes"  validate:"omitempty,min=1024"`
	MaxResponseBytes int           `mapstructure:"maxResponseBytes" validate:"omitempty,min=1024"`
}

func (c *IdempotencyConfig) GetRecordTTL() time.Duration {
	return timeutils.WithDefaultDuration(c.RecordTTL, defaultIdempotencyRecordTTL)
}

func (c *IdempotencyConfig) GetLockTTL() time.Duration {
	return timeutils.WithDefaultDuration(c.LockTTL, defaultIdempotencyLockTTL)
}

func (c *IdempotencyConfig) GetStoreTimeout() time.Duration {
	return timeutils.WithDefaultDuration(c.StoreTimeout, defaultIdempotencyStoreTimeout)
}

func (c *IdempotencyConfig) GetMaxRequestBytes() int64 {
	if c.MaxRequestBytes <= 0 {
		return defaultIdempotencyMaxRequestBytes
	}
	return c.MaxRequestBytes
}

func (c *IdempotencyConfig) GetMaxResponseBytes() int {
	return intutils.WithDefault(c.MaxResponseBytes, defaultIdempotencyMaxResponseBytes)
}

type GraphQLConfig struct {
	MaxRequestBodyBytes    int64                   `mapstructure:"maxRequestBodyBytes"    validate:"omitempty,min=1024"`
	PersistedDocumentsPath string                  `mapstructure:"persistedDocumentsPath"`
	CostBudget             GraphQLCostBudgetConfig `mapstructure:"costBudget"`
}

func (c *GraphQLConfig) GetMaxRequestBodyBytes() int64 {
	if c.MaxRequestBodyBytes <= 0 {
		return defaultGraphQLMaxRequestBodyBytes
	}
	return c.MaxRequestBodyBytes
}

type GraphQLCostBudgetConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	PointsPerMinute int           `mapstructure:"pointsPerMinute" validate:"omitempty,min=1"`
	Burst           int           `mapstructure:"burst"           validate:"omitempty,min=1"`
	CleanupInterval time.Duration `mapstructure:"cleanupInterval"`
}

func (c *GraphQLCostBudgetConfig) GetPointsPerMinute() int {
	if c.PointsPerMinute <= 0 {
		return defaultGraphQLCostBudgetPointsPerMin
	}
	return c.PointsPerMinute
}

func (c *GraphQLCostBudgetConfig) GetBurst() int {
	if c.Burst <= 0 {
		return defaultGraphQLCostBudgetBurst
	}
	return c.Burst
}

func (c *GraphQLCostBudgetConfig) GetCleanupInterval() time.Duration {
	if c.CleanupInterval <= 0 {
		return defaultGraphQLCostBudgetCleanupInterval
	}
	return c.CleanupInterval
}

type SessionConfig struct {
	Secret        string        `mapstructure:"secret"        validate:"required,min=32"`
	Name          string        `mapstructure:"name"          validate:"required"`
	MaxAge        time.Duration `mapstructure:"maxAge"        validate:"required,min=1m"`
	IdleTimeout   time.Duration `mapstructure:"idleTimeout"   validate:"omitempty,min=5m"`
	IdleWarning   time.Duration `mapstructure:"idleWarning"   validate:"omitempty,min=10s"`
	HTTPOnly      bool          `mapstructure:"httpOnly"`
	Secure        bool          `mapstructure:"secure"`
	SameSite      string        `mapstructure:"sameSite"      validate:"required,oneof=strict lax none"`
	Domain        string        `mapstructure:"domain"`
	Path          string        `mapstructure:"path"          validate:"required"`
	RefreshWindow time.Duration `mapstructure:"refreshWindow"`
}

func (s *SessionConfig) IdleTimeoutEnabled() bool {
	return s.IdleTimeout > 0
}

func (s *SessionConfig) GetSameSite() http.SameSite {
	switch strings.ToLower(s.SameSite) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

type APITokenConfig struct {
	Enabled            bool          `mapstructure:"enabled"`
	DefaultExpiry      time.Duration `mapstructure:"defaultExpiry"`
	MaxExpiry          time.Duration `mapstructure:"maxExpiry"`
	MaxTokensPerUser   int           `mapstructure:"maxTokensPerUser"`
	UsageFlushInterval time.Duration `mapstructure:"usageFlushInterval"`
	UsageUpdateTimeout time.Duration `mapstructure:"usageUpdateTimeout"`
	UsageMaxPending    int           `mapstructure:"usageMaxPending"`
}

func (c *APITokenConfig) GetUsageFlushInterval() time.Duration {
	if c.UsageFlushInterval <= 0 {
		return 10 * time.Second
	}
	return c.UsageFlushInterval
}

func (c *APITokenConfig) GetUsageUpdateTimeout() time.Duration {
	if c.UsageUpdateTimeout <= 0 {
		return 3 * time.Second
	}
	return c.UsageUpdateTimeout
}

func (c *APITokenConfig) GetUsageMaxPending() int {
	if c.UsageMaxPending <= 0 {
		return 10000
	}
	return c.UsageMaxPending
}

type CSRFConfig struct {
	TokenName      string                 `mapstructure:"tokenName"      validate:"required"`
	HeaderName     string                 `mapstructure:"headerName"     validate:"required"`
	TrustedOrigins []string               `mapstructure:"trustedOrigins" validate:"omitempty,dive,origin_or_wildcard"`
	BrowserGuard   CSRFBrowserGuardConfig `mapstructure:"browserGuard"`
}

type CSRFBrowserGuardConfig struct {
	Mode string `mapstructure:"mode" validate:"required,oneof=enforce report off"`
}

var InsecureDefaultValues = []string{"change", "secret", "example"}

func Defaults(env string) map[string]any {
	defaults := map[string]any{
		"session.name":                                "trv-session-id",
		"session.maxAge":                              "12h",
		"session.idleTimeout":                         "30m",
		"session.idleWarning":                         "2m",
		"session.httpOnly":                            true,
		"session.secure":                              false,
		"session.sameSite":                            "lax",
		"session.path":                                "/",
		"session.refreshWindow":                       "1h",
		"apiToken.enabled":                            true,
		"apiToken.defaultExpiry":                      "0s",
		"apiToken.maxExpiry":                          "8760h",
		"apiToken.maxTokensPerUser":                   100,
		"csrf.tokenName":                              "csrf_token",
		"csrf.headerName":                             "X-CSRF-Token",
		"csrf.browserGuard.mode":                      "report",
		"rateLimit.enabled":                           true,
		"rateLimit.cleanupInterval":                   "1m",
		"rateLimit.store":                             RateLimitStoreRedis,
		"rateLimit.failureMode":                       RateLimitFailureModeLocal,
		"rateLimit.keyPrefix":                         defaultRateLimitKeyPrefix,
		"rateLimit.storeTimeout":                      "250ms",
		"rateLimit.exemptPathPrefixes":                []string{},
		"encryption.allowLocalKeyManagerInProduction": false,
	}
	if env == appconfig.EnvProduction || env == appconfig.EnvStaging {
		defaults["csrf.browserGuard.mode"] = "enforce"
	}
	for _, scope := range []RateLimitScope{
		RateLimitScopeAnonymous,
		RateLimitScopeAPIKey,
		RateLimitScopeTenant,
		RateLimitScopePublicToken,
	} {
		prefix := "rateLimit." + string(scope)
		defaults[prefix+".disabled"] = false
		defaults[prefix+".requestsPerMinute"] = 0
		defaults[prefix+".burstSize"] = 0
	}

	return defaults
}

func (c *Config) Validate() error {
	if err := c.validateHostPrefixCookie(); err != nil {
		return err
	}

	return c.validateSessionLifetime()
}

func (c *Config) ValidateProduction() error {
	if !c.Session.Secure ||
		!c.Session.HTTPOnly ||
		!strings.EqualFold(c.Session.SameSite, "strict") {
		return ErrProductionSessionCookieRequired
	}
	encryptionMode := strings.ToLower(strings.TrimSpace(c.Encryption.Mode))
	if encryptionMode != EncryptionModeEnvelope {
		return ErrProductionEncryptionModeRequired
	}
	if err := c.Encryption.validateProductionKeyManager(); err != nil {
		return err
	}

	return validateProductionRecoveryCodePepper(c.MFA.RecoveryCodePepper)
}

func (c *Config) validateHostPrefixCookie() error {
	if !strings.HasPrefix(c.Session.Name, "__Host-") {
		return nil
	}

	if !c.Session.Secure ||
		!c.Session.HTTPOnly ||
		c.Session.Domain != "" ||
		c.Session.Path != "/" ||
		!strings.EqualFold(c.Session.SameSite, "strict") {
		return ErrInvalidHostPrefixCookie
	}

	return nil
}

func (c *Config) validateSessionLifetime() error {
	sessionCfg := c.Session
	if !sessionCfg.IdleTimeoutEnabled() {
		return nil
	}
	if sessionCfg.IdleTimeout > sessionCfg.MaxAge {
		return ErrSessionIdleTimeoutExceedsMaxAge
	}
	if sessionCfg.IdleWarning <= 0 || sessionCfg.IdleWarning >= sessionCfg.IdleTimeout {
		return ErrSessionIdleWarningOutOfRange
	}

	return nil
}
