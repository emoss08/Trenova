package config

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/pkg/dbdialect"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

type ServerConfig struct {
	Host              string        `mapstructure:"host"              validate:"required,hostname|ip"`
	Port              int           `mapstructure:"port"              validate:"required,min=1,max=65535"`
	Mode              string        `mapstructure:"mode"              validate:"required,oneof=debug release test"`
	ReadTimeout       time.Duration `mapstructure:"readTimeout"`
	ReadHeaderTimeout time.Duration `mapstructure:"readHeaderTimeout"`
	WriteTimeout      time.Duration `mapstructure:"writeTimeout"`
	IdleTimeout       time.Duration `mapstructure:"idleTimeout"`
	ShutdownTimeout   time.Duration `mapstructure:"shutdownTimeout"`
	RequestTimeout    time.Duration `mapstructure:"requestTimeout"`
	TrustedProxies    []string      `mapstructure:"trustedProxies"`
	TrustedPlatform   string        `mapstructure:"trustedPlatform"   validate:"omitempty,oneof=cloudflare google-app-engine flyio"`
	CORS              CORSConfig    `mapstructure:"cors,omitempty"`
}

type MonitoringConfig struct {
	Metrics MetricsConfig       `mapstructure:"metrics,omitempty"`
	Tracing TracingConfig       `mapstructure:"tracing,omitempty"`
	Health  HealthConfig        `mapstructure:"health"`
	Pprof   PprofConfig         `mapstructure:"pprof"`
	GraphQL GraphQLObservConfig `mapstructure:"graphql,omitempty"`
}

const (
	defaultGraphQLSlowOperationAfter   = time.Second
	defaultGraphQLMaxTrackedOperations = 2000
)

type GraphQLObservConfig struct {
	ResolverMetrics      bool          `mapstructure:"resolverMetrics"`
	SlowOperationAfter   time.Duration `mapstructure:"slowOperationAfter"`
	MaxTrackedOperations int           `mapstructure:"maxTrackedOperations" validate:"omitempty,min=1"`
}

func (c *GraphQLObservConfig) GetSlowOperationAfter() time.Duration {
	if c.SlowOperationAfter <= 0 {
		return defaultGraphQLSlowOperationAfter
	}
	return c.SlowOperationAfter
}

func (c *GraphQLObservConfig) GetMaxTrackedOperations() int {
	if c.MaxTrackedOperations <= 0 {
		return defaultGraphQLMaxTrackedOperations
	}
	return c.MaxTrackedOperations
}

type TwilioConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	AccountSID string `mapstructure:"accountSID" validate:"required_if=Enabled true"`
	AuthToken  string `mapstructure:"authToken"  validate:"required_if=Enabled true"`
	FromNumber string `mapstructure:"fromNumber" validate:"required_if=Enabled true"`
}

type FoonyConfig struct {
	APIKey string `mapstructure:"apiKey" validate:"required"`
}

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"    validate:"omitempty,hostname|ip"`
	Port    int    `mapstructure:"port"    validate:"required_if=Enabled true,min=1,max=65535"`
	Path    string `mapstructure:"path"    validate:"required_if=Enabled true"`
}

func (c *MetricsConfig) GetHost() string {
	if strings.TrimSpace(c.Host) == "" {
		return "127.0.0.1"
	}
	return c.Host
}

type TracingConfig struct {
	Enabled      bool    `mapstructure:"enabled"`
	Provider     string  `mapstructure:"provider"     validate:"required_if=Enabled true,oneof=jaeger otlp otlp-grpc stdout"`
	Endpoint     string  `mapstructure:"endpoint"     validate:"required_if=Enabled true"`
	ServiceName  string  `mapstructure:"serviceName"  validate:"required_if=Enabled true"`
	SamplingRate float64 `mapstructure:"samplingRate" validate:"min=0,max=1"`
}

type HealthConfig struct {
	Path          string        `mapstructure:"path"          validate:"required"`
	ReadinessPath string        `mapstructure:"readinessPath"`
	LivenessPath  string        `mapstructure:"livenessPath"`
	CheckInterval time.Duration `mapstructure:"checkInterval"`
	Timeout       time.Duration `mapstructure:"timeout"`
}

type PprofConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"    validate:"omitempty,hostname|ip"`
	Port    int    `mapstructure:"port"    validate:"omitempty,min=1,max=65535"`
}

func (c *PprofConfig) GetHost() string {
	if strings.TrimSpace(c.Host) == "" {
		return "127.0.0.1"
	}
	return c.Host
}

func (c *PprofConfig) GetPort() int {
	if c.Port <= 0 {
		return 6060
	}
	return c.Port
}

type LoggingConfig struct {
	Level      string         `mapstructure:"level"          validate:"required,oneof=debug info warn error"`
	Format     string         `mapstructure:"format"         validate:"required,oneof=json text"`
	Output     string         `mapstructure:"output"         validate:"required,oneof=stdout stderr file"`
	File       *LogFileConfig `mapstructure:"file,omitempty"`
	Sampling   bool           `mapstructure:"sampling"`
	Stacktrace bool           `mapstructure:"stacktrace"`
}

type LogFileConfig struct {
	Path       string `mapstructure:"path"       validate:"required"`
	MaxSize    int    `mapstructure:"maxSize"    validate:"min=1,max=1000"`
	MaxAge     int    `mapstructure:"maxAge"     validate:"min=1,max=365"`
	MaxBackups int    `mapstructure:"maxBackups" validate:"min=0,max=100"`
	Compress   bool   `mapstructure:"compress"`
}

type SecurityConfig struct {
	Session    SessionConfig         `mapstructure:"session"    validate:"required"`
	APIToken   APITokenConfig        `mapstructure:"apiToken"`
	RateLimit  RateLimitConfig       `mapstructure:"rateLimit"`
	CSRF       CSRFConfig            `mapstructure:"csrf"`
	Encryption EncryptionConfig      `mapstructure:"encryption"`
	GraphQL    GraphQLSecurityConfig `mapstructure:"graphql"`

	PasswordReset PasswordResetConfig `mapstructure:"passwordReset"`
}

