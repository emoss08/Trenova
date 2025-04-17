package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// TLSPolicy represents the TLS policy for SMTP connections
type TLSPolicy string

const (
	// TLSMandatory requires TLS for connections
	TLSMandatory TLSPolicy = "mandatory"
	// TLSOpportunistic attempts TLS but falls back to plaintext
	TLSOpportunistic TLSPolicy = "opportunistic"
	// TLSNone disables TLS
	TLSNone TLSPolicy = "none"
)

// AppConfig holds all the configuration for the application
type AppConfig struct {
	Environment string
	Port        int
	RabbitMQ    RabbitMQConfig
	SMTP        SMTPConfig
	SendGrid    SendGridConfig
	RateLimit   RateLimitConfig
}

// RabbitMQConfig holds the RabbitMQ configuration
type RabbitMQConfig struct {
	Host          string
	Port          int
	User          string
	Password      string
	VHost         string
	ExchangeName  string
	QueueName     string
	PrefetchCount int
	Timeout       time.Duration
}

// URL returns the RabbitMQ connection URL
func (c *RabbitMQConfig) URL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		c.User, c.Password, c.Host, c.Port, c.VHost)
}

// SMTPConfig holds the SMTP configuration
type SMTPConfig struct {
	Host      string
	Port      int
	User      string
	Password  string
	From      string
	FromName  string
	TLSPolicy TLSPolicy
	Timeout   time.Duration
}

// SendGridConfig holds the SendGrid configuration
type SendGridConfig struct {
	APIKey string
	From   string
	Name   string
}

// RateLimitConfig holds the rate limit configuration
type RateLimitConfig struct {
	PerMinute int
	PerHour   int
}

// LoadConfig loads the configuration from environment variables
func LoadConfig() *AppConfig {
	return &AppConfig{
		Environment: getEnv("EMAIL_ENV", "development"),
		Port:        getEnvAsInt("EMAIL_PORT", 8082),
		RabbitMQ: RabbitMQConfig{
			Host:          getEnv("EMAIL_RABBITMQ_HOST", "localhost"),
			Port:          getEnvAsInt("EMAIL_RABBITMQ_PORT", 5672),
			User:          getEnv("EMAIL_RABBITMQ_USER", "guest"),
			Password:      getEnv("EMAIL_RABBITMQ_PASSWORD", "guest"),
			VHost:         getEnv("EMAIL_RABBITMQ_VHOST", "/"),
			ExchangeName:  getEnv("EMAIL_RABBITMQ_EXCHANGE", "trenova.events"),
			QueueName:     getEnv("EMAIL_RABBITMQ_QUEUE", "email.service"),
			PrefetchCount: getEnvAsInt("EMAIL_RABBITMQ_PREFETCH_COUNT", 10),
			Timeout:       getEnvAsDuration("EMAIL_RABBITMQ_TIMEOUT", 5*time.Second),
		},
		SMTP: SMTPConfig{
			Host:      getEnv("EMAIL_SMTP_HOST", "smtp.example.com"),
			Port:      getEnvAsInt("EMAIL_SMTP_PORT", 587),
			User:      getEnv("EMAIL_SMTP_USER", ""),
			Password:  getEnv("EMAIL_SMTP_PASSWORD", ""),
			From:      getEnv("EMAIL_SMTP_FROM", "no-reply@example.com"),
			FromName:  getEnv("EMAIL_SMTP_FROM_NAME", "Trenova"),
			TLSPolicy: getTLSPolicy(getEnv("EMAIL_SMTP_TLS_POLICY", string(TLSMandatory))),
			Timeout:   getEnvAsDuration("EMAIL_SMTP_TIMEOUT", 30*time.Second),
		},
		SendGrid: SendGridConfig{
			APIKey: getEnv("EMAIL_SENDGRID_API_KEY", ""),
			From:   getEnv("EMAIL_SMTP_FROM", "no-reply@example.com"), // Fallback to SMTP
			Name:   getEnv("EMAIL_SMTP_FROM_NAME", "Trenova"),         // Fallback to SMTP
		},
		RateLimit: RateLimitConfig{
			PerMinute: getEnvAsInt("EMAIL_RATE_LIMIT_PER_MINUTE", 100),
			PerHour:   getEnvAsInt("EMAIL_RATE_LIMIT_PER_HOUR", 1000),
		},
	}
}

// Helper functions to get environment variables with defaults
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// getTLSPolicy converts a string to a TLSPolicy
func getTLSPolicy(policy string) TLSPolicy {
	switch TLSPolicy(policy) {
	case TLSMandatory:
		return TLSMandatory
	case TLSOpportunistic:
		return TLSOpportunistic
	case TLSNone:
		return TLSNone
	default:
		return TLSMandatory
	}
}
