package config

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDatabaseRuntimeConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*Config)
		want   error
	}{
		{name: "defaults pass", mutate: func(*Config) {}},
		{name: "local cache with a shared tier", mutate: func(c *Config) {
			c.Database.Cache = DatabaseCacheConfig{
				Enabled: true,
				Shared:  DatabaseSharedCacheConfig{Enabled: true},
			}
		}},
		{name: "shared tier without the cache", mutate: func(c *Config) {
			c.Database.Cache = DatabaseCacheConfig{Shared: DatabaseSharedCacheConfig{Enabled: true}}
		}, want: ErrDatabaseSharedCacheNeedsCache},
		{name: "listen port without a host", mutate: func(c *Config) {
			c.Database.Cache = DatabaseCacheConfig{Enabled: true, ListenPort: 5432}
		}, want: ErrDatabaseCacheListenPortNeedsHost},
		{name: "local cache under a mebibyte", mutate: func(c *Config) {
			c.Database.Cache = DatabaseCacheConfig{Enabled: true, LocalBytes: 4096}
		}, want: ErrDatabaseCacheLocalBytesTooSmall},
		{name: "tenant concurrency within the pool", mutate: func(c *Config) {
			c.Database.TenantConcurrency = c.Database.MaxOpenConns
		}},
		{name: "tenant concurrency beyond the pool", mutate: func(c *Config) {
			c.Database.TenantConcurrency = c.Database.MaxOpenConns + 1
		}, want: ErrDatabaseTenantConcurrencyExceedsPool},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := newValidConfig()
			tc.mutate(cfg)

			err := validateDatabaseRuntimeConfig(cfg)
			if tc.want == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestDatabaseConfig_GetTenantConcurrency(t *testing.T) {
	t.Parallel()

	db := DatabaseConfig{}
	assert.Equal(t, 25, db.GetTenantConcurrency(100))
	assert.Equal(t, 1, db.GetTenantConcurrency(3))
	assert.Equal(t, 0, db.GetTenantConcurrency(0))

	db.TenantConcurrency = 12
	assert.Equal(t, 12, db.GetTenantConcurrency(100))
	assert.Equal(t, 8, db.GetTenantConcurrency(8))
}

func TestDatabaseCacheConfig_Defaults(t *testing.T) {
	t.Parallel()

	var cache DatabaseCacheConfig
	assert.Equal(t, DefaultDatabaseCacheLocalBytes, cache.GetLocalBytes())
	assert.Equal(t, "trenova:keel:", cache.Shared.GetPrefix())
	assert.False(t, cache.SharedEnabled())

	cache.Shared.Enabled = true
	assert.False(t, cache.SharedEnabled(), "the shared tier needs the cache")
	cache.Enabled = true
	assert.True(t, cache.SharedEnabled())

	cache.LocalBytes = 64 << 20
	cache.Shared.Prefix = " tenant-cache: "
	assert.Equal(t, int64(64<<20), cache.GetLocalBytes())
	assert.Equal(t, "tenant-cache:", cache.Shared.GetPrefix())

	var budget DatabaseBudgetConfig
	assert.Equal(t, 10, budget.GetMaxRepeats())
	budget.MaxRepeats = 3
	assert.Equal(t, 3, budget.GetMaxRepeats())
}

func TestConfig_GetCacheListenDSN(t *testing.T) {
	t.Parallel()

	cfg := newValidConfig()
	assert.Empty(t, cfg.GetCacheListenDSN(), "no listen host keeps the pool's settings")

	cfg.Database.Cache.ListenHost = "pg-primary.internal"
	parsed, err := url.Parse(cfg.GetCacheListenDSN())
	require.NoError(t, err)
	assert.Equal(t, "pg-primary.internal:5432", parsed.Host)
	assert.Equal(t, "/trenova", parsed.Path)
	assert.Equal(t, "postgres", parsed.User.Username())

	cfg.Database.Cache.ListenPort = 5433
	parsed, err = url.Parse(cfg.GetCacheListenDSN())
	require.NoError(t, err)
	assert.Equal(t, "pg-primary.internal:5433", parsed.Host)
	assert.Equal(t, "disable", parsed.Query().Get("sslmode"))
}
