package database

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxConnections  int
	MaxIdleTime     time.Duration
	ConnMaxLifetime time.Duration
	AppName         string
}

func NewConfig() *Config {
	return &Config{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnvAsInt("DB_PORT", 5432),
		User:            getEnv("DB_USER", "postgres"),
		Password:        getEnv("DB_PASSWORD", ""),
		Database:        getEnv("DB_NAME", "trenova_edi"),
		SSLMode:         getEnv("DB_SSL_MODE", "disable"),
		MaxConnections:  getEnvAsInt("DB_MAX_CONNECTIONS", 25),
		MaxIdleTime:     getEnvAsDuration("DB_MAX_IDLE_TIME", 15*time.Minute),
		ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 1*time.Hour),
		AppName:         getEnv("APP_NAME", "edi-service"),
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s&application_name=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode, c.AppName)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}