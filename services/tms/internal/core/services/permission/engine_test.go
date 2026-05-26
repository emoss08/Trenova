package permission

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/rbactest"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupTestEngine(
	t *testing.T,
) (*engine, *mocks.MockRoleRepository, *mocks.MockPermissionCacheRepository, *mocks.MockUserRepository) {
	t.Helper()

	roleRepo := mocks.NewMockRoleRepository(t)
	cacheRepo := mocks.NewMockPermissionCacheRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	logger := zap.NewNop()

	e := &engine{
		roleRepo:      roleRepo,
		rbacRepo:      &rbactest.Repository{},
		cacheRepo:     cacheRepo,
		userRepo:      userRepo,
		registry:      permission.NewRegistry(),
		routeRegistry: permission.NewRouteRegistry(),
		l:             logger.Named("test.permission-engine"),
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

	cacheRepo.On("Get", ctx, userID, orgID).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*permission.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).Return([]*permission.Role{
		{
			ID:             roleID,
			Name:           "Driver Manager",
			MaxSensitivity: permission.SensitivityInternal,
			Permissions: []*permission.ResourcePermission{
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
	cacheRepo.On("Set", ctx, userID, orgID, mock.AnythingOfType("*repositories.CachedPermissions"), cacheTTL).
		Return(nil)

	result, err := eng.Check(ctx, &services.PermissionCheckRequest{
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

	cacheRepo.On("Get", ctx, userID, orgID).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*permission.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).Return([]*permission.Role{
		{
			ID:             roleID,
			Name:           "Reader",
			MaxSensitivity: permission.SensitivityPublic,
			Permissions: []*permission.ResourcePermission{
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
	cacheRepo.On("Set", ctx, userID, orgID, mock.AnythingOfType("*repositories.CachedPermissions"), cacheTTL).
		Return(nil)

	result, err := eng.Check(ctx, &services.PermissionCheckRequest{
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

	cacheRepo.On("Get", ctx, userID, orgID).Return(&repositories.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityInternal),
		Resources: map[string]*repositories.CachedResourcePermission{
			"shipment": {
				Operations: []string{"read", "create"},
				DataScope:  string(permission.DataScopeOrganization),
			},
		},
		ExpiresAt: timeutils.NowUnix() + 3600,
	}, nil)

	result, err := eng.Check(ctx, &services.PermissionCheckRequest{
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

func TestCheck_NoRoles(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, userRepo := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	cacheRepo.On("Get", ctx, userID, orgID).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*permission.UserRoleAssignment{}, nil)
	cacheRepo.On("Set", ctx, userID, orgID, mock.AnythingOfType("*repositories.CachedPermissions"), cacheTTL).
		Return(nil)

	result, err := eng.Check(ctx, &services.PermissionCheckRequest{
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

	cacheRepo.On("Get", ctx, userID, orgID).Return(nil, nil).Once()
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*permission.UserRoleAssignment{}, nil).
		Once()
	cacheRepo.On("Set", ctx, userID, orgID, mock.AnythingOfType("*repositories.CachedPermissions"), cacheTTL).
		Return(nil).
		Once()
	cacheRepo.On("Get", ctx, userID, orgID).Return(&repositories.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityConfidential),
		Resources:      map[string]*repositories.CachedResourcePermission{},
		ExpiresAt:      timeutils.NowUnix() + 3600,
	}, nil).Once()

	result, err := eng.CheckBatch(ctx, &services.BatchPermissionCheckRequest{
		UserID:         userID,
		OrganizationID: orgID,
		Checks: []services.ResourceOperationCheck{
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

	userRepo.On("GetUserOrganizationSummaries", ctx, userID).Return([]repositories.OrgSummary{
		{ID: orgID, Name: "Test Org"},
	}, nil)
	cacheRepo.On("Get", ctx, userID, orgID).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*permission.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).Return([]*permission.Role{
		{
			ID:             roleID,
			Name:           "Dispatcher",
			MaxSensitivity: permission.SensitivityRestricted,
			Permissions: []*permission.ResourcePermission{
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
	cacheRepo.On("Set", ctx, userID, orgID, mock.AnythingOfType("*repositories.CachedPermissions"), cacheTTL).
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
		AuthorizedRoles: []*permission.Role{
			{
				ID:          roleID,
				Name:        "Dispatcher",
				Description: "Coordinates loads",
			},
		},
	}

	userRepo.On("GetUserOrganizationSummaries", ctx, userID).Return([]repositories.OrgSummary{
		{ID: orgID, Name: "Test Org"},
	}, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*permission.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)

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
	cacheRepo.AssertNotCalled(t, "Get")
	cacheRepo.AssertNotCalled(t, "Set")
}

func TestGetLightManifest_DoesNotRequireRoleActivationWithoutAuthorizedRoles(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, userRepo := setupTestEngine(t)
	ctx := authctx.WithSessionRoleActivation(t.Context(), []pulid.ID{}, true)
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	userRepo.On("GetUserOrganizationSummaries", ctx, userID).Return([]repositories.OrgSummary{
		{ID: orgID, Name: "Test Org"},
	}, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*permission.UserRoleAssignment{}, nil)

	manifest, err := eng.GetLightManifest(ctx, userID, orgID)

	require.NoError(t, err)
	assert.False(t, manifest.RequiresRoleActivation)
	assert.Empty(t, manifest.AuthorizedRoleIDs)
	assert.Empty(t, manifest.ActiveRoleIDs)
	assert.Empty(t, manifest.AuthorizedRoles)
	assert.Empty(t, manifest.ActiveRoles)

	userRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	cacheRepo.AssertNotCalled(t, "Get")
	cacheRepo.AssertNotCalled(t, "Set")
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
		Return([]*permission.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).Return([]*permission.Role{
		{
			ID:             roleID,
			Name:           "Manager",
			IsSystem:       true,
			MaxSensitivity: permission.SensitivityRestricted,
			Permissions: []*permission.ResourcePermission{
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

func TestSimulatePermissions(t *testing.T) {
	t.Parallel()

	eng, roleRepo, _, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	existingRoleID := pulid.MustNew("rol_")
	newRoleID := pulid.MustNew("rol_")

	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*permission.UserRoleAssignment{
			{
				ID:             pulid.MustNew("ura_"),
				RoleID:         existingRoleID,
				UserID:         userID,
				OrganizationID: orgID,
			},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{existingRoleID, newRoleID}).
		Return([]*permission.Role{
			{
				ID:             existingRoleID,
				Name:           "Viewer",
				MaxSensitivity: permission.SensitivityInternal,
				Permissions: []*permission.ResourcePermission{
					{
						ID:         pulid.MustNew("rp_"),
						RoleID:     existingRoleID,
						Resource:   "shipment",
						Operations: []permission.Operation{permission.OpRead},
						DataScope:  permission.DataScopeOwn,
					},
				},
			},
			{
				ID:             newRoleID,
				Name:           "Editor",
				MaxSensitivity: permission.SensitivityRestricted,
				Permissions: []*permission.ResourcePermission{
					{
						ID:         pulid.MustNew("rp_"),
						RoleID:     newRoleID,
						Resource:   "shipment",
						Operations: []permission.Operation{permission.OpUpdate},
						DataScope:  permission.DataScopeOrganization,
					},
				},
			},
		}, nil)

	result, err := eng.SimulatePermissions(ctx, &services.SimulatePermissionsRequest{
		UserID:         userID,
		OrganizationID: orgID,
		AddRoleIDs:     []pulid.ID{newRoleID},
		RemoveRoleIDs:  []pulid.ID{},
	})

	require.NoError(t, err)
	assert.Len(t, result.Roles, 2)
	assert.Equal(t, permission.SensitivityRestricted, result.MaxSensitivity)
	assert.Contains(t, result.Resources, "shipment")
	assert.Equal(t, permission.DataScopeOrganization, result.Resources["shipment"].DataScope)

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

func TestExpiredAssignmentsIgnored(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")
	expiredRoleID := pulid.MustNew("rol_")

	expiredTime := timeutils.NowUnix() - 3600

	cacheRepo.On("Get", ctx, userID, orgID).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*permission.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
			{
				ID:             pulid.MustNew("ura_"),
				RoleID:         expiredRoleID,
				UserID:         userID,
				OrganizationID: orgID,
				ExpiresAt:      &expiredTime,
			},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).Return([]*permission.Role{
		{
			ID:             roleID,
			Name:           "Active Role",
			MaxSensitivity: permission.SensitivityInternal,
			Permissions:    []*permission.ResourcePermission{},
		},
	}, nil)
	cacheRepo.On("Set", ctx, userID, orgID, mock.AnythingOfType("*repositories.CachedPermissions"), cacheTTL).
		Return(nil)

	result, err := eng.Check(ctx, &services.PermissionCheckRequest{
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

	cacheRepo.On("Get", ctx, userID, orgID).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*permission.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: role1ID, UserID: userID, OrganizationID: orgID},
			{ID: pulid.MustNew("ura_"), RoleID: role2ID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{role1ID, role2ID}).
		Return([]*permission.Role{
			{
				ID:             role1ID,
				Name:           "Reader",
				MaxSensitivity: permission.SensitivityInternal,
				Permissions: []*permission.ResourcePermission{
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
				Permissions: []*permission.ResourcePermission{
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
	cacheRepo.On("Set", ctx, userID, orgID, mock.MatchedBy(func(perms *repositories.CachedPermissions) bool {
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

	result, err := eng.Check(ctx, &services.PermissionCheckRequest{
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

	cacheRepo.On("Get", ctx, userID, orgID).Return(&repositories.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityInternal),
		Resources: map[string]*repositories.CachedResourcePermission{
			"shipment": {
				Operations: []string{"read"},
				DataScope:  string(permission.DataScopeOrganization),
			},
		},
		ExpiresAt: timeutils.NowUnix() + 3600,
	}, nil)

	result, err := eng.Check(ctx, &services.PermissionCheckRequest{
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

	cacheRepo.On("Get", ctx, userID, orgID).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return(nil, errors.New("role lookup error"))

	result, err := eng.CheckBatch(ctx, &services.BatchPermissionCheckRequest{
		UserID:         userID,
		OrganizationID: orgID,
		Checks: []services.ResourceOperationCheck{
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

	cacheRepo.On("Get", ctx, userID, orgID).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*permission.UserRoleAssignment{}, nil)
	cacheRepo.On("Set", ctx, userID, orgID, mock.AnythingOfType("*repositories.CachedPermissions"), cacheTTL).
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

	cacheRepo.On("Get", ctx, userID, orgID).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*permission.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).Return([]*permission.Role{
		{
			ID:             roleID,
			Name:           "Viewer",
			MaxSensitivity: permission.SensitivityInternal,
			Permissions: []*permission.ResourcePermission{
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
	cacheRepo.On("Set", ctx, userID, orgID, mock.AnythingOfType("*repositories.CachedPermissions"), cacheTTL).
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

func TestGetResourcePermissions_NoPermissionForResource(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, userRepo := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	cacheRepo.On("Get", ctx, userID, orgID).Return(nil, nil)
	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*permission.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).Return([]*permission.Role{
		{
			ID:             roleID,
			Name:           "Viewer",
			MaxSensitivity: permission.SensitivityPublic,
			Permissions: []*permission.ResourcePermission{
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
	cacheRepo.On("Set", ctx, userID, orgID, mock.AnythingOfType("*repositories.CachedPermissions"), cacheTTL).
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
		Return([]*permission.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).
		Return(nil, errors.New("roles error"))

	result, err := eng.GetEffectivePermissions(ctx, userID, orgID)

	require.Error(t, err)
	assert.Nil(t, result)
	roleRepo.AssertExpectations(t)
}

func TestSimulatePermissions_WithRemoval(t *testing.T) {
	t.Parallel()

	eng, roleRepo, _, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	existingRoleID := pulid.MustNew("rol_")
	removeRoleID := pulid.MustNew("rol_")

	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*permission.UserRoleAssignment{
			{
				ID:             pulid.MustNew("ura_"),
				RoleID:         existingRoleID,
				UserID:         userID,
				OrganizationID: orgID,
			},
			{
				ID:             pulid.MustNew("ura_"),
				RoleID:         removeRoleID,
				UserID:         userID,
				OrganizationID: orgID,
			},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{existingRoleID}).
		Return([]*permission.Role{
			{
				ID:             existingRoleID,
				Name:           "Viewer",
				MaxSensitivity: permission.SensitivityInternal,
				Permissions: []*permission.ResourcePermission{
					{
						ID:         pulid.MustNew("rp_"),
						RoleID:     existingRoleID,
						Resource:   "shipment",
						Operations: []permission.Operation{permission.OpRead},
						DataScope:  permission.DataScopeOwn,
					},
				},
			},
		}, nil)

	result, err := eng.SimulatePermissions(ctx, &services.SimulatePermissionsRequest{
		UserID:         userID,
		OrganizationID: orgID,
		AddRoleIDs:     []pulid.ID{},
		RemoveRoleIDs:  []pulid.ID{removeRoleID},
	})

	require.NoError(t, err)
	assert.Len(t, result.Roles, 1)
	assert.Equal(t, "Viewer", result.Roles[0].Name)
	roleRepo.AssertExpectations(t)
}

func TestSimulatePermissions_Error(t *testing.T) {
	t.Parallel()

	eng, roleRepo, _, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).Return(nil, errors.New("db error"))

	result, err := eng.SimulatePermissions(ctx, &services.SimulatePermissionsRequest{
		UserID:         userID,
		OrganizationID: orgID,
		AddRoleIDs:     []pulid.ID{},
		RemoveRoleIDs:  []pulid.ID{},
	})

	require.Error(t, err)
	assert.Nil(t, result)
	roleRepo.AssertExpectations(t)
}

func TestSimulatePermissions_GetRolesError(t *testing.T) {
	t.Parallel()

	eng, roleRepo, _, _ := setupTestEngine(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return([]*permission.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: roleID, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{roleID}).
		Return(nil, errors.New("roles error"))

	result, err := eng.SimulatePermissions(ctx, &services.SimulatePermissionsRequest{
		UserID:         userID,
		OrganizationID: orgID,
		AddRoleIDs:     []pulid.ID{},
		RemoveRoleIDs:  []pulid.ID{},
	})

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
		Return([]*permission.UserRoleAssignment{
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
	roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{activeRoleID}).Return([]*permission.Role{
		{
			ID:             activeRoleID,
			Name:           "Active",
			MaxSensitivity: permission.SensitivityInternal,
			Permissions:    []*permission.ResourcePermission{},
		},
	}, nil)

	result, err := eng.GetEffectivePermissions(ctx, userID, orgID)

	require.NoError(t, err)
	assert.Len(t, result.Roles, 1)
	assert.Equal(t, "Active", result.Roles[0].Name)
	roleRepo.AssertExpectations(t)
}