const (
	defaultPasswordResetTokenTTL           = 30 * time.Minute
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
	TokenTTL time.Duration `mapstructure:"tokenTtl"           validate:"omitempty,min=0"`
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

type GraphQLSecurityConfig struct {
	MaxRequestBodyBytes    int64                   `mapstructure:"maxRequestBodyBytes"    validate:"omitempty,min=1024"`
	PersistedDocumentsPath string                  `mapstructure:"persistedDocumentsPath"`
	CostBudget             GraphQLCostBudgetConfig `mapstructure:"costBudget"`
}

func (c *GraphQLSecurityConfig) GetMaxRequestBodyBytes() int64 {
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

const (
	EncryptionModeEnvelope = "envelope"
	EncryptionModeDisabled = "disabled"

	EncryptionKeyManagerLocal      = "local"
	EncryptionKeyManagerGCPAutokey = "gcp-autokey"
	EncryptionKeyManagerDisabled   = "disabled"
)

type EncryptionConfig struct {
	Mode       string       `mapstructure:"mode"       validate:"omitempty,oneof=envelope disabled"`
	KeyManager string       `mapstructure:"keyManager" validate:"omitempty,oneof=local gcp-autokey disabled"`
	Key        string       `mapstructure:"key"        validate:"omitempty,min=32"`
	GCPKMS     GCPKMSConfig `mapstructure:"gcpKms"`
}

type GCPKMSConfig struct {
	CryptoKey       string        `mapstructure:"cryptoKey"`
	KeyResource     string        `mapstructure:"keyResource"`
	CredentialsMode string        `mapstructure:"credentialsMode" validate:"omitempty,oneof=adc workload-identity credentials-file"`
	CredentialsFile string        `mapstructure:"credentialsFile"`
	Timeout         time.Duration `mapstructure:"timeout"`
	RetryAttempts   int           `mapstructure:"retryAttempts"   validate:"omitempty,min=1,max=10"`
}

type SessionConfig struct {
	Secret        string        `mapstructure:"secret"        validate:"required,min=32"`
	Name          string        `mapstructure:"name"          validate:"required"`
	MaxAge        time.Duration `mapstructure:"maxAge"        validate:"required,min=1m"`
	HTTPOnly      bool          `mapstructure:"httpOnly"`
	Secure        bool          `mapstructure:"secure"`
	SameSite      string        `mapstructure:"sameSite"      validate:"required,oneof=strict lax none"`
	Domain        string        `mapstructure:"domain"`
	Path          string        `mapstructure:"path"          validate:"required"`
	RefreshWindow time.Duration `mapstructure:"refreshWindow"`
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

type RateLimitScope string

const (
	RateLimitScopeAnonymous   RateLimitScope = "anonymous"
	RateLimitScopeUser        RateLimitScope = "user"
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

	defaultRateLimitRequestsPerMinute = 60
	defaultRateLimitBurstSize         = 10
	defaultRateLimitCleanupInterval   = time.Minute
	defaultRateLimitStoreTimeout      = 250 * time.Millisecond
	defaultRateLimitKeyPrefix         = "ratelimit:v1"
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
	RequestsPerMinute  int                  `mapstructure:"requestsPerMinute"  validate:"min=1,max=10000"`
	BurstSize          int                  `mapstructure:"burstSize"          validate:"min=1,max=1000"`
	CleanupInterval    time.Duration        `mapstructure:"cleanupInterval"`
	ExemptPathPrefixes []string             `mapstructure:"exemptPathPrefixes"`
	Anonymous          RateLimitScopeConfig `mapstructure:"anonymous"`
	User               RateLimitScopeConfig `mapstructure:"user"`
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
	RateLimitScopeUser:        {requestsPerMinute: 600, burstSize: 60},
	RateLimitScopeAPIKey:      {requestsPerMinute: 300, burstSize: 30},
	RateLimitScopeTenant:      {requestsPerMinute: 6000, burstSize: 600},
	RateLimitScopePublicToken: {requestsPerMinute: 10, burstSize: 5},
}

func (c *RateLimitConfig) GetRequestsPerMinute() int {
	return intutils.WithDefault(c.RequestsPerMinute, defaultRateLimitRequestsPerMinute)
}

func (c *RateLimitConfig) GetBurstSize() int {
	return intutils.WithDefault(c.BurstSize, defaultRateLimitBurstSize)
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
	defaults := rateLimitScopeDefaults{
		requestsPerMinute: c.GetRequestsPerMinute(),
		burstSize:         c.GetBurstSize(),
	}

	switch scope {
	case RateLimitScopeAnonymous:
		scopeCfg = c.Anonymous
	case RateLimitScopeUser:
		scopeCfg = c.User
	case RateLimitScopeAPIKey:
		scopeCfg = c.APIKey
	case RateLimitScopeTenant:
		scopeCfg = c.Tenant
	case RateLimitScopePublicToken:
		scopeCfg = c.PublicToken
	default:
		return RateLimitScopePolicy{}
	}

	if d, ok := rateLimitScopeDefaultTable[scope]; ok {
		defaults = d
	}

	return RateLimitScopePolicy{
		Enabled: c.Enabled && !scopeCfg.Disabled,
		RequestsPerMinute: intutils.WithDefault(
			scopeCfg.RequestsPerMinute,
			defaults.requestsPerMinute,
		),
		BurstSize: intutils.WithDefault(scopeCfg.BurstSize, defaults.burstSize),
	}
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

type DatabaseConfig struct {
	Driver           string        `mapstructure:"driver"                          validate:"omitempty,oneof=postgres sqlite"`
	Host             string        `mapstructure:"host"                            validate:"required_if=Driver postgres"`
	Port             int           `mapstructure:"port"                            validate:"required_if=Driver postgres,omitempty,min=1,max=65535"`
	Name             string        `mapstructure:"name"                            validate:"required_if=Driver postgres,omitempty,min=1,max=63"`
	User             string        `mapstructure:"user"                            validate:"required_if=Driver postgres,omitempty,min=1,max=63"`
	Password         string        `mapstructure:"password"                        validate:"required_if=Driver postgres"`
	SSLMode          string        `mapstructure:"sslMode"                         validate:"required_if=Driver postgres,omitempty,oneof=disable require verify-ca verify-full"`
	MaxIdleConns     int           `mapstructure:"maxIdleConns"                    validate:"min=1,max=1000"`
	MaxOpenConns     int           `mapstructure:"maxOpenConns"                    validate:"min=1,max=1000"`
	Verbose          bool          `mapstructure:"verbose"`
	ConnMaxLifetime  time.Duration `mapstructure:"connMaxLifetime"`
	ConnMaxIdleTime  time.Duration `mapstructure:"connMaxIdleTime"`
	StatementTimeout time.Duration `mapstructure:"statementTimeout"`
	LockTimeout      time.Duration `mapstructure:"lockTimeout"`
	IdleTxTimeout    time.Duration `mapstructure:"idleInTransactionSessionTimeout"`
	SQLite           SQLiteConfig  `mapstructure:"sqlite"`
}

type SQLiteConfig struct {
	Path        string        `mapstructure:"path"`
	JournalMode string        `mapstructure:"journalMode" validate:"omitempty,oneof=WAL DELETE TRUNCATE PERSIST MEMORY OFF"`
	Synchronous string        `mapstructure:"synchronous" validate:"omitempty,oneof=OFF NORMAL FULL EXTRA"`
	BusyTimeout time.Duration `mapstructure:"busyTimeout"`
	ForeignKeys *bool         `mapstructure:"foreignKeys"`
	CacheSizeKB int           `mapstructure:"cacheSizeKb"`
}

func (c *SQLiteConfig) GetPath() string {
	if c.Path == "" {
		return "trenova.db"
	}
	return c.Path
}

func (c *SQLiteConfig) GetJournalMode() string {
	if c.JournalMode == "" {
		return "WAL"
	}
	return c.JournalMode
}

func (c *SQLiteConfig) GetSynchronous() string {
	if c.Synchronous == "" {
		return "NORMAL"
	}
	return c.Synchronous
}

func (c *SQLiteConfig) GetBusyTimeout() time.Duration {
	if c.BusyTimeout <= 0 {
		return 5 * time.Second
	}
	return c.BusyTimeout
}

func (c *SQLiteConfig) GetForeignKeys() bool {
	if c.ForeignKeys == nil {
		return true
	}
	return *c.ForeignKeys
}

func (c *SQLiteConfig) GetCacheSizeKB() int {
	if c.CacheSizeKB <= 0 {
		return 64000
	}
	return c.CacheSizeKB
}

func (c *DatabaseConfig) GetDialect() dbdialect.Kind {
	kind, err := dbdialect.Parse(c.Driver)
	if err != nil {
		return dbdialect.DefaultKind
	}
	return kind
}

func (c *DatabaseConfig) GetStatementTimeout() time.Duration {
	if c.StatementTimeout <= 0 {
		return 10 * time.Second
	}
	return c.StatementTimeout
}

func (c *DatabaseConfig) GetLockTimeout() time.Duration {
	if c.LockTimeout <= 0 {
		return 5 * time.Second
	}
	return c.LockTimeout
}

func (c *DatabaseConfig) GetIdleTxTimeout() time.Duration {
	if c.IdleTxTimeout <= 0 {
		return 30 * time.Second
	}
	return c.IdleTxTimeout
}

type CORSConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	AllowedOrigins []string `mapstructure:"allowedOrigins" validate:"required_if=Enabled true,dive,origin_or_wildcard"`
	AllowedMethods []string `mapstructure:"allowedMethods" validate:"required_if=Enabled true"`
	AllowedHeaders []string `mapstructure:"allowedHeaders" validate:"required_if=Enabled true"`
	ExposeHeaders  []string `mapstructure:"exposeHeaders"`
	Credentials    bool     `mapstructure:"credentials"`
	MaxAge         int      `mapstructure:"maxAge"         validate:"min=0,max=86400"`
}

type CacheConfig struct {
	Host            string        `mapstructure:"host"            validate:"required"`
	Port            int           `mapstructure:"port"            validate:"required,min=1,max=65535"`
	Username        string        `mapstructure:"username"        validate:"omitempty,min=1,max=63"`
	Password        string        `mapstructure:"password"`
	DB              int           `mapstructure:"db"              validate:"min=0,max=15"`
	PoolSize        int           `mapstructure:"poolSize"        validate:"min=0,max=1000"`
	MinIdleConns    int           `mapstructure:"minIdleConns"    validate:"min=0,max=1000"`
	DialTimeout     time.Duration `mapstructure:"dialTimeout"`
	ReadTimeout     time.Duration `mapstructure:"readTimeout"`
	WriteTimeout    time.Duration `mapstructure:"writeTimeout"`
	PoolTimeout     time.Duration `mapstructure:"poolTimeout"`
	ConnMaxIdleTime time.Duration `mapstructure:"connMaxIdleTime"`
	ConnMaxLifetime time.Duration `mapstructure:"connMaxLifetime"`
	MaxRetries      int           `mapstructure:"maxRetries"      validate:"min=0,max=10"`
	MinRetryBackoff time.Duration `mapstructure:"minRetryBackoff"`
	MaxRetryBackoff time.Duration `mapstructure:"maxRetryBackoff"`
}

type SearchConfig struct {
	Enabled      bool              `mapstructure:"enabled"`
	DefaultLimit int               `mapstructure:"defaultLimit" validate:"omitempty,min=1,max=100"`
	Meilisearch  MeilisearchConfig `mapstructure:"meilisearch"`
}

func (c *SearchConfig) GetDefaultLimit() int {
	if c.DefaultLimit <= 0 {
		return 8
	}
	return c.DefaultLimit
}

// AIConfig is everything model-backed, in one section.
//
// The assistant, the agents, insight narration, the formula assistant and
// document extraction all run through the same completion router, so they
// share a master switch, the same timeouts and the same retry budget.
// Splitting them across a second section named after one feature is what
// produced the original confusion: the gate used to be
// documentIntelligence.enableAI, which ships false, so configuring a
// provider under AI Control produced "AI features are disabled" from a key
// named after a feature the operator was not using. The OCR settings live
// here too, because every one of them exists to feed the extraction above
// them.
type AIConfig struct {
	// Enabled is a pointer so an absent key means enabled rather than
	// disabled. Leaving it out is the common case and must not turn the
	// assistant off; the real gate is per-organization anyway, since the
	// router can only reach a provider row somebody configured and enabled.
	// This switch is here for an operator who wants to stop all of it at once.
	Enabled *bool `mapstructure:"enabled"`
	// VerdictCacheTTL is how long a scope verdict is shared between replicas.
	// A verdict is a pure function of the question, so it can be long; it is
	// bounded so a change to the classifier's prompt reaches every replica
	// within a day rather than never.
	VerdictCacheTTL time.Duration `mapstructure:"verdictCacheTtl" validate:"omitempty,min=0"`

	// Timeout bounds a plain reachability check against an endpoint.
	Timeout time.Duration `mapstructure:"timeout"`
	// ProbeTimeout bounds the Test Connection probe, which waits on a real
	// generation rather than only on a connection.
	ProbeTimeout time.Duration `mapstructure:"probeTimeout"`
	// CompletionTimeout bounds one blocking call to a model, including the
	// wait before it says anything.
	CompletionTimeout time.Duration `mapstructure:"completionTimeout"`
	// StreamIdleTimeout is how long a streaming reply may go silent.
	StreamIdleTimeout time.Duration `mapstructure:"streamIdleTimeout"`
	MaxRetries        int           `mapstructure:"maxRetries"        validate:"omitempty,min=0,max=10"`

	// TurnStreamKeyPrefix names the redis streams a turn's events are
	// published to, one per turn. They are a tail buffer a reader can rejoin,
	// never the transcript: that is in postgres and outlives all of this.
	TurnStreamKeyPrefix string `mapstructure:"turnStreamKeyPrefix" validate:"omitempty,max=64"`
	// TurnStreamMaxLen bounds one turn's stream. A turn that somehow produced
	// more events than this loses its oldest, which costs a reader who
	// reattaches the beginning of a reply they already watched arrive.
	TurnStreamMaxLen int64 `mapstructure:"turnStreamMaxLen" validate:"omitempty,min=100,max=1000000"`
	// TurnStreamTTL is how long a turn's events outlive the turn itself, for
	// a reader whose tab slept. Past it they read the saved conversation
	// instead, which is the real record anyway.
	TurnStreamTTL time.Duration `mapstructure:"turnStreamTtl"`
	// TurnStreamBlockTimeout is how long one read waits for the next event.
	// It bounds how quickly the relay notices a worker that died without
	// closing its stream, so it is short; the cost of a wake-up is one redis
	// round trip on an idle connection.
	TurnStreamBlockTimeout time.Duration `mapstructure:"turnStreamBlockTimeout"`

	// DocumentExtraction lets a model classify and extract uploaded
	// documents. Off by default; the OCR pipeline below runs either way.
	DocumentExtraction  bool `mapstructure:"documentExtraction"`
	MaxInputChars       int  `mapstructure:"maxInputChars"      validate:"omitempty,min=1000,max=500000"`
	ExtractionMaxTokens int  `mapstructure:"extractionMaxTokens" validate:"omitempty,min=256,max=32768"`

	// The OCR pipeline that turns a scan into the text a model reads. It
	// lives here rather than in a section of its own because every one of
	// these settings exists to feed the extraction above it.
	OCRCommand              string        `mapstructure:"ocrCommand"`
	OCRLanguage             string        `mapstructure:"ocrLanguage"`
	OCRTimeout              time.Duration `mapstructure:"ocrTimeout"`
	EnableOCRPreprocessing  bool          `mapstructure:"enableOcrPreprocessing"`
	OCRPreprocessingMode    string        `mapstructure:"ocrPreprocessingMode"`
	OCRMaxImageDimension    int           `mapstructure:"ocrMaxImageDimension"    validate:"omitempty,min=512,max=12000"`
	MaxOCRPages             int           `mapstructure:"maxOcrPages"             validate:"omitempty,min=1,max=500"`
	MaxExtractedChars       int           `mapstructure:"maxExtractedChars"       validate:"omitempty,min=1000,max=1000000"`
	ReconcileBatchSize      int           `mapstructure:"reconcileBatchSize"      validate:"omitempty,min=1,max=1000"`
	MaxConcurrentActivities int           `mapstructure:"maxConcurrentActivities" validate:"omitempty,min=1,max=64"`
}

const defaultVerdictCacheTTL = 24 * time.Hour

// GetVerdictCacheTTL is nil-safe: a guard built without this section shares
// verdicts for a day.
func (c *AIConfig) GetVerdictCacheTTL() time.Duration {
	if c == nil || c.VerdictCacheTTL <= 0 {
		return defaultVerdictCacheTTL
	}

	return c.VerdictCacheTTL
}

// AIEnabledKey names the configuration key in error messages, so a disabled
// install says which switch to look at rather than leaving it to be guessed.
const AIEnabledKey = "ai.enabled"

// AIEnabled is nil-safe on the receiver as well as the field. A service built
// without this section — which a test does, and which a future caller that
// forgets to wire it would — must fall to the same "available" default as an
// absent key, rather than panicking inside a gate whose whole job is to answer
// a yes-or-no question.
func (c *AIConfig) AIEnabled() bool {
	return c == nil || c.Enabled == nil || *c.Enabled
}

func (c *AIConfig) GetOCRCommand() string {
	if c.OCRCommand == "" {
		return "tesseract"
	}

	return c.OCRCommand
}

func (c *AIConfig) GetOCRLanguage() string {
	if c.OCRLanguage == "" {
		return "eng"
	}

	return c.OCRLanguage
}

func (c *AIConfig) GetOCRTimeout() time.Duration {
	if c.OCRTimeout <= 0 {
		return 45 * time.Second
	}

	return c.OCRTimeout
}

// DocumentExtractionEnabled says whether a model may read uploaded
// documents. It is separate from the master switch on purpose: reading a
// customer's paperwork with a model is a decision an operator makes on its
// own terms, and it ships off, while the assistant an organization has
// configured a provider for ships on.
func (c *AIConfig) DocumentExtractionEnabled() bool {
	return c != nil && c.DocumentExtraction
}

func (c *AIConfig) GetTimeout() time.Duration {
	if c.Timeout <= 0 {
		return 20 * time.Second
	}

	return c.Timeout
}

// GetProbeTimeout bounds the Test Connection call against a provider.
//
// A probe is not a reachability check: it asks the model to emit a two-field
// JSON object, because what the test is really for is finding out whether the
// endpoint honours a schema. So it has to wait for a real generation, and on
// a queued free tier — NVIDIA NIM, a shared router, a cold self-hosted model —
// the first response header can be thirty seconds away. Twenty seconds failed
// those providers with "could not reach the endpoint" when the endpoint was
// fine and merely busy. The ceiling is kept under the server's own request
// timeout so a slow probe returns a verdict rather than a 504.
func (c *AIConfig) GetProbeTimeout() time.Duration {
	if c.ProbeTimeout <= 0 {
		return 45 * time.Second
	}

	return c.ProbeTimeout
}

// GetCompletionTimeout bounds one blocking call to a model.
//
// Separate from GetTimeout, which is sized for a reachability probe: asking
// whether an endpoint answers is a second or two of work, while asking a model
// to read a prompt and write an answer is minutes on a loaded or self-hosted
// one. Sharing the probe's budget made a long answer fail outright on providers
// that do not stream, which looks to the reader exactly like an outage.
func (c *AIConfig) GetCompletionTimeout() time.Duration {
	if c.CompletionTimeout <= 0 {
		return 5 * time.Minute
	}

	return c.CompletionTimeout
}

// GetStreamIdleTimeout is how long a streaming reply may go silent.
//
// A stream cannot use GetTimeout: that one bounds a whole request, and
// http.Client applies it to reading the body, so a long answer arriving
// perfectly well would be severed partway through. What a stream needs bounded
// is silence, measured between reads, which is long by default because the gap
// before the first token of a considered answer is ordinary rather than a
// fault.
func (c *AIConfig) GetStreamIdleTimeout() time.Duration {
	if c.StreamIdleTimeout <= 0 {
		return 5 * time.Minute
	}

	return c.StreamIdleTimeout
}

// GetTurnStreamKeyPrefix names the redis key space a turn's events live in.
func (c *AIConfig) GetTurnStreamKeyPrefix() string {
	if c == nil || c.TurnStreamKeyPrefix == "" {
		return "assistant:turn"
	}

	return c.TurnStreamKeyPrefix
}

func (c *AIConfig) GetTurnStreamMaxLen() int64 {
	if c == nil || c.TurnStreamMaxLen <= 0 {
		return 10000
	}

	return c.TurnStreamMaxLen
}

func (c *AIConfig) GetTurnStreamTTL() time.Duration {
	if c == nil || c.TurnStreamTTL <= 0 {
		return time.Hour
	}

	return c.TurnStreamTTL
}

// GetTurnStreamBlockTimeout bounds one blocking read.
//
// It is deliberately shorter than the SSE keepalive, so a relay waiting on a
// silent turn wakes, writes its heartbeat and checks whether the turn is still
// alive, rather than sitting on a read until a proxy gives up on the reader.
func (c *AIConfig) GetTurnStreamBlockTimeout() time.Duration {
	if c == nil || c.TurnStreamBlockTimeout <= 0 {
		return 5 * time.Second
	}

	return c.TurnStreamBlockTimeout
}

func (c *AIConfig) GetMaxInputChars() int {
	if c.MaxInputChars <= 0 {
		return 24000
	}

	return c.MaxInputChars
}

func (c *AIConfig) GetExtractionMaxTokens() int {
	if c.ExtractionMaxTokens <= 0 {
		return 5000
	}

	return c.ExtractionMaxTokens
}

func (c *AIConfig) GetMaxRetries() int {
	if c.MaxRetries <= 0 {
		return 2
	}

	return c.MaxRetries
}

func (c *AIConfig) OCRPreprocessingEnabled() bool {
	return c.EnableOCRPreprocessing
}

func (c *AIConfig) GetOCRPreprocessingMode() string {
	if c.OCRPreprocessingMode == "" {
		return "standard"
	}

	return c.OCRPreprocessingMode
}

func (c *AIConfig) GetOCRMaxImageDimension() int {
	if c.OCRMaxImageDimension <= 0 {
		return 2400
	}

	return c.OCRMaxImageDimension
}

func (c *AIConfig) GetMaxOCRPages() int {
	if c.MaxOCRPages <= 0 {
		return 25
	}

	return c.MaxOCRPages
}

func (c *AIConfig) GetMaxConcurrentActivities() int {
	if c.MaxConcurrentActivities <= 0 {
		return 2
	}

	return c.MaxConcurrentActivities
}

func (c *AIConfig) GetMaxExtractedChars() int {
	if c.MaxExtractedChars <= 0 {
		return 200000
	}

	return c.MaxExtractedChars
}

func (c *AIConfig) GetReconcileBatchSize() int {
	if c.ReconcileBatchSize <= 0 {
		return 100
	}

	return c.ReconcileBatchSize
}

type MeilisearchConfig struct {
	URL     string                 `mapstructure:"url"     validate:"omitempty,url,no_trailing_slash"`
	APIKey  string                 `mapstructure:"apiKey"`
	Timeout time.Duration          `mapstructure:"timeout"`
	Indexes MeilisearchIndexConfig `mapstructure:"indexes"`
}

func (c *MeilisearchConfig) GetTimeout() time.Duration {
	if c.Timeout <= 0 {
		return 3 * time.Second
	}

	return c.Timeout
}

type MeilisearchIndexConfig struct {
	Shipments string `mapstructure:"shipments"`
	Customers string `mapstructure:"customers"`
	Workers   string `mapstructure:"workers"`
	Documents string `mapstructure:"documents"`
}

type TemporalConfig struct {
	HostPort     string                    `mapstructure:"hostPort"     validate:"required_without=Profile,omitempty,hostname_port"`
	Namespace    string                    `mapstructure:"namespace"`
	Identity     string                    `mapstructure:"identity"`
	Profile      string                    `mapstructure:"profile"      validate:"omitempty,max=128"`
	ConfigFile   string                    `mapstructure:"configFile"`
	APIKey       string                    `mapstructure:"apiKey"       validate:"omitempty,max=8192"`
	TLS          TemporalTLSConfig         `mapstructure:"tls"`
	Security     TemporalSecurityConfig    `mapstructure:"security"     validate:"required"`
	Interceptors TemporalInterceptorConfig `mapstructure:"interceptors"`
	Schedule     TemporalScheduleConfig    `mapstructure:"schedule"`
	Worker       TemporalWorkerConfig      `mapstructure:"worker"`
}

func (c *TemporalConfig) GetNamespace() string {
	if c.Namespace == "" {
		return "default"
	}

	return c.Namespace
}

func (c *TemporalConfig) GetIdentity() string {
	if c.Identity == "" {
		return "trenova-tms"
	}

	return c.Identity
}

func (c *TemporalConfig) UsesProfile() bool {
	return strings.TrimSpace(c.Profile) != ""
}

type TemporalTLSConfig struct {
	Enabled          bool   `mapstructure:"enabled"`
	ServerName       string `mapstructure:"serverName"`
	ServerCACertPath string `mapstructure:"serverCACertPath"`
	ClientCertPath   string `mapstructure:"clientCertPath"   validate:"required_with=ClientKeyPath"`
	ClientKeyPath    string `mapstructure:"clientKeyPath"    validate:"required_with=ClientCertPath"`
}

func (c *TemporalTLSConfig) IsEnabled() bool {
	return c.Enabled ||
		c.ServerName != "" ||
		c.ServerCACertPath != "" ||
		c.ClientCertPath != "" ||
		c.ClientKeyPath != ""
}

type TemporalSecurityConfig struct {
	EnableEncryption     bool   `mapstructure:"enableEncryption"`
	EncryptionKeyID      string `mapstructure:"encryptionKeyID"      validate:"required_if=EnableEncryption true,min=1,max=100"`
	EnableCompression    bool   `mapstructure:"enableCompression"`
	CompressionThreshold int    `mapstructure:"compressionThreshold" validate:"min=0,max=1048576"`
}

type TemporalInterceptorConfig struct {
	EnableLogging bool   `mapstructure:"enableLogging"`
	LogLevel      string `mapstructure:"logLevel"      validate:"omitempty,oneof=debug info warn error"`
}

func (c *TemporalInterceptorConfig) GetLogLevel() string {
	if c.LogLevel == "" {
		return "info"
	}

	return c.LogLevel
}

type TemporalScheduleConfig struct {
	PersistOnStop bool `mapstructure:"persistOnStop"`
}

type TemporalWorkerConfig struct {
	MaxConcurrentActivities int           `mapstructure:"maxConcurrentActivities" validate:"min=0,max=1000"`
	MaxConcurrentWorkflows  int           `mapstructure:"maxConcurrentWorkflows"  validate:"min=0,max=1000"`
	MaxActivityPollers      int           `mapstructure:"maxActivityPollers"      validate:"min=0,max=100"`
	MaxWorkflowPollers      int           `mapstructure:"maxWorkflowPollers"      validate:"min=0,max=100"`
	WorkerStopTimeout       time.Duration `mapstructure:"workerStopTimeout"`
	Queues                  []string      `mapstructure:"queues"`
}

func (c *TemporalWorkerConfig) GetMaxConcurrentActivities() int {
	if c.MaxConcurrentActivities == 0 {
		return 10
	}

	return c.MaxConcurrentActivities
}

func (c *TemporalWorkerConfig) GetMaxConcurrentWorkflows() int {
	if c.MaxConcurrentWorkflows == 0 {
		return 10
	}

	return c.MaxConcurrentWorkflows
}

func (c *TemporalWorkerConfig) GetMaxActivityPollers() int {
	if c.MaxActivityPollers == 0 {
		return 2
	}

	return c.MaxActivityPollers
}

func (c *TemporalWorkerConfig) GetMaxWorkflowPollers() int {
	if c.MaxWorkflowPollers == 0 {
		return 2
	}

	return c.MaxWorkflowPollers
}

func (c *TemporalWorkerConfig) GetWorkerStopTimeout() time.Duration {
	if c.WorkerStopTimeout == 0 {
		return 30 * time.Second
	}
	return c.WorkerStopTimeout
}

type AuditConfig struct {
	BufferFlushInterval time.Duration `mapstructure:"bufferFlushInterval"`
	BatchSize           int           `mapstructure:"batchSize"           validate:"min=1,max=5000"`
	MaxEntriesPerFlush  int           `mapstructure:"maxEntriesPerFlush"  validate:"min=100,max=50000"`
	DLQRetryInterval    time.Duration `mapstructure:"dlqRetryInterval"`
	DLQMaxRetries       int           `mapstructure:"dlqMaxRetries"       validate:"min=1,max=20"`
}

type StorageConfig struct {
	Provider           string        `mapstructure:"provider"           validate:"omitempty,oneof=minio r2"`
	Endpoint           string        `mapstructure:"endpoint"           validate:"required"`
	PublicEndpoint     string        `mapstructure:"publicEndpoint"`
	AccessKey          string        `mapstructure:"accessKey"          validate:"required"`
	SecretKey          string        `mapstructure:"secretKey"          validate:"required"`
	SessionToken       string        `mapstructure:"sessionToken"`
	Bucket             string        `mapstructure:"bucket"             validate:"required"`
	UseSSL             bool          `mapstructure:"useSSL"`
	Region             string        `mapstructure:"region"`
	AutoCreateBucket   *bool         `mapstructure:"autoCreateBucket"`
	MaxFileSize        int64         `mapstructure:"maxFileSize"`
	MaxFilesPerUpload  int           `mapstructure:"maxFilesPerUpload"  validate:"min=0,max=100"`
	PresignedURLExpiry time.Duration `mapstructure:"presignedUrlExpiry"`
	AllowedMIMETypes   []string      `mapstructure:"allowedMimeTypes"`
}

const (
	StorageProviderMinio = "minio"
	StorageProviderR2    = "r2"
)

func (c *StorageConfig) GetProvider() string {
	if c.Provider == "" {
		return StorageProviderMinio
	}
	return c.Provider
}

func (c *StorageConfig) ShouldAutoCreateBucket() bool {
	if c.AutoCreateBucket == nil {
		return true
	}
	return *c.AutoCreateBucket
}

func (c *StorageConfig) GetMaxFileSize() int64 {
	if c.MaxFileSize == 0 {
		return 52428800
	}
	return c.MaxFileSize
}

func (c *StorageConfig) GetMaxFilesPerUpload() int {
	if c.MaxFilesPerUpload == 0 {
		return 10
	}
	return c.MaxFilesPerUpload
}

func (c *StorageConfig) GetPresignedURLExpiry() time.Duration {
	if c.PresignedURLExpiry == 0 {
		return 15 * time.Minute
	}
	return c.PresignedURLExpiry
}

func (c *StorageConfig) GetAllowedMIMETypes() []string {
	if len(c.AllowedMIMETypes) == 0 {
		return []string{
			"application/pdf",
			"image/jpeg",
			"image/png",
			"image/webp",
			"image/gif",
			"application/msword",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/vnd.ms-excel",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			"text/plain",
			"text/csv",
		}
	}

	return c.AllowedMIMETypes
}

func (c *AuditConfig) GetBufferFlushInterval() time.Duration {
	if c.BufferFlushInterval == 0 {
		return time.Minute
	}

	return c.BufferFlushInterval
}

func (c *AuditConfig) GetBatchSize() int {
	if c.BatchSize == 0 {
		return 500
	}

	return c.BatchSize
}

func (c *AuditConfig) GetMaxEntriesPerFlush() int {
	if c.MaxEntriesPerFlush == 0 {
		return 5000
	}

	return c.MaxEntriesPerFlush
}

func (c *AuditConfig) GetDLQRetryInterval() time.Duration {
	if c.DLQRetryInterval == 0 {
		return 5 * time.Minute
	}

	return c.DLQRetryInterval
}

func (c *AuditConfig) GetDLQMaxRetries() int {
	if c.DLQMaxRetries == 0 {
		return 5
	}

	return c.DLQMaxRetries
}

func (c *CacheConfig) GetRedisAddr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

type ReportingConfig struct {
	PoolMaxOpenConns        int           `mapstructure:"poolMaxOpenConns"        validate:"min=0,max=64"`
	PoolMaxIdleConns        int           `mapstructure:"poolMaxIdleConns"        validate:"min=0,max=64"`
	StatementTimeout        time.Duration `mapstructure:"statementTimeout"`
	PreviewStatementTimeout time.Duration `mapstructure:"previewStatementTimeout"`
	MaxRunDuration          time.Duration `mapstructure:"maxRunDuration"`
	MaxRows                 int64         `mapstructure:"maxRows"                 validate:"min=0"`
	MaxArtifactBytes        int64         `mapstructure:"maxArtifactBytes"        validate:"min=0"`
	PreviewRowLimit         int           `mapstructure:"previewRowLimit"         validate:"min=0,max=1000"`
	PDFMaxRows              int64         `mapstructure:"pdfMaxRows"              validate:"min=0"`
	MaxToOneJoins           int           `mapstructure:"maxToOneJoins"           validate:"min=0,max=16"`
	MaxToManySubqueries     int           `mapstructure:"maxToManySubqueries"     validate:"min=0,max=8"`
	MaxDimensions           int           `mapstructure:"maxDimensions"           validate:"min=0,max=16"`
	MaxPivotColumns         int           `mapstructure:"maxPivotColumns"         validate:"min=0,max=200"`
	MaxPathDepth            int           `mapstructure:"maxPathDepth"            validate:"min=0,max=6"`
	MaxDefinitionLimit      int           `mapstructure:"maxDefinitionLimit"      validate:"min=0"`
	MaxConcurrentRunsPerOrg int           `mapstructure:"maxConcurrentRunsPerOrg" validate:"min=0,max=32"`
	MaxQueuedRunsPerOrg     int           `mapstructure:"maxQueuedRunsPerOrg"     validate:"min=0,max=256"`
	ArtifactRetention       time.Duration `mapstructure:"artifactRetention"`
	ResultCacheTTL          time.Duration `mapstructure:"resultCacheTtl"`
	ArtifactPrefix          string        `mapstructure:"artifactPrefix"`
	CSVIncludeBOM           bool          `mapstructure:"csvIncludeBom"`
	ExplainCostLimit        float64       `mapstructure:"explainCostLimit"        validate:"min=0"`
	ExplainRowLimit         float64       `mapstructure:"explainRowLimit"         validate:"min=0"`
	DeliveryLinkBaseURL     string        `mapstructure:"deliveryLinkBaseUrl"     validate:"omitempty,url"`
	EmailMaxAttachmentBytes int64         `mapstructure:"emailMaxAttachmentBytes" validate:"min=0"`
}

func (c *ReportingConfig) GetPoolMaxOpenConns() int {
	if c.PoolMaxOpenConns == 0 {
		return 4
	}
	return c.PoolMaxOpenConns
}

func (c *ReportingConfig) GetPoolMaxIdleConns() int {
	if c.PoolMaxIdleConns == 0 {
		return 2
	}
	return c.PoolMaxIdleConns
}

func (c *ReportingConfig) GetStatementTimeout() time.Duration {
	if c.StatementTimeout == 0 {
		return 5 * time.Minute
	}
	return c.StatementTimeout
}

func (c *ReportingConfig) GetPreviewStatementTimeout() time.Duration {
	if c.PreviewStatementTimeout == 0 {
		return 10 * time.Second
	}
	return c.PreviewStatementTimeout
}

func (c *ReportingConfig) GetMaxRunDuration() time.Duration {
	if c.MaxRunDuration == 0 {
		return 30 * time.Minute
	}
	return c.MaxRunDuration
}

func (c *ReportingConfig) GetMaxRows() int64 {
	if c.MaxRows == 0 {
		return 5_000_000
	}
	return c.MaxRows
}

func (c *ReportingConfig) GetMaxArtifactBytes() int64 {
	if c.MaxArtifactBytes == 0 {
		return 1 << 30
	}
	return c.MaxArtifactBytes
}

func (c *ReportingConfig) GetPreviewRowLimit() int {
	if c.PreviewRowLimit == 0 {
		return 100
	}
	return c.PreviewRowLimit
}

func (c *ReportingConfig) GetPDFMaxRows() int64 {
	if c.PDFMaxRows == 0 {
		return 5000
	}
	return c.PDFMaxRows
}

func (c *ReportingConfig) GetMaxToOneJoins() int {
	if c.MaxToOneJoins == 0 {
		return 6
	}
	return c.MaxToOneJoins
}

func (c *ReportingConfig) GetMaxToManySubqueries() int {
	if c.MaxToManySubqueries == 0 {
		return 3
	}
	return c.MaxToManySubqueries
}

func (c *ReportingConfig) GetMaxDimensions() int {
	if c.MaxDimensions == 0 {
		return 6
	}
	return c.MaxDimensions
}

func (c *ReportingConfig) GetMaxPivotColumns() int {
	if c.MaxPivotColumns == 0 {
		return 50
	}
	return c.MaxPivotColumns
}

func (c *ReportingConfig) GetMaxPathDepth() int {
	if c.MaxPathDepth == 0 {
		return 3
	}
	return c.MaxPathDepth
}

func (c *ReportingConfig) GetMaxDefinitionLimit() int {
	if c.MaxDefinitionLimit == 0 {
		return 100_000
	}
	return c.MaxDefinitionLimit
}

func (c *ReportingConfig) GetMaxConcurrentRunsPerOrg() int {
	if c.MaxConcurrentRunsPerOrg == 0 {
		return 2
	}
	return c.MaxConcurrentRunsPerOrg
}

func (c *ReportingConfig) GetMaxQueuedRunsPerOrg() int {
	if c.MaxQueuedRunsPerOrg == 0 {
		return 10
	}
	return c.MaxQueuedRunsPerOrg
}

func (c *ReportingConfig) GetArtifactRetention() time.Duration {
	if c.ArtifactRetention == 0 {
		return 7 * 24 * time.Hour
	}
	return c.ArtifactRetention
}

func (c *ReportingConfig) GetResultCacheTTL() time.Duration {
	if c.ResultCacheTTL == 0 {
		return 15 * time.Minute
	}
	return c.ResultCacheTTL
}

func (c *ReportingConfig) GetArtifactPrefix() string {
	if c.ArtifactPrefix == "" {
		return "reports"
	}
	return c.ArtifactPrefix
}

func (c *ReportingConfig) GetExplainCostLimit() float64 {
	if c.ExplainCostLimit == 0 {
		return 5_000_000
	}
	return c.ExplainCostLimit
}

func (c *ReportingConfig) GetExplainRowLimit() float64 {
	if c.ExplainRowLimit == 0 {
		return 10_000_000
	}
	return c.ExplainRowLimit
}

func (c *ReportingConfig) GetDeliveryLinkBaseURL() string {
	return strings.TrimSuffix(c.DeliveryLinkBaseURL, "/")
}

func (c *ReportingConfig) GetEmailMaxAttachmentBytes() int64 {
	if c.EmailMaxAttachmentBytes == 0 {
		return 10 << 20
	}
	return c.EmailMaxAttachmentBytes
}

// RendererConfig governs HTML-to-PDF rendering and the limits applied to
// organization-authored templates.
//
// GotenbergURL points at a sidecar that owns the browser. It is deliberately not
// reachable from the host and is never proxied: it renders untrusted markup, so
// the only thing that should be able to reach it is this process.
type RendererConfig struct {
	GotenbergURL   string        `mapstructure:"gotenbergUrl"   validate:"omitempty,url"`
	RequestTimeout time.Duration `mapstructure:"requestTimeout"`
	HealthTimeout  time.Duration `mapstructure:"healthTimeout"`
	RenderTimeout  time.Duration `mapstructure:"renderTimeout"`
	MaxConcurrent  int           `mapstructure:"maxConcurrent"  validate:"min=0,max=32"`
	MaxRetries     int           `mapstructure:"maxRetries"     validate:"min=0,max=5"`
	MaxSourceBytes int64         `mapstructure:"maxSourceBytes" validate:"min=0"`
	MaxHTMLBytes   int64         `mapstructure:"maxHtmlBytes"   validate:"min=0"`
	MaxTextBytes   int64         `mapstructure:"maxTextBytes"   validate:"min=0"`
	MaxAssetBytes  int64         `mapstructure:"maxAssetBytes"  validate:"min=0"`
	MaxAssets      int           `mapstructure:"maxAssets"      validate:"min=0,max=64"`
	MaxAssetBudget int64         `mapstructure:"maxAssetBudget" validate:"min=0"`
	MaxPDFBytes    int64         `mapstructure:"maxPdfBytes"    validate:"min=0"`
}

// Enabled reports whether a rendering backend is configured. When it is not,
// PDF generation surfaces a business error instead of failing startup — the
// same treatment web push and search get.
func (c *RendererConfig) Enabled() bool { return c.GotenbergURL != "" }

func (c *RendererConfig) GetGotenbergURL() string {
	return strings.TrimSuffix(c.GotenbergURL, "/")
}

func (c *RendererConfig) GetRequestTimeout() time.Duration {
	if c.RequestTimeout == 0 {
		return 45 * time.Second
	}
	return c.RequestTimeout
}

func (c *RendererConfig) GetHealthTimeout() time.Duration {
	if c.HealthTimeout == 0 {
		return 5 * time.Second
	}
	return c.HealthTimeout
}

func (c *RendererConfig) GetRenderTimeout() time.Duration {
	if c.RenderTimeout == 0 {
		return 2 * time.Second
	}
	return c.RenderTimeout
}

func (c *RendererConfig) GetMaxConcurrent() int {
	if c.MaxConcurrent == 0 {
		return 4
	}
	return c.MaxConcurrent
}

func (c *RendererConfig) GetMaxRetries() int {
	if c.MaxRetries == 0 {
		return 3
	}
	return c.MaxRetries
}

func (c *RendererConfig) GetMaxSourceBytes() int64 {
	if c.MaxSourceBytes == 0 {
		return 256 << 10
	}
	return c.MaxSourceBytes
}

func (c *RendererConfig) GetMaxHTMLBytes() int64 {
	if c.MaxHTMLBytes == 0 {
		return 2 << 20
	}
	return c.MaxHTMLBytes
}

func (c *RendererConfig) GetMaxTextBytes() int64 {
	if c.MaxTextBytes == 0 {
		return 64 << 10
	}
	return c.MaxTextBytes
}

func (c *RendererConfig) GetMaxAssetBytes() int64 {
	if c.MaxAssetBytes == 0 {
		return 1 << 20
	}
	return c.MaxAssetBytes
}

func (c *RendererConfig) GetMaxAssets() int {
	if c.MaxAssets == 0 {
		return 10
	}
	return c.MaxAssets
}

func (c *RendererConfig) GetMaxAssetBudget() int64 {
	if c.MaxAssetBudget == 0 {
		return 4 << 20
	}
	return c.MaxAssetBudget
}

func (c *RendererConfig) GetMaxPDFBytes() int64 {
	if c.MaxPDFBytes == 0 {
		return 20 << 20
	}
	return c.MaxPDFBytes
}

type AppConfig struct {
	Name               string `mapstructure:"name"               validate:"required,min=1,max=100"`
	Env                string `mapstructure:"env"                validate:"required,oneof=development staging production test"`
	Debug              bool   `mapstructure:"debug"`
	Version            string `mapstructure:"version"            validate:"required"`
	ProblemTypeBaseURI string `mapstructure:"problemTypeBaseUri"`
	WebBaseURL         string `mapstructure:"webBaseUrl"         validate:"omitempty,url"`
}

func (c *AppConfig) GetWebBaseURL() string {
	return strings.TrimSuffix(c.WebBaseURL, "/")
}

func (c *AppConfig) IsDevelopment() bool { return c.Env == EnvDevelopment }

func (c *AppConfig) IsProduction() bool { return c.Env == EnvProduction }

func (c *AppConfig) IsStaging() bool { return c.Env == EnvStaging }

func (c *AppConfig) IsTest() bool { return c.Env == EnvTest }

func (c *AppConfig) GetProblemTypeBaseURI() string {
	if c.ProblemTypeBaseURI != "" {
		return strings.TrimSuffix(c.ProblemTypeBaseURI, "/") + "/"
	}

	return "https://api.trenova.app/problems/"
}

type UpdateConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	CheckInterval   time.Duration `mapstructure:"checkInterval"`
	GitHubOwner     string        `mapstructure:"githubOwner"`
	GitHubRepo      string        `mapstructure:"githubRepo"`
	AllowPrerelease bool          `mapstructure:"allowPrerelease"`
	ProxyURL        string        `mapstructure:"proxyUrl"`
	OfflineMode     bool          `mapstructure:"offlineMode"`
}

func (c *UpdateConfig) GetCheckInterval() time.Duration {
	if c.CheckInterval == 0 {
		return 1 * time.Hour
	}
	return c.CheckInterval
}

func (c *UpdateConfig) GetGitHubOwner() string {
	if c.GitHubOwner == "" {
		return "emoss08"
	}
	return c.GitHubOwner
}

func (c *UpdateConfig) GetGitHubRepo() string {
	if c.GitHubRepo == "" {
		return "trenova"
	}
	return c.GitHubRepo
}

type PlatformConfig struct {
	Mode         PlatformMode               `mapstructure:"mode"         validate:"omitempty,oneof=community self_hosted development cloud enterprise"`
	InstanceID   string                     `mapstructure:"instanceId"`
	ControlPlane PlatformControlPlaneConfig `mapstructure:"controlPlane"`
}

func (c *PlatformConfig) IsCloudBacked() bool {
	return c.ControlPlane.Enabled
}

func (c *PlatformConfig) GetMode() PlatformMode {
	if c.Mode == "" {
		return PlatformModeSelfHosted
	}

	if c.Mode == PlatformModeCommunity || c.Mode == PlatformModeEnterprise {
		return PlatformModeSelfHosted
	}

	return c.Mode
}

func (c *PlatformConfig) IsDevelopmentDeployment() bool {
	return c.GetMode() == PlatformModeDevelopment
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

type SystemConfig struct {
	SystemUserPassword string             `mapstructure:"systemUserPassword" validate:"required,min=1,max=100"`
	NetworkPulse       NetworkPulseConfig `mapstructure:"networkPulse"`
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

type Config struct {
	App                 AppConfig                 `mapstructure:"app"                  validate:"required"`
	Database            DatabaseConfig            `mapstructure:"database"             validate:"required"`
	Monitoring          MonitoringConfig          `mapstructure:"monitoring"           validate:"required"`
	Cache               CacheConfig               `mapstructure:"cache"                validate:"required"`
	Server              ServerConfig              `mapstructure:"server"               validate:"required"`
	Security            SecurityConfig            `mapstructure:"security"             validate:"required"`
	Logging             LoggingConfig             `mapstructure:"logging"              validate:"required"`
	Temporal            TemporalConfig            `mapstructure:"temporal"             validate:"required"`
	Storage             StorageConfig             `mapstructure:"storage"              validate:"required"`
	System              SystemConfig              `mapstructure:"system"               validate:"required"`
	Foony               FoonyConfig               `mapstructure:"foony"                validate:"required"`
	Search              SearchConfig              `mapstructure:"search"`
	AI                  AIConfig                  `mapstructure:"ai"`
	Audit               AuditConfig               `mapstructure:"audit"`
	Update              UpdateConfig              `mapstructure:"update"`
	Twilio              TwilioConfig              `mapstructure:"twilio"`
	Platform            PlatformConfig            `mapstructure:"platform"`
	Reporting           ReportingConfig           `mapstructure:"reporting"`
	Renderer            RendererConfig            `mapstructure:"renderer"`
	Portal              PortalConfig              `mapstructure:"portal"`
	Push                PushConfig                `mapstructure:"push"`
	Tendering           TenderingConfig           `mapstructure:"tendering"`
	CarrierIntelligence CarrierIntelligenceConfig `mapstructure:"carrierIntelligence"`
}

type CarrierIntelligenceConfig struct {
	AllowedHosts    []string      `mapstructure:"allowedHosts"`
	SandboxAllowed  *bool         `mapstructure:"sandboxAllowed"`
	InteractiveWait time.Duration `mapstructure:"interactiveWait"`
	ReplicaHint     int           `mapstructure:"replicaHint"     validate:"omitempty,min=1,max=1000"`
}

var defaultCarrierIntelligenceHosts = []string{"api.carrierok.com", "mobile.fmcsa.dot.gov"}

func (c *CarrierIntelligenceConfig) GetAllowedHosts() []string {
	if len(c.AllowedHosts) == 0 {
		return defaultCarrierIntelligenceHosts
	}
	return c.AllowedHosts
}

func (c *CarrierIntelligenceConfig) IsSandboxAllowed(app *AppConfig) bool {
	if c.SandboxAllowed != nil {
		return *c.SandboxAllowed
	}
	return !app.IsProduction()
}

func (c *CarrierIntelligenceConfig) GetInteractiveWait() time.Duration {
	if c.InteractiveWait <= 0 {
		return 8 * time.Second
	}
	return c.InteractiveWait
}

func (c *CarrierIntelligenceConfig) GetReplicaHint() int {
	if c.ReplicaHint <= 0 {
		return 1
	}
	return c.ReplicaHint
}

type PortalConfig struct {
	BaseURL string `mapstructure:"baseUrl" validate:"omitempty,url"`
}

type TenderingConfig struct {
	PublicBaseURL string `mapstructure:"publicBaseUrl" validate:"omitempty,url"`
}

func (c *TenderingConfig) GetPublicBaseURL() string {
	return strings.TrimSuffix(c.PublicBaseURL, "/")
}

type PushConfig struct {
	VAPIDPublicKey  string `mapstructure:"vapidPublicKey"`
	VAPIDPrivateKey string `mapstructure:"vapidPrivateKey"`
	Subject         string `mapstructure:"subject"         validate:"omitempty"`
}

func (c *PushConfig) Enabled() bool {
	return c.VAPIDPublicKey != "" && c.VAPIDPrivateKey != ""
}

func (c *PortalConfig) GetBaseURL() string {
	return strings.TrimSuffix(c.BaseURL, "/")
}

func (c *Config) GetCacheConfig() *CacheConfig { return &c.Cache }

func (c *Config) GetSearchConfig() *SearchConfig { return &c.Search }

func (c *Config) GetAIConfig() *AIConfig { return &c.AI }

func (c *Config) GetTemporalConfig() *TemporalConfig { return &c.Temporal }

func (c *Config) GetMetricsConfig() *MetricsConfig { return &c.Monitoring.Metrics }

func (c *Config) GetGraphQLObservConfig() *GraphQLObservConfig { return &c.Monitoring.GraphQL }

func (c *Config) GetStorageConfig() *StorageConfig { return &c.Storage }

func (c *Config) GetTwilioConfig() *TwilioConfig { return &c.Twilio }

func (c *Config) GetFoonyConfig() *FoonyConfig { return &c.Foony }

func (c *Config) GetSystemConfig() *SystemConfig { return &c.System }

func (c *Config) GetPlatformConfig() *PlatformConfig { return &c.Platform }

func (c *Config) GetReportingConfig() *ReportingConfig { return &c.Reporting }

func (c *Config) GetRendererConfig() *RendererConfig { return &c.Renderer }

func (c *Config) GetDSN(password string) string {
	if c.Database.GetDialect().IsSQLite() {
		return c.getSQLiteDSN()
	}

	escapedPassword := url.QueryEscape(password)

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=%s",
		c.Database.User,
		escapedPassword,
		net.JoinHostPort(c.Database.Host, strconv.Itoa(c.Database.Port)),
		c.Database.Name,
		c.Database.SSLMode,
	)

	dsn += fmt.Sprintf("&application_name=%s", url.QueryEscape(c.App.Name))

	dsn += "&dial_timeout=10s"

	return dsn
}

func (c *Config) getSQLiteDSN() string {
	sqlite := &c.Database.SQLite

	pragmas := []string{
		fmt.Sprintf("_pragma=journal_mode(%s)", sqlite.GetJournalMode()),
		fmt.Sprintf("_pragma=synchronous(%s)", sqlite.GetSynchronous()),
		fmt.Sprintf("_pragma=busy_timeout(%d)", sqlite.GetBusyTimeout().Milliseconds()),
		fmt.Sprintf("_pragma=foreign_keys(%d)", boolToPragma(sqlite.GetForeignKeys())),
		fmt.Sprintf("_pragma=cache_size(-%d)", sqlite.GetCacheSizeKB()),
		"_time_format=sqlite",
		"_txlock=immediate",
	}

	return fmt.Sprintf("file:%s?%s", sqlite.GetPath(), strings.Join(pragmas, "&"))
}

func boolToPragma(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (c *Config) GetDSNMasked() string {
	if c.Database.GetDialect().IsSQLite() {
		return fmt.Sprintf("sqlite://%s", c.Database.SQLite.GetPath())
	}

	return fmt.Sprintf(
		"postgres://%s:****@%s/%s?sslmode=%s",
		c.Database.User,
		net.JoinHostPort(c.Database.Host, strconv.Itoa(c.Database.Port)),
		c.Database.Name,
		c.Database.SSLMode,
	)
}

func (c *Config) CorsEnabled() bool { return c.Server.CORS.Enabled }
