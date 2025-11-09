package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"
)

// Config represents the main application configuration
type Config struct {
	App             AppConfig              `mapstructure:"app"                validate:"required"`
	Server          ServerConfig           `mapstructure:"server"             validate:"required"`
	Database        DatabaseConfig         `mapstructure:"database"           validate:"required"`
	Cache           *CacheConfig           `mapstructure:"cache,omitempty"`
	Queue           *QueueConfig           `mapstructure:"queue,omitempty"`
	AI              *AIConfig              `mapstructure:"ai"                 validate:"required"`
	Google          *GoogleConfig          `mapstructure:"google"             validate:"required"`
	CDC             *CDCConfig             `mapstructure:"cdc,omitempty"`
	Storage         *StorageConfig         `mapstructure:"storage,omitempty"`
	Temporal        *TemporalConfig        `mapstructure:"temporal,omitempty"`
	Email           *EmailConfig           `mapstructure:"email,omitempty"    validate:"required"`
	PermissionCache *PermissionCacheConfig `mapstructure:"permissionCache"    validate:"required"`
	Search          *SearchConfig          `mapstructure:"search,omitempty"`
	Security        SecurityConfig         `mapstructure:"security"           validate:"required"`
	Logging         LoggingConfig          `mapstructure:"logging"            validate:"required"`
	Monitoring      MonitoringConfig       `mapstructure:"monitoring"         validate:"required"`
	Streaming       StreamingConfig        `mapstructure:"streaming"          validate:"required"`
	WebSocket       WebSocketConfig        `mapstructure:"websocket"          validate:"required"`
}

// AppConfig contains application-level settings
type AppConfig struct {
	Name    string `mapstructure:"name"    validate:"required,min=1,max=100"`
	Env     string `mapstructure:"env"     validate:"required,oneof=development staging production test"`
	Debug   bool   `mapstructure:"debug"`
	Version string `mapstructure:"version" validate:"required"`
}

// ServerConfig contains HTTP server settings
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

type PermissionCacheConfig struct {
	Workers    int `mapstructure:"workers"    validate:"min=1,max=100"`
	BufferSize int `mapstructure:"bufferSize" validate:"min=1,max=10000"`
}

type GoogleConfig struct {
	APIKey string `mapstructure:"apiKey"`
}

type AIConfig struct {
	OpenAIAPIKey string `mapstructure:"openaiAPIKey"`
}

// CORSConfig contains CORS settings
type CORSConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	AllowedOrigins []string `mapstructure:"allowedOrigins" validate:"required_if=Enabled true,dive,url,no_trailing_slash|eq=*"`
	AllowedMethods []string `mapstructure:"allowedMethods" validate:"required_if=Enabled true"`
	AllowedHeaders []string `mapstructure:"allowedHeaders" validate:"required_if=Enabled true"`
	ExposeHeaders  []string `mapstructure:"exposeHeaders"`
	Credentials    bool     `mapstructure:"credentials"`
	MaxAge         int      `mapstructure:"maxAge"         validate:"min=0,max=86400"`
}

// DatabaseConfig contains database connection settings
type DatabaseConfig struct {
	PasswordSource  string        `mapstructure:"passwordSource"  validate:"required,oneof=env file secret"`
	Host            string        `mapstructure:"host"            validate:"required"`
	Port            int           `mapstructure:"port"            validate:"required,min=1,max=65535"`
	Name            string        `mapstructure:"name"            validate:"required,min=1,max=63"`
	User            string        `mapstructure:"user"            validate:"required,min=1,max=63"`
	Password        string        `mapstructure:"password"        validate:"required_if=PasswordSource env"`
	PasswordFile    string        `mapstructure:"passwordFile"    validate:"required_if=PasswordSource file"`
	PasswordSecret  string        `mapstructure:"passwordSecret"  validate:"required_if=PasswordSource secret"`
	SSLMode         string        `mapstructure:"sslMode"         validate:"required,oneof=disable require verify-ca verify-full"`
	MaxIdleConns    int           `mapstructure:"maxIdleConns"    validate:"min=1,max=1000"`
	MaxOpenConns    int           `mapstructure:"maxOpenConns"    validate:"min=1,max=1000"`
	ConnMaxLifetime time.Duration `mapstructure:"connMaxLifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"connMaxIdleTime"`
	Verbose         bool          `mapstructure:"verbose"`
}

