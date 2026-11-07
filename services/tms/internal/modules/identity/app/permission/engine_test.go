package permission

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/monitoringconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/identity/domain/apikey"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/domain/routeaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/mocks/apikeyportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/iamportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/rbactest"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func allRolesKey(userID, orgID pulid.ID) permissionports.PermissionCacheKey {
	return permissionports.PermissionCacheKey{
		UserID:  userID,
		OrgID:   orgID,
		Variant: permissionports.PermissionCacheVariantAllRoles,
	}
}

func activeRolesKey(
	ctx context.Context,
	userID, orgID pulid.ID,
) permissionports.PermissionCacheKey {
	return permissionports.PermissionCacheKey{
		UserID:  userID,
		OrgID:   orgID,
		Variant: permissionCacheVariant(ctx),
	}
}

const testEpoch int64 = 7

func setupTestEngine(
	t *testing.T,
) (*engine, *permissionportsmock.MockRoleRepository, *permissionportsmock.MockPermissionCacheRepository, *tenantportsmock.MockUserRepository) {
	t.Helper()

	roleRepo := permissionportsmock.NewMockRoleRepository(t)
	roleRepo.On("GetMembershipExpiry", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil).
		Maybe()
	cacheRepo := permissionportsmock.NewMockPermissionCacheRepository(t)
	cacheRepo.On("Epoch", mock.Anything, mock.Anything, mock.Anything).
		Return(testEpoch, nil).
		Maybe()
	userRepo := tenantportsmock.NewMockUserRepository(t)
	logger := zap.NewNop()

	metricsRegistry, err := metrics.NewRegistry(&monitoringconfig.Config{}, logger)
	require.NoError(t, err)

	noPolicies := iamportsmock.NewMockIAMRepository(t)
	noPolicies.EXPECT().
		ListEnabledTenantAccessPolicies(mock.Anything, mock.Anything).
		Return(nil, nil).
		Maybe()

	e := &engine{
		roleRepo:          roleRepo,
		rbacRepo:          &rbactest.Repository{},
		cacheRepo:         cacheRepo,
		userRepo:          userRepo,
		iamRepo:           noPolicies,
		agentGrants:       &fakeAgentGrants{},
		settings:          &stubAccessSettings{},
		registry:          permissioncatalog.NewRegistry(),
		routeRegistry:     routeaccess.NewRouteRegistry(),
		databaseEnforcing: true,
		metrics:           metricsRegistry,
		l:                 logger.Named("test.permission-engine"),
	}

	return e, roleRepo, cacheRepo, userRepo
}

func TestCheck_AllowedByPermission(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, userRepo := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).Return([]*rbac.Role{
		{
			ID:             roleID,
			Name:           "Driver Manager",
			MaxSensitivity: permission.SensitivityInternal,
			Permissions: []*rbac.ResourcePermission{
				{
					ID:         pulid.MustNew("rp_"),
					RoleID:     roleID,
					Resource:   "shipment",
					Operations: []permission.Operation{permission.OpRead, permission.OpCreate},
					DataScope:  permission.DataScopeOrganization,
				},
			},
		},
	}, nil)
	cacheRepo.On("Set", ctx, allRolesKey(userID, orgID), mock.AnythingOfType("*permissionports.CachedPermissions"), cacheTTL).
		Return(nil)

	result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
		UserID:         userID,
		OrganizationID: orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, "allowed", result.Reason)
	assert.Equal(t, permission.DataScopeOrganization, result.DataScope)

	userRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	cacheRepo.AssertExpectations(t)
}

func TestCheck_NoPermission(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, userRepo := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).Return([]*rbac.Role{
		{
			ID:             roleID,
			Name:           "Reader",
			MaxSensitivity: permission.SensitivityPublic,
			Permissions: []*rbac.ResourcePermission{
				{
					ID:         pulid.MustNew("rp_"),
					RoleID:     roleID,
					Resource:   "customer",
					Operations: []permission.Operation{permission.OpRead},
					DataScope:  permission.DataScopeOwn,
				},
			},
		},
	}, nil)
	cacheRepo.On("Set", ctx, allRolesKey(userID, orgID), mock.AnythingOfType("*permissionports.CachedPermissions"), cacheTTL).
		Return(nil)

	result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
		UserID:         userID,
		OrganizationID: orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})

	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, "no_permission", result.Reason)

	userRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	cacheRepo.AssertExpectations(t)
}

func TestCheck_CacheHit(t *testing.T) {
	t.Parallel()

	eng, _, cacheRepo, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(&permissionports.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityInternal),
		Resources: map[string]*permissionports.CachedResourcePermission{
			"shipment": {
				Operations: []string{"read", "create"},
				DataScope:  string(permission.DataScopeOrganization),
			},
		},
		ExpiresAt: timeutils.NowUnix() + 3600,
	}, nil)

	result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
		UserID:         userID,
		OrganizationID: orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.True(t, result.CacheHit)

	cacheRepo.AssertExpectations(t)
}

func TestCheck_EnforcesResourceAttributes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		dataScope         permission.DataScope
		attrs             permissionports.ResourceAttributes
		contextAttributes permissionports.RequestContextAttributes
		wantReason        string
	}{
		{
			name:       "owner scope denies mismatched owner",
			dataScope:  permission.DataScopeOwn,
			attrs:      permissionports.ResourceAttributes{OwnerID: pulid.MustNew("usr_")},
			wantReason: "abac_owner_scope",
		},
		{
			name:       "organization mismatch denies globally",
			dataScope:  permission.DataScopeOrganization,
			attrs:      permissionports.ResourceAttributes{OrganizationID: pulid.MustNew("org_")},
			wantReason: "abac_organization_mismatch",
		},
		{
			name:       "business unit mismatch denies globally",
			dataScope:  permission.DataScopeBusinessUnit,
			attrs:      permissionports.ResourceAttributes{BusinessUnitID: pulid.MustNew("bu_")},
			wantReason: "abac_business_unit_mismatch",
		},
		{
			name:       "active role requirement denies when inactive",
			dataScope:  permission.DataScopeOrganization,
			attrs:      permissionports.ResourceAttributes{ActiveRoleID: pulid.MustNew("rol_")},
			wantReason: "abac_active_role_required",
		},
		{
			name:      "risk deny blocks request",
			dataScope: permission.DataScopeOrganization,
			attrs:     permissionports.ResourceAttributes{},
			contextAttributes: permissionports.RequestContextAttributes{
				RiskDecision: "deny",
			},
			wantReason: "abac_risk_denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			eng, _, cacheRepo, _ := setupTestEngine(t)
			ctx := t.Context()
			userID := pulid.MustNew("usr_")
			orgID := pulid.MustNew("org_")
			buID := pulid.MustNew("bu_")

			cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).
				Return(&permissionports.CachedPermissions{
					MaxSensitivity: string(permission.SensitivityInternal),
					Resources: map[string]*permissionports.CachedResourcePermission{
						"shipment": {
							Operations: []string{"read"},
							DataScope:  string(tt.dataScope),
						},
					},
					ExpiresAt: timeutils.NowUnix() + 3600,
				}, nil)

			result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
				UserID:             userID,
				BusinessUnitID:     buID,
				OrganizationID:     orgID,
				Resource:           "shipment",
				Operation:          permission.OpRead,
				ResourceAttributes: tt.attrs,
				ContextAttributes:  tt.contextAttributes,
			})

			require.NoError(t, err)
			assert.False(t, result.Allowed)
			assert.Equal(t, tt.wantReason, result.Reason)
			assert.Equal(t, tt.dataScope, result.DataScope)
			cacheRepo.AssertExpectations(t)
		})
	}
}

