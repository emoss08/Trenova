package databaseconfig

import (
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
			c.Cache = CacheConfig{
				Enabled: true,
				Shared:  SharedCacheConfig{Enabled: true},
			}
		}},
		{name: "shared tier without the cache", mutate: func(c *Config) {
			c.Cache = CacheConfig{Shared: SharedCacheConfig{Enabled: true}}
		}, want: ErrSharedCacheNeedsCache},
		{name: "listen port without a host", mutate: func(c *Config) {
			c.Cache = CacheConfig{Enabled: true, ListenPort: 5432}
		}, want: ErrCacheListenPortNeedsHost},
		{name: "local cache under a mebibyte", mutate: func(c *Config) {
			c.Cache = CacheConfig{Enabled: true, LocalBytes: 4096}
		}, want: ErrCacheLocalBytesTooSmall},
		{name: "tenant concurrency within the pool", mutate: func(c *Config) {
			c.TenantConcurrency = c.MaxOpenConns
		}},
		{name: "tenant concurrency beyond the pool", mutate: func(c *Config) {
			c.TenantConcurrency = c.MaxOpenConns + 1
		}, want: ErrTenantConcurrencyExceedsPool},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := validDatabaseConfig()
			tc.mutate(cfg)

			err := cfg.validateRuntime()
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

	db := Config{}
	assert.Equal(t, 25, db.GetTenantConcurrency(100))
	assert.Equal(t, 1, db.GetTenantConcurrency(3))
	assert.Equal(t, 0, db.GetTenantConcurrency(0))

	db.TenantConcurrency = 12
	assert.Equal(t, 12, db.GetTenantConcurrency(100))
	assert.Equal(t, 8, db.GetTenantConcurrency(8))
}

func TestDatabaseCacheConfig_Defaults(t *testing.T) {
	t.Parallel()

	var cache CacheConfig
	assert.Equal(t, DefaultCacheLocalBytes, cache.GetLocalBytes())
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

	var budget BudgetConfig
	assert.Equal(t, 10, budget.GetMaxRepeats())
	budget.MaxRepeats = 3
	assert.Equal(t, 3, budget.GetMaxRepeats())
}
