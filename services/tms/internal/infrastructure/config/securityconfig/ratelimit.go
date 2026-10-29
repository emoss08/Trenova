package securityconfig

import (
	"time"

	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

type RateLimitScope string

const (
	RateLimitScopeAnonymous   RateLimitScope = "anonymous"
	RateLimitScopeAPIKey      RateLimitScope = "apiKey"
	RateLimitScopeTenant      RateLimitScope = "tenant"
	RateLimitScopePublicToken RateLimitScope = "publicToken"
)

const (
	RateLimitStoreRedis  = "redis"
	RateLimitStoreMemory = "memory"

	RateLimitFailureModeLocal = "local"
	RateLimitFailureModeAllow = "allow"
	RateLimitFailureModeDeny  = "deny"

	defaultRateLimitCleanupInterval = time.Minute
	defaultRateLimitStoreTimeout    = 250 * time.Millisecond
	defaultRateLimitKeyPrefix       = "ratelimit:v1"
)

type RateLimitScopeConfig struct {
	Disabled          bool `mapstructure:"disabled"`
	RequestsPerMinute int  `mapstructure:"requestsPerMinute" validate:"omitempty,min=1,max=1000000"`
	BurstSize         int  `mapstructure:"burstSize"         validate:"omitempty,min=1,max=100000"`
}

type RateLimitConfig struct {
	Enabled            bool                 `mapstructure:"enabled"`
	Store              string               `mapstructure:"store"              validate:"omitempty,oneof=redis memory"`
	FailureMode        string               `mapstructure:"failureMode"        validate:"omitempty,oneof=local allow deny"`
	KeyPrefix          string               `mapstructure:"keyPrefix"`
	StoreTimeout       time.Duration        `mapstructure:"storeTimeout"`
	CleanupInterval    time.Duration        `mapstructure:"cleanupInterval"`
	ExemptPathPrefixes []string             `mapstructure:"exemptPathPrefixes"`
	Anonymous          RateLimitScopeConfig `mapstructure:"anonymous"`
	APIKey             RateLimitScopeConfig `mapstructure:"apiKey"`
	Tenant             RateLimitScopeConfig `mapstructure:"tenant"`
	PublicToken        RateLimitScopeConfig `mapstructure:"publicToken"`
}

type RateLimitScopePolicy struct {
	Enabled           bool
	RequestsPerMinute int
	BurstSize         int
}

type rateLimitScopeDefaults struct {
	requestsPerMinute int
	burstSize         int
}

var rateLimitScopeDefaultTable = map[RateLimitScope]rateLimitScopeDefaults{
	RateLimitScopeAnonymous:   {requestsPerMinute: 300, burstSize: 60},
	RateLimitScopeAPIKey:      {requestsPerMinute: 300, burstSize: 30},
	RateLimitScopeTenant:      {requestsPerMinute: 6000, burstSize: 600},
	RateLimitScopePublicToken: {requestsPerMinute: 60, burstSize: 20},
}

func (c *RateLimitConfig) GetCleanupInterval() time.Duration {
	return timeutils.WithDefaultDuration(c.CleanupInterval, defaultRateLimitCleanupInterval)
}

func (c *RateLimitConfig) GetStore() string {
	if c.Store == "" {
		return RateLimitStoreRedis
	}
	return c.Store
}

func (c *RateLimitConfig) GetFailureMode() string {
	if c.FailureMode == "" {
		return RateLimitFailureModeLocal
	}
	return c.FailureMode
}

func (c *RateLimitConfig) GetKeyPrefix() string {
	if c.KeyPrefix == "" {
		return defaultRateLimitKeyPrefix
	}
	return c.KeyPrefix
}

func (c *RateLimitConfig) GetStoreTimeout() time.Duration {
	return timeutils.WithDefaultDuration(c.StoreTimeout, defaultRateLimitStoreTimeout)
}

func (c *RateLimitConfig) ScopePolicy(scope RateLimitScope) RateLimitScopePolicy {
	var scopeCfg RateLimitScopeConfig
	switch scope {
	case RateLimitScopeAnonymous:
		scopeCfg = c.Anonymous
	case RateLimitScopeAPIKey:
		scopeCfg = c.APIKey
	case RateLimitScopeTenant:
		scopeCfg = c.Tenant
	case RateLimitScopePublicToken:
		scopeCfg = c.PublicToken
	default:
		return RateLimitScopePolicy{}
	}

	defaults := rateLimitScopeDefaultTable[scope]

	return RateLimitScopePolicy{
		Enabled: c.Enabled && !scopeCfg.Disabled,
		RequestsPerMinute: intutils.WithDefault(
			scopeCfg.RequestsPerMinute,
			defaults.requestsPerMinute,
		),
		BurstSize: intutils.WithDefault(scopeCfg.BurstSize, defaults.burstSize),
	}
}