// CacheConfig contains Redis cache settings (optional)
type CacheConfig struct {
	Provider         string        `mapstructure:"provider"         validate:"required,oneof=redis memory"`
	Host             string        `mapstructure:"host"             validate:"required_if=Provider redis"`
	Port             int           `mapstructure:"port"             validate:"required_if=Provider redis,min=1,max=65535"`
	Password         string        `mapstructure:"password"`
	Username         string        `mapstructure:"username"`
	DB               int           `mapstructure:"db"               validate:"min=0,max=15"`
	PoolSize         int           `mapstructure:"poolSize"         validate:"min=1,max=1000"`
	MinIdleConns     int           `mapstructure:"minIdleConns"     validate:"min=0,max=1000"`
	MaxRetries       int           `mapstructure:"maxRetries"       validate:"min=0,max=10"`
	DefaultTTL       time.Duration `mapstructure:"defaultTTL"`
	MaxRetryBackoff  time.Duration `mapstructure:"maxRetryBackoff"`
	MinRetryBackoff  time.Duration `mapstructure:"minRetryBackoff"`
	DialTimeout      time.Duration `mapstructure:"dialTimeout"`
	ReadTimeout      time.Duration `mapstructure:"readTimeout"`
	WriteTimeout     time.Duration `mapstructure:"writeTimeout"`
	PoolTimeout      time.Duration `mapstructure:"poolTimeout"`
	ConnMaxIdleTime  time.Duration `mapstructure:"connMaxIdleTime"`
	ConnMaxLifetime  time.Duration `mapstructure:"connMaxLifetime"`
	ClusterMode      bool          `mapstructure:"clusterMode"`
	ClusterNodes     []string      `mapstructure:"clusterNodes"     validate:"required_if=ClusterMode true"`
	SentinelMode     bool          `mapstructure:"sentinelMode"`
	MasterName       string        `mapstructure:"masterName"       validate:"required_if=SentinelMode true"`
	SentinelAddrs    []string      `mapstructure:"sentinelAddrs"    validate:"required_if=SentinelMode true"`
	SentinelPassword string        `mapstructure:"sentinelPassword"`
	EnablePipelining bool          `mapstructure:"enablePipelining"`
	SlowLogThreshold time.Duration `mapstructure:"slowLogThreshold"`
}

