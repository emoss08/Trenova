//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func cacheKey(userID, orgID pulid.ID) permissionports.PermissionCacheKey {
	return permissionports.PermissionCacheKey{
		UserID:  userID,
		OrgID:   orgID,
		Variant: permissionports.PermissionCacheVariantAllRoles,
	}
}

func setCurrent(
	t *testing.T,
	repo *permissionCacheRepository,
	key permissionports.PermissionCacheKey,
	perms *permissionports.CachedPermissions,
	ttl time.Duration,
) {
	t.Helper()

	epoch, err := repo.Epoch(t.Context(), key.UserID, key.OrgID)
	require.NoError(t, err)
	epoch.Stamp(perms)
	require.NoError(t, repo.Set(t.Context(), key, perms, ttl))
}

func setupTestCacheRepository(t *testing.T) (*permissionCacheRepository, *redis.Client, func()) {
	t.Helper()

	client := testutil.SetupTestRedis(t)
	logger := zap.NewNop()

	repo := &permissionCacheRepository{
		client: client,
		l:      logger.Named("test.permission-cache-repository"),
		prefix: permissionCachePrefix + ":test",
	}

	cleanup := func() {
		client.FlushDB(t.Context())
	}

	return repo, client, cleanup
}

func TestPermissionCacheRepository_SetAndGet_Integration(t *testing.T) {
	repo, _, cleanup := setupTestCacheRepository(t)
	defer cleanup()

	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	perms := &permissionports.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityRestricted),
		Resources: map[string]*permissionports.CachedResourcePermission{
			"shipment": {
				Operations: []string{
					string(permission.OpRead),
					string(permission.OpCreate),
					string(permission.OpUpdate),
				},
				DataScope: string(permission.DataScopeOrganization),
			},
			"driver": {
				Operations: []string{string(permission.OpRead)},
				DataScope:  string(permission.DataScopeOrganization),
			},
		},
	}

	setCurrent(t, repo, cacheKey(userID, orgID), perms, 5*time.Minute)

	retrieved, err := repo.Get(ctx, cacheKey(userID, orgID))
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, perms.MaxSensitivity, retrieved.MaxSensitivity)
	assert.Len(t, retrieved.Resources, 2)
	assert.Contains(t, retrieved.Resources, "shipment")
	assert.Contains(t, retrieved.Resources, "driver")
}

