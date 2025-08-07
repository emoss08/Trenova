/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package config

import (
	"time"
)

// Config is the configuration for the app.
type Config struct {
	// App is the app configuration.
	App AppConfig `mapstructure:"app"`

	// Log is the log configuration.
	Log LogConfig `mapstructure:"logging"`

	// Server is the server configuration.
	Server ServerConfig `mapstructure:"server"`

	// DB is the database configuration.
	DB DatabaseConfig `mapstructure:"db"`

	// Redis is the redis configuration.
	Redis RedisConfig `mapstructure:"redis"`

	// Auth is the auth configuration.
	Auth AuthConfig `mapstructure:"auth"`

	// Audit is the audit configuration.
	Audit AuditConfig `mapstructure:"audit"`

	// Minio is the minio configuration.
	// TODO(Wolfred): Rename this to FileStorage
	Minio MinioConfig `mapstructure:"minio"`

	// Cors is the cors configuration.
	Cors CorsConfig `mapstructure:"cors"`

	// Static is the static configuration.
	Static StaticConfig `mapstructure:"static"`

	// Backup is the backup configuration.
	Backup BackupConfig `mapstructure:"backup"`

	// Kafka is the kafka configuration for CDC.
	Kafka KafkaConfig `mapstructure:"kafka"`

	// Streaming is the streaming configuration.
	Streaming StreamingConfig `mapstructure:"streaming"`

	// AI is the AI configuration.
	AI AIConfig `mapstructure:"ai"`

	// CronScheduler is the cron scheduler configuration.
	CronScheduler CronSchedulerConfig `mapstructure:"cronScheduler"`

	// Telemetry is the telemetry configuration.
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
}

type LogConfig struct {
	// LogLevel is the app log level.
	// trace, debug, info, warn, error, fatal, panic
	Level string `mapstructure:"level"`

	// SamplingPeriod is the sampling period.
	// This is the period at which the log is sampled.
	// Defaults to 10 seconds.
	SamplingPeriod time.Duration `mapstructure:"samplingPeriod"`

	// SamplingInterval is the sampling interval.
	// This is the interval at which the log is sampled.
	// Defaults to 1000.
	SamplingInterval uint32 `mapstructure:"samplingInterval"`

	// FileConfig is the file configuration.
	FileConfig FileConfig `mapstructure:"file"`
}

type FileConfig struct {
	// Enabled is the enabled flag.
	Enabled bool `mapstructure:"enabled"`

	// Path is the path to the log file.
	Path string `mapstructure:"path"`

	// FileName is the name of the log file.
	FileName string `mapstructure:"fileName"`

	// MaxSize is the max size of the log file.
	MaxSize int `mapstructure:"maxSize"`

	// MaxBackups is the max backups.
	MaxBackups int `mapstructure:"maxBackups"`

	// MaxAge is the max age of the log file.
	MaxAge int `mapstructure:"maxAge"`

	// Compress is the compress flag.
	Compress bool `mapstructure:"compress"`
}

// StaticConfig is the configuration for the static files.
type StaticConfig struct {
	// Path is the path to the static files.
	Path string `mapstructure:"path"`

	// Browse is the browse flag.
	Browse bool `mapstructure:"browse"`

	// Root is the root directory.
	Root string `mapstructure:"root"`
}

