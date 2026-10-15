package permission

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/apikey"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/apikeyportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func creatorHolds(grants map[string][]string) *permissionports.CachedPermissions {
	resources := make(map[string]*permissionports.CachedResourcePermission, len(grants))
	for resource, ops := range grants {
		resources[resource] = &permissionports.CachedResourcePermission{
			Operations: ops,
			DataScope:  string(permission.DataScopeOrganization),
		}
	}
	return &permissionports.CachedPermissions{
		Resources: resources,
		ExpiresAt: timeutils.NowUnix() + 3600,
	}
}

func expectAPIKeyCreator(
	users *tenantportsmock.MockUserRepository,
	cache *permissionportsmock.MockPermissionCacheRepository,
	held *permissionports.CachedPermissions,
) pulid.ID {
	creatorID := pulid.MustNew("usr_")
	users.On("GetTenantMember", mock.Anything, mock.Anything).
		Return(&tenant.User{ID: creatorID, Status: domaintypes.StatusActive}, nil)
	cache.On("Get", mock.Anything, mock.MatchedBy(func(key permissionports.PermissionCacheKey) bool {
		return key.UserID == creatorID
	})).Return(held, nil)
	return creatorID
}

type apiKeyCase struct {
	eng   *engine
	keys  *apikeyportsmock.MockAPIKeyRepository
	users *tenantportsmock.MockUserRepository
	cache *permissionportsmock.MockPermissionCacheRepository
	keyID pulid.ID
	orgID pulid.ID
	buID  pulid.ID
}

func newAPIKeyCase(t *testing.T) *apiKeyCase {
	t.Helper()

	eng, _, cache, users := setupTestEngine(t)
	keys := apikeyportsmock.NewMockAPIKeyRepository(t)
	eng.apiKeyRepo = keys
	return &apiKeyCase{
		eng:   eng,
		keys:  keys,
		users: users,
		cache: cache,
		keyID: pulid.MustNew("ak_"),
		orgID: pulid.MustNew("org_"),
		buID:  pulid.MustNew("bu_"),
	}
}

func (c *apiKeyCase) withKey(creatorID pulid.ID, scope permission.DataScope, ops ...permission.Operation) {
	c.keys.EXPECT().GetByID(mock.Anything, mock.Anything, c.keyID).Return(&apikey.Key{
		ID:          c.keyID,
		CreatedByID: creatorID,
		Permissions: []*apikey.Permission{{
			Resource:   "customer",
			Operations: ops,
			DataScope:  scope,
		}},
	}, nil)
}

func (c *apiKeyCase) check(t *testing.T, op permission.Operation) *permissionports.PermissionCheckResult {
	t.Helper()

	result, err := c.eng.Check(t.Context(), &permissionports.PermissionCheckRequest{
		PrincipalType:  permissionports.PrincipalTypeAPIKey,
		PrincipalID:    c.keyID,
		APIKeyID:       c.keyID,
		BusinessUnitID: c.buID,
		OrganizationID: c.orgID,
		Resource:       "customer",
		Operation:      op,
	})
	require.NoError(t, err)
	return result
}

func TestAPIKey_GrantsOnlyWhatItsCreatorStillHolds(t *testing.T) {
	t.Parallel()

	c := newAPIKeyCase(t)
	creatorID := expectAPIKeyCreator(c.users, c.cache, creatorHolds(map[string][]string{
		"customer": {"read"},
	}))
	c.withKey(creatorID, permission.DataScopeOrganization, permission.OpRead, permission.OpUpdate)

	assert.True(t, c.check(t, permission.OpRead).Allowed)
	assert.False(t, c.check(t, permission.OpUpdate).Allowed)
}

func TestAPIKey_NarrowsToTheCreatorsScope(t *testing.T) {
	t.Parallel()

	c := newAPIKeyCase(t)
	held := creatorHolds(map[string][]string{"customer": {"read"}})
	held.Resources["customer"].DataScope = string(permission.DataScopeOwn)
	creatorID := expectAPIKeyCreator(c.users, c.cache, held)
	c.withKey(creatorID, permission.DataScopeAll, permission.OpRead)

	result := c.check(t, permission.OpRead)

	assert.True(t, result.Allowed)
	assert.Equal(t, permission.DataScopeOwn, result.DataScope)
}

func TestAPIKey_GrantsNothingOnceItsCreatorCannotSignIn(t *testing.T) {
	t.Parallel()

	c := newAPIKeyCase(t)
	creatorID := pulid.MustNew("usr_")
	c.users.On("GetTenantMember", mock.Anything, mock.Anything).
		Return(&tenant.User{ID: creatorID, Status: domaintypes.StatusInactive}, nil)
	c.withKey(creatorID, permission.DataScopeOrganization, permission.OpRead)

	assert.False(t, c.check(t, permission.OpRead).Allowed)
}

func TestAPIKey_GrantsNothingOnceItsCreatorLeftTheOrganization(t *testing.T) {
	t.Parallel()

	c := newAPIKeyCase(t)
	c.users.On("GetTenantMember", mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("User not found"))
	c.withKey(pulid.MustNew("usr_"), permission.DataScopeOrganization, permission.OpRead)

	assert.False(t, c.check(t, permission.OpRead).Allowed)
}