func (c *CacheConfig) GetRedisAddr() string {
	if c.Provider != "redis" {
		panic("provider must be redis")
	}

	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

type QueueConfig struct {
	Provider      string            `mapstructure:"provider"      validate:"required,oneof=kafka rabbitmq redis"`
	Brokers       []string          `mapstructure:"brokers"       validate:"required_if=Provider kafka,min=1"`
	ConsumerGroup string            `mapstructure:"consumerGroup" validate:"required_if=Provider kafka,min=1,max=255"`
	Topics        map[string]string `mapstructure:"topics"`
}

// CDCConfig contains CDC (Change Data Capture) settings
type CDCConfig struct {
	Enabled           bool                 `mapstructure:"enabled"`
	Brokers           []string             `mapstructure:"brokers"           validate:"required_if=Enabled true,min=1"`
	ConsumerGroup     string               `mapstructure:"consumerGroup"     validate:"required_if=Enabled true"`
	TopicPattern      string               `mapstructure:"topicPattern"      validate:"required_if=Enabled true"`
	SchemaRegistryURL string               `mapstructure:"schemaRegistryURL" validate:"required_if=Enabled true,url"`
	StartOffset       string               `mapstructure:"startOffset"       validate:"oneof=earliest latest"`
	MaxRetryAttempts  int                  `mapstructure:"maxRetryAttempts"  validate:"min=0,max=10"`
	Processing        CDCProcessingConfig  `mapstructure:"processing"`
	SchemaCache       CDCSchemaCacheConfig `mapstructure:"schemaCache"`
	Retry             CDCRetryConfig       `mapstructure:"retry"`
}

// CDCProcessingConfig contains CDC message processing settings
type CDCProcessingConfig struct {
	WorkerCount              int           `mapstructure:"workerCount"              validate:"min=1,max=100"`
	MessageChannelSize       int           `mapstructure:"messageChannelSize"       validate:"min=1,max=10000"`
	ProcessingTimeout        time.Duration `mapstructure:"processingTimeout"`
	ShutdownTimeout          time.Duration `mapstructure:"shutdownTimeout"`
	EnableParallelProcessing bool          `mapstructure:"enableParallelProcessing"`
}

// CDCSchemaCacheConfig contains schema cache settings
type CDCSchemaCacheConfig struct {
	MaxSize         int           `mapstructure:"maxSize"         validate:"min=10,max=10000"`
	TTL             time.Duration `mapstructure:"ttl"`
	CleanupInterval time.Duration `mapstructure:"cleanupInterval"`
	EvictionPolicy  string        `mapstructure:"evictionPolicy"  validate:"oneof=lru fifo"`
}

// CDCRetryConfig contains retry behavior settings
type CDCRetryConfig struct {
	InitialBackoff time.Duration `mapstructure:"initialBackoff"`
	MaxBackoff     time.Duration `mapstructure:"maxBackoff"`
	BackoffFactor  float64       `mapstructure:"backoffFactor"  validate:"min=1,max=10"`
	MaxAttempts    int           `mapstructure:"maxAttempts"    validate:"min=1,max=20"`
}

// StorageConfig contains object storage settings (optional)
type StorageConfig struct {
	Provider     string `mapstructure:"provider"     validate:"required,oneof=minio s3 local"`
	Endpoint     string `mapstructure:"endpoint"`
	AccessKey    string `mapstructure:"accessKey"`
	SecretKey    string `mapstructure:"secretKey"`
	SessionToken string `mapstructure:"sessionToken"`
	Region       string `mapstructure:"region"`
	Bucket       string `mapstructure:"bucket"`
	UseSSL       bool   `mapstructure:"useSSL"`
}

// TemporalConfig contains Temporal workflow engine settings
type TemporalConfig struct {
	HostPort string                 `mapstructure:"hostPort" validate:"required,hostname_port"`
	Security TemporalSecurityConfig `mapstructure:"security" validate:"required"`
}

// TemporalSecurityConfig configures payload encryption and compression
type TemporalSecurityConfig struct {
	EnableEncryption     bool   `mapstructure:"enableEncryption"`
	EncryptionKeyID      string `mapstructure:"encryptionKeyID"      validate:"required_if=EnableEncryption true,min=1,max=100"`
	EnableCompression    bool   `mapstructure:"enableCompression"`
	CompressionThreshold int    `mapstructure:"compressionThreshold" validate:"min=0,max=1048576"`
}

// EmailConfig contains email settings (optional)
type EmailConfig struct {
	Provider string     `mapstructure:"provider" validate:"required,oneof=smtp resend"`
	From     string     `mapstructure:"from"     validate:"required,email"`
	SMTP     SMTPConfig `mapstructure:"smtp"`
	APIKey   string     `mapstructure:"apiKey"`
}

// SMTPConfig contains SMTP settings
type SMTPConfig struct {
	Host     string `mapstructure:"host"     validate:"required_if=Provider smtp"`
	Port     int    `mapstructure:"port"     validate:"required_if=Provider smtp"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	UseTLS   bool   `mapstructure:"useTLS"`
}

// SecurityConfig contains security settings
type SecurityConfig struct {
	Session    SessionConfig    `mapstructure:"session"    validate:"required"`
	APIToken   APITokenConfig   `mapstructure:"apiToken"`
	RateLimit  RateLimitConfig  `mapstructure:"rateLimit"`
	CSRF       CSRFConfig       `mapstructure:"csrf"`
	Encryption EncryptionConfig `mapstructure:"encryption"`
}

// EncryptionConfig contains encryption settings
type EncryptionConfig struct {
	Key string `mapstructure:"key" validate:"required,min=32"`
}

// SessionConfig contains session settings
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

// APITokenConfig contains API token settings
type APITokenConfig struct {
	Enabled          bool          `mapstructure:"enabled"`
	DefaultExpiry    time.Duration `mapstructure:"defaultExpiry"`
	MaxExpiry        time.Duration `mapstructure:"maxExpiry"`
	MaxTokensPerUser int           `mapstructure:"maxTokensPerUser"`
}

// RateLimitConfig contains rate limiting settings
type RateLimitConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	RequestsPerMinute int           `mapstructure:"requestsPerMinute" validate:"min=1,max=10000"`
	BurstSize         int           `mapstructure:"burstSize"         validate:"min=1,max=1000"`
	CleanupInterval   time.Duration `mapstructure:"cleanupInterval"`
}

// CSRFConfig contains CSRF protection settings
type CSRFConfig struct {
	TokenName  string `mapstructure:"tokenName"  validate:"required"`
	HeaderName string `mapstructure:"headerName" validate:"required"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level      string         `mapstructure:"level"          validate:"required,oneof=debug info warn error"`
	Format     string         `mapstructure:"format"         validate:"required,oneof=json text"`
	Output     string         `mapstructure:"output"         validate:"required,oneof=stdout stderr file"`
	File       *LogFileConfig `mapstructure:"file,omitempty"`
	Sampling   bool           `mapstructure:"sampling"`
	Stacktrace bool           `mapstructure:"stacktrace"`
}

// LogFileConfig contains log file settings
type LogFileConfig struct {
	Path       string `mapstructure:"path"       validate:"required"`
	MaxSize    int    `mapstructure:"maxSize"    validate:"min=1,max=1000"`
	MaxAge     int    `mapstructure:"maxAge"     validate:"min=1,max=365"`
	MaxBackups int    `mapstructure:"maxBackups" validate:"min=0,max=100"`
	Compress   bool   `mapstructure:"compress"`
}

// MonitoringConfig contains monitoring settings
type MonitoringConfig struct {
	Metrics *MetricsConfig `mapstructure:"metrics,omitempty"`
	Tracing *TracingConfig `mapstructure:"tracing,omitempty"`
	Health  HealthConfig   `mapstructure:"health"`
}

// MetricsConfig contains metrics settings
type MetricsConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Provider  string `mapstructure:"provider"  validate:"required_if=Enabled true,oneof=prometheus datadog"`
	Port      int    `mapstructure:"port"      validate:"required_if=Provider prometheus,min=1,max=65535"`
	Path      string `mapstructure:"path"      validate:"required_if=Provider prometheus"`
	Namespace string `mapstructure:"namespace" validate:"required_if=Enabled true"`
	Subsystem string `mapstructure:"subsystem" validate:"required_if=Enabled true"`
	APIKey    string `mapstructure:"apiKey"    validate:"required_if=Provider datadog"`
}

