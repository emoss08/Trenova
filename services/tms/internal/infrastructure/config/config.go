package config

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ServerConfig struct {
	Host            string        `mapstructure:"host"            validate:"required,hostname|ip"`
	Port            int           `mapstructure:"port"            validate:"required,min=1,max=65535"`
	Mode            string        `mapstructure:"mode"            validate:"required,oneof=debug release test"`
	ReadTimeout     time.Duration `mapstructure:"readTimeout"`
	WriteTimeout    time.Duration `mapstructure:"writeTimeout"`
	IdleTimeout     time.Duration `mapstructure:"idleTimeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdownTimeout"`
	CORS            CORSConfig    `mapstructure:"cors,omitempty"`
}

type MonitoringConfig struct {
	Metrics MetricsConfig `mapstructure:"metrics,omitempty"`
	Tracing TracingConfig `mapstructure:"tracing,omitempty"`
	Health  HealthConfig  `mapstructure:"health"`
}

type TwilioConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	AccountSID string `mapstructure:"accountSID" validate:"required_if=Enabled true"`
	AuthToken  string `mapstructure:"authToken"  validate:"required_if=Enabled true"`
	FromNumber string `mapstructure:"fromNumber" validate:"required_if=Enabled true"`
}

type AblyConfig struct {
	APIKey string `mapstructure:"apiKey" validate:"required"`
}

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"    validate:"required_if=Enabled true,min=1,max=65535"`
	Path    string `mapstructure:"path"    validate:"required_if=Enabled true"`
}

type TracingConfig struct {
	Enabled      bool    `mapstructure:"enabled"`
	Provider     string  `mapstructure:"provider"     validate:"required_if=Enabled true,oneof=jaeger zipkin otlp otlp-grpc stdout"`
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
	Session    SessionConfig    `mapstructure:"session"    validate:"required"`
	APIToken   APITokenConfig   `mapstructure:"apiToken"`
	RateLimit  RateLimitConfig  `mapstructure:"rateLimit"`
	CSRF       CSRFConfig       `mapstructure:"csrf"`
	Encryption EncryptionConfig `mapstructure:"encryption"`
}

type EncryptionConfig struct {
	Key string `mapstructure:"key" validate:"required,min=32"`
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

type RateLimitConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	RequestsPerMinute int           `mapstructure:"requestsPerMinute" validate:"min=1,max=10000"`
	BurstSize         int           `mapstructure:"burstSize"         validate:"min=1,max=1000"`
	CleanupInterval   time.Duration `mapstructure:"cleanupInterval"`
}

type CSRFConfig struct {
	TokenName  string `mapstructure:"tokenName"  validate:"required"`
	HeaderName string `mapstructure:"headerName" validate:"required"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"            validate:"required"`
	Port            int           `mapstructure:"port"            validate:"required,min=1,max=65535"`
	Name            string        `mapstructure:"name"            validate:"required,min=1,max=63"`
	User            string        `mapstructure:"user"            validate:"required,min=1,max=63"`
	Password        string        `mapstructure:"password"        validate:"required"`
	SSLMode         string        `mapstructure:"sslMode"         validate:"required,oneof=disable require verify-ca verify-full"`
	MaxIdleConns    int           `mapstructure:"maxIdleConns"    validate:"min=1,max=1000"`
	MaxOpenConns    int           `mapstructure:"maxOpenConns"    validate:"min=1,max=1000"`
	Verbose         bool          `mapstructure:"verbose"`
	ConnMaxLifetime time.Duration `mapstructure:"connMaxLifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"connMaxIdleTime"`
}

type CORSConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	AllowedOrigins []string `mapstructure:"allowedOrigins" validate:"required_if=Enabled true,dive,url,no_trailing_slash|eq=*"`
	AllowedMethods []string `mapstructure:"allowedMethods" validate:"required_if=Enabled true"`
	AllowedHeaders []string `mapstructure:"allowedHeaders" validate:"required_if=Enabled true"`
	ExposeHeaders  []string `mapstructure:"exposeHeaders"`
	Credentials    bool     `mapstructure:"credentials"`
	MaxAge         int      `mapstructure:"maxAge"         validate:"min=0,max=86400"`
}