// AuditConfig is the configuration for the audit service.
type AuditConfig struct {
	// BufferSize is the buffer size.
	// This is the size of the buffer.
	// Defaults to 1000.
	// You may need to increase this if you're using a high-traffic service.
	BufferSize int `mapstructure:"bufferSize"`

	// FlushInterval is the flush interval.
	// This is the interval at which the buffer is flushed.
	// Defaults to 10 seconds.
	// You may need to increase this if you're using a high-traffic service.
	FlushInterval int `mapstructure:"flushInterval"`

	// MaxRetries is the max retries.
	// This is the maximum number of retries for the audit service.
	// Defaults to 3.
	MaxRetries int `mapstructure:"maxRetries"`

	// CompressionEnabled enables compression of large audit entries.
	// When enabled, large state data will be compressed to reduce storage requirements.
	// Defaults to false.
	CompressionEnabled bool `mapstructure:"compressionEnabled"`

	// CompressionLevel sets the compression level (1-9).
	// 1 is fastest but least compression, 9 is slowest but best compression.
	// Defaults to 6 (medium compression).
	CompressionLevel int `mapstructure:"compressionLevel"`

	// CompressionThreshold is the size in KB before compression is applied.
	// State data smaller than this threshold will not be compressed.
	// Defaults to 10 KB.
	CompressionThreshold int `mapstructure:"compressionThreshold"`

	// BatchSize is the batch size for processing audit entries.
	// This is the number of entries processed in a single batch.
	// Defaults to 50.
	BatchSize int `mapstructure:"batchSize"`

	// Workers is the number of worker goroutines for processing audit entries.
	// More workers can process entries faster but use more resources.
	// Defaults to 2.
	Workers int `mapstructure:"workers"`
}

// AppConfig is the configuration for the app.
type AppConfig struct {
	// Name is the app name.
	Name string `mapstructure:"name"`

	// Environment is the app environment.
	// development, staging, production, testing
	Environment string `mapstructure:"environment"`

	// Version is the app version.
	Version string `mapstructure:"version"`
}

// ServerConfig is the configuration for the server.
// To understand the configuration options, please refer to the Fiber documentation:
// https://docs.gofiber.io/api/fiber#config
type ServerConfig struct {
	// SecretKey is the server secret key.
	// This is used to sign the session cookie.
	SecretKey string `mapstructure:"secretKey"`

	// ListenAddress is the server listen address.
	ListenAddress string `mapstructure:"listenAddress"`

	// Immutable is the immutable mode.
	// When enabled, all values returned by context methods are immutable.
	// By default, they are valid until you return from the handler.
	Immutable bool `mapstructure:"immutable"`

	// ReadBufferSize is the read buffer size.
	// per-connection buffer size for requests' reading. This also limits
	// the maximum header size. Increase this buffer if your clients send
	// multi-KB RequestURIs and/or multi-KB headers
	// (for example, BIG cookies).
	ReadBufferSize int `mapstructure:"readBufferSize"`

	// WriteBufferSize is the write buffer size.
	// Per-connection buffer size for responses' writing.
	WriteBufferSize int `mapstructure:"writeBufferSize"`

	// PassLocalsToViews enables passing of the locals set on a fiber.Ctx
	// to the template engine. See our Template Middleware for supported
	// engines.
	PassLocalsToViews bool `mapstructure:"passLocalsToViews"`

	// DisableStartupMessage disables the startup message.
	DisableStartupMessage bool `mapstructure:"disableStartupMessage"`

	// StreamRequestBody enables streaming of the request body.
	StreamRequestBody bool `mapstructure:"streamRequestBody"`

	// StrictRouting enables strict routing.
	// If enabled, the router will not allow trailing slashes.
	StrictRouting bool `mapstructure:"strictRouting"`

	// CaseSensitive enables case-sensitive routing.
	// If enabled, the router will be case-sensitive.
	CaseSensitive bool `mapstructure:"caseSensitive"`

	// EnableIPValidation enables IP validation.
	EnableIPValidation bool `mapstructure:"enableIPValidation"`

	// EnableTrustedProxyCheck enables trusted proxy check.
	EnableTrustedProxyCheck bool `mapstructure:"enableTrustedProxyCheck"`

	// ProxyHeader is the proxy header.
	ProxyHeader string `mapstructure:"proxyHeader"`

	// EnablePrefork is the enable prefork.
	// Enables use of the SO_REUSEPORT socket option. This will spawn
	// multiple Go processes listening on the same port. Learn more about
	// socket sharding. NOTE: if enabled, the application will need to be
	// ran through a shell because prefork mode sets environment variables.
	// If you're using Docker, make sure the app is ran with CMD ./app or
	// CMD ["sh", "-c", "/app"]. For more info, see this issue comment.
	// ! No longer available after performance benchmark.
	// EnablePrefork bool `mapstructure:"enablePrefork"`

	// EnablePrintRoutes enables print all routes with their method,
	// path, name and handler..
	EnablePrintRoutes bool `mapstructure:"enablePrintRoutes"`
}

