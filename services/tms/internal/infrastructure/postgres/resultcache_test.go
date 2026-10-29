package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func cacheTenant() dbscope.Tenant {
	return dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
	}
}

func withGrants(ctx context.Context, grants readscope.Grants) context.Context {
	return readscope.With(ctx, func(context.Context) (readscope.Grants, error) {
		return grants, nil
	})
}

func TestReadScopeCacheKeyIsEmptyForAnOrganizationWideRead(t *testing.T) {
	t.Parallel()

	ctx := dbscope.WithTenant(t.Context(), cacheTenant())
	assert.Empty(t, readScopeCacheKey(ctx))
	assert.Empty(t, readScopeCacheKey(withGrants(ctx, readscope.Grants{})))
}

func TestReadScopeCacheKeySeparatesUsersOfANarrowedScope(t *testing.T) {
	t.Parallel()

	own := readscope.Grants{"shipment": {Scope: readscope.ScopeOwn}}
	first, second := cacheTenant(), cacheTenant()
	second.OrganizationID, second.BusinessUnitID = first.OrganizationID, first.BusinessUnitID

	firstKey := readScopeCacheKey(withGrants(dbscope.WithTenant(t.Context(), first), own))
	secondKey := readScopeCacheKey(withGrants(dbscope.WithTenant(t.Context(), second), own))
	againKey := readScopeCacheKey(withGrants(dbscope.WithTenant(t.Context(), first), own))

	assert.Equal(t, "shipment:own|user:"+first.UserID.String(), firstKey)
	assert.NotEqual(
		t,
		firstKey,
		secondKey,
		"two people who read only their own rows never share an entry",
	)
	assert.Equal(t, firstKey, againKey)
}

func TestReadScopeCacheKeyNeverMatchesAResolvedScopeWhenResolutionFails(t *testing.T) {
	t.Parallel()

	ctx := readscope.With(
		dbscope.WithTenant(t.Context(), cacheTenant()),
		func(context.Context) (readscope.Grants, error) {
			return nil, errors.New("permission engine unavailable")
		},
	)

	key := readScopeCacheKey(ctx)
	assert.True(t, strings.HasPrefix(key, unresolvedReadScopeKey), key)
	assert.Contains(t, key, "permission engine unavailable")
}

func TestResultCacheFollowsTheConfiguration(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.Database.Host, cfg.Database.Port, cfg.Database.Name = "db.internal", 6432, "trenova"
	cfg.Database.User, cfg.Database.Password, cfg.Database.SSLMode = "trenova_app", "secret", "disable"
	cfg.Database.Cache = databaseconfig.CacheConfig{Enabled: true}
	conn := &Connection{cfg: cfg, logger: observability.NewContextLogger(zap.NewNop())}

	cache, err := conn.resultCache()
	require.NoError(t, err)
	assert.NotNil(t, cache.Local)
	assert.NotNil(t, cache.Key)
	assert.NotNil(t, cache.OnError)
	assert.Nil(t, cache.Listen, "without a listen host the listener uses the pool's settings")
	assert.Nil(t, cache.Shared, "the Redis tier is off unless configured")

	cfg.Database.Cache.ListenHost = "pg-primary.internal"
	cfg.Database.Cache.ListenPort = 5432
	cache, err = conn.resultCache()
	require.NoError(t, err)
	require.NotNil(t, cache.Listen)
	assert.Equal(t, "pg-primary.internal", cache.Listen.Host)
	assert.Equal(t, uint16(5432), cache.Listen.Port)

	cfg.Database.Cache.Shared.Enabled = true
	_, err = conn.resultCache()
	require.ErrorIs(t, err, ErrSharedResultCacheNeedsRedis)
}

func TestCacheErrorLogThrottlesARunOfFailures(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zapcore.WarnLevel)
	log := newCacheErrorLog(observability.NewContextLogger(zap.New(core)))

	for range 5 {
		log.report(errors.New("redis: connection refused"))
	}

	entries := logs.All()
	require.Len(t, entries, 1, "a failing store logs once per interval, not once per read")
	assert.Equal(t, int64(0), entries[0].ContextMap()["suppressed_since_last"])
	assert.Equal(t, 4, log.suppressed)
}
