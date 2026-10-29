package realtimeconfig

import (
	"time"
)

// RealtimeConfig tunes the server-sent event fan-out. Every value has a
// production default, so an empty section is a working one. ShardCount must be
// identical on every instance that shares a Redis, because it decides which
// stream a tenant's events are written to and read from.
type Config struct {
	ShardCount            int           `mapstructure:"shardCount"            validate:"omitempty,min=1,max=256"`
	StreamMaxLen          int64         `mapstructure:"streamMaxLen"          validate:"omitempty,min=100"`
	ReplayScanLimit       int64         `mapstructure:"replayScanLimit"       validate:"omitempty,min=1"`
	SubscriberBuffer      int           `mapstructure:"subscriberBuffer"      validate:"omitempty,min=8"`
	MaxConnections        int           `mapstructure:"maxConnections"        validate:"omitempty,min=1"`
	MaxConnectionsPerUser int           `mapstructure:"maxConnectionsPerUser" validate:"omitempty,min=1"`
	HeartbeatInterval     time.Duration `mapstructure:"heartbeatInterval"`
	PresenceTTL           time.Duration `mapstructure:"presenceTTL"`
	MaxStreamLifetime     time.Duration `mapstructure:"maxStreamLifetime"`
	PublishQueueSize      int           `mapstructure:"publishQueueSize"      validate:"omitempty,min=1"`
	PublishBatchSize      int           `mapstructure:"publishBatchSize"      validate:"omitempty,min=1"`
	PublishTimeout        time.Duration `mapstructure:"publishTimeout"`
	MaxEntityBytes        int           `mapstructure:"maxEntityBytes"        validate:"omitempty,min=0"`
	TypingThrottle        time.Duration `mapstructure:"typingThrottle"`
}

const (
	DefaultShardCount            = 16
	DefaultStreamMaxLen          = 10_000
	DefaultReplayScanLimit       = 5_000
	DefaultSubscriberBuffer      = 256
	DefaultMaxConnections        = 20_000
	DefaultMaxConnectionsPerUser = 20
	DefaultHeartbeatInterval     = 15 * time.Second
	DefaultPresenceTTL           = 45 * time.Second
	DefaultMaxStreamLifetime     = 30 * time.Minute
	DefaultPublishQueueSize      = 16_384
	DefaultPublishBatchSize      = 256
	DefaultPublishTimeout        = 5 * time.Second
	DefaultMaxEntityBytes        = 64 * 1024
	DefaultTypingThrottle        = time.Second
)

func (c *Config) GetShardCount() int {
	if c.ShardCount <= 0 {
		return DefaultShardCount
	}
	return c.ShardCount
}

func (c *Config) GetStreamMaxLen() int64 {
	if c.StreamMaxLen <= 0 {
		return DefaultStreamMaxLen
	}
	return c.StreamMaxLen
}

func (c *Config) GetReplayScanLimit() int64 {
	if c.ReplayScanLimit <= 0 {
		return DefaultReplayScanLimit
	}
	return c.ReplayScanLimit
}

func (c *Config) GetSubscriberBuffer() int {
	if c.SubscriberBuffer <= 0 {
		return DefaultSubscriberBuffer
	}
	return c.SubscriberBuffer
}

func (c *Config) GetMaxConnections() int {
	if c.MaxConnections <= 0 {
		return DefaultMaxConnections
	}
	return c.MaxConnections
}

func (c *Config) GetMaxConnectionsPerUser() int {
	if c.MaxConnectionsPerUser <= 0 {
		return DefaultMaxConnectionsPerUser
	}
	return c.MaxConnectionsPerUser
}

func (c *Config) GetHeartbeatInterval() time.Duration {
	if c.HeartbeatInterval <= 0 {
		return DefaultHeartbeatInterval
	}
	return c.HeartbeatInterval
}

// GetPresenceTTL is never shorter than two heartbeats, so one late refresh
// never drops a member who is still connected.
func (c *Config) GetPresenceTTL() time.Duration {
	ttl := c.PresenceTTL
	if ttl <= 0 {
		ttl = DefaultPresenceTTL
	}
	return max(ttl, 2*c.GetHeartbeatInterval())
}

func (c *Config) GetMaxStreamLifetime() time.Duration {
	if c.MaxStreamLifetime <= 0 {
		return DefaultMaxStreamLifetime
	}
	return c.MaxStreamLifetime
}

func (c *Config) GetPublishQueueSize() int {
	if c.PublishQueueSize <= 0 {
		return DefaultPublishQueueSize
	}
	return c.PublishQueueSize
}

func (c *Config) GetPublishBatchSize() int {
	if c.PublishBatchSize <= 0 {
		return DefaultPublishBatchSize
	}
	return c.PublishBatchSize
}

func (c *Config) GetPublishTimeout() time.Duration {
	if c.PublishTimeout <= 0 {
		return DefaultPublishTimeout
	}
	return c.PublishTimeout
}

func (c *Config) GetMaxEntityBytes() int {
	if c.MaxEntityBytes <= 0 {
		return DefaultMaxEntityBytes
	}
	return c.MaxEntityBytes
}

func (c *Config) GetTypingThrottle() time.Duration {
	if c.TypingThrottle <= 0 {
		return DefaultTypingThrottle
	}
	return c.TypingThrottle
}