type DatabaseDriver string

const (
	DatabaseDriverPostgres DatabaseDriver = "postgresql"
	DatabaseDriverMySQL    DatabaseDriver = "mysql"
	DatabaseDriverSQLite   DatabaseDriver = "sqlite"
	DatabaseDriverMSSQL    DatabaseDriver = "mssql"
	DatabaseDriverOracle   DatabaseDriver = "oracle"
)

// DatabaseConfig is the configuration for the database.
// To understand the configuration options, please refer to the bun documentation:
// https://bun.uptrace.dev/guide/drivers.html
type DatabaseConfig struct {
	// Driver is the database driver to use.
	Driver DatabaseDriver `mapstructure:"driver" json:"driver"`

	// Host is the database host.
	Host string `mapstructure:"host" json:"host"`

	// Port is the database port.
	Port int `mapstructure:"port" json:"port"`

	// Username is the database username.
	Username string `mapstructure:"username" json:"username"`

	// Password is the database password.
	Password string `mapstructure:"password" json:"password"`

	// Database is the database name.
	Database string `mapstructure:"database" json:"database"`

	// SSLMode is the database SSL mode.
	SSLMode string `mapstructure:"sslMode" json:"sslMode"`

	// MaxConnections is the maximum number of connections in the pool.
	MaxConnections int `mapstructure:"maxConnections" json:"maxConnections"`

	// MaxIdleConns is the maximum number of connections in the idle pool.
	MaxIdleConns int `mapstructure:"maxIdleConns" json:"maxIdleConns"`

	// ConnMaxLifetime is the maximum amount of time a connection can be reused.
	ConnMaxLifetime int `mapstructure:"connMaxLifetime" json:"connMaxLifetime"`

	// ConnMaxIdleTime is the maximum amount of time a connection can be idle.
	ConnMaxIdleTime int `mapstructure:"connMaxIdleTime" json:"connMaxIdleTime"`

	// Debug is the debug mode.
	Debug bool `mapstructure:"debug" json:"debug"`

	// ReadReplicas is the configuration for read replicas.
	// When provided, read operations will be distributed across these replicas.
	ReadReplicas []ReadReplicaConfig `mapstructure:"readReplicas" json:"readReplicas"`

	// EnableReadWriteSeparation enables automatic routing of read queries to replicas.
	EnableReadWriteSeparation bool `mapstructure:"enableReadWriteSeparation" json:"enableReadWriteSeparation"`

	// ReplicaLagThreshold is the maximum allowed replication lag in seconds.
	// If a replica is lagging more than this threshold, it will be temporarily removed from the pool.
	ReplicaLagThreshold int `mapstructure:"replicaLagThreshold" json:"replicaLagThreshold"`
}

// ReadReplicaConfig is the configuration for a read replica.
type ReadReplicaConfig struct {
	// Name is a unique identifier for the replica.
	Name string `mapstructure:"name" json:"name"`

	// Host is the replica host.
	Host string `mapstructure:"host" json:"host"`

	// Port is the replica port.
	Port int `mapstructure:"port" json:"port"`

	// Weight is the relative weight for load balancing (higher = more traffic).
	// Default is 1.
	Weight int `mapstructure:"weight" json:"weight"`

	// MaxConnections is the maximum number of connections for this replica.
	// If not specified, uses the primary database's MaxConnections.
	MaxConnections int `mapstructure:"maxConnections" json:"maxConnections"`

	// MaxIdleConns is the maximum number of idle connections for this replica.
	// If not specified, uses the primary database's MaxIdleConns.
	MaxIdleConns int `mapstructure:"maxIdleConns" json:"maxIdleConns"`
}

// RedisConfig is the configuration for the redis.
// To understand the configuration options, please refer to the redis documentation:
// https://pkg.go.dev/github.com/go-redis/redis/v9#Options
type RedisConfig struct {
	// Addr is the redis address.
	Addr string `mapstructure:"addr"`

	// // Username is the redis username.
	// // This is used to authenticate the redis connection.
	// // If the username is not set, the redis connection will not be authenticated.
	// Username string `mapstructure:"username"`

	// Password is the redis password.
	// This is used to authenticate the redis connection.
	// If the password is not set, the redis connection will not be authenticated.
	Password string `mapstructure:"password"`

	// DB is the redis database.
	// This is the database number to select after connecting to the server.
	// It is recommended to use a different database for each service.
	DB int `mapstructure:"db"`

	// ConnTimeout is the redis connection timeout.
	// This is the timeout for the redis connection.
	ConnTimeout time.Duration `mapstructure:"connTimeout"`

	// ReadTimeout is the redis read timeout.
	// This is the timeout for the redis read operation.
	ReadTimeout time.Duration `mapstructure:"readTimeout"`

	// WriteTimeout is the redis write timeout.
	// This is the timeout for the redis write operation.
	WriteTimeout time.Duration `mapstructure:"writeTimeout"`

	// PoolSize is the redis pool size.
	// This is the size of the redis connection pool.
	// Defaults to 10. If you're using a high-traffic service, you may want to increase this.
	PoolSize int `mapstructure:"poolSize"`

	// MinIdleConns is the redis minimum idle connections.
	// This is the minimum number of idle connections in the redis connection pool.
	// Defaults to 10. If you're using a high-traffic service, you may want to increase this.
	MinIdleConns int `mapstructure:"minIdleConns"`
}

// AuthConfig is the configuration for the auth.
type AuthConfig struct {
	// SessionCookieName is the session cookie name.
	// This is the name of the session cookie.
	// Preferrably this should not be changed. It is used to identify the session cookie.
	SessionCookieName string `mapstructure:"sessionCookieName"`

	// CookiePath is the cookie path.
	// This is the path of the cookie.
	// Defaults to "/".
	// Should be set to "/" for the root path. If you're using a sub-path, set it to the sub-path.
	CookiePath string `mapstructure:"cookiePath"`

	// CookieDomain is the cookie domain.
	// This is the domain of the cookie.
	// Defaults to "".
	CookieDomain string `mapstructure:"cookieDomain"`

	// CookieSecure is the cookie secure.
	// This is the secure flag of the cookie.
	// Defaults to false.
	// ! In Production, this should always be set to `true`.
	CookieSecure bool `mapstructure:"cookieSecure"`

	// CookieHTTPOnly is the cookie HTTP only.
	// This is the HTTP only flag of the cookie.
	// Defaults to false.
	// ! In Production, this should always be set to `true`.
	CookieHTTPOnly bool `mapstructure:"cookieHTTPOnly"`

	// CookieSameSite is the cookie same site.
	// This is the same site flag of the cookie.
	// Defaults to "Lax".
	// ! In Production, this should always be set to `Strict`.
	CookieSameSite string `mapstructure:"cookieSameSite"`
}

// MinioConfig is the configuration for the minio.
type MinioConfig struct {
	// Endpoint is the minio endpoint.
	Endpoint string `mapstructure:"endpoint"`

	// AccessKey is the minio access key.
	AccessKey string `mapstructure:"accessKey"`

	// SecretKey is the minio secret key.
	SecretKey string `mapstructure:"secretKey"`

	// Region is the minio region.
	Region string `mapstructure:"region"`

	// UseSSL is the minio use SSL.
	UseSSL bool `mapstructure:"useSSL"`

	// ConnectionTimeout is the minio connection timeout.
	ConnectionTimeout time.Duration `mapstructure:"connectionTimeout"`

	// RequestTimeout is the minio request timeout.
	RequestTimeout time.Duration `mapstructure:"requestTimeout"`

	// MaxRetries is the max retries.
	MaxRetries int `mapstructure:"maxRetries"`

	// MaxIdleConns is the max idle connections.
	MaxIdleConns int `mapstructure:"maxIdleConns"`

	// MaxConnsPerHost is the max connections per host.
	MaxConnsPerHost int `mapstructure:"maxConnsPerHost"`

	// IdleConnTimeout is the idle connection timeout.
	IdleConnTimeout time.Duration `mapstructure:"idleConnTimeout"`
}

