/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package config

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/robfig/cron/v3"
	"github.com/rotisserie/eris"
	"github.com/spf13/viper"
)

type Manager struct {
	Cfg   *Config
	Viper *viper.Viper
}

func NewManager() *Manager {
	return &Manager{
		Viper: viper.New(),
	}
}

// expandWithDefault parses a string like "${VAR:-default}" and expands VAR,
// using "default" if VAR is not set or is empty.
func expandWithDefault(placeholder string) string {
	// Regex to capture VAR and default_val from "${VAR:-default_val}"
	// It also handles cases like "${VAR}" (no default)
	r := regexp.MustCompile(`^\$\{(?P<var>[A-Z0-9_]+)(?::-(?P<def>.*?))?\}$`)
	matches := r.FindStringSubmatch(placeholder)

	if matches == nil {
		// Not a recognized placeholder format, or just a regular string that might contain $
		// Let os.ExpandEnv handle simple $VAR or ${VAR} cases if any
		return os.ExpandEnv(placeholder)
	}

	varMap := make(map[string]string)
	for i, name := range r.SubexpNames() {
		if i != 0 && name != "" {
			varMap[name] = matches[i]
		}
	}

	varName := varMap["var"]
	envValue := os.Getenv(varName)

	if envValue != "" {
		return envValue
	}
	// If envValue is empty, use the default value
	// varMap["def"] will be empty string if ":-default_val" part was not present
	return varMap["def"]
}

func (m *Manager) Load() (*Config, error) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	// Set default values
	m.setDefaults()

	m.Viper.SetConfigName(fmt.Sprintf("config.%s", env))
	m.Viper.SetConfigType("yaml")
	m.Viper.AddConfigPath(fmt.Sprintf("config/%s", env))
	m.Viper.AddConfigPath("config")
	m.Viper.AddConfigPath(".")

	// Environment variables
	m.Viper.SetEnvPrefix("TRENOVA")
	m.Viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	m.Viper.AutomaticEnv()

	if err := m.Viper.ReadInConfig(); err != nil {
		return nil, eris.Wrap(err, "failed to read config")
	}

	config := &Config{}
	if err := m.Viper.Unmarshal(config); err != nil {
		return nil, eris.Wrap(err, "failed to unmarshal config")
	}

	// Expand for DB User and Password as well, as they use the same pattern
	if config.DB.Username != "" {
		config.DB.Username = expandWithDefault(config.DB.Username)
	}
	if config.DB.Password != "" {
		config.DB.Password = expandWithDefault(config.DB.Password)
	}

	// Expand for Minio AccessKey and SecretKey
	if config.Minio.AccessKey != "" {
		config.Minio.AccessKey = expandWithDefault(config.Minio.AccessKey)
	}
	if config.Minio.SecretKey != "" {
		config.Minio.SecretKey = expandWithDefault(config.Minio.SecretKey)
	}

	// TODO: Consider a more generic way to expand env vars for ALL string fields
	// in the config struct using reflection, if this pattern is widespread.

	m.Cfg = config
	return config, nil
}

