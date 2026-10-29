package databaseconfig

import "strings"

const (
	DefaultCacheLocalBytes   int64 = 256 << 20
	DefaultCacheSharedPrefix       = "trenova:keel:"
	DefaultBudgetMaxRepeats        = 10

	minDatabaseCacheLocalBytes int64 = 1 << 20
	tenantConcurrencyShare           = 4
)

type CacheConfig struct {
	Enabled    bool              `mapstructure:"enabled"`
	LocalBytes int64             `mapstructure:"localBytes" validate:"min=0"`
	ListenHost string            `mapstructure:"listenHost" validate:"omitempty,max=253"`
	ListenPort int               `mapstructure:"listenPort" validate:"omitempty,min=1,max=65535"`
	Shared     SharedCacheConfig `mapstructure:"shared"`
}

type SharedCacheConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Prefix  string `mapstructure:"prefix"  validate:"omitempty,max=64"`
}

type BudgetConfig struct {
	MaxRepeats int `mapstructure:"maxRepeats" validate:"omitempty,min=1,max=10000"`
}

func (c *CacheConfig) GetLocalBytes() int64 {
	if c.LocalBytes <= 0 {
		return DefaultCacheLocalBytes
	}
	return c.LocalBytes
}

func (c *CacheConfig) SharedEnabled() bool {
	return c.Enabled && c.Shared.Enabled
}

func (c *SharedCacheConfig) GetPrefix() string {
	if prefix := strings.TrimSpace(c.Prefix); prefix != "" {
		return prefix
	}
	return DefaultCacheSharedPrefix
}

func (c *BudgetConfig) GetMaxRepeats() int {
	if c.MaxRepeats <= 0 {
		return DefaultBudgetMaxRepeats
	}
	return c.MaxRepeats
}

func (c *Config) validateRuntime() error {
	if c.Cache.Shared.Enabled && !c.Cache.Enabled {
		return ErrSharedCacheNeedsCache
	}
	if c.Cache.ListenPort > 0 && strings.TrimSpace(c.Cache.ListenHost) == "" {
		return ErrCacheListenPortNeedsHost
	}
	if c.Cache.LocalBytes > 0 && c.Cache.LocalBytes < minDatabaseCacheLocalBytes {
		return ErrCacheLocalBytesTooSmall
	}
	if c.MaxOpenConns > 0 && c.TenantConcurrency > c.MaxOpenConns {
		return ErrTenantConcurrencyExceedsPool
	}

	return nil
}