// CorsConfig is the configuration for the cors.
type CorsConfig struct {
	// AllowedOrigins is the allowed origins.
	// This is the allowed origins for the cors.
	// Defaults to "*".
	AllowedOrigins string `mapstructure:"allowedOrigins"`

	// AllowedHeaders is the allowed headers.
	// This is the allowed headers for the cors.
	// Defaults to "*".
	AllowedHeaders string `mapstructure:"allowedHeaders"`

	// AllowedMethods is the allowed methods.
	// This is the allowed methods for the cors.
	// Defaults to "*".
	AllowedMethods string `mapstructure:"allowedMethods"`

	// AllowCredentials is the allow credentials.
	// This is the allow credentials for the cors.
	// Defaults to false.
	AllowCredentials bool `mapstructure:"allowCredentials"`

	// MaxAge is the max age.
	// This is the max age for the cors.
	// Defaults to 0.
	MaxAge int `mapstructure:"maxAge"`
}

// BackupConfig is the configuration database backups
type BackupConfig struct {
	// Enabled determines whether the backup service is active.
	Enabled bool `mapstructure:"enabled"`

	// BackupDir is the directory where backups will be stored.
	// Default: "./backups"
	BackupDir string `mapstructure:"backupDir"`

	// RetentionDays is the number of days to keep backups.
	// Backups older than this will be automatically deleted.
	// Default: 30
	RetentionDays int `mapstructure:"retentionDays"`

	// Schedule is the cron schedule for automatic backups.
	// Examples:
	// - "0 0 * * *" (daily at midnight)
	// - "0 0 * * 0" (weekly on Sunday at midnight)
	// - "0 0 1 * *" (monthly on the 1st at midnight)
	// Default: "0 0 * * *" (daily at midnight)
	Schedule string `mapstructure:"schedule"`

	// Compression determines the compression level (1-9).
	// Higher values result in better compression but slower speed.
	// Default: 6
	Compression int `mapstructure:"compression"`

	// MaxConcurrentBackups is the maximum number of backup operations
	// that can run simultaneously.
	// Default: 1
	MaxConcurrentBackups int `mapstructure:"maxConcurrentBackups"`

	// BackupTimeout is the maximum time allowed for a backup operation.
	// Default: 30 minutes
	BackupTimeout time.Duration `mapstructure:"backupTimeout"`

	// NotifyOnFailure determines whether to send notifications on backup failures.
	// Default: true
	NotifyOnFailure bool `mapstructure:"notifyOnFailure"`

	// NotifyOnSuccess determines whether to send notifications on backup success.
	// Default: false
	NotifyOnSuccess bool `mapstructure:"notifyOnSuccess"`

	// NotificationEmail is the email address to send notifications to.
	// If empty, email notifications will be disabled.
	NotificationEmail string `mapstructure:"notificationEmail"`
}

