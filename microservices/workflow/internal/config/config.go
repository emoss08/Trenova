package config

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"
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

type AppConfig struct {
	LogLevel    string
	Environment string
	RabbitMQ    *RabbitMQConfig
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *AppConfig {
	return &AppConfig{
		LogLevel:    getEnvOrDefault("LOG_LEVEL", "info"),
		Environment: getEnvOrDefault("ENVIRONMENT", "development"),
		RabbitMQ:    loadRabbitMQConfig(),
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

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	log.Printf("Warning: %s not set, using default value: %s", key, defaultValue)
	return defaultValue
}
