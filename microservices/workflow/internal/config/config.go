package config

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/emoss08/trenova/microservices/workflow/internal/model"
)

type RabbitMQConfig struct {
	Host          string
	Port          int
	Username      string
	Password      string
	VHost         string
	ExchangeName  string
	QueueName     string
	PrefetchCount int
	Timeout       time.Duration
}

func (c *RabbitMQConfig) URL() string {
	hostPort := net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	return fmt.Sprintf("amqp://%s:%s@%s/%s", c.Username, c.Password, hostPort, c.VHost)
}

type DBConfig struct {
	Host            string
	Port            int
	Database        string
	Username        string
	Password        string
	SSLMode         string
	MaxConnections  int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	Debug           bool
}

func (c *DBConfig) DSN() string {
	hostPort := net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s", c.Username, c.Password, hostPort, c.Database, c.SSLMode)
}

type AppConfig struct {
	LogLevel      string
	Environment   string
	RabbitMQ      *RabbitMQConfig
	DB            *DBConfig
	WorkflowTypes []model.WorkflowType
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *AppConfig {
	return &AppConfig{
		LogLevel:    getEnvOrDefault("LOG_LEVEL", "info"),
		Environment: getEnvOrDefault("ENVIRONMENT", "development"),
		RabbitMQ:    loadRabbitMQConfig(),
		DB:          loadDBConfig(),
	}
}

// loadRabbitMQConfig loads RabbitMQ specific configuration
func loadRabbitMQConfig() *RabbitMQConfig {
	port, _ := strconv.Atoi(getEnvOrDefault("RABBITMQ_PORT", "5674"))
	prefetchCount, _ := strconv.Atoi(getEnvOrDefault("RABBITMQ_PREFETCH_COUNT", "10"))
	timeoutSec, _ := strconv.Atoi(getEnvOrDefault("RABBITMQ_TIMEOUT_SEC", "30"))

	return &RabbitMQConfig{
		Host:          getEnvOrDefault("RABBITMQ_HOST", "localhost"),
		Port:          port,
		Username:      getEnvOrDefault("RABBITMQ_USERNAME", "user"),
		Password:      getEnvOrDefault("RABBITMQ_PASSWORD", "password"),
		VHost:         getEnvOrDefault("RABBITMQ_VHOST", "/"),
		ExchangeName:  getEnvOrDefault("RABBITMQ_EXCHANGE", "trenova"),
		QueueName:     getEnvOrDefault("RABBITMQ_QUEUE", "trenovas"),
		PrefetchCount: prefetchCount,
		Timeout:       time.Duration(timeoutSec) * time.Second,
	}
}

func loadDBConfig() *DBConfig {
	port, _ := strconv.Atoi(getEnvOrDefault("DB_PORT", "5432"))
	maxConnections, _ := strconv.Atoi(getEnvOrDefault("DB_MAX_CONNECTIONS", "10"))
	maxIdleConns, _ := strconv.Atoi(getEnvOrDefault("DB_MAX_IDLE_CONNS", "10"))
	connMaxLifetime, _ := strconv.Atoi(getEnvOrDefault("DB_CONN_MAX_LIFETIME", "10"))
	connMaxIdleTime, _ := strconv.Atoi(getEnvOrDefault("DB_CONN_MAX_IDLE_TIME", "10"))
	debug, _ := strconv.ParseBool(getEnvOrDefault("DB_DEBUG", "false"))

	return &DBConfig{
		Host:            getEnvOrDefault("DB_HOST", "localhost"),
		Port:            port,
		Database:        getEnvOrDefault("DB_DATABASE", "trenova"),
		Username:        getEnvOrDefault("DB_USERNAME", "user"),
		Password:        getEnvOrDefault("DB_PASSWORD", "password"),
		SSLMode:         getEnvOrDefault("DB_SSLMODE", "disable"),
		MaxConnections:  maxConnections,
		MaxIdleConns:    maxIdleConns,
		ConnMaxLifetime: time.Duration(connMaxLifetime) * time.Second,
		ConnMaxIdleTime: time.Duration(connMaxIdleTime) * time.Second,
		Debug:           debug,
	}
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	log.Printf("Warning: %s not set, using default value: %s", key, defaultValue)
	return defaultValue
}
