package config

import (
	"crypto/rsa"
	"time"

	"github.com/emoss08/trenova/pkg/utils"
	"github.com/rs/zerolog"
)

type FiberServer struct {
	// ListenAddress is the address that the server will listen on.
	ListenAddress string

	// Enable prefork on the server instance.
	//
	// When prefork is enabled, the server will fork itself into multiple processes to handle incoming requests.
	// This can be useful to take advantage of multiple CPU cores.
	//
	// When prefork is disabled, the server will run in a single process and will only be able to take advantage of a single CPU core.
	// By default, prefork is disabled.
	//
	// Read More: https://github.com/gofiber/fiber/issues/180
	EnablePrefork bool

	// Print out all the routes to the console.
	EnablePrintRoutes bool

	// Enable logging middleware on the server instance.
	//
	// Read More: https://docs.gofiber.io/contrib/fiberzap/
	EnableLoggingMiddleware bool

	// Enable helmet middleware on the server instance.
	//
	// Read More: https://docs.gofiber.io/api/middleware/helmet
	EnableHelmetMiddleware bool

	// Enable request ID middleware on the server instance.
	//
	// Read More: https://docs.gofiber.io/api/middleware/requestid
	EnableRequestIDMiddleware bool

	// Enable recover middleware on the server instance.
	//
	// Read More: https://docs.gofiber.io/api/middleware/recover
	EnableRecoverMiddleware bool

	// Enable CORS middleware on the server instance.
	//
	// Read More: https://docs.gofiber.io/api/middleware/cors
	EnableCORSMiddleware bool

	// Enable Idempotency middleware on the server instance.
	//
	// Read More: https://docs.gofiber.io/api/middleware/idempotency
	EnableIdempotencyMiddleware bool

	// Enable Prometheus middleware on the server instance.
	EnablePrometheusMiddleware bool
}

type Integration struct {
	// GenerateReportEndpoint is the URL of the endpoint that will generate a report.
	GenerateReportEndpoint string
}

type Meilisearch struct {
	// Host is the URL of the MeiliSearch server.
	Host string

	// Token is the API key used to authenticate with the MeiliSearch server.
	Token string
}

type Auth struct {
	// PrivateKey is the RSA private key used to sign JWT tokens.
	PrivateKey *rsa.PrivateKey

	// PublicKey is the RSA public key used to verify JWT tokens.
	PublicKey *rsa.PublicKey
}

type Cors struct {
	// Allowed Origins for Cors Middleware.
	// Example: "https://localhost:5173, https://localhost:4173"
	AllowedOrigins string

	// Allowed Headers for Cors Middleware.
	// Example: "Authorization, Origin, Content-Type, Accept, X-CSRF-Token, X-Idempotency-Key"
	AllowedHeaders string

	// Allowed Methods for Cors Middleware.
	// Example: "GET, POST, PUT, DELETE, OPTIONS"
	AllowedMethods string

	// AllowCredentials for Cors Middleware.
	AllowCredentials bool

	// MaxAge for Cors Middleware.
	MaxAge int
}

type Logger struct {
	// Level is the log level for the logger.
	Level zerolog.Level

	// PrettyPrintConsole will print the logs in a human-readable format.
	PrettyPrintConsole bool
}

type Minio struct {
	// Endpoint is the URL of the Minio server.
	Endpoint string `json:"-"`

	// AccessKey is the access key used to authenticate with the Minio server.
	AccessKey string `json:"-"`

	// SecretKey is the secret key used to authenticate with the Minio server.
	SecretKey string `json:"-"`

	// UseSSL is a flag to determine if the Minio client should use SSL.
	UseSSL bool
}

type KafkaServer struct {
	// Brokers is the list of Kafka brokers.
	// Example: "localhost:9092, localhost:9093"
	Broker string
}

type Server struct {
	// FiberServer contains configuration options for the Fiber server.
	Fiber FiberServer

	// Database contains configuration options for the database.
	DB Database

	// Mellisearch contains configuration options for the MeiliSearch server.
	// Meilisearch Meilisearch

	// Auth contains configuration options for the JWT authentication.
	Auth Auth

	// Cors contains configuration options for the CORS middleware.
	Cors Cors

	// Logger contains configuration options for the logger.
	Logger Logger

	// Minio contains configuration options for the Minio server.
	Minio Minio

	// Kafka contains configuration options for the Kafka server.
	Kafka KafkaServer

	// Integration contains configuration options for the integration services.
	Integration Integration
}