func TestCheckBatch_PropagatesResourceAttributes(t *testing.T) {
	t.Parallel()

	eng, _, cacheRepo, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	activeRoleID := pulid.MustNew("rol_")
	inactiveRoleID := pulid.MustNew("rol_")

	cachedPermissions := &permissionports.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityInternal),
		Resources: map[string]*permissionports.CachedResourcePermission{
			"shipment": {
				Operations: []string{"read"},
				DataScope:  string(permission.DataScopeOrganization),
			},
		},
		ExpiresAt: timeutils.NowUnix() + 3600,
	}
	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(cachedPermissions, nil).Once()

	result, err := eng.CheckBatch(ctx, &permissionports.BatchPermissionCheckRequest{
		UserID:         userID,
		BusinessUnitID: buID,
		OrganizationID: orgID,
		ContextAttributes: permissionports.RequestContextAttributes{
			ActiveRoleIDs: []pulid.ID{activeRoleID},
		},
		Checks: []permissionports.ResourceOperationCheck{
			{
				Resource:  "shipment",
				Operation: permission.OpRead,
				ResourceAttributes: permissionports.ResourceAttributes{
					ActiveRoleID: activeRoleID,
				},
			},
			{
				Resource:  "shipment",
				Operation: permission.OpRead,
				ResourceAttributes: permissionports.ResourceAttributes{
					ActiveRoleID: inactiveRoleID,
				},
			},
		},
	})

	require.NoError(t, err)
	require.Len(t, result.Results, 2)
	assert.True(t, result.Results[0].Allowed)
	assert.False(t, result.Results[1].Allowed)
	assert.Equal(t, "abac_active_role_required", result.Results[1].Reason)
	cacheRepo.AssertExpectations(t)
}

func TestCheck_NoRoles(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, userRepo := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{}, nil)
	cacheRepo.On("Set", ctx, allRolesKey(userID, orgID), mock.AnythingOfType("*permissionports.CachedPermissions"), cacheTTL).
		Return(nil)

	result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
		UserID:         userID,
		OrganizationID: orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})

	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, "no_permission", result.Reason)

	userRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	cacheRepo.AssertExpectations(t)
}

func TestCheckBatch(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(nil, nil).Once()
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{}, nil).
		Once()
	cacheRepo.On("Set", ctx, allRolesKey(userID, orgID), mock.AnythingOfType("*permissionports.CachedPermissions"), cacheTTL).
		Return(nil).
		Once()

	result, err := eng.CheckBatch(ctx, &permissionports.BatchPermissionCheckRequest{
		UserID:         userID,
		OrganizationID: orgID,
		Checks: []permissionports.ResourceOperationCheck{
			{Resource: "shipment", Operation: permission.OpRead},
			{Resource: "customer", Operation: permission.OpCreate},
		},
	})

	require.NoError(t, err)
	assert.Len(t, result.Results, 2)
	assert.False(t, result.Results[0].Allowed)
	assert.False(t, result.Results[1].Allowed)

	cacheRepo.AssertExpectations(t)
}

func TestGetLightManifest_RegularUser(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, userRepo := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	userRepo.On("GetUserOrganizationSummaries", ctx, userID).Return([]tenantports.OrgSummary{
		{ID: orgID, Name: "Test Org"},
	}, nil)
	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).Return([]*rbac.Role{
		{
			ID:             roleID,
			Name:           "Dispatcher",
			MaxSensitivity: permission.SensitivityRestricted,
			Permissions: []*rbac.ResourcePermission{
				{
					ID:         pulid.MustNew("rp_"),
					RoleID:     roleID,
					Resource:   "shipment",
					Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
					DataScope:  permission.DataScopeOrganization,
				},
			},
		},
	}, nil)
	cacheRepo.On("Set", ctx, allRolesKey(userID, orgID), mock.AnythingOfType("*permissionports.CachedPermissions"), cacheTTL).
		Return(nil)
	manifest, err := eng.GetLightManifest(ctx, userID, orgID)

	require.NoError(t, err)
	assert.Equal(t, permission.SensitivityRestricted, manifest.MaxSensitivity)
	assert.Contains(t, manifest.Permissions, "shipment")
	assert.NotEmpty(t, manifest.Checksum)

	userRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	cacheRepo.AssertExpectations(t)
}

func TestGetLightManifest_IncludesAuthorizedRolesWhenActivationRequired(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, userRepo := setupTestEngine(t)
	ctx := authctx.WithSessionRoleActivation(t.Context(), []pulid.ID{}, true)
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")
	eng.rbacRepo = &rbactest.Repository{
		AuthorizedRoles: []*rbac.Role{
			{
				ID:          roleID,
				Name:        "Dispatcher",
				Description: "Coordinates loads",
			},
		},
	}

	userRepo.On("GetUserOrganizationSummaries", ctx, userID).Return([]tenantports.OrgSummary{
		{ID: orgID, Name: "Test Org"},
	}, nil)
	cacheRepo.On("Get", ctx, activeRolesKey(ctx, userID, orgID)).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	cacheRepo.On("Set", ctx, activeRolesKey(ctx, userID, orgID), mock.MatchedBy(func(perms *permissionports.CachedPermissions) bool {
		return len(perms.Resources) == 0
	}), cacheTTL).
		Return(nil)

	manifest, err := eng.GetLightManifest(ctx, userID, orgID)

	require.NoError(t, err)
	assert.True(t, manifest.RequiresRoleActivation)
	assert.Equal(t, []pulid.ID{roleID}, manifest.AuthorizedRoleIDs)
	assert.Empty(t, manifest.ActiveRoleIDs)
	require.Len(t, manifest.AuthorizedRoles, 1)
	assert.Equal(t, "Dispatcher", manifest.AuthorizedRoles[0].Name)
	assert.Empty(t, manifest.ActiveRoles)

	userRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	cacheRepo.AssertExpectations(t)
}

