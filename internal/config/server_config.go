package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"

	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"
)

type FiberServer struct {
	ListenAddress                  string
	HideInternalServerErrorDetails bool
	BaseURL                        string
	EnableLoggerMiddleware         bool
	EnableCORSMiddleware           bool
	EnableHelmetMiddleware         bool
	EnableIdempotencyMiddleware    bool
	EnableRequestIDMiddleware      bool
	EnableETagMiddleware           bool
	EnableSessionMiddleware        bool
	EnableCompressMiddleware       bool
	EnableRecoverMiddleware        bool
	EnableEncryptCookieMiddleware  bool
	EnableMonitorMiddleware        bool
}

type Integration struct {
	GenerateReportEndpoint string
}

type RedisServer struct {
	Host     string
	Port     int
	Username string
	Password string `json:"-"`
	Database int
	Addr     string
}

type LoggerServer struct {
	Level              zerolog.Level
	RequestLevel       zerolog.Level
	LogRequestBody     bool
	LogRequestHeader   bool
	LogRequestQuery    bool
	LogResponseBody    bool
	LogResponseHeader  bool
	LogCaller          bool
	PrettyPrintConsole bool
}

type MinioServer struct {
	Endpoint  string `json:"-"`
	AccessKey string `json:"-"`
	SecretKey string `json:"-"`
	UseSSL    bool
}

type KafkaServer struct {
	Broker string
}

type EncryptCookie struct {
	Key string
}

type Monitor struct {
	Path string
}

type Server struct {
	DB           Database       `json:"database"`
	Fiber        FiberServer    `json:"fiber"`
	Logger       LoggerServer   `json:"logger"`
	Redis        RedisServer    `json:"redis"`
	SessionStore *session.Store `json:"-"`
	Kafka        KafkaServer    `json:"kafka"`
	Cookie       EncryptCookie  `json:"cookie"`
	Monitor      Monitor        `json:"monitor"`
	Minio        MinioServer    `json:"minio"`
	Integration  Integration    `json:"integration_server"`
}