func DefaultServiceConfigFromEnv() Server {
	return Server{
		Fiber: FiberServer{
			ListenAddress:               utils.GetEnv("SERVER_FIBER_LISTEN_ADDRESS", ":3001"),
			EnableLoggingMiddleware:     utils.GetEnvAsBool("SERVER_FIBER_ENABLE_LOGGER_MIDDLEWARE", true),
			EnableCORSMiddleware:        utils.GetEnvAsBool("SERVER_FIBER_ENABLE_CORS_MIDDLEWARE", true),
			EnableRequestIDMiddleware:   utils.GetEnvAsBool("SERVER_FIBER_ENABLE_REQUEST_ID_MIDDLEWARE", true),
			EnableHelmetMiddleware:      utils.GetEnvAsBool("SERVER_FIBER_ENABLE_HELMET_MIDDLEWARE", true),
			EnableIdempotencyMiddleware: utils.GetEnvAsBool("SERVER_FIBER_ENABLE_IDEMPOTENCY_MIDDLEWARE", true),
			EnableRecoverMiddleware:     utils.GetEnvAsBool("SERVER_FIBER_ENABLE_RECOVER_MIDDLEWARE", true),
			EnablePrometheusMiddleware:  utils.GetEnvAsBool("SERVER_FIBER_ENABLE_PROMETHEUS_MIDDLEWARE", true),
			EnablePrefork:               utils.GetEnvAsBool("SERVER_FIBER_ENABLE_PREFORK", false),
			EnablePrintRoutes:           utils.GetEnvAsBool("SERVER_FIBER_ENABLE_PRINT_ROUTES", false),
		},
		DB: Database{
			Host:     utils.GetEnv("SERVER_DB_HOST", "localhost"),
			Port:     utils.GetEnvAsInt("SERVER_DB_PORT", 5432),
			Database: utils.GetEnv("SERVER_DB_NAME", "trenova_go_db"),
			Username: utils.GetEnv("SERVER_DB_USER", "postgres"),
			Password: utils.GetEnv("SERVER_DB_PASSWORD", "postgres"),
			AdditionalParams: map[string]string{
				"sslmode": utils.GetEnv("SERVER_DB_SSL_MODE", "disable"),
			},
			MaxOpenConns:    utils.GetEnvAsInt("SERVER_DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    utils.GetEnvAsInt("SERVER_DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: time.Second * time.Duration(utils.GetEnvAsInt("SERVER_DB_CONN_MAX_LIFETIME_SECONDS", 300)),
			VerboseLogging:  utils.GetEnvAsBool("SERVER_DB_VERBOSE_LOGGING", true),
			Debug:           utils.GetEnvAsBool("SERVER_DB_DEBUG", true),
		},
		Logger: Logger{
			Level:              utils.LogLevelFromString(utils.GetEnv("SERVER_LOGGER_LEVEL", zerolog.DebugLevel.String())),
			PrettyPrintConsole: utils.GetEnvAsBool("SERVER_LOGGER_PRETTY_PRINT_CONSOLE", true),
		},
		Kafka: KafkaServer{
			Broker: utils.GetEnv("SERVER_KAFKA_BROKER", "localhost:9092"),
		},
		// Meilisearch: Meilisearch{
		// 	Host:  utils.GetEnv("SERVER_MELLISEARCH_HOST", "http://localhost:7700"),
		// 	Token: utils.GetEnv("SERVER_MELLISEARCH_TOKEN", "private-meilisearch-token-for-dev-only"),
		// },
		Cors: Cors{
			AllowedOrigins:   utils.GetEnv("SEVER_CORS_ALLOWED_ORIGINS", "https://localhost:5173, http://localhost:5173, https://localhost:4173, http://localhost:4173"),
			AllowedHeaders:   utils.GetEnv("SEVER_CORS_ALLOWED_HEADERS", "Authorization, Origin, Content-Type, Accept, X-CSRF-Token, X-Idempotency-Key"),
			AllowedMethods:   utils.GetEnv("SERVER_CORS_ALLOWED_METHODS", "GET, POST, PUT, DELETE, OPTIONS"),
			AllowCredentials: utils.GetEnvAsBool("SERVER_CORS_ALLOWED_METHODS", true),
			MaxAge:           utils.GetEnvAsInt("SERVER_CORS_MAX_AGE", 300),
		},
		Integration: Integration{
			GenerateReportEndpoint: utils.GetEnv("SERVER_INTEGRATION_GENERATE_REPORT_ENDPOINT", "http://localhost:8000/report/generate-report/"),
		},
		Minio: Minio{
			Endpoint:  utils.GetEnv("SERVER_MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: utils.GetEnv("SERVER_MINIO_ACCESS_KEY", "minio"),
			SecretKey: utils.GetEnv("SERVER_MINIO_SECRET_KEY", "minio123"),
			UseSSL:    utils.GetEnvAsBool("SERVER_MINIO_USE_SSL", false),
		},
	}
}