func TestGetLightManifest_DoesNotRequireRoleActivationWithoutAuthorizedRoles(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, userRepo := setupTestEngine(t)
	ctx := authctx.WithSessionRoleActivation(t.Context(), []pulid.ID{}, true)
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	userRepo.On("GetUserOrganizationSummaries", ctx, userID).Return([]tenantports.OrgSummary{
		{ID: orgID, Name: "Test Org"},
	}, nil)
	cacheRepo.On("Get", ctx, activeRolesKey(ctx, userID, orgID)).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{}, nil)
	cacheRepo.On("Set", ctx, activeRolesKey(ctx, userID, orgID), mock.AnythingOfType("*permissionports.CachedPermissions"), cacheTTL).
		Return(nil)

	manifest, err := eng.GetLightManifest(ctx, userID, orgID)

	require.NoError(t, err)
	assert.False(t, manifest.RequiresRoleActivation)
	assert.Empty(t, manifest.AuthorizedRoleIDs)
	assert.Empty(t, manifest.ActiveRoleIDs)
	assert.Empty(t, manifest.AuthorizedRoles)
	assert.Empty(t, manifest.ActiveRoles)

	userRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	cacheRepo.AssertExpectations(t)
}

func TestInvalidateUser(t *testing.T) {
	t.Parallel()

	eng, _, cacheRepo, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	cacheRepo.On("Delete", ctx, userID, orgID).Return(nil)

	err := eng.InvalidateUser(ctx, userID, orgID)

	require.NoError(t, err)
	cacheRepo.AssertExpectations(t)
}

func TestGetEffectivePermissions(t *testing.T) {
	t.Parallel()

	eng, roleRepo, _, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).Return([]*rbac.Role{
		{
			ID:             roleID,
			Name:           "Manager",
			IsSystem:       true,
			MaxSensitivity: permission.SensitivityRestricted,
			Permissions: []*rbac.ResourcePermission{
				{
					ID:       pulid.MustNew("rp_"),
					RoleID:   roleID,
					Resource: "worker",
					Operations: []permission.Operation{
						permission.OpRead,
						permission.OpCreate,
						permission.OpUpdate,
					},
					DataScope: permission.DataScopeOrganization,
				},
			},
		},
	}, nil)

	result, err := eng.GetEffectivePermissions(ctx, userID, orgID)

	require.NoError(t, err)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, orgID, result.OrganizationID)
	assert.Len(t, result.Roles, 1)
	assert.Equal(t, "Manager", result.Roles[0].Name)
	assert.True(t, result.Roles[0].IsSystem)
	assert.Equal(t, permission.SensitivityRestricted, result.MaxSensitivity)
	assert.Contains(t, result.Resources, "worker")
	assert.Contains(t, result.Resources["worker"].GrantedBy, "Manager")

	roleRepo.AssertExpectations(t)
}

func TestGetResourcePermissions_UnknownResource(t *testing.T) {
	t.Parallel()

	eng, _, _, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	result, err := eng.GetResourcePermissions(ctx, userID, orgID, "unknown_resource")

	require.NoError(t, err)
	assert.Nil(t, result)

}

func TestCachedPermissionsEndWithTheEarliestAssignment(t *testing.T) {
	t.Parallel()

	eng, roleRepo, _, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")
	soon := timeutils.NowUnix() + 60
	later := timeutils.NowUnix() + 300
	past := timeutils.NowUnix() - 60

	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{RoleID: roleID, UserID: userID, OrganizationID: orgID, ExpiresAt: &later},
			{RoleID: roleID, UserID: userID, OrganizationID: orgID, ExpiresAt: &soon},
			{
				RoleID:         pulid.MustNew("rol_"),
				UserID:         userID,
				OrganizationID: orgID,
				ExpiresAt:      &past,
			},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).Return([]*rbac.Role{
		{ID: roleID, MaxSensitivity: permission.SensitivityInternal},
	}, nil)

	perms, err := eng.computePermissions(ctx, userID, orgID)

	require.NoError(t, err)
	assert.Equal(t, soon, perms.ExpiresAt)
}

func TestPermissionsExpireAt_KeepsTheCacheLifetimeWithoutAnEarlierExpiry(t *testing.T) {
	t.Parallel()

	far := timeutils.NowUnix() + 2*int64(cacheTTL.Seconds())
	before := timeutils.NowUnix()

	expiresAt := permissionsExpireAt([]*rbac.UserRoleAssignment{{}, {ExpiresAt: &far}}, nil)

	assert.GreaterOrEqual(t, expiresAt, before+int64(cacheTTL.Seconds()))
	assert.Less(t, expiresAt, far)
}