// DefaultServiceConfigFromEnv returns the server config as parsed from environment variables
// and their respective defaults defined below.
// We don't expect that ENV_VARs change while we are running our application or our tests
// (and it would be a bad thing to do anyway with parallel testing).
// Do NOT use os.Setenv / os.Unsetenv in tests utilizing DefaultServiceConfigFromEnv()!
func DefaultServiceConfigFromEnv() Server {
	if !util.RunningInTest() {
		DotEnvTryLoad(filepath.Join(util.GetProjectRootDir(), ".env.local"), os.Setenv)
	}

	return Server{
		DB: Database{
			Host:     util.GetEnv("SERVER_DB_HOST", "localhost"),
			Port:     util.GetEnvAsInt("SERVER_DB_PORT", 5432),
			Database: util.GetEnv("SERVER_DB_NAME", "trenova_go_db"),
			Username: util.GetEnv("SERVER_DB_USER", "postgres"),
			Password: util.GetEnv("SERVER_DB_PASSWORD", "postgres"),
			AdditionalParams: map[string]string{
				"sslmode": util.GetEnv("SERVER_DB_SSL_MODE", "disable"),
			},
			MaxOpenConns:    util.GetEnvAsInt("SERVER_DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    util.GetEnvAsInt("SERVER_DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: time.Second * time.Duration(util.GetEnvAsInt("DB_CONN_MAX_LIFETIME_SEC", 300)),
		},
		Fiber: FiberServer{
			ListenAddress:                  util.GetEnv("SERVER_FIBER_LISTEN_ADDRESS", ":3000"),
			HideInternalServerErrorDetails: util.GetEnvAsBool("SERVER_FIBER_HIDE_INTERNAL_SERVER_ERROR_DETAILS", true),
			BaseURL:                        util.GetEnv("SERVER_FIBER_BASE_URL", "http://localhost:3000"),
			EnableLoggerMiddleware:         util.GetEnvAsBool("SERVER_FIBER_ENABLE_LOGGER_MIDDLEWARE", true),
			EnableCORSMiddleware:           util.GetEnvAsBool("SERVER_FIBER_ENABLE_CORS_MIDDLEWARE", true),
			EnableRequestIDMiddleware:      util.GetEnvAsBool("SERVER_FIBER_ENABLE_REQUEST_ID_MIDDLEWARE", true),
			EnableHelmetMiddleware:         util.GetEnvAsBool("SERVER_FIBER_ENABLE_HELMET_MIDDLEWARE", true),
			EnableIdempotencyMiddleware:    util.GetEnvAsBool("SERVER_FIBER_ENABLE_IDEMPOTENCY_MIDDLEWARE", true),
			EnableETagMiddleware:           util.GetEnvAsBool("SERVER_FIBER_ENABLE_ETAG_MIDDLEWARE", true),
			EnableSessionMiddleware:        util.GetEnvAsBool("SERVER_FIBER_ENABLE_SESSION_MIDDLEWARE", true),
			EnableCompressMiddleware:       util.GetEnvAsBool("SERVER_FIBER_ENABLE_COMPRESS_MIDDLEWARE", true),
			EnableRecoverMiddleware:        util.GetEnvAsBool("SERVER_FIBER_ENABLE_RECOVER_MIDDLEWARE", true),
			EnableEncryptCookieMiddleware:  util.GetEnvAsBool("SERVER_FIBER_ENABLE_ENCRYPT_COOKIE_MIDDLEWARE", true),
			EnableMonitorMiddleware:        util.GetEnvAsBool("SERVER_FIBER_ENABLE_MONITOR_MIDDLEWARE", true),
		},
		Logger: LoggerServer{
			Level:              util.LogLevelFromString(util.GetEnv("SERVER_LOGGER_LEVEL", zerolog.DebugLevel.String())),
			RequestLevel:       util.LogLevelFromString(util.GetEnv("SERVER_LOGGER_REQUEST_LEVEL", zerolog.DebugLevel.String())),
			LogRequestBody:     util.GetEnvAsBool("SERVER_LOGGER_LOG_REQUEST_BODY", false),
			LogRequestHeader:   util.GetEnvAsBool("SERVER_LOGGER_LOG_REQUEST_HEADER", false),
			LogRequestQuery:    util.GetEnvAsBool("SERVER_LOGGER_LOG_REQUEST_QUERY", false),
			LogResponseBody:    util.GetEnvAsBool("SERVER_LOGGER_LOG_RESPONSE_BODY", false),
			LogResponseHeader:  util.GetEnvAsBool("SERVER_LOGGER_LOG_RESPONSE_HEADER", false),
			LogCaller:          util.GetEnvAsBool("SERVER_LOGGER_LOG_CALLER", false),
			PrettyPrintConsole: util.GetEnvAsBool("SERVER_LOGGER_PRETTY_PRINT_CONSOLE", false),
		},
		Redis: RedisServer{
			Host:     util.GetEnv("SERVER_REDIS_HOST", "localhost"),
			Port:     util.GetEnvAsInt("SERVER_REDIS_PORT", 6379),
			Username: util.GetEnv("SERVER_REDIS_USER", ""),
			Password: util.GetEnv("SERVER_REDIS_PASSWORD", ""),
			Database: util.GetEnvAsInt("SERVER_REDIS_DB", 0),
			Addr:     util.GetEnv("SERVER_REDIS_ADDR", "localhost:6379"),
		},
		Kafka: KafkaServer{
			Broker: util.GetEnv("SERVER_KAFKA_BROKER", "localhost:9092"),
		},
		Cookie: EncryptCookie{
			Key: util.GetEnv("SERVER_COOKIE_KEY", "octxhyEw4TS8RjK8ahe0M1ti9StS+xqFvk+iFS7d3qk="), // NOTE: this value is only used in development
		},
		Monitor: Monitor{
			Path: util.GetEnv("SERVER_METRICS_PATH", "/metrics"),
		},
		Minio: MinioServer{
			Endpoint:  util.GetEnv("SERVER_MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: util.GetEnv("SERVER_MINIO_ACCESS_KEY", "minio"),
			SecretKey: util.GetEnv("SERVER_MINIO_SECRET_KEY", "minio123"),
			UseSSL:    util.GetEnvAsBool("SERVER_MINIO_USE_SSL", false),
		},
		Integration: Integration{
			GenerateReportEndpoint: util.GetEnv("SERVER_INTEGRATION_GENERATE_REPORT_ENDPOINT", "http://localhost:8000/generate-report/"),
		},
	}
}
