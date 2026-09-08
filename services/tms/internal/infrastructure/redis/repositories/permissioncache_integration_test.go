//go:build integration

package repositories

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func cacheKey(userID, orgID pulid.ID) repositories.PermissionCacheKey {
	return repositories.PermissionCacheKey{
		UserID:  userID,
		OrgID:   orgID,
		Variant: repositories.PermissionCacheVariantAllRoles,
	}
}

func setupTestCacheRepository(t *testing.T) (*permissionCacheRepository, *redis.Client, func()) {
	t.Helper()

	client := testutil.SetupTestRedis(t)
	logger := zap.NewNop()

	repo := &permissionCacheRepository{
		client: client,
		l:      logger.Named("test.permission-cache-repository"),
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

	perms := &repositories.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityRestricted),
		Resources: map[string]*repositories.CachedResourcePermission{
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

	err := repo.Set(ctx, cacheKey(userID, orgID), perms, 5*time.Minute)
	require.NoError(t, err)

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

	perms := &repositories.CachedPermissions{
		Resources: map[string]*repositories.CachedResourcePermission{},
	}

	err := repo.Set(ctx, cacheKey(userID, orgID), perms, 5*time.Minute)
	require.NoError(t, err)

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

	perms := &repositories.CachedPermissions{
		Resources: map[string]*repositories.CachedResourcePermission{},
	}

	require.NoError(t, repo.Set(ctx, cacheKey(user1ID, orgID), perms, 5*time.Minute))
	require.NoError(t, repo.Set(ctx, cacheKey(user2ID, orgID), perms, 5*time.Minute))
	require.NoError(t, repo.Set(ctx, cacheKey(user3ID, otherOrgID), perms, 5*time.Minute))

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

func TestPermissionCacheRepository_InvalidateByRole_Integration(t *testing.T) {
	repo, _, cleanup := setupTestCacheRepository(t)
	defer cleanup()

	ctx := t.Context()
	roleID := pulid.MustNew("rol_")
	orgID := pulid.MustNew("org_")

	user1ID := pulid.MustNew("usr_")
	user2ID := pulid.MustNew("usr_")
	user3ID := pulid.MustNew("usr_")

	perms := &repositories.CachedPermissions{
		Resources: map[string]*repositories.CachedResourcePermission{},
	}

	require.NoError(t, repo.Set(ctx, cacheKey(user1ID, orgID), perms, 5*time.Minute))
	require.NoError(t, repo.Set(ctx, cacheKey(user2ID, orgID), perms, 5*time.Minute))
	require.NoError(t, repo.Set(ctx, cacheKey(user3ID, orgID), perms, 5*time.Minute))

	mockRoleRepo := mocks.NewMockRoleRepository(t)
	mockRoleRepo.EXPECT().
		GetUsersWithRole(mock.Anything, roleID).
		Return(
			[]repositories.ImpactedUser{
				{UserID: user1ID, OrganizationID: orgID},
				{UserID: user2ID, OrganizationID: orgID},
			},
			nil,
		)

	err := repo.InvalidateByRole(ctx, roleID, mockRoleRepo)
	require.NoError(t, err)

	r1, _ := repo.Get(ctx, cacheKey(user1ID, orgID))
	r2, _ := repo.Get(ctx, cacheKey(user2ID, orgID))
	r3, _ := repo.Get(ctx, cacheKey(user3ID, orgID))

	assert.Nil(t, r1)
	assert.Nil(t, r2)
	assert.NotNil(t, r3)
}

func TestPermissionCacheRepository_TTL_Integration(t *testing.T) {
	repo, _, cleanup := setupTestCacheRepository(t)
	defer cleanup()

	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	perms := &repositories.CachedPermissions{
		Resources: map[string]*repositories.CachedResourcePermission{},
	}

	err := repo.Set(ctx, cacheKey(userID, orgID), perms, 1*time.Second)
	require.NoError(t, err)

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
	activeSubset := repositories.PermissionCacheKey{
		UserID:  userID,
		OrgID:   orgID,
		Variant: "roles:abc",
	}

	require.NoError(t, repo.Set(ctx, allRoles, &repositories.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityRestricted),
		Resources:      map[string]*repositories.CachedResourcePermission{},
	}, 5*time.Minute))
	require.NoError(t, repo.Set(ctx, activeSubset, &repositories.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityPublic),
		Resources:      map[string]*repositories.CachedResourcePermission{},
	}, 5*time.Minute))

	all, err := repo.Get(ctx, allRoles)
	require.NoError(t, err)
	require.NotNil(t, all)
	assert.Equal(t, string(permission.SensitivityRestricted), all.MaxSensitivity)

	subset, err := repo.Get(ctx, activeSubset)
	require.NoError(t, err)
	require.NotNil(t, subset)
	assert.Equal(t, string(permission.SensitivityPublic), subset.MaxSensitivity)

	missing, err := repo.Get(ctx, repositories.PermissionCacheKey{
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