func membershipEnding(
	t *testing.T,
	eng *engine,
	userID, orgID, roleID pulid.ID,
	ends int64,
) *permissionportsmock.MockRoleRepository {
	t.Helper()

	roles := permissionportsmock.NewMockRoleRepository(t)
	eng.roleRepo = roles
	roles.On("GetUserRoleAssignments", mock.Anything, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roles.On("GetMembershipExpiry", mock.Anything, userID, orgID).Return(&ends, nil)

	return roles
}

func TestCachedPermissionsEndWithTheMembership(t *testing.T) {
	t.Parallel()

	eng, _, _, _ := setupTestEngine(t)
	userID, orgID, roleID := pulid.MustNew("usr_"), pulid.MustNew("org_"), pulid.MustNew("rol_")
	ends := timeutils.NowUnix() + 60
	roles := membershipEnding(t, eng, userID, orgID, roleID, ends)
	roles.On("GetRolesWithInheritance", mock.Anything, []pulid.ID{roleID}).Return([]*rbac.Role{
		{ID: roleID, MaxSensitivity: permission.SensitivityInternal},
	}, nil)

	perms, err := eng.computePermissions(t.Context(), userID, orgID)

	require.NoError(t, err)
	assert.Equal(t, ends, perms.ExpiresAt)
	assert.Equal(t, string(permission.SensitivityInternal), perms.MaxSensitivity)
}

func TestAnEndedMembershipGrantsNothing(t *testing.T) {
	t.Parallel()

	eng, _, _, _ := setupTestEngine(t)
	userID, orgID, roleID := pulid.MustNew("usr_"), pulid.MustNew("org_"), pulid.MustNew("rol_")
	roles := membershipEnding(t, eng, userID, orgID, roleID, timeutils.NowUnix()-60)
	roles.On("GetRolesWithInheritance", mock.Anything, []pulid.ID{}).
		Return([]*rbac.Role{}, nil)

	perms, err := eng.computePermissions(t.Context(), userID, orgID)

	require.NoError(t, err)
	assert.Empty(t, perms.Resources)
	assert.Equal(t, string(permission.SensitivityPublic), perms.MaxSensitivity)
	assert.Greater(t, perms.ExpiresAt, timeutils.NowUnix())

	result, err := eng.GetEffectivePermissions(t.Context(), userID, orgID)

	require.NoError(t, err)
	assert.Empty(t, result.Resources)
	roles.AssertNotCalled(t, "GetRolesWithInheritance", mock.Anything, []pulid.ID{roleID})
}

func TestExpiredAssignmentsIgnored(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")
	expiredRoleID := pulid.MustNew("rol_")

	expiredTime := timeutils.NowUnix() - 3600

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
			{
				ID:             pulid.MustNew("ura_"),
				RoleID:         expiredRoleID,
				UserID:         userID,
				OrganizationID: orgID,
				ExpiresAt:      &expiredTime,
			},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).Return([]*rbac.Role{
		{
			ID:             roleID,
			Name:           "Active Role",
			MaxSensitivity: permission.SensitivityInternal,
			Permissions:    []*rbac.ResourcePermission{},
		},
	}, nil)
	cacheRepo.On("Set", ctx, allRolesKey(userID, orgID), mock.AnythingOfType("*permissionports.CachedPermissions"), cacheTTL).
		Return(nil)

	result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
		UserID:         userID,
		OrganizationID: orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})

	require.NoError(t, err)
	assert.False(t, result.Allowed)

	roleRepo.AssertCalled(t, "GetRolesWithInheritance", ctx, []pulid.ID{roleID})
}

func TestMultipleRolesMergePermissions(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, userRepo := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	role1ID := pulid.MustNew("rol_")
	role2ID := pulid.MustNew("rol_")

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: role1ID, UserID: userID, OrganizationID: orgID},
			{ID: pulid.MustNew("ura_"), RoleID: role2ID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{role1ID, role2ID}).
		Return([]*rbac.Role{
			{
				ID:             role1ID,
				Name:           "Reader",
				MaxSensitivity: permission.SensitivityInternal,
				Permissions: []*rbac.ResourcePermission{
					{
						ID:         pulid.MustNew("rp_"),
						RoleID:     role1ID,
						Resource:   "shipment",
						Operations: []permission.Operation{permission.OpRead},
						DataScope:  permission.DataScopeOwn,
					},
				},
			},
			{
				ID:             role2ID,
				Name:           "Editor",
				MaxSensitivity: permission.SensitivityRestricted,
				Permissions: []*rbac.ResourcePermission{
					{
						ID:         pulid.MustNew("rp_"),
						RoleID:     role2ID,
						Resource:   "shipment",
						Operations: []permission.Operation{permission.OpUpdate},
						DataScope:  permission.DataScopeOrganization,
					},
				},
			},
		}, nil)
	cacheRepo.On("Set", ctx, allRolesKey(userID, orgID), mock.MatchedBy(func(perms *permissionports.CachedPermissions) bool {
		rp, ok := perms.Resources["shipment"]
		if !ok {
			return false
		}
		hasRead := false
		hasUpdate := false
		for _, op := range rp.Operations {
			if op == "read" {
				hasRead = true
			}
			if op == "update" {
				hasUpdate = true
			}
		}
		return hasRead && hasUpdate && rp.DataScope == "organization" &&
			perms.MaxSensitivity == "restricted"
	}), cacheTTL).
		Return(nil)

	result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
		UserID:         userID,
		OrganizationID: orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, permission.DataScopeOrganization, result.DataScope)

	userRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	cacheRepo.AssertExpectations(t)
}

func TestCheck_OperationNotAllowed(t *testing.T) {
	t.Parallel()

	eng, _, cacheRepo, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(&permissionports.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityInternal),
		Resources: map[string]*permissionports.CachedResourcePermission{
			"shipment": {
				Operations: []string{"read"},
				DataScope:  string(permission.DataScopeOrganization),
			},
		},
		ExpiresAt: timeutils.NowUnix() + 3600,
	}, nil)

	result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
		UserID:         userID,
		OrganizationID: orgID,
		Resource:       "shipment",
		Operation:      permission.OpDelete,
	})

	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, "no_permission", result.Reason)
	cacheRepo.AssertExpectations(t)
}

func TestCheckBatch_Error(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return(nil, errors.New("role lookup error"))

	result, err := eng.CheckBatch(ctx, &permissionports.BatchPermissionCheckRequest{
		UserID:         userID,
		OrganizationID: orgID,
		Checks: []permissionports.ResourceOperationCheck{
			{Resource: "shipment", Operation: permission.OpRead},
		},
	})

	require.Error(t, err)
	assert.Nil(t, result)
	cacheRepo.AssertExpectations(t)
}

func TestGetLightManifest_OrgSummariesError(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, userRepo := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{}, nil)
	cacheRepo.On("Set", ctx, allRolesKey(userID, orgID), mock.AnythingOfType("*permissionports.CachedPermissions"), cacheTTL).
		Return(nil)
	userRepo.On("GetUserOrganizationSummaries", ctx, userID).
		Return(nil, errors.New("summaries error"))

	manifest, err := eng.GetLightManifest(ctx, userID, orgID)

	require.Error(t, err)
	assert.Nil(t, manifest)
	userRepo.AssertExpectations(t)
	cacheRepo.AssertExpectations(t)
}