func TestPermissionCacheRepository_GetNonExistent_Integration(t *testing.T) {
	repo, _, cleanup := setupTestCacheRepository(t)
	defer cleanup()

	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	retrieved, err := repo.Get(ctx, cacheKey(userID, orgID))
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestPermissionCacheRepository_Delete_Integration(t *testing.T) {
	repo, _, cleanup := setupTestCacheRepository(t)
	defer cleanup()

	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	perms := &permissionports.CachedPermissions{
		Resources: map[string]*permissionports.CachedResourcePermission{},
	}

	setCurrent(t, repo, cacheKey(userID, orgID), perms, 5*time.Minute)

	retrieved, err := repo.Get(ctx, cacheKey(userID, orgID))
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	err = repo.Delete(ctx, userID, orgID)
	require.NoError(t, err)

	retrieved, err = repo.Get(ctx, cacheKey(userID, orgID))
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestPermissionCacheRepository_InvalidateOrganization_Integration(t *testing.T) {
	repo, client, cleanup := setupTestCacheRepository(t)
	defer cleanup()

	ctx := t.Context()
	orgID := pulid.MustNew("org_")

	user1ID := pulid.MustNew("usr_")
	user2ID := pulid.MustNew("usr_")
	user3ID := pulid.MustNew("usr_")
	otherOrgID := pulid.MustNew("org_")

	perms := &permissionports.CachedPermissions{
		Resources: map[string]*permissionports.CachedResourcePermission{},
	}

	setCurrent(t, repo, cacheKey(user1ID, orgID), perms, 5*time.Minute)
	setCurrent(t, repo, cacheKey(user2ID, orgID), perms, 5*time.Minute)
	setCurrent(t, repo, cacheKey(user3ID, otherOrgID), perms, 5*time.Minute)

	keys, err := client.Keys(ctx, permissionCachePrefix+":*").Result()
	require.NoError(t, err)
	assert.Len(t, keys, 3)

	err = repo.InvalidateOrganization(ctx, orgID)
	require.NoError(t, err)

	r1, _ := repo.Get(ctx, cacheKey(user1ID, orgID))
	r2, _ := repo.Get(ctx, cacheKey(user2ID, orgID))
	r3, _ := repo.Get(ctx, cacheKey(user3ID, otherOrgID))

	assert.Nil(t, r1)
	assert.Nil(t, r2)
	assert.NotNil(t, r3)
}

func TestPermissionCacheRepository_ServesOnlyTheCurrentEpoch_Integration(t *testing.T) {
	repo, client, cleanup := setupTestCacheRepository(t)
	defer cleanup()

	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	stale := &permissionports.CachedPermissions{
		Resources: map[string]*permissionports.CachedResourcePermission{},
	}
	setCurrent(t, repo, cacheKey(userID, orgID), stale, 5*time.Minute)

	require.NoError(t, repo.InvalidateOrganization(ctx, orgID))
	retrieved, err := repo.Get(ctx, cacheKey(userID, orgID))
	require.NoError(t, err)
	assert.Nil(t, retrieved, "an entry computed before the change is not served")

	fresh := &permissionports.CachedPermissions{
		Resources: map[string]*permissionports.CachedResourcePermission{},
	}
	setCurrent(t, repo, cacheKey(userID, orgID), fresh, 5*time.Minute)
	retrieved, err = repo.Get(ctx, cacheKey(userID, orgID))
	require.NoError(t, err)
	assert.NotNil(t, retrieved)

	require.NoError(t, client.Del(ctx, repo.getEpochKey(orgID)).Err())
	retrieved, err = repo.Get(ctx, cacheKey(userID, orgID))
	require.NoError(t, err)
	assert.Nil(t, retrieved, "without an epoch nothing cached is trusted")

	epoch, err := repo.Epoch(ctx, userID, orgID)
	require.NoError(t, err)
	assert.NotEqual(
		t,
		fresh.Epoch,
		epoch.Organization,
		"a reseeded epoch never matches an entry from before",
	)
}

// A check that read the epoch before a user's roles changed, and computed
// their grants from the roles as they were, finishes after the change has
// cleared the cache. What it writes back is the old grants, and it must not
// be served for the rest of the entry's life.
func TestPermissionCacheRepository_RefusesGrantsComputedBeforeAUserChanged_Integration(
	t *testing.T,
) {
	repo, _, cleanup := setupTestCacheRepository(t)
	defer cleanup()

	ctx := t.Context()
	key := cacheKey(pulid.MustNew("usr_"), pulid.MustNew("org_"))

	before, err := repo.Epoch(ctx, key.UserID, key.OrgID)
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, key.UserID, key.OrgID))

	stale := &permissionports.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityRestricted),
		Resources:      map[string]*permissionports.CachedResourcePermission{},
	}
	before.Stamp(stale)
	require.NoError(t, repo.Set(ctx, key, stale, 5*time.Minute))

	retrieved, err := repo.Get(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, retrieved, "grants computed before the change are not served")

	fresh := &permissionports.CachedPermissions{
		Resources: map[string]*permissionports.CachedResourcePermission{},
	}
	setCurrent(t, repo, key, fresh, 5*time.Minute)
	retrieved, err = repo.Get(ctx, key)
	require.NoError(t, err)
	assert.NotNil(t, retrieved, "grants computed after the change are")

	other := cacheKey(pulid.MustNew("usr_"), key.OrgID)
	setCurrent(t, repo, other, &permissionports.CachedPermissions{
		Resources: map[string]*permissionports.CachedResourcePermission{},
	}, 5*time.Minute)
	require.NoError(t, repo.Delete(ctx, key.UserID, key.OrgID))
	retrieved, err = repo.Get(ctx, other)
	require.NoError(t, err)
	assert.NotNil(t, retrieved, "one user's change leaves the others' entries alone")
}

func TestPermissionCacheRepository_TTL_Integration(t *testing.T) {
	repo, _, cleanup := setupTestCacheRepository(t)
	defer cleanup()

	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	perms := &permissionports.CachedPermissions{
		Resources: map[string]*permissionports.CachedResourcePermission{},
	}

	setCurrent(t, repo, cacheKey(userID, orgID), perms, 1*time.Second)

	retrieved, err := repo.Get(ctx, cacheKey(userID, orgID))
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	time.Sleep(2 * time.Second)

	retrieved, err = repo.Get(ctx, cacheKey(userID, orgID))
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestPermissionCacheRepository_VariantsAreIsolated_Integration(t *testing.T) {
	repo, _, cleanup := setupTestCacheRepository(t)
	defer cleanup()

	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	allRoles := cacheKey(userID, orgID)
	activeSubset := permissionports.PermissionCacheKey{
		UserID:  userID,
		OrgID:   orgID,
		Variant: "roles:abc",
	}

	setCurrent(t, repo, allRoles, &permissionports.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityRestricted),
		Resources:      map[string]*permissionports.CachedResourcePermission{},
	}, 5*time.Minute)
	setCurrent(t, repo, activeSubset, &permissionports.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityPublic),
		Resources:      map[string]*permissionports.CachedResourcePermission{},
	}, 5*time.Minute)

	all, err := repo.Get(ctx, allRoles)
	require.NoError(t, err)
	require.NotNil(t, all)
	assert.Equal(t, string(permission.SensitivityRestricted), all.MaxSensitivity)

	subset, err := repo.Get(ctx, activeSubset)
	require.NoError(t, err)
	require.NotNil(t, subset)
	assert.Equal(t, string(permission.SensitivityPublic), subset.MaxSensitivity)

	missing, err := repo.Get(ctx, permissionports.PermissionCacheKey{
		UserID:  userID,
		OrgID:   orgID,
		Variant: "roles:other",
	})
	require.NoError(t, err)
	assert.Nil(t, missing)

	require.NoError(t, repo.Delete(ctx, userID, orgID))

	all, err = repo.Get(ctx, allRoles)
	require.NoError(t, err)
	assert.Nil(t, all)
	subset, err = repo.Get(ctx, activeSubset)
	require.NoError(t, err)
	assert.Nil(t, subset)
}
