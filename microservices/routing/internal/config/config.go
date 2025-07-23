// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package config

import (
	"time"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port string `mapstructure:"port"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	DSN            string `mapstructure:"dsn"`
	AutoMigrate    bool   `mapstructure:"auto_migrate"`
	MaxConnections int    `mapstructure:"max_connections"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// KafkaConfig represents Kafka configuration
type KafkaConfig struct {
	Brokers  []string       `mapstructure:"brokers"`
	Producer ProducerConfig `mapstructure:"producer"`
	Consumer ConsumerConfig `mapstructure:"consumer"`
	Topics   TopicsConfig   `mapstructure:"topics"`
}

// ProducerConfig represents Kafka producer configuration
type ProducerConfig struct {
	BatchSize    int           `mapstructure:"batch_size"`
	BatchTimeout time.Duration `mapstructure:"batch_timeout"`
	Async        bool          `mapstructure:"async"`
	Compression  string        `mapstructure:"compression"`
}

// ConsumerConfig represents Kafka consumer configuration
type ConsumerConfig struct {
	GroupID        string        `mapstructure:"group_id"`
	MinBytes       int           `mapstructure:"min_bytes"`
	MaxBytes       int           `mapstructure:"max_bytes"`
	MaxWait        time.Duration `mapstructure:"max_wait"`
	CommitInterval time.Duration `mapstructure:"commit_interval"`
}

// TopicsConfig represents Kafka topic configuration
type TopicsConfig struct {
	RouteEvents        string `mapstructure:"route_events"`
	BatchRequests      string `mapstructure:"batch_requests"`
	OSMUpdates         string `mapstructure:"osm_updates"`
	RestrictionUpdates string `mapstructure:"restriction_updates"`
	CacheInvalidation  string `mapstructure:"cache_invalidation"`
}