type CacheConfig struct {
	Host            string        `mapstructure:"host"            validate:"required"`
	Port            int           `mapstructure:"port"            validate:"required,min=1,max=65535"`
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

type DocumentIntelligenceConfig struct {
	Enabled                 bool          `mapstructure:"enabled"`
	OCRCommand              string        `mapstructure:"ocrCommand"`
	OCRLanguage             string        `mapstructure:"ocrLanguage"`
	OCRTimeout              time.Duration `mapstructure:"ocrTimeout"`
	EnableAI                bool          `mapstructure:"enableAI"`
	AITimeout               time.Duration `mapstructure:"aiTimeout"`
	AIMaxInputChars         int           `mapstructure:"aiMaxInputChars"         validate:"omitempty,min=1000,max=500000"`
	AIExtractionMaxTokens   int           `mapstructure:"aiExtractionMaxTokens"   validate:"omitempty,min=256,max=32768"`
	AIClassificationModel   string        `mapstructure:"aiClassificationModel"`
	AIExtractionModel       string        `mapstructure:"aiExtractionModel"`
	AIMaxRetries            int           `mapstructure:"aiMaxRetries"            validate:"omitempty,min=0,max=10"`
	EnableOCRPreprocessing  bool          `mapstructure:"enableOCRPreprocessing"`
	OCRPreprocessingMode    string        `mapstructure:"ocrPreprocessingMode"`
	OCRMaxImageDimension    int           `mapstructure:"ocrMaxImageDimension"    validate:"omitempty,min=512,max=12000"`
	MaxOCRPages             int           `mapstructure:"maxOCRPages"             validate:"omitempty,min=1,max=500"`
	MaxExtractedChars       int           `mapstructure:"maxExtractedChars"       validate:"omitempty,min=1000,max=1000000"`
	ReconcileBatchSize      int           `mapstructure:"reconcileBatchSize"      validate:"omitempty,min=1,max=1000"`
	MaxConcurrentActivities int           `mapstructure:"maxConcurrentActivities" validate:"omitempty,min=1,max=64"`
}

func (c *DocumentIntelligenceConfig) GetOCRCommand() string {
	if c.OCRCommand == "" {
		return "tesseract"
	}
	return c.OCRCommand
}

func (c *DocumentIntelligenceConfig) GetOCRLanguage() string {
	if c.OCRLanguage == "" {
		return "eng"
	}
	return c.OCRLanguage
}

func (c *DocumentIntelligenceConfig) GetOCRTimeout() time.Duration {
	if c.OCRTimeout <= 0 {
		return 45 * time.Second
	}
	return c.OCRTimeout
}

func (c *DocumentIntelligenceConfig) AIEnabled() bool {
	return c.EnableAI
}

func (c *DocumentIntelligenceConfig) GetAITimeout() time.Duration {
	if c.AITimeout <= 0 {
		return 20 * time.Second
	}
	return c.AITimeout
}

func (c *DocumentIntelligenceConfig) GetAIMaxInputChars() int {
	if c.AIMaxInputChars <= 0 {
		return 24000
	}
	return c.AIMaxInputChars
}

func (c *DocumentIntelligenceConfig) GetAIExtractionMaxTokens() int {
	if c.AIExtractionMaxTokens <= 0 {
		return 5000
	}
	return c.AIExtractionMaxTokens
}

func (c *DocumentIntelligenceConfig) GetAIClassificationModel() string {
	if c.AIClassificationModel == "" {
		return "gpt-5-nano-2025-08-07"
	}
	return c.AIClassificationModel
}

func (c *DocumentIntelligenceConfig) GetAIExtractionModel() string {
	if c.AIExtractionModel == "" {
		return "gpt-5-mini-2025-08-07"
	}
	return c.AIExtractionModel
}

func (c *DocumentIntelligenceConfig) GetAIMaxRetries() int {
	if c.AIMaxRetries <= 0 {
		return 2
	}
	return c.AIMaxRetries
}

func (c *DocumentIntelligenceConfig) OCRPreprocessingEnabled() bool {
	return c.EnableOCRPreprocessing
}

func (c *DocumentIntelligenceConfig) GetOCRPreprocessingMode() string {
	if c.OCRPreprocessingMode == "" {
		return "standard"
	}
	return c.OCRPreprocessingMode
}

func (c *DocumentIntelligenceConfig) GetOCRMaxImageDimension() int {
	if c.OCRMaxImageDimension <= 0 {
		return 2400
	}
	return c.OCRMaxImageDimension
}

func (c *DocumentIntelligenceConfig) GetMaxOCRPages() int {
	if c.MaxOCRPages <= 0 {
		return 25
	}
	return c.MaxOCRPages
}

func (c *DocumentIntelligenceConfig) GetMaxConcurrentActivities() int {
	if c.MaxConcurrentActivities <= 0 {
		return 2
	}
	return c.MaxConcurrentActivities
}

func (c *DocumentIntelligenceConfig) GetMaxExtractedChars() int {
	if c.MaxExtractedChars <= 0 {
		return 200000
	}
	return c.MaxExtractedChars
}

func (c *DocumentIntelligenceConfig) GetReconcileBatchSize() int {
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
	HostPort     string                    `mapstructure:"hostPort"     validate:"required,hostname_port"`
	Namespace    string                    `mapstructure:"namespace"`
	Identity     string                    `mapstructure:"identity"`
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
	Endpoint           string        `mapstructure:"endpoint"           validate:"required"`
	PublicEndpoint     string        `mapstructure:"publicEndpoint"`
	AccessKey          string        `mapstructure:"accessKey"          validate:"required"`
	SecretKey          string        `mapstructure:"secretKey"          validate:"required"`
	SessionToken       string        `mapstructure:"sessionToken"`
	Bucket             string        `mapstructure:"bucket"             validate:"required"`
	UseSSL             bool          `mapstructure:"useSSL"`
	Region             string        `mapstructure:"region"`
	MaxFileSize        int64         `mapstructure:"maxFileSize"`
	PresignedURLExpiry time.Duration `mapstructure:"presignedUrlExpiry"`
	AllowedMIMETypes   []string      `mapstructure:"allowedMimeTypes"`
}

func (c *StorageConfig) GetMaxFileSize() int64 {
	if c.MaxFileSize == 0 {
		return 52428800
	}
	return c.MaxFileSize
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
			"text/html",
		}
	}
	return c.AllowedMIMETypes
}

func (c *AuditConfig) GetBufferFlushInterval() time.Duration {
	if c.BufferFlushInterval == 0 {
		return 10 * time.Second
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

type AppConfig struct {
	Name               string `mapstructure:"name"               validate:"required,min=1,max=100"`
	Env                string `mapstructure:"env"                validate:"required,oneof=development staging production test"`
	Debug              bool   `mapstructure:"debug"`
	Version            string `mapstructure:"version"            validate:"required"`
	ProblemTypeBaseURI string `mapstructure:"problemTypeBaseUri"`
}

func (c *AppConfig) IsDevelopment() bool {
	return c.Env == EnvDevelopment
}

func (c *AppConfig) IsProduction() bool {
	return c.Env == EnvProduction
}

func (c *AppConfig) IsStaging() bool {
	return c.Env == EnvStaging
}

func (c *AppConfig) IsTest() bool {
	return c.Env == EnvTest
}

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

type SystemConfig struct {
	SystemUserPassword string `mapstructure:"systemUserPassword" validate:"required,min=1,max=100"`
}

type Config struct {
	App                  AppConfig                  `mapstructure:"app"                  validate:"required"`
	Database             DatabaseConfig             `mapstructure:"database"             validate:"required"`
	Monitoring           MonitoringConfig           `mapstructure:"monitoring"           validate:"required"`
	Cache                CacheConfig                `mapstructure:"cache"                validate:"required"`
	Server               ServerConfig               `mapstructure:"server"               validate:"required"`
	Security             SecurityConfig             `mapstructure:"security"             validate:"required"`
	Logging              LoggingConfig              `mapstructure:"logging"              validate:"required"`
	Temporal             TemporalConfig             `mapstructure:"temporal"             validate:"required"`
	Storage              StorageConfig              `mapstructure:"storage"              validate:"required"`
	System               SystemConfig               `mapstructure:"system"               validate:"required"`
	Ably                 AblyConfig                 `mapstructure:"ably"                 validate:"required"`
	Search               SearchConfig               `mapstructure:"search"`
	DocumentIntelligence DocumentIntelligenceConfig `mapstructure:"documentIntelligence"`
	Audit                AuditConfig                `mapstructure:"audit"`
	Update               UpdateConfig               `mapstructure:"update"`
	Twilio               TwilioConfig               `mapstructure:"twilio"`
}

func (c *Config) GetCacheConfig() *CacheConfig {
	return &c.Cache
}

func (c *Config) GetSearchConfig() *SearchConfig {
	return &c.Search
}

func (c *Config) GetDocumentIntelligenceConfig() *DocumentIntelligenceConfig {
	return &c.DocumentIntelligence
}

func (c *Config) GetTemporalConfig() *TemporalConfig {
	return &c.Temporal
}

func (c *Config) GetMetricsConfig() *MetricsConfig {
	return &c.Monitoring.Metrics
}

func (c *Config) GetStorageConfig() *StorageConfig {
	return &c.Storage
}

func (c *Config) GetTwilioConfig() *TwilioConfig {
	return &c.Twilio
}

func (c *Config) GetAblyConfig() *AblyConfig {
	return &c.Ably
}

func (c *Config) GetSystemConfig() *SystemConfig {
	return &c.System
}

func (c *Config) GetDSN(password string) string {
	escapedPassword := url.QueryEscape(password)

	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		c.Database.User,
		escapedPassword,
		net.JoinHostPort(c.Database.Host, strconv.Itoa(c.Database.Port)),
		c.Database.Name,
		c.Database.SSLMode,
	)

	dsn += fmt.Sprintf("&application_name=%s", url.QueryEscape(c.App.Name))

	dsn += "&connect_timeout=10&statement_timeout=30000&idle_in_transaction_session_timeout=30000"

	return dsn
}

func (c *Config) GetDSNMasked() string {
	return fmt.Sprintf("postgres://%s:****@%s/%s?sslmode=%s",
		c.Database.User,
		net.JoinHostPort(c.Database.Host, strconv.Itoa(c.Database.Port)),
		c.Database.Name,
		c.Database.SSLMode,
	)
}

func (c *Config) CorsEnabled() bool {
	return c.Server.CORS.Enabled
}