func (m *Manager) setDefaults() {
	// App defaults
	m.Viper.SetDefault("app.environment", "development")
	m.Viper.SetDefault("app.logLevel", "info")
	m.Viper.SetDefault("app.version", "0.0.1")

	// Server defaults
	m.Viper.SetDefault("server.host", "0.0.0.0")
	m.Viper.SetDefault("server.port", 8080)
	m.Viper.SetDefault("server.readTimeout", 15)
	m.Viper.SetDefault("server.writeTimeout", 15)
	m.Viper.SetDefault("server.maxHeaderBytes", 1<<20) // 1 MB

	// Database defaults
	m.Viper.SetDefault("db.driver", "postgresql")
	m.Viper.SetDefault("db.sslMode", "disable")
	m.Viper.SetDefault("db.maxConnections", 100)
	m.Viper.SetDefault("db.maxIdleConns", 10)
	m.Viper.SetDefault("db.connMaxLifetime", 3600) // 1 hour

	// Backup defaults
	m.Viper.SetDefault("backup.enabled", false)
	m.Viper.SetDefault("backup.backupDir", "./backups")
	m.Viper.SetDefault("backup.retentionDays", 30)
	m.Viper.SetDefault("backup.schedule", "0 0 * * *") // Daily at midnight
	m.Viper.SetDefault("backup.compression", 6)
	m.Viper.SetDefault("backup.maxConcurrentBackups", 1)
	m.Viper.SetDefault("backup.backupTimeout", 30*60) // 30 minutes in seconds
	m.Viper.SetDefault("backup.notifyOnFailure", true)
	m.Viper.SetDefault("backup.notifyOnSuccess", false)

	// Kafka defaults
	m.Viper.SetDefault("kafka.enabled", false)
	m.Viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	m.Viper.SetDefault("kafka.consumerGroupId", "trenova-shipment-streaming")
	m.Viper.SetDefault("kafka.topicPattern", "trenova.public.*")
	m.Viper.SetDefault("kafka.commitInterval", "1s")
	m.Viper.SetDefault("kafka.startOffset", "latest")
	m.Viper.SetDefault("kafka.maxRetries", 3)
	m.Viper.SetDefault("kafka.retryBackoff", "1s")
	m.Viper.SetDefault("kafka.readTimeout", "10s")
	m.Viper.SetDefault("kafka.writeTimeout", "10s")

	// Streaming defaults
	m.Viper.SetDefault("streaming.pollInterval", "2s")
	m.Viper.SetDefault("streaming.maxConnections", 100)
	m.Viper.SetDefault("streaming.streamTimeout", "30m")
	m.Viper.SetDefault("streaming.enableHeartbeat", true)
	m.Viper.SetDefault("streaming.heartbeatInterval", "30s")
	m.Viper.SetDefault("streaming.maxConnectionsPerUser", 5)

	// gRPC server defaults
	m.Viper.SetDefault("grpc.enabled", false)
	m.Viper.SetDefault("grpc.listenAddress", ":9090")
	m.Viper.SetDefault("grpc.maxRecvMsgSize", 16*1024*1024) // 16MB
	m.Viper.SetDefault("grpc.maxSendMsgSize", 16*1024*1024) // 16MB
	m.Viper.SetDefault("grpc.reflection", true)
    m.Viper.SetDefault("grpc.tls.enabled", false)
    m.Viper.SetDefault("grpc.tls.certFile", "")
    m.Viper.SetDefault("grpc.tls.keyFile", "")
    m.Viper.SetDefault("grpc.tls.clientCAFile", "")
    m.Viper.SetDefault("grpc.tls.requireClientCert", false)
    m.Viper.SetDefault("grpc.auth.enabled", false)
    m.Viper.SetDefault("grpc.auth.bearerTokens", []string{})
    m.Viper.SetDefault("grpc.auth.apiKeys", []string{})
}

func (m *Manager) Get() *Config {
	return m.Cfg
}

func (m *Manager) Validate() error {
	if m.Cfg == nil {
		return ErrConfigNotLoaded
	}

	if err := m.validateApp(); err != nil {
		return eris.Wrap(err, "app config validation failed")
	}

	if err := m.validateServer(); err != nil {
		return eris.Wrap(err, "server config validation failed")
	}

	if err := m.validateDatabase(); err != nil {
		return eris.Wrap(err, "database config validation failed")
	}

	if err := m.validateBackup(); err != nil {
		return eris.Wrap(err, "backup config validation failed")
	}

	if err := m.validateGRPC(); err != nil {
		return eris.Wrap(err, "grpc config validation failed")
	}

	return nil
}

func (m *Manager) validateApp() error {
	if m.Cfg.App.Name == "" {
		return ErrInvalidAppName
	}
	return nil
}

func (m *Manager) validateServer() error {
	if m.Cfg.Server.ListenAddress == "" {
		return ErrInvalidServerAddress
	}
	return nil
}