// TracingConfig contains tracing settings
type TracingConfig struct {
	Enabled      bool    `mapstructure:"enabled"`
	Provider     string  `mapstructure:"provider"     validate:"required_if=Enabled true,oneof=jaeger zipkin otlp otlp-grpc stdout"`
	Endpoint     string  `mapstructure:"endpoint"     validate:"required_if=Enabled true"`
	ServiceName  string  `mapstructure:"serviceName"  validate:"required_if=Enabled true"`
	SamplingRate float64 `mapstructure:"samplingRate" validate:"min=0,max=1"`
}

// HealthConfig contains health check settings
type HealthConfig struct {
	Path          string        `mapstructure:"path"          validate:"required"`
	ReadinessPath string        `mapstructure:"readinessPath"`
	LivenessPath  string        `mapstructure:"livenessPath"`
	CheckInterval time.Duration `mapstructure:"checkInterval"`
	Timeout       time.Duration `mapstructure:"timeout"`
}

// StreamingConfig is the configuration for CDC-based real-time streaming
type StreamingConfig struct {
	MaxConnections        int           `mapstructure:"maxConnections"        validate:"min=1,max=100"`
	StreamTimeout         time.Duration `mapstructure:"streamTimeout"         validate:"min=0"`
	MaxConnectionsPerUser int           `mapstructure:"maxConnectionsPerUser" validate:"min=1,max=100"`
}

// WebSocketConfig contains WebSocket connection settings
type WebSocketConfig struct {
	MaxTotalConnections   int           `mapstructure:"maxTotalConnections"   validate:"min=1,max=50000"`
	MaxConnectionsPerUser int           `mapstructure:"maxConnectionsPerUser" validate:"min=1,max=20"`
	ConnectionTimeout     time.Duration `mapstructure:"connectionTimeout"     validate:"min=0"`
	PingInterval          time.Duration `mapstructure:"pingInterval"          validate:"min=0"`
	ReadBufferSize        int           `mapstructure:"readBufferSize"        validate:"min=1024"`
	WriteBufferSize       int           `mapstructure:"writeBufferSize"       validate:"min=1024"`
}

// SearchConfig contains Meilisearch settings
type SearchConfig struct {
	Host        string        `mapstructure:"host"        validate:"required_if=Enabled true"`
	APIKey      string        `mapstructure:"apiKey"      validate:"required_if=Enabled true"`
	IndexPrefix string        `mapstructure:"indexPrefix"`
	Timeout     time.Duration `mapstructure:"timeout"`
}

// Environment check methods
func (c *Config) IsDevelopment() bool {
	return c.App.Env == EnvDevelopment
}

func (c *Config) IsProduction() bool {
	return c.App.Env == EnvProduction
}

func (c *Config) IsStaging() bool {
	return c.App.Env == EnvStaging
}

func (c *Config) IsTest() bool {
	return c.App.Env == EnvTest
}

// GetDSN returns a secure PostgreSQL connection string
func (c *Config) GetDSN(password string) string {
	escapedPassword := url.QueryEscape(password)

	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		c.Database.User,
		escapedPassword,
		net.JoinHostPort(c.Database.Host, strconv.Itoa(c.Database.Port)),
		c.Database.Name,
		c.Database.SSLMode,
	)

	// Add application name for monitoring
	dsn += fmt.Sprintf("&application_name=%s", url.QueryEscape(c.App.Name))

	// Add timeouts
	dsn += "&connect_timeout=10&statement_timeout=30000&idle_in_transaction_session_timeout=30000"

	return dsn
}

// GetDSNMasked returns the DSN with password masked for logging
func (c *Config) GetDSNMasked() string {
	return fmt.Sprintf("postgres://%s:****@%s/%s?sslmode=%s",
		c.Database.User,
		net.JoinHostPort(c.Database.Host, strconv.Itoa(c.Database.Port)),
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// GetServerAddr returns the server address
func (c *Config) GetServerAddr() string {
	return net.JoinHostPort(c.Server.Host, strconv.Itoa(c.Server.Port))
}
