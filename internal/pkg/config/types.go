package config

import "time"

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

	// Search is the search configuration.
	Search SearchConfig `mapstructure:"search"`

	// Static is the static configuration.
	Static StaticConfig `mapstructure:"static"`
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
	EnablePrefork bool `mapstructure:"enablePrefork"`

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
	Driver DatabaseDriver `mapstructure:"driver"`

	// Host is the database host.
	Host string `mapstructure:"host"`

	// Port is the database port.
	Port int `mapstructure:"port"`

	// Username is the database username.
	Username string `mapstructure:"username"`

	// Password is the database password.
	Password string `mapstructure:"password"`

	// Database is the database name.
	Database string `mapstructure:"database"`

	// SSLMode is the database SSL mode.
	SSLMode string `mapstructure:"sslMode"`

	// MaxConnections is the maximum number of connections in the pool.
	MaxConnections int `mapstructure:"maxConnections"`

	// MaxIdleConns is the maximum number of connections in the idle pool.
	MaxIdleConns int `mapstructure:"maxIdleConns"`

	// ConnMaxLifetime is the maximum amount of time a connection can be reused.
	ConnMaxLifetime int `mapstructure:"connMaxLifetime"`

	// ConnMaxIdleTime is the maximum amount of time a connection can be idle.
	ConnMaxIdleTime int `mapstructure:"connMaxIdleTime"`

	// Debug is the debug mode.
	Debug bool `mapstructure:"debug"`
}

// RedisConfig is the configuration for the redis.
// To understand the configuration options, please refer to the redis documentation:
// https://pkg.go.dev/github.com/go-redis/redis/v9#Options
type RedisConfig struct {
	// Addr is the redis address.
	Addr string `mapstructure:"addr"`

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

// SearchConfig is the configuration for the search client.
// Note: Some of these options may be irrelevant depending on the search client you are using.
// These configurations are for Meilisearch, but can be used for other search clients.
// You may need to adjust the configurations based on the search client you are using.
type SearchConfig struct {
	// Host is the search host.
	Host string `mapstructure:"host"`

	// APIKey is the search API key.
	APIKey string `mapstructure:"apiKey"`

	// IndexPrefix is the search index prefix.
	IndexPrefix string `mapstructure:"indexPrefix"`

	// MaxBatchSize is the search max batch size.
	MaxBatchSize int `mapstructure:"maxBatchSize"`

	// BatchInterval is the search batch interval.
	BatchInterval int `mapstructure:"batchInterval"`
}