// KafkaConfig is the configuration for Kafka CDC
type KafkaConfig struct {
	// Enabled determines whether Kafka CDC is active
	Enabled bool `mapstructure:"enabled"`

	// Brokers is the list of Kafka broker addresses
	Brokers []string `mapstructure:"brokers"`

	// ConsumerGroupID is the Kafka consumer group ID
	ConsumerGroupID string `mapstructure:"consumerGroupId"`

	// TopicPattern is the Kafka topic pattern for all table changes (e.g., "trenova.public.*")
	TopicPattern string `mapstructure:"topicPattern"`

	// CommitInterval is the interval for committing offsets
	CommitInterval time.Duration `mapstructure:"commitInterval"`

	// StartOffset determines where to start reading (earliest/latest)
	StartOffset string `mapstructure:"startOffset"`

	// MaxRetries is the maximum number of retries for failed operations
	MaxRetries int `mapstructure:"maxRetries"`

	// RetryBackoff is the backoff duration between retries
	RetryBackoff time.Duration `mapstructure:"retryBackoff"`

	// ReadTimeout is the timeout for reading messages
	ReadTimeout time.Duration `mapstructure:"readTimeout"`

	// WriteTimeout is the timeout for writing messages
	WriteTimeout time.Duration `mapstructure:"writeTimeout"`

	// SchemaRegistryURL is the URL of the schema registry
	SchemaRegistryURL string `mapstructure:"schemaRegistryURL"`
}

// StreamingConfig is the configuration for CDC-based real-time streaming
type StreamingConfig struct {
	// MaxConnections is the maximum number of concurrent connections per stream
	MaxConnections int `mapstructure:"maxConnections"`

	// StreamTimeout is the maximum duration for a stream connection (0 for no timeout)
	StreamTimeout time.Duration `mapstructure:"streamTimeout"`

	// EnableHeartbeat enables periodic heartbeat messages to keep connections alive
	EnableHeartbeat bool `mapstructure:"enableHeartbeat"`

	// HeartbeatInterval is the interval for sending heartbeat messages
	HeartbeatInterval time.Duration `mapstructure:"heartbeatInterval"`

	// MaxConnectionsPerUser limits connections per user
	MaxConnectionsPerUser int `mapstructure:"maxConnectionsPerUser"`
}

// AIConfig is the configuration for AI services
type AIConfig struct {
	// ClaudeAPIKey is the Anthropic Claude API key
	ClaudeAPIKey string `mapstructure:"claudeApiKey"`

	// ClaudeModel is the Claude model to use (e.g., "claude-3-haiku-20240307")
	ClaudeModel string `mapstructure:"claudeModel"`

	// MaxTokens is the maximum number of tokens for AI responses
	MaxTokens int `mapstructure:"maxTokens"`

	// Temperature controls the randomness of AI responses (0.0-1.0)
	Temperature float64 `mapstructure:"temperature"`

	// CacheEnabled enables caching of AI responses
	CacheEnabled bool `mapstructure:"cacheEnabled"`

	// CacheTTL is the time-to-live for cached AI responses in seconds
	CacheTTL int `mapstructure:"cacheTtl"`
}

type CronSchedulerConfig struct {
	// Enabled determines whether the cron scheduler is active
	Enabled bool `mapstructure:"enabled"`

	// LogLevel sets the logging level for the scheduler (debug, info, warn, error)
	LogLevel string `mapstructure:"logLevel"`

	// TimeZone for cron schedules (e.g., "America/New_York")
	TimeZone string `mapstructure:"timeZone"`

	// GlobalRetryPolicy is the default retry policy for all jobs
	GlobalRetryPolicy JobRetryPolicy `mapstructure:"globalRetryPolicy"`

	// ShipmentJobs is the configuration for shipment jobs
	ShipmentJobs ShipmentJobsConfig `mapstructure:"shipmentJobs"`

	// PatternAnalysisJobs is the configuration for pattern analysis jobs
	PatternAnalysisJobs PatternAnalysisJobsConfig `mapstructure:"patternAnalysisJobs"`

	// EmailQueueJobs is the configuration for email queue jobs
	EmailQueueJobs EmailQueueJobsConfig `mapstructure:"emailQueueJobs"`

	// SystemJobs is the configuration for system maintenance jobs
	SystemJobs SystemJobsConfig `mapstructure:"systemJobs"`

	// ComplianceJobs is the configuration for compliance jobs
	ComplianceJobs ComplianceJobsConfig `mapstructure:"complianceJobs"`
}

// JobRetryPolicy defines retry behavior for jobs
type JobRetryPolicy struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int `mapstructure:"maxRetries"`

	// RetryDelay is the initial delay between retries
	RetryDelay time.Duration `mapstructure:"retryDelay"`

	// MaxRetryDelay is the maximum delay between retries
	MaxRetryDelay time.Duration `mapstructure:"maxRetryDelay"`

	// RetryBackoffMultiplier is the exponential backoff multiplier
	RetryBackoffMultiplier float64 `mapstructure:"retryBackoffMultiplier"`
}

// JobConfig is the base configuration for all job types
type JobConfig struct {
	// Enabled determines whether this job is active
	Enabled bool `mapstructure:"enabled"`

	// Schedule is the cron schedule expression
	Schedule string `mapstructure:"schedule"`

	// Timeout is the maximum duration for job execution
	Timeout time.Duration `mapstructure:"timeout"`

	// Queue is the queue name for this job
	Queue string `mapstructure:"queue"`

	// Priority sets the job priority (0-10, higher is more important)
	Priority int `mapstructure:"priority"`

	// Retention is how long to keep completed job records
	Retention time.Duration `mapstructure:"retention"`

	// UniqueKey prevents duplicate jobs within this duration
	UniqueKey time.Duration `mapstructure:"uniqueKey"`

	// RetryPolicy overrides the global retry policy for this job
	RetryPolicy *JobRetryPolicy `mapstructure:"retryPolicy"`
}

type ShipmentJobsConfig struct {
	// DelayShipment job configuration
	DelayShipment JobConfig `mapstructure:"delayShipment"`

	// StatusUpdate job configuration
	StatusUpdate JobConfig `mapstructure:"statusUpdate"`

	// DuplicateCheck job configuration
	DuplicateCheck JobConfig `mapstructure:"duplicateCheck"`

	// NotificationJob configuration
	NotificationJob JobConfig `mapstructure:"notificationJob"`

	// DelayThreshold is the time after which shipments are considered delayed
	DelayThreshold time.Duration `mapstructure:"delayThreshold"`

	// BatchSize for processing shipments
	BatchSize int `mapstructure:"batchSize"`
}

type PatternAnalysisJobsConfig struct {
	// DailyAnalysis job configuration
	DailyAnalysis JobConfig `mapstructure:"dailyAnalysis"`

	// WeeklyAnalysis job configuration
	WeeklyAnalysis JobConfig `mapstructure:"weeklyAnalysis"`

	// ExpireSuggestions job configuration
	ExpireSuggestions JobConfig `mapstructure:"expireSuggestions"`

	// MinFrequency is the minimum occurrence frequency for pattern detection
	MinFrequency int64 `mapstructure:"minFrequency"`

	// AnalysisLookbackDays is how many days of data to analyze
	AnalysisLookbackDays int `mapstructure:"analysisLookbackDays"`

	// SuggestionTTL is how long to keep pattern suggestions
	SuggestionTTL time.Duration `mapstructure:"suggestionTtl"`

	// MaxConcurrentAnalysis limits concurrent pattern analysis
	MaxConcurrentAnalysis int `mapstructure:"maxConcurrentAnalysis"`
}

type EmailQueueJobsConfig struct {
	// ProcessQueue job configuration
	ProcessQueue JobConfig `mapstructure:"processQueue"`

	// RetryFailed job configuration
	RetryFailed JobConfig `mapstructure:"retryFailed"`

	// CleanupOld job configuration
	CleanupOld JobConfig `mapstructure:"cleanupOld"`

	// BatchSize for processing emails
	BatchSize int `mapstructure:"batchSize"`

	// MaxSendRate is emails per minute
	MaxSendRate int `mapstructure:"maxSendRate"`

	// FailedRetentionDays is how long to keep failed email records
	FailedRetentionDays int `mapstructure:"failedRetentionDays"`
}