func (m *Manager) validateDatabase() error {
	if m.Cfg.DB.Host == "" {
		return ErrInvalidDBHost
	}
	if m.Cfg.DB.Port == 0 {
		return ErrInvalidDBPort
	}
	if m.Cfg.DB.Database == "" {
		return ErrInvalidDBName
	}
	if m.Cfg.DB.Username == "" {
		return ErrInvalidDBUser
	}
	return nil
}

func (m *Manager) validateGRPC() error {
    if !m.Cfg.GRPC.Enabled {
        return nil
    }
    if m.Cfg.GRPC.ListenAddress == "" {
        return ErrInvalidServerAddress
    }
    return nil
}

func (m *Manager) validateBackup() error {
	// Only validate if backup is enabled
	if !m.Cfg.Backup.Enabled {
		return nil
	}

	// Validate compression level
	if m.Cfg.Backup.Compression < 1 || m.Cfg.Backup.Compression > 9 {
		return ErrInvalidBackupCompression
	}

	// Validate cron schedule
	if m.Cfg.Backup.Schedule != "" {
		_, err := cron.ParseStandard(m.Cfg.Backup.Schedule)
		if err != nil {
			return eris.Wrap(err, "invalid backup schedule")
		}
	}

	return nil
}

// Helper methods for easier access to config sections
func (m *Manager) App() *AppConfig {
	if m.Cfg == nil {
		return nil
	}
	return &m.Cfg.App
}

func (m *Manager) Server() *ServerConfig {
	if m.Cfg == nil {
		return nil
	}
	return &m.Cfg.Server
}

func (m *Manager) Log() *LogConfig {
	if m.Cfg == nil {
		return nil
	}
	return &m.Cfg.Log
}

func (m *Manager) Database() *DatabaseConfig {
	if m.Cfg == nil {
		return nil
	}
	return &m.Cfg.DB
}

func (m *Manager) Redis() *RedisConfig {
	if m.Cfg == nil {
		return nil
	}
	return &m.Cfg.Redis
}

func (m *Manager) Auth() *AuthConfig {
	if m.Cfg == nil {
		return nil
	}
	return &m.Cfg.Auth
}

func (m *Manager) Audit() *AuditConfig {
	if m.Cfg == nil {
		return nil
	}
	return &m.Cfg.Audit
}

func (m *Manager) Minio() *MinioConfig {
	if m.Cfg == nil {
		return nil
	}
	return &m.Cfg.Minio
}

func (m *Manager) Cors() *CorsConfig {
	if m.Cfg == nil {
		return nil
	}
	return &m.Cfg.Cors
}

func (m *Manager) Static() *StaticConfig {
	if m.Cfg == nil {
		return nil
	}
	return &m.Cfg.Static
}

func (m *Manager) Backup() *BackupConfig {
	if m.Cfg == nil {
		return nil
	}
	return &m.Cfg.Backup
}

func (m *Manager) Kafka() *KafkaConfig {
	if m.Cfg == nil {
		return nil
	}
	return &m.Cfg.Kafka
}

func (m *Manager) Streaming() *StreamingConfig {
	if m.Cfg == nil {
		return nil
	}
	return &m.Cfg.Streaming
}

// GetDSN returns a formatted database connection string
func (m *Manager) GetDSN() string {
	db := m.Database()
	if db == nil {
		return ""
	}

	// Get the type of database driver
	driver := db.Driver
	hostPort := net.JoinHostPort(db.Host, strconv.Itoa(db.Port))

	switch driver {
	case DatabaseDriverPostgres:
		return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
			db.Username,
			db.Password,
			hostPort,
			db.Database,
			db.SSLMode,
		)
	case DatabaseDriverMSSQL:
		return fmt.Sprintf("sqlserver://%s:%s@%s?database=%s",
			db.Username,
			db.Password,
			hostPort,
			db.Database,
		)
	case DatabaseDriverMySQL:
		return fmt.Sprintf("%s:%s@/%s",
			db.Username,
			db.Password,
			db.Database,
		)
	case DatabaseDriverSQLite:
		return "file::memory:?cache=shared"
	case DatabaseDriverOracle:
		return fmt.Sprintf("%s/%s",
			db.Username,
			db.Password,
		)
	default:
		return ""
	}
}
