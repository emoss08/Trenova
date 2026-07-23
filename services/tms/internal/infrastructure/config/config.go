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
	Host              string        `mapstructure:"host"              validate:"required,hostname|ip"`
	Port              int           `mapstructure:"port"              validate:"required,min=1,max=65535"`
	Mode              string        `mapstructure:"mode"              validate:"required,oneof=debug release test"`
	ReadTimeout       time.Duration `mapstructure:"readTimeout"`
	ReadHeaderTimeout time.Duration `mapstructure:"readHeaderTimeout"`
	WriteTimeout      time.Duration `mapstructure:"writeTimeout"`
	IdleTimeout       time.Duration `mapstructure:"idleTimeout"`
	ShutdownTimeout   time.Duration `mapstructure:"shutdownTimeout"`
	RequestTimeout    time.Duration `mapstructure:"requestTimeout"`
	CORS              CORSConfig    `mapstructure:"cors,omitempty"`
}

type MonitoringConfig struct {
	Metrics MetricsConfig `mapstructure:"metrics,omitempty"`
	Tracing TracingConfig `mapstructure:"tracing,omitempty"`
	Health  HealthConfig  `mapstructure:"health"`
	Pprof   PprofConfig   `mapstructure:"pprof"`
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
	Session    SessionConfig    `mapstructure:"session"    validate:"required"`
	APIToken   APITokenConfig   `mapstructure:"apiToken"`
	RateLimit  RateLimitConfig  `mapstructure:"rateLimit"`
	CSRF       CSRFConfig       `mapstructure:"csrf"`
	Encryption EncryptionConfig `mapstructure:"encryption"`
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
	RetryAttempts   int           `mapstructure:"retryAttempts" validate:"omitempty,min=1,max=10"`
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

func (c *RateLimitConfig) GetRequestsPerMinute() int {
	if c.RequestsPerMinute == 0 {
		return 60
	}
	return c.RequestsPerMinute
}

func (c *RateLimitConfig) GetBurstSize() int {
	if c.BurstSize == 0 {
		return 10
	}
	return c.BurstSize
}

func (c *RateLimitConfig) GetCleanupInterval() time.Duration {
	if c.CleanupInterval == 0 {
		return time.Minute
	}
	return c.CleanupInterval
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
	Host             string        `mapstructure:"host"            validate:"required"`
	Port             int           `mapstructure:"port"            validate:"required,min=1,max=65535"`
	Name             string        `mapstructure:"name"            validate:"required,min=1,max=63"`
	User             string        `mapstructure:"user"            validate:"required,min=1,max=63"`
	Password         string        `mapstructure:"password"        validate:"required"`
	SSLMode          string        `mapstructure:"sslMode"         validate:"required,oneof=disable require verify-ca verify-full"`
	MaxIdleConns     int           `mapstructure:"maxIdleConns"    validate:"min=1,max=1000"`
	MaxOpenConns     int           `mapstructure:"maxOpenConns"    validate:"min=1,max=1000"`
	Verbose          bool          `mapstructure:"verbose"`
	ConnMaxLifetime  time.Duration `mapstructure:"connMaxLifetime"`
	ConnMaxIdleTime  time.Duration `mapstructure:"connMaxIdleTime"`
	StatementTimeout time.Duration `mapstructure:"statementTimeout"`
	LockTimeout      time.Duration `mapstructure:"lockTimeout"`
	IdleTxTimeout    time.Duration `mapstructure:"idleInTransactionSessionTimeout"`
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

type AppConfig struct {
	Name               string `mapstructure:"name"               validate:"required,min=1,max=100"`
	Env                string `mapstructure:"env"                validate:"required,oneof=development staging production test"`
	Debug              bool   `mapstructure:"debug"`
	Version            string `mapstructure:"version"            validate:"required"`
	ProblemTypeBaseURI string `mapstructure:"problemTypeBaseUri"`
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

type PlatformControlPlaneConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	Endpoint          string        `mapstructure:"endpoint"          validate:"omitempty,url,no_trailing_slash"`
	APIKey            string        `mapstructure:"apiKey"`
	Timeout           time.Duration `mapstructure:"timeout"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeatInterval"`
	FailOpenOnError   bool          `mapstructure:"failOpenOnError"`
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
	Foony                FoonyConfig                `mapstructure:"foony"                validate:"required"`
	Search               SearchConfig               `mapstructure:"search"`
	DocumentIntelligence DocumentIntelligenceConfig `mapstructure:"documentIntelligence"`
	Audit                AuditConfig                `mapstructure:"audit"`
	Update               UpdateConfig               `mapstructure:"update"`
	Twilio               TwilioConfig               `mapstructure:"twilio"`
	Platform             PlatformConfig             `mapstructure:"platform"`
	Reporting            ReportingConfig            `mapstructure:"reporting"`
	Portal               PortalConfig               `mapstructure:"portal"`
	Push                 PushConfig                 `mapstructure:"push"`
}

type PortalConfig struct {
	BaseURL string `mapstructure:"baseUrl" validate:"omitempty,url"`
}

type PushConfig struct {
	VAPIDPublicKey  string `mapstructure:"vapidPublicKey"`
	VAPIDPrivateKey string `mapstructure:"vapidPrivateKey"`
	Subject         string `mapstructure:"subject" validate:"omitempty"`
}

func (c *PushConfig) Enabled() bool {
	return c.VAPIDPublicKey != "" && c.VAPIDPrivateKey != ""
}

func (c *PortalConfig) GetBaseURL() string {
	return strings.TrimSuffix(c.BaseURL, "/")
}

func (c *Config) GetCacheConfig() *CacheConfig { return &c.Cache }

func (c *Config) GetSearchConfig() *SearchConfig { return &c.Search }

func (c *Config) GetDocumentIntelligenceConfig() *DocumentIntelligenceConfig {
	return &c.DocumentIntelligence
}

func (c *Config) GetTemporalConfig() *TemporalConfig { return &c.Temporal }

func (c *Config) GetMetricsConfig() *MetricsConfig { return &c.Monitoring.Metrics }

func (c *Config) GetStorageConfig() *StorageConfig { return &c.Storage }

func (c *Config) GetTwilioConfig() *TwilioConfig { return &c.Twilio }

func (c *Config) GetFoonyConfig() *FoonyConfig { return &c.Foony }

func (c *Config) GetSystemConfig() *SystemConfig { return &c.System }

func (c *Config) GetPlatformConfig() *PlatformConfig { return &c.Platform }

func (c *Config) GetReportingConfig() *ReportingConfig { return &c.Reporting }

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

	dsn += "&dial_timeout=10s"

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

func (c *Config) CorsEnabled() bool { return c.Server.CORS.Enabled }