func TestGetResourcePermissions_RegularUser(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, userRepo := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).Return([]*rbac.Role{
		{
			ID:             roleID,
			Name:           "Viewer",
			MaxSensitivity: permission.SensitivityInternal,
			Permissions: []*rbac.ResourcePermission{
				{
					ID:         pulid.MustNew("rp_"),
					RoleID:     roleID,
					Resource:   "shipment",
					Operations: []permission.Operation{permission.OpRead},
					DataScope:  permission.DataScopeOwn,
				},
			},
		},
	}, nil)
	cacheRepo.On("Set", ctx, allRolesKey(userID, orgID), mock.AnythingOfType("*permissionports.CachedPermissions"), cacheTTL).
		Return(nil)

	result, err := eng.GetResourcePermissions(ctx, userID, orgID, "shipment")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "shipment", result.Resource)
	assert.Equal(t, permission.DataScopeOwn, result.DataScope)
	assert.Contains(t, result.Operations, permission.OpRead)
	userRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	cacheRepo.AssertExpectations(t)
}

func TestGetResourcePermissionsBatch_ReadsThePermissionsOnce(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(nil, nil).Once()
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil).Once()
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).Return([]*rbac.Role{
		{
			ID:             roleID,
			Name:           "Viewer",
			MaxSensitivity: permission.SensitivityInternal,
			Permissions: []*rbac.ResourcePermission{
				{
					ID:         pulid.MustNew("rp_"),
					RoleID:     roleID,
					Resource:   "shipment",
					Operations: []permission.Operation{permission.OpRead},
					DataScope:  permission.DataScopeOwn,
				},
			},
		},
	}, nil).Once()
	cacheRepo.On("Set", ctx, allRolesKey(userID, orgID), mock.AnythingOfType("*permissionports.CachedPermissions"), cacheTTL).
		Return(nil).
		Once()

	details, err := eng.GetResourcePermissionsBatch(
		ctx, userID, orgID, []string{"shipment", "customer", "shipment", "not_a_resource"},
	)

	require.NoError(t, err)
	require.Len(t, details, 2, "each registered resource once; an unknown one has no entry")
	assert.Contains(t, details["shipment"].Operations, permission.OpRead)
	assert.Equal(t, permission.DataScopeOwn, details["shipment"].DataScope)
	assert.Empty(t, details["customer"].Operations)
	assert.Equal(t, permission.SensitivityInternal, details["customer"].MaxSensitivity)
	roleRepo.AssertExpectations(t)
	cacheRepo.AssertExpectations(t)
}

func TestGetResourcePermissions_NoPermissionForResource(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, userRepo := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).Return([]*rbac.Role{
		{
			ID:             roleID,
			Name:           "Viewer",
			MaxSensitivity: permission.SensitivityPublic,
			Permissions: []*rbac.ResourcePermission{
				{
					ID:         pulid.MustNew("rp_"),
					RoleID:     roleID,
					Resource:   "customer",
					Operations: []permission.Operation{permission.OpRead},
					DataScope:  permission.DataScopeOwn,
				},
			},
		},
	}, nil)
	cacheRepo.On("Set", ctx, allRolesKey(userID, orgID), mock.AnythingOfType("*permissionports.CachedPermissions"), cacheTTL).
		Return(nil)

	result, err := eng.GetResourcePermissions(ctx, userID, orgID, "shipment")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "shipment", result.Resource)
	assert.Empty(t, result.Operations)
	assert.Empty(t, result.DataScope)
	userRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	cacheRepo.AssertExpectations(t)
}

func TestGetEffectivePermissions_CountsOnlyActivatedRoles(t *testing.T) {
	t.Parallel()

	eng, roleRepo, _, _ := setupTestEngine(t)
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	activeID := pulid.MustNew("rol_")
	idleID := pulid.MustNew("rol_")
	ctx := authctx.WithSessionRoleActivation(t.Context(), []pulid.ID{activeID}, true)

	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{RoleID: activeID, UserID: userID, OrganizationID: orgID},
			{RoleID: idleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{activeID}).Return([]*rbac.Role{
		{ID: activeID, MaxSensitivity: permission.SensitivityInternal},
	}, nil)

	result, err := eng.GetEffectivePermissions(ctx, userID, orgID)

	require.NoError(t, err)
	assert.Equal(t, permission.SensitivityInternal, result.MaxSensitivity)
	roleRepo.AssertNotCalled(
		t,
		"GetRolesWithInheritance",
		ctx,
		[]pulid.ID{activeID, idleID},
	)
}

func TestGetEffectivePermissions_WithoutActivationCountsEveryRole(t *testing.T) {
	t.Parallel()

	eng, roleRepo, _, _ := setupTestEngine(t)
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	firstID := pulid.MustNew("rol_")
	secondID := pulid.MustNew("rol_")
	bound := authctx.WithSessionRoleActivation(t.Context(), []pulid.ID{firstID}, true)
	ctx := authctx.WithoutSessionRoleActivation(bound)

	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{RoleID: firstID, UserID: userID, OrganizationID: orgID},
			{RoleID: secondID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{firstID, secondID}).
		Return([]*rbac.Role{}, nil).
		Once()

	_, err := eng.GetEffectivePermissions(ctx, userID, orgID)

	require.NoError(t, err)
}

func TestGetEffectivePermissions_Error(t *testing.T) {
	t.Parallel()

	eng, roleRepo, _, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).Return(nil, errors.New("db error"))

	result, err := eng.GetEffectivePermissions(ctx, userID, orgID)

	require.Error(t, err)
	assert.Nil(t, result)
	roleRepo.AssertExpectations(t)
}

func TestGetEffectivePermissions_GetRolesError(t *testing.T) {
	t.Parallel()

	eng, roleRepo, _, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).
		Return(nil, errors.New("roles error"))

	result, err := eng.GetEffectivePermissions(ctx, userID, orgID)

	require.Error(t, err)
	assert.Nil(t, result)
	roleRepo.AssertExpectations(t)
}

func TestInvalidateUser_Error(t *testing.T) {
	t.Parallel()

	eng, _, cacheRepo, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	cacheRepo.On("Delete", ctx, userID, orgID).Return(errors.New("cache error"))

	err := eng.InvalidateUser(ctx, userID, orgID)

	require.Error(t, err)
	assert.Equal(t, "cache error", err.Error())
	cacheRepo.AssertExpectations(t)
}

