package config

import "strings"

const (
	DefaultDatabaseCacheLocalBytes   int64 = 256 << 20
	DefaultDatabaseCacheSharedPrefix       = "trenova:keel:"
	DefaultDatabaseBudgetMaxRepeats        = 10

	minDatabaseCacheLocalBytes int64 = 1 << 20
	tenantConcurrencyShare           = 4
)

type DatabaseCacheConfig struct {
	Enabled    bool                      `mapstructure:"enabled"`
	LocalBytes int64                     `mapstructure:"localBytes" validate:"min=0"`
	ListenHost string                    `mapstructure:"listenHost" validate:"omitempty,max=253"`
	ListenPort int                       `mapstructure:"listenPort" validate:"omitempty,min=1,max=65535"`
	Shared     DatabaseSharedCacheConfig `mapstructure:"shared"`
}

type DatabaseSharedCacheConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Prefix  string `mapstructure:"prefix"  validate:"omitempty,max=64"`
}

type DatabaseBudgetConfig struct {
	MaxRepeats int `mapstructure:"maxRepeats" validate:"omitempty,min=1,max=10000"`
}

func (c *DatabaseCacheConfig) GetLocalBytes() int64 {
	if c.LocalBytes <= 0 {
		return DefaultDatabaseCacheLocalBytes
	}
	return c.LocalBytes
}

func (c *DatabaseCacheConfig) SharedEnabled() bool {
	return c.Enabled && c.Shared.Enabled
}

func (c *DatabaseSharedCacheConfig) GetPrefix() string {
	if prefix := strings.TrimSpace(c.Prefix); prefix != "" {
		return prefix
	}
	return DefaultDatabaseCacheSharedPrefix
}

func (c *DatabaseBudgetConfig) GetMaxRepeats() int {
	if c.MaxRepeats <= 0 {
		return DefaultDatabaseBudgetMaxRepeats
	}
	return c.MaxRepeats
}

func (c *DatabaseConfig) GetTenantConcurrency(maxConns int) int {
	if maxConns <= 0 {
		return max(c.TenantConcurrency, 0)
	}
	if c.TenantConcurrency > 0 {
		return min(c.TenantConcurrency, maxConns)
	}
	return max(maxConns/tenantConcurrencyShare, 1)
}

func (c *Config) GetCacheListenDSN() string {
	host := strings.TrimSpace(c.Database.Cache.ListenHost)
	if host == "" {
		return ""
	}

	port := c.Database.Cache.ListenPort
	if port <= 0 {
		port = c.Database.Port
	}

	return c.postgresDSNAt(host, port, c.Database.User, c.Database.Password)
}

func validateDatabaseRuntimeConfig(config *Config) error {
	db := &config.Database

	if db.Cache.Shared.Enabled && !db.Cache.Enabled {
		return ErrDatabaseSharedCacheNeedsCache
	}
	if db.Cache.ListenPort > 0 && strings.TrimSpace(db.Cache.ListenHost) == "" {
		return ErrDatabaseCacheListenPortNeedsHost
	}
	if db.Cache.LocalBytes > 0 && db.Cache.LocalBytes < minDatabaseCacheLocalBytes {
		return ErrDatabaseCacheLocalBytesTooSmall
	}
	if db.MaxOpenConns > 0 && db.TenantConcurrency > db.MaxOpenConns {
		return ErrDatabaseTenantConcurrencyExceedsPool
	}

	return nil
}