type SystemJobsConfig struct {
	// CleanupTempFiles job configuration
	CleanupTempFiles JobConfig `mapstructure:"cleanupTempFiles"`

	// GenerateReports job configuration
	GenerateReports JobConfig `mapstructure:"generateReports"`

	// DataBackup job configuration
	DataBackup JobConfig `mapstructure:"dataBackup"`

	// TempFileMaxAge is how old temp files must be before cleanup
	TempFileMaxAge time.Duration `mapstructure:"tempFileMaxAge"`

	// ReportFormats supported report formats
	ReportFormats []string `mapstructure:"reportFormats"`

	// BackupRetentionDays is how long to keep backups
	BackupRetentionDays int `mapstructure:"backupRetentionDays"`
}

type ComplianceJobsConfig struct {
	// ComplianceCheck job configuration
	ComplianceCheck JobConfig `mapstructure:"complianceCheck"`

	// HazmatExpiration job configuration
	HazmatExpiration JobConfig `mapstructure:"hazmatExpiration"`

	// ExpirationWarningDays is days before expiration to start warnings
	ExpirationWarningDays int `mapstructure:"expirationWarningDays"`

	// ComplianceCheckDepth is how thorough compliance checks should be
	ComplianceCheckDepth string `mapstructure:"complianceCheckDepth"`
}

// TelemetryConfig is the configuration for telemetry and observability
type TelemetryConfig struct {
	// Enabled determines whether telemetry is active
	Enabled bool `mapstructure:"enabled"`

	// MetricsEnabled determines whether metrics collection is active
	MetricsEnabled bool `mapstructure:"metricsEnabled"`

	// TracingEnabled determines whether distributed tracing is active
	TracingEnabled bool `mapstructure:"tracingEnabled"`

	// LoggingEnabled determines whether structured logging with telemetry is active
	LoggingEnabled bool `mapstructure:"loggingEnabled"`

	// ServiceName is the name of the service for telemetry
	ServiceName string `mapstructure:"serviceName"`

	// ServiceVersion is the version of the service
	ServiceVersion string `mapstructure:"serviceVersion"`

	// Environment is the deployment environment
	Environment string `mapstructure:"environment"`

	// MetricsPort is the port for exposing Prometheus metrics
	MetricsPort int `mapstructure:"metricsPort"`

	// MetricsPath is the path for exposing Prometheus metrics
	MetricsPath string `mapstructure:"metricsPath"`

	// OTLP is the OpenTelemetry Protocol configuration
	OTLP OTLPConfig `mapstructure:"otlp"`

	// Sampling is the trace sampling configuration
	Sampling SamplingConfig `mapstructure:"sampling"`
}

// OTLPConfig is the configuration for OpenTelemetry Protocol
type OTLPConfig struct {
	// Endpoint is the OTLP endpoint (e.g., "localhost:4317")
	Endpoint string `mapstructure:"endpoint"`

	// Insecure determines whether to use insecure connection
	Insecure bool `mapstructure:"insecure"`

	// Headers are additional headers to send with OTLP requests
	Headers map[string]string `mapstructure:"headers"`

	// Timeout is the timeout for OTLP requests
	Timeout time.Duration `mapstructure:"timeout"`

	// RetryConfig is the retry configuration for OTLP
	RetryConfig OTLPRetryConfig `mapstructure:"retry"`
}

// OTLPRetryConfig is the retry configuration for OTLP
type OTLPRetryConfig struct {
	// Enabled determines whether retries are enabled
	Enabled bool `mapstructure:"enabled"`

	// InitialInterval is the initial retry interval
	InitialInterval time.Duration `mapstructure:"initialInterval"`

	// MaxInterval is the maximum retry interval
	MaxInterval time.Duration `mapstructure:"maxInterval"`

	// MaxElapsedTime is the maximum elapsed time for retries
	MaxElapsedTime time.Duration `mapstructure:"maxElapsedTime"`
}

// SamplingConfig is the configuration for trace sampling
type SamplingConfig struct {
	// Probability is the sampling probability (0.0 to 1.0)
	Probability float64 `mapstructure:"probability"`

	// ParentBased determines whether to use parent-based sampling
	ParentBased bool `mapstructure:"parentBased"`
}