func TestGetEffectivePermissions_ExpiredAssignmentsSkipped(t *testing.T) {
	t.Parallel()

	eng, roleRepo, _, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	activeRoleID := pulid.MustNew("rol_")
	expiredRoleID := pulid.MustNew("rol_")

	expiredTime := timeutils.NowUnix() - 3600

	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{
				ID:             pulid.MustNew("ura_"),
				RoleID:         activeRoleID,
				UserID:         userID,
				OrganizationID: orgID,
			},
			{
				ID:             pulid.MustNew("ura_"),
				RoleID:         expiredRoleID,
				UserID:         userID,
				OrganizationID: orgID,
				ExpiresAt:      &expiredTime,
			},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{activeRoleID}).Return([]*rbac.Role{
		{
			ID:             activeRoleID,
			Name:           "Active",
			MaxSensitivity: permission.SensitivityInternal,
			Permissions:    []*rbac.ResourcePermission{},
		},
	}, nil)

	result, err := eng.GetEffectivePermissions(ctx, userID, orgID)

	require.NoError(t, err)
	assert.Len(t, result.Roles, 1)
	assert.Equal(t, "Active", result.Roles[0].Name)
	roleRepo.AssertExpectations(t)
}

func TestPermissionCacheVariant(t *testing.T) {
	t.Parallel()

	roleA := pulid.MustNew("rol_")
	roleB := pulid.MustNew("rol_")

	assert.Equal(
		t,
		permissionports.PermissionCacheVariantAllRoles,
		permissionCacheVariant(t.Context()),
	)

	ab := permissionCacheVariant(
		authctx.WithSessionRoleActivation(t.Context(), []pulid.ID{roleA, roleB}, false),
	)
	ba := permissionCacheVariant(
		authctx.WithSessionRoleActivation(t.Context(), []pulid.ID{roleB, roleA}, false),
	)
	aab := permissionCacheVariant(
		authctx.WithSessionRoleActivation(t.Context(), []pulid.ID{roleA, roleA, roleB}, false),
	)
	a := permissionCacheVariant(
		authctx.WithSessionRoleActivation(t.Context(), []pulid.ID{roleA}, false),
	)
	none := permissionCacheVariant(
		authctx.WithSessionRoleActivation(t.Context(), []pulid.ID{}, true),
	)

	assert.Equal(t, ab, ba)
	assert.Equal(t, ab, aab)
	assert.NotEqual(t, ab, a)
	assert.NotEqual(t, a, none)
	assert.NotEqual(t, permissionports.PermissionCacheVariantAllRoles, none)
}

func TestCheck_WithRoleActivationUsesCache(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, _ := setupTestEngine(t)
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")
	ctx := authctx.WithSessionRoleActivation(t.Context(), []pulid.ID{roleID}, false)

	cacheRepo.On("Get", ctx, activeRolesKey(ctx, userID, orgID)).
		Return(&permissionports.CachedPermissions{
			MaxSensitivity: string(permission.SensitivityInternal),
			Resources: map[string]*permissionports.CachedResourcePermission{
				"shipment": {
					Operations: []string{"read"},
					DataScope:  string(permission.DataScopeOrganization),
				},
			},
			ExpiresAt: timeutils.NowUnix() + 3600,
		}, nil).
		Once()

	result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
		UserID:         userID,
		OrganizationID: orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.True(t, result.CacheHit)
	roleRepo.AssertNotCalled(
		t,
		"GetUserRoleAssignments",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
	cacheRepo.AssertExpectations(t)
}

func TestCheck_WithRoleActivationCachesActiveSubset(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, _ := setupTestEngine(t)
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	activeRoleID := pulid.MustNew("rol_")
	inactiveRoleID := pulid.MustNew("rol_")
	ctx := authctx.WithSessionRoleActivation(t.Context(), []pulid.ID{activeRoleID}, false)

	cacheRepo.On("Get", ctx, activeRolesKey(ctx, userID, orgID)).Return(nil, nil).Once()
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{
				ID:             pulid.MustNew("ura_"),
				RoleID:         activeRoleID,
				UserID:         userID,
				OrganizationID: orgID,
			},
			{
				ID:             pulid.MustNew("ura_"),
				RoleID:         inactiveRoleID,
				UserID:         userID,
				OrganizationID: orgID,
			},
		}, nil).Once()
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{activeRoleID}).Return([]*rbac.Role{
		{
			ID:             activeRoleID,
			Name:           "Dispatcher",
			MaxSensitivity: permission.SensitivityInternal,
			Permissions: []*rbac.ResourcePermission{
				{
					ID:         pulid.MustNew("rp_"),
					RoleID:     activeRoleID,
					Resource:   "shipment",
					Operations: []permission.Operation{permission.OpRead},
					DataScope:  permission.DataScopeOrganization,
				},
			},
		},
	}, nil).Once()
	cacheRepo.On("Set", ctx, activeRolesKey(ctx, userID, orgID), mock.MatchedBy(func(perms *permissionports.CachedPermissions) bool {
		rp, ok := perms.Resources["shipment"]
		return ok && len(rp.Operations) == 1 && rp.Operations[0] == "read"
	}), cacheTTL).
		Return(nil).
		Once()

	result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
		UserID:         userID,
		OrganizationID: orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.False(t, result.CacheHit)
	roleRepo.AssertExpectations(t)
	cacheRepo.AssertExpectations(t)
}

func TestCheckBatch_LoadsPermissionsAndPoliciesOnce(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, _ := setupTestEngine(t)
	iamRepo := iamportsmock.NewMockIAMRepository(t)
	eng.iamRepo = iamRepo
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(&permissionports.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityInternal),
		Resources: map[string]*permissionports.CachedResourcePermission{
			"shipment": {
				Operations: []string{"read", "create"},
				DataScope:  string(permission.DataScopeOrganization),
			},
			"customer": {
				Operations: []string{"create"},
				DataScope:  string(permission.DataScopeOrganization),
			},
		},
		ExpiresAt: timeutils.NowUnix() + 3600,
	}, nil).Once()
	iamRepo.EXPECT().
		ListEnabledTenantAccessPolicies(ctx, iamports.IAMTenantPolicyLookupRequest{
			OrganizationID: orgID,
			BusinessUnitID: buID,
		}).
		Return([]*iam.AccessPolicy{
			{
				ID:             pulid.MustNew("ap_"),
				OrganizationID: orgID,
				BusinessUnitID: buID,
				Resource:       "customer",
				Operation:      string(permission.OpCreate),
				Effect:         iam.PolicyEffectDeny,
				Priority:       10,
				Enabled:        true,
			},
			{
				ID:             pulid.MustNew("ap_"),
				OrganizationID: orgID,
				BusinessUnitID: buID,
				Resource:       "shipment",
				Operation:      string(permission.OpRead),
				Effect:         iam.PolicyEffectAllow,
				Priority:       5,
				Enabled:        true,
			},
		}, nil).
		Once()

	result, err := eng.CheckBatch(ctx, &permissionports.BatchPermissionCheckRequest{
		UserID:         userID,
		BusinessUnitID: buID,
		OrganizationID: orgID,
		Checks: []permissionports.ResourceOperationCheck{
			{Resource: "shipment", Operation: permission.OpRead},
			{Resource: "shipment", Operation: permission.OpCreate},
			{Resource: "customer", Operation: permission.OpCreate},
			{Resource: "worker", Operation: permission.OpRead},
		},
	})

	require.NoError(t, err)
	require.Len(t, result.Results, 4)
	assert.True(t, result.CacheHit)
	assert.True(t, result.Results[0].Allowed)
	assert.True(t, result.Results[1].Allowed)
	assert.False(t, result.Results[2].Allowed)
	assert.Equal(t, "iam_policy_denied", result.Results[2].Reason)
	assert.False(t, result.Results[3].Allowed)
	assert.Equal(t, "no_permission", result.Results[3].Reason)

	roleRepo.AssertNotCalled(
		t,
		"GetUserRoleAssignments",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
	iamRepo.AssertNotCalled(t, "ListEnabledAccessPolicies", mock.Anything, mock.Anything)
	cacheRepo.AssertExpectations(t)
	iamRepo.AssertExpectations(t)
}

func TestCheckBatch_SkipsPolicyLookupWhenNothingIsAllowed(t *testing.T) {
	t.Parallel()

	eng, _, cacheRepo, _ := setupTestEngine(t)
	iamRepo := iamportsmock.NewMockIAMRepository(t)
	eng.iamRepo = iamRepo
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(&permissionports.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityPublic),
		Resources:      map[string]*permissionports.CachedResourcePermission{},
		ExpiresAt:      timeutils.NowUnix() + 3600,
	}, nil).Once()

	result, err := eng.CheckBatch(ctx, &permissionports.BatchPermissionCheckRequest{
		UserID:         userID,
		OrganizationID: orgID,
		Checks: []permissionports.ResourceOperationCheck{
			{Resource: "shipment", Operation: permission.OpRead},
			{Resource: "customer", Operation: permission.OpCreate},
		},
	})

	require.NoError(t, err)
	require.Len(t, result.Results, 2)
	iamRepo.AssertNotCalled(t, "ListEnabledTenantAccessPolicies", mock.Anything, mock.Anything)
	cacheRepo.AssertExpectations(t)
}

func TestCheck_UsesTenantPolicyLookup(t *testing.T) {
	t.Parallel()

	eng, _, cacheRepo, _ := setupTestEngine(t)
	iamRepo := iamportsmock.NewMockIAMRepository(t)
	eng.iamRepo = iamRepo
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(&permissionports.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityInternal),
		Resources: map[string]*permissionports.CachedResourcePermission{
			"shipment": {
				Operations: []string{"read"},
				DataScope:  string(permission.DataScopeOrganization),
			},
		},
		ExpiresAt: timeutils.NowUnix() + 3600,
	}, nil).Once()
	iamRepo.EXPECT().
		ListEnabledTenantAccessPolicies(ctx, iamports.IAMTenantPolicyLookupRequest{
			OrganizationID: orgID,
			BusinessUnitID: buID,
		}).
		Return([]*iam.AccessPolicy{
			{
				ID:        pulid.MustNew("ap_"),
				Resource:  "shipment",
				Operation: string(permission.OpRead),
				Effect:    iam.PolicyEffectDeny,
				Enabled:   true,
			},
		}, nil).
		Once()

	result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
		UserID:         userID,
		BusinessUnitID: buID,
		OrganizationID: orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})

	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, "iam_policy_denied", result.Reason)
	iamRepo.AssertNotCalled(t, "ListEnabledAccessPolicies", mock.Anything, mock.Anything)
	cacheRepo.AssertExpectations(t)
	iamRepo.AssertExpectations(t)
}

func TestCheckBatch_APIKeyLoadsKeyOnce(t *testing.T) {
	t.Parallel()

	eng, _, cacheRepo, userRepo := setupTestEngine(t)
	apiKeyRepo := apikeyportsmock.NewMockAPIKeyRepository(t)
	eng.apiKeyRepo = apiKeyRepo
	ctx := t.Context()
	apiKeyID := pulid.MustNew("ak_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	creatorID := expectAPIKeyCreator(userRepo, cacheRepo, creatorHolds(map[string][]string{
		"shipment": {"read", "create"},
		"customer": {"read"},
	}))

	apiKeyRepo.EXPECT().
		GetByID(ctx, pagination.TenantInfo{OrgID: orgID, BuID: buID}, apiKeyID).
		Return(&apikey.Key{
			ID:          apiKeyID,
			CreatedByID: creatorID,
			Permissions: []*apikey.Permission{
				{
					Resource:   "shipment",
					Operations: []permission.Operation{permission.OpRead},
					DataScope:  permission.DataScopeOrganization,
				},
			},
		}, nil).
		Once()

	result, err := eng.CheckBatch(ctx, &permissionports.BatchPermissionCheckRequest{
		PrincipalType:  permissionports.PrincipalTypeAPIKey,
		PrincipalID:    apiKeyID,
		APIKeyID:       apiKeyID,
		BusinessUnitID: buID,
		OrganizationID: orgID,
		Checks: []permissionports.ResourceOperationCheck{
			{Resource: "shipment", Operation: permission.OpRead},
			{Resource: "shipment", Operation: permission.OpCreate},
			{Resource: "customer", Operation: permission.OpRead},
		},
	})

	require.NoError(t, err)
	require.Len(t, result.Results, 3)
	assert.True(t, result.Results[0].Allowed)
	assert.False(t, result.Results[1].Allowed)
	assert.False(t, result.Results[2].Allowed)
	apiKeyRepo.AssertExpectations(t)
}

func TestCheckBatch_Empty(t *testing.T) {
	t.Parallel()

	eng, _, cacheRepo, _ := setupTestEngine(t)

	result, err := eng.CheckBatch(t.Context(), &permissionports.BatchPermissionCheckRequest{
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
	})

	require.NoError(t, err)
	assert.Empty(t, result.Results)
	assert.False(t, result.CacheHit)
	cacheRepo.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
}

func readOnlyShipmentPerms() *permissionports.CachedPermissions {
	return &permissionports.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityInternal),
		Resources: map[string]*permissionports.CachedResourcePermission{
			"shipment": {
				Operations: []string{"read"},
				DataScope:  string(permission.DataScopeOrganization),
			},
		},
		ExpiresAt: timeutils.NowUnix() + 3600,
	}
}

func TestCheck_AnEnabledDenyPolicyFromTheRepositoryDenies(t *testing.T) {
	t.Parallel()

	eng, _, cacheRepo, _ := setupTestEngine(t)
	iamRepo := iamportsmock.NewMockIAMRepository(t)
	eng.iamRepo = iamRepo
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	lookup := iamports.IAMTenantPolicyLookupRequest{OrganizationID: orgID, BusinessUnitID: buID}

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).
		Return(readOnlyShipmentPerms(), nil).
		Once()
	iamRepo.EXPECT().ListEnabledTenantAccessPolicies(ctx, lookup).Return([]*iam.AccessPolicy{
		{
			ID:        pulid.MustNew("ap_"),
			Resource:  "shipment",
			Operation: string(permission.OpRead),
			Effect:    iam.PolicyEffectDeny,
			Enabled:   true,
		},
	}, nil).Once()

	result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
		UserID:         userID,
		BusinessUnitID: buID,
		OrganizationID: orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})

	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, "iam_policy_denied", result.Reason)
}

func TestCheck_NoEnabledPoliciesLeaveTheRoleDecision(t *testing.T) {
	t.Parallel()

	eng, _, cacheRepo, _ := setupTestEngine(t)
	iamRepo := iamportsmock.NewMockIAMRepository(t)
	eng.iamRepo = iamRepo
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	lookup := iamports.IAMTenantPolicyLookupRequest{OrganizationID: orgID, BusinessUnitID: buID}

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).
		Return(readOnlyShipmentPerms(), nil).
		Once()
	iamRepo.EXPECT().
		ListEnabledTenantAccessPolicies(ctx, lookup).
		Return([]*iam.AccessPolicy{}, nil).
		Once()

	result, err := eng.Check(ctx, &permissionports.PermissionCheckRequest{
		UserID:         userID,
		BusinessUnitID: buID,
		OrganizationID: orgID,
		Resource:       "shipment",
		Operation:      permission.OpRead,
	})

	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestCheckBatch_ReadsTheAccessPoliciesOncePerBatch(t *testing.T) {
	t.Parallel()

	eng, _, cacheRepo, _ := setupTestEngine(t)
	iamRepo := iamportsmock.NewMockIAMRepository(t)
	eng.iamRepo = iamRepo
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	lookup := iamports.IAMTenantPolicyLookupRequest{OrganizationID: orgID, BusinessUnitID: buID}

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(&permissionports.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityInternal),
		Resources: map[string]*permissionports.CachedResourcePermission{
			"shipment": {
				Operations: []string{"read", "create", "update"},
				DataScope:  string(permission.DataScopeOrganization),
			},
		},
		ExpiresAt: timeutils.NowUnix() + 3600,
	}, nil).Once()
	iamRepo.EXPECT().
		ListEnabledTenantAccessPolicies(ctx, lookup).
		Return([]*iam.AccessPolicy{}, nil).
		Once()

	result, err := eng.CheckBatch(ctx, &permissionports.BatchPermissionCheckRequest{
		UserID:         userID,
		BusinessUnitID: buID,
		OrganizationID: orgID,
		Checks: []permissionports.ResourceOperationCheck{
			{Resource: "shipment", Operation: permission.OpRead},
			{Resource: "shipment", Operation: permission.OpCreate},
			{Resource: "shipment", Operation: permission.OpUpdate},
		},
	})

	require.NoError(t, err)
	require.Len(t, result.Results, 3)
	for _, r := range result.Results {
		assert.True(t, r.Allowed)
	}
}

func TestCheck_AllowsWhatAGrantImpliesAtEvaluation(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	cacheRepo.On("Get", ctx, allRolesKey(userID, orgID)).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).Return([]*rbac.Role{
		{
			ID:             roleID,
			Name:           "Billing approver",
			MaxSensitivity: permission.SensitivityInternal,
			Permissions: []*rbac.ResourcePermission{
				{
					ID:         pulid.MustNew("rp_"),
					RoleID:     roleID,
					Resource:   "invoice",
					Operations: []permission.Operation{permission.OpApprove},
					DataScope:  permission.DataScopeOrganization,
				},
			},
		},
	}, nil)
	cacheRepo.On("Set", ctx, allRolesKey(userID, orgID), mock.AnythingOfType("*permissionports.CachedPermissions"), cacheTTL).
		Return(nil)

	results, err := eng.CheckBatch(ctx, &permissionports.BatchPermissionCheckRequest{
		UserID:         userID,
		OrganizationID: orgID,
		Checks: []permissionports.ResourceOperationCheck{
			{Resource: "invoice", Operation: permission.OpApprove},
			{Resource: "invoice", Operation: permission.OpUpdate},
			{Resource: "invoice", Operation: permission.OpRead},
			{Resource: "invoice", Operation: permission.OpDelete},
		},
	})

	require.NoError(t, err)
	require.Len(t, results.Results, 4)
	assert.True(t, results.Results[0].Allowed, "approve is granted")
	assert.True(t, results.Results[1].Allowed, "approve implies update")
	assert.True(t, results.Results[2].Allowed, "approve implies read")
	assert.False(t, results.Results[3].Allowed, "approve does not imply delete")
}

func TestGetEffectivePermissions_ReportsImpliedOperations(t *testing.T) {
	t.Parallel()

	eng, roleRepo, _, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*rbac.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).Return([]*rbac.Role{
		{
			ID:             roleID,
			Name:           "Memory keeper",
			MaxSensitivity: permission.SensitivityInternal,
			Permissions: []*rbac.ResourcePermission{
				{
					ID:         pulid.MustNew("rp_"),
					RoleID:     roleID,
					Resource:   "agent_memory",
					Operations: []permission.Operation{permission.OpManage},
					DataScope:  permission.DataScopeOrganization,
				},
			},
		},
	}, nil)

	result, err := eng.GetEffectivePermissions(ctx, userID, orgID)

	require.NoError(t, err)
	assert.Equal(
		t,
		[]permission.Operation{
			permission.OpDelete,
			permission.OpManage,
			permission.OpRead,
			permission.OpUpdate,
		},
		result.Resources["agent_memory"].Operations,
	)
}
