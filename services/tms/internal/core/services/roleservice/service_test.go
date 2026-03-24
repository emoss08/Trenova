package roleservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newStubValidator() *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*permission.Role]().
			WithModelName("Role").
			Build(),
	}
}

type testServiceDeps struct {
	roleRepo   *mocks.MockRoleRepository
	userRepo   *mocks.MockUserRepository
	permCache  *mocks.MockPermissionCacheRepository
	permEngine *mocks.MockPermissionEngine
	svc        *Service
}

func setupTestService(t *testing.T) *testServiceDeps {
	t.Helper()

	roleRepo := mocks.NewMockRoleRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	permCache := mocks.NewMockPermissionCacheRepository(t)
	permEngine := mocks.NewMockPermissionEngine(t)
	logger := zap.NewNop()

	svc := &Service{
		l:          logger.Named("test.role"),
		roleRepo:   roleRepo,
		userRepo:   userRepo,
		permCache:  permCache,
		permEngine: permEngine,
		validator:  newStubValidator(),
		registry:   permission.NewRegistry(),
	}

	return &testServiceDeps{
		roleRepo:   roleRepo,
		userRepo:   userRepo,
		permCache:  permCache,
		permEngine: permEngine,
		svc:        svc,
	}
}

func TestCreateRole_Success_PlatformAdmin(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	role := &permission.Role{
		Name:           "Test Role",
		MaxSensitivity: permission.SensitivityInternal,
		IsOrgAdmin:     false,
		Permissions:    []*permission.ResourcePermission{},
	}

	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(true, nil)
	deps.roleRepo.On("Create", ctx, role).Return(nil)

	err := deps.svc.CreateRole(ctx, CreateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.NoError(t, err)
	assert.Equal(t, orgID, role.OrganizationID)
	assert.Equal(t, actorID, role.CreatedBy)

	deps.userRepo.AssertExpectations(t)
	deps.roleRepo.AssertExpectations(t)
}

func TestCreateRole_OrgAdminWithoutPlatformAdmin_Fails(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	role := &permission.Role{
		Name:       "Admin Role",
		IsOrgAdmin: true,
	}

	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(false, nil)

	err := deps.svc.CreateRole(ctx, CreateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.Error(t, err)
	assert.Equal(t, ErrCannotCreateOrgAdmin, err)

	deps.userRepo.AssertExpectations(t)
}

func TestCreateRole_PrivilegeEscalation_Fails(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	role := &permission.Role{
		Name:           "Elevated Role",
		MaxSensitivity: permission.SensitivityConfidential,
		Permissions:    []*permission.ResourcePermission{},
	}

	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(false, nil)
	deps.permEngine.On("GetEffectivePermissions", ctx, actorID, orgID).
		Return(&services.EffectivePermissions{
			MaxSensitivity: permission.SensitivityInternal,
			Resources:      make(map[string]services.EffectiveResourcePermission),
		}, nil)

	err := deps.svc.CreateRole(ctx, CreateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.True(t, errors.As(err, &multiErr))

	deps.userRepo.AssertExpectations(t)
	deps.permEngine.AssertExpectations(t)
}

func TestUpdateRole_SystemRole_Fails(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	role := &permission.Role{
		ID:   roleID,
		Name: "Updated Role",
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:       roleID,
		IsSystem: true,
	}, nil)

	err := deps.svc.UpdateRole(ctx, UpdateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.Error(t, err)
	assert.Equal(t, ErrCannotModifySystemRole, err)

	deps.roleRepo.AssertExpectations(t)
}

func TestUpdateRole_Success(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	role := &permission.Role{
		ID:             roleID,
		Name:           "Updated Role",
		MaxSensitivity: permission.SensitivityInternal,
		Permissions:    []*permission.ResourcePermission{},
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:       roleID,
		IsSystem: false,
	}, nil)
	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(true, nil)
	deps.roleRepo.On("Update", ctx, role).Return(nil)
	deps.permCache.On("InvalidateByRole", ctx, roleID, deps.roleRepo).Return(nil)

	err := deps.svc.UpdateRole(ctx, UpdateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.NoError(t, err)

	deps.roleRepo.AssertExpectations(t)
	deps.userRepo.AssertExpectations(t)
	deps.permCache.AssertExpectations(t)
}

func TestListRoles(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	orgID := pulid.MustNew("org_")

	expectedRoles := []*permission.Role{
		{ID: pulid.MustNew("rol_"), Name: "Role 1"},
		{ID: pulid.MustNew("rol_"), Name: "Role 2"},
	}

	req := &repositories.ListRolesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{OrgID: orgID},
			Pagination: pagination.Info{Limit: 10, Offset: 0},
		},
	}

	deps.roleRepo.On("List", ctx, req).Return(&pagination.ListResult[*permission.Role]{
		Items: expectedRoles,
		Total: 2,
	}, nil)

	result, err := deps.svc.ListRoles(ctx, req)

	require.NoError(t, err)
	assert.Len(t, result.Items, 2)

	deps.roleRepo.AssertExpectations(t)
}

func TestAssignRole_Success(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	targetUserID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	assignment := &permission.UserRoleAssignment{
		UserID: targetUserID,
		RoleID: roleID,
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:          roleID,
		IsOrgAdmin:  false,
		Permissions: []*permission.ResourcePermission{},
	}, nil)
	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(true, nil)
	deps.roleRepo.On("CreateAssignment", ctx, assignment).Return(nil)
	deps.permEngine.On("InvalidateUser", ctx, targetUserID, orgID).Return(nil)

	err := deps.svc.AssignRole(ctx, AssignRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Assignment:     assignment,
	})

	require.NoError(t, err)
	assert.Equal(t, orgID, assignment.OrganizationID)
	assert.Equal(t, actorID, assignment.AssignedBy)

	deps.roleRepo.AssertExpectations(t)
	deps.userRepo.AssertExpectations(t)
	deps.permEngine.AssertExpectations(t)
}

func TestAssignRole_OrgAdmin_NotPlatformAdmin_NotOrgAdmin_Fails(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	targetUserID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	assignment := &permission.UserRoleAssignment{
		UserID: targetUserID,
		RoleID: roleID,
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:          roleID,
		IsOrgAdmin:  true,
		Permissions: []*permission.ResourcePermission{},
	}, nil)
	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(false, nil)
	deps.permEngine.On("GetEffectivePermissions", ctx, actorID, orgID).
		Return(&services.EffectivePermissions{
			Roles: []services.RoleSummary{
				{ID: pulid.MustNew("rol_"), Name: "Regular", IsOrgAdmin: false},
			},
		}, nil)

	err := deps.svc.AssignRole(ctx, AssignRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Assignment:     assignment,
	})

	require.Error(t, err)
	assert.Equal(t, ErrCannotEscalatePrivileges, err)

	deps.roleRepo.AssertExpectations(t)
	deps.userRepo.AssertExpectations(t)
	deps.permEngine.AssertExpectations(t)
}

func TestUnassignRole_Success(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	assignmentID := pulid.MustNew("ura_")

	deps.roleRepo.On("DeleteAssignment", ctx, assignmentID).Return(nil)

	err := deps.svc.UnassignRole(ctx, UnassignRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		AssignmentID:   assignmentID,
	})

	require.NoError(t, err)

	deps.roleRepo.AssertExpectations(t)
}

func TestGetImpactedUsers(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	roleID := pulid.MustNew("rol_")

	expected := []repositories.ImpactedUser{
		{UserID: pulid.MustNew("usr_"), UserName: "User 1"},
		{UserID: pulid.MustNew("usr_"), UserName: "User 2"},
	}

	deps.roleRepo.On("GetUsersWithRole", ctx, roleID).Return(expected, nil)

	users, err := deps.svc.GetImpactedUsers(ctx, roleID)

	require.NoError(t, err)
	assert.Len(t, users, 2)

	deps.roleRepo.AssertExpectations(t)
}

func TestInitializeOrganizationRoles(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	orgID := pulid.MustNew("org_")
	creatorID := pulid.MustNew("usr_")

	deps.roleRepo.On("Create", ctx, mock.MatchedBy(func(role *permission.Role) bool {
		return role.OrganizationID == orgID &&
			role.Name == "Organization Administrator" &&
			role.IsSystem &&
			role.IsOrgAdmin &&
			role.CreatedBy == creatorID
	})).Return(nil)
	deps.roleRepo.On("CreateAssignment", ctx, mock.MatchedBy(func(a *permission.UserRoleAssignment) bool {
		return a.UserID == creatorID &&
			a.OrganizationID == orgID &&
			a.AssignedBy == creatorID
	})).
		Return(nil)

	err := deps.svc.InitializeOrganizationRoles(ctx, orgID, creatorID)

	require.NoError(t, err)

	deps.roleRepo.AssertExpectations(t)
}

func TestCreateResourcePermission_Success(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	rp := &permission.ResourcePermission{
		RoleID:     roleID,
		Resource:   "shipment",
		Operations: []permission.Operation{permission.OpRead},
		DataScope:  permission.DataScopeOrganization,
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:       roleID,
		IsSystem: false,
	}, nil)
	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(true, nil)
	deps.roleRepo.On("CreateResourcePermission", ctx, rp).Return(nil)
	deps.permCache.On("InvalidateByRole", ctx, roleID, deps.roleRepo).Return(nil)

	err := deps.svc.CreateResourcePermission(ctx, actorID, orgID, rp)

	require.NoError(t, err)

	deps.roleRepo.AssertExpectations(t)
	deps.userRepo.AssertExpectations(t)
	deps.permCache.AssertExpectations(t)
}

func TestCreateResourcePermission_SystemRole_Fails(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	rp := &permission.ResourcePermission{
		RoleID:   roleID,
		Resource: "shipment",
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:       roleID,
		IsSystem: true,
	}, nil)

	err := deps.svc.CreateResourcePermission(ctx, actorID, orgID, rp)

	require.Error(t, err)
	assert.Equal(t, ErrCannotModifySystemRole, err)

	deps.roleRepo.AssertExpectations(t)
}

func TestDeleteResourcePermission_Success(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")
	permID := pulid.MustNew("rp_")

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:       roleID,
		IsSystem: false,
	}, nil)
	deps.roleRepo.On("DeleteResourcePermission", ctx, permID).Return(nil)
	deps.permCache.On("InvalidateByRole", ctx, roleID, deps.roleRepo).Return(nil)

	err := deps.svc.DeleteResourcePermission(ctx, orgID, permID, roleID)

	require.NoError(t, err)

	deps.roleRepo.AssertExpectations(t)
	deps.permCache.AssertExpectations(t)
}

func TestCircularInheritance_DirectSelfReference(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	role := &permission.Role{
		ID:            roleID,
		Name:          "Self Referencing Role",
		ParentRoleIDs: []pulid.ID{roleID},
	}

	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(true, nil)

	err := deps.svc.CreateRole(ctx, CreateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.Error(t, err)
	assert.Equal(t, ErrCircularInheritance, err)

	deps.userRepo.AssertExpectations(t)
}

func TestValidateNoEscalation_ResourceNotAllowed(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	role := &permission.Role{
		Name:           "Role with forbidden resource",
		MaxSensitivity: permission.SensitivityInternal,
		Permissions: []*permission.ResourcePermission{
			{
				Resource:   "billing",
				Operations: []permission.Operation{permission.OpRead},
				DataScope:  permission.DataScopeOrganization,
			},
		},
	}

	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(false, nil)
	deps.permEngine.On("GetEffectivePermissions", ctx, actorID, orgID).
		Return(&services.EffectivePermissions{
			MaxSensitivity: permission.SensitivityInternal,
			Resources: map[string]services.EffectiveResourcePermission{
				"shipment": {
					Operations: []permission.Operation{permission.OpRead},
					DataScope:  permission.DataScopeOrganization,
				},
			},
		}, nil)

	err := deps.svc.CreateRole(ctx, CreateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.True(t, errors.As(err, &multiErr))

	deps.userRepo.AssertExpectations(t)
	deps.permEngine.AssertExpectations(t)
}

func TestValidateNoEscalation_OperationNotAllowed(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	role := &permission.Role{
		Name:           "Role with forbidden operation",
		MaxSensitivity: permission.SensitivityInternal,
		Permissions: []*permission.ResourcePermission{
			{
				Resource:   "shipment",
				Operations: []permission.Operation{permission.OpRead, permission.OpCreate},
				DataScope:  permission.DataScopeOrganization,
			},
		},
	}

	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(false, nil)
	deps.permEngine.On("GetEffectivePermissions", ctx, actorID, orgID).
		Return(&services.EffectivePermissions{
			MaxSensitivity: permission.SensitivityInternal,
			Resources: map[string]services.EffectiveResourcePermission{
				"shipment": {
					Operations: []permission.Operation{permission.OpRead},
					DataScope:  permission.DataScopeOrganization,
				},
			},
		}, nil)

	err := deps.svc.CreateRole(ctx, CreateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.True(t, errors.As(err, &multiErr))

	deps.userRepo.AssertExpectations(t)
	deps.permEngine.AssertExpectations(t)
}

func TestValidateNoEscalation_DataScopeNotAllowed(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	role := &permission.Role{
		Name:           "Role with forbidden data scope",
		MaxSensitivity: permission.SensitivityInternal,
		Permissions: []*permission.ResourcePermission{
			{
				Resource:   "shipment",
				Operations: []permission.Operation{permission.OpRead},
				DataScope:  permission.DataScopeAll,
			},
		},
	}

	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(false, nil)
	deps.permEngine.On("GetEffectivePermissions", ctx, actorID, orgID).
		Return(&services.EffectivePermissions{
			MaxSensitivity: permission.SensitivityInternal,
			Resources: map[string]services.EffectiveResourcePermission{
				"shipment": {
					Operations: []permission.Operation{permission.OpRead},
					DataScope:  permission.DataScopeOrganization,
				},
			},
		}, nil)

	err := deps.svc.CreateRole(ctx, CreateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.True(t, errors.As(err, &multiErr))

	deps.userRepo.AssertExpectations(t)
	deps.permEngine.AssertExpectations(t)
}

func TestGetRoleByID(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	roleID := pulid.MustNew("rol_")
	orgID := pulid.MustNew("org_")

	expectedRole := &permission.Role{
		ID:   roleID,
		Name: "Test Role",
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(expectedRole, nil)

	role, err := deps.svc.GetRoleByID(ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	})

	require.NoError(t, err)
	assert.Equal(t, roleID, role.ID)
	assert.Equal(t, "Test Role", role.Name)

	deps.roleRepo.AssertExpectations(t)
}

func TestGetRoleByID_NotFound(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	roleID := pulid.MustNew("rol_")
	orgID := pulid.MustNew("org_")

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(nil, errors.New("not found"))

	role, err := deps.svc.GetRoleByID(ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	})

	require.Error(t, err)
	assert.Nil(t, role)

	deps.roleRepo.AssertExpectations(t)
}

func TestUpdateResourcePermission_Success(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")
	permID := pulid.MustNew("rp_")

	rp := &permission.ResourcePermission{
		ID:         permID,
		RoleID:     roleID,
		Resource:   "shipment",
		Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
		DataScope:  permission.DataScopeOrganization,
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:       roleID,
		IsSystem: false,
	}, nil)
	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(true, nil)
	deps.roleRepo.On("UpdateResourcePermission", ctx, rp).Return(nil)
	deps.permCache.On("InvalidateByRole", ctx, roleID, deps.roleRepo).Return(nil)

	err := deps.svc.UpdateResourcePermission(ctx, actorID, orgID, rp)

	require.NoError(t, err)

	deps.roleRepo.AssertExpectations(t)
	deps.userRepo.AssertExpectations(t)
	deps.permCache.AssertExpectations(t)
}

func TestUpdateResourcePermission_SystemRole_Fails(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")
	permID := pulid.MustNew("rp_")

	rp := &permission.ResourcePermission{
		ID:       permID,
		RoleID:   roleID,
		Resource: "shipment",
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:       roleID,
		IsSystem: true,
	}, nil)

	err := deps.svc.UpdateResourcePermission(ctx, actorID, orgID, rp)

	require.Error(t, err)
	assert.Equal(t, ErrCannotModifySystemRole, err)

	deps.roleRepo.AssertExpectations(t)
}

func TestUpdateResourcePermission_PrivilegeEscalation_Fails(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")
	permID := pulid.MustNew("rp_")

	rp := &permission.ResourcePermission{
		ID:         permID,
		RoleID:     roleID,
		Resource:   "billing",
		Operations: []permission.Operation{permission.OpRead},
		DataScope:  permission.DataScopeOrganization,
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:       roleID,
		IsSystem: false,
	}, nil)
	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(false, nil)
	deps.permEngine.On("GetEffectivePermissions", ctx, actorID, orgID).
		Return(&services.EffectivePermissions{
			MaxSensitivity: permission.SensitivityInternal,
			Resources: map[string]services.EffectiveResourcePermission{
				"shipment": {
					Operations: []permission.Operation{permission.OpRead},
					DataScope:  permission.DataScopeOrganization,
				},
			},
		}, nil)

	err := deps.svc.UpdateResourcePermission(ctx, actorID, orgID, rp)

	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.True(t, errors.As(err, &multiErr))

	deps.roleRepo.AssertExpectations(t)
	deps.userRepo.AssertExpectations(t)
	deps.permEngine.AssertExpectations(t)
}

func TestCircularInheritance_IndirectCycle(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleA := pulid.MustNew("rol_")
	roleB := pulid.MustNew("rol_")
	roleC := pulid.MustNew("rol_")

	role := &permission.Role{
		ID:            roleC,
		Name:          "Role C",
		ParentRoleIDs: []pulid.ID{roleB},
		Permissions:   []*permission.ResourcePermission{},
	}

	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(true, nil)
	deps.roleRepo.On("GetByID", ctx, mock.MatchedBy(func(req repositories.GetRoleByIDRequest) bool {
		return req.ID == roleB
	})).Return(&permission.Role{
		ID:            roleB,
		ParentRoleIDs: []pulid.ID{roleA},
	}, nil)
	deps.roleRepo.On("GetByID", ctx, mock.MatchedBy(func(req repositories.GetRoleByIDRequest) bool {
		return req.ID == roleA
	})).Return(&permission.Role{
		ID:            roleA,
		ParentRoleIDs: []pulid.ID{roleC},
	}, nil)

	err := deps.svc.CreateRole(ctx, CreateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.Error(t, err)
	assert.Equal(t, ErrCircularInheritance, err)

	deps.userRepo.AssertExpectations(t)
}

func TestCircularInheritance_NoParents(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	role := &permission.Role{
		Name:           "Role without parents",
		ParentRoleIDs:  []pulid.ID{},
		MaxSensitivity: permission.SensitivityInternal,
		Permissions:    []*permission.ResourcePermission{},
	}

	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(true, nil)
	deps.roleRepo.On("Create", ctx, role).Return(nil)

	err := deps.svc.CreateRole(ctx, CreateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.NoError(t, err)

	deps.userRepo.AssertExpectations(t)
	deps.roleRepo.AssertExpectations(t)
}

func TestCircularInheritance_ValidChain(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	parentRoleID := pulid.MustNew("rol_")

	role := &permission.Role{
		Name:           "Child Role",
		ParentRoleIDs:  []pulid.ID{parentRoleID},
		MaxSensitivity: permission.SensitivityInternal,
		Permissions:    []*permission.ResourcePermission{},
	}

	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(true, nil)
	deps.roleRepo.On("GetByID", ctx, mock.MatchedBy(func(req repositories.GetRoleByIDRequest) bool {
		return req.ID == parentRoleID
	})).Return(&permission.Role{
		ID:            parentRoleID,
		ParentRoleIDs: []pulid.ID{},
	}, nil)
	deps.roleRepo.On("Create", ctx, role).Return(nil)

	err := deps.svc.CreateRole(ctx, CreateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.NoError(t, err)

	deps.userRepo.AssertExpectations(t)
	deps.roleRepo.AssertExpectations(t)
}

func TestAssignRole_OrgAdminByOrgAdmin_Success(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	targetUserID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	assignment := &permission.UserRoleAssignment{
		UserID: targetUserID,
		RoleID: roleID,
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:          roleID,
		IsOrgAdmin:  true,
		Permissions: []*permission.ResourcePermission{},
	}, nil)
	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(false, nil)
	deps.permEngine.On("GetEffectivePermissions", ctx, actorID, orgID).
		Return(&services.EffectivePermissions{
			MaxSensitivity: permission.SensitivityConfidential,
			Roles: []services.RoleSummary{
				{ID: pulid.MustNew("rol_"), Name: "Org Admin", IsOrgAdmin: true},
			},
			Resources: make(map[string]services.EffectiveResourcePermission),
		}, nil)
	deps.roleRepo.On("CreateAssignment", ctx, assignment).Return(nil)
	deps.permEngine.On("InvalidateUser", ctx, targetUserID, orgID).Return(nil)

	err := deps.svc.AssignRole(ctx, AssignRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Assignment:     assignment,
	})

	require.NoError(t, err)

	deps.roleRepo.AssertExpectations(t)
	deps.userRepo.AssertExpectations(t)
	deps.permEngine.AssertExpectations(t)
}

func TestUnassignRole_Error(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	assignmentID := pulid.MustNew("ura_")

	deps.roleRepo.On("DeleteAssignment", ctx, assignmentID).Return(errors.New("database error"))

	err := deps.svc.UnassignRole(ctx, UnassignRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		AssignmentID:   assignmentID,
	})

	require.Error(t, err)

	deps.roleRepo.AssertExpectations(t)
}

func TestDeleteResourcePermission_SystemRole_Fails(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")
	permID := pulid.MustNew("rp_")

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:       roleID,
		IsSystem: true,
	}, nil)

	err := deps.svc.DeleteResourcePermission(ctx, orgID, permID, roleID)

	require.Error(t, err)
	assert.Equal(t, ErrCannotModifySystemRole, err)

	deps.roleRepo.AssertExpectations(t)
}

func TestInitializeOrganizationRoles_CreateRoleError(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	orgID := pulid.MustNew("org_")
	creatorID := pulid.MustNew("usr_")

	deps.roleRepo.On("Create", ctx, mock.Anything).Return(errors.New("create failed"))

	err := deps.svc.InitializeOrganizationRoles(ctx, orgID, creatorID)

	require.Error(t, err)

	deps.roleRepo.AssertExpectations(t)
}

func TestInitializeOrganizationRoles_AssignmentError(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	orgID := pulid.MustNew("org_")
	creatorID := pulid.MustNew("usr_")

	deps.roleRepo.On("Create", ctx, mock.Anything).Return(nil)
	deps.roleRepo.On("CreateAssignment", ctx, mock.Anything).Return(errors.New("assignment failed"))

	err := deps.svc.InitializeOrganizationRoles(ctx, orgID, creatorID)

	require.Error(t, err)

	deps.roleRepo.AssertExpectations(t)
}

func TestCreateRole_IsPlatformAdminError(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	role := &permission.Role{
		Name: "Test Role",
	}

	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(false, errors.New("db error"))

	err := deps.svc.CreateRole(ctx, CreateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.Error(t, err)

	deps.userRepo.AssertExpectations(t)
}

func TestUpdateRole_GetByIDError(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	role := &permission.Role{
		ID:   roleID,
		Name: "Updated Role",
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(nil, errors.New("not found"))

	err := deps.svc.UpdateRole(ctx, UpdateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.Error(t, err)

	deps.roleRepo.AssertExpectations(t)
}

func TestAssignRole_GetRoleError(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	assignment := &permission.UserRoleAssignment{
		UserID: pulid.MustNew("usr_"),
		RoleID: roleID,
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(nil, errors.New("not found"))

	err := deps.svc.AssignRole(ctx, AssignRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Assignment:     assignment,
	})

	require.Error(t, err)

	deps.roleRepo.AssertExpectations(t)
}

func TestCreateResourcePermission_GetRoleError(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	rp := &permission.ResourcePermission{
		RoleID:   roleID,
		Resource: "shipment",
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(nil, errors.New("not found"))

	err := deps.svc.CreateResourcePermission(ctx, actorID, orgID, rp)

	require.Error(t, err)

	deps.roleRepo.AssertExpectations(t)
}

func TestUpdateResourcePermission_GetRoleError(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	rp := &permission.ResourcePermission{
		ID:       pulid.MustNew("rp_"),
		RoleID:   roleID,
		Resource: "shipment",
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(nil, errors.New("not found"))

	err := deps.svc.UpdateResourcePermission(ctx, actorID, orgID, rp)

	require.Error(t, err)

	deps.roleRepo.AssertExpectations(t)
}

func TestDeleteResourcePermission_GetRoleError(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")
	permID := pulid.MustNew("rp_")

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(nil, errors.New("not found"))

	err := deps.svc.DeleteResourcePermission(ctx, orgID, permID, roleID)

	require.Error(t, err)

	deps.roleRepo.AssertExpectations(t)
}

func TestNew(t *testing.T) {
	t.Parallel()

	roleRepo := mocks.NewMockRoleRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	permCache := mocks.NewMockPermissionCacheRepository(t)
	permEngine := mocks.NewMockPermissionEngine(t)
	validator := newStubValidator()
	registry := permission.NewRegistry()

	svc := New(Params{
		Logger:           zap.NewNop(),
		RoleRepo:         roleRepo,
		UserRepo:         userRepo,
		PermissionCache:  permCache,
		PermissionEngine: permEngine,
		Validator:        validator,
		Registry:         registry,
	})

	require.NotNil(t, svc)
}

func TestNewTestValidator(t *testing.T) {
	t.Parallel()

	v := NewTestValidator()
	require.NotNil(t, v)
}

func TestGetUserRoleAssignments_Success(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	expected := []*permission.UserRoleAssignment{
		{
			UserID:         userID,
			RoleID:         roleID,
			OrganizationID: orgID,
			AssignedBy:     pulid.MustNew("usr_"),
		},
	}

	deps.roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).Return(expected, nil)

	assignments, err := deps.svc.GetUserRoleAssignments(ctx, userID, orgID)

	require.NoError(t, err)
	assert.Len(t, assignments, 1)
	assert.Equal(t, userID, assignments[0].UserID)

	deps.roleRepo.AssertExpectations(t)
}

func TestGetUserRoleAssignments_Error(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	userID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	deps.roleRepo.On("GetUserRoleAssignments", ctx, userID, orgID).
		Return(nil, errors.New("db error"))

	assignments, err := deps.svc.GetUserRoleAssignments(ctx, userID, orgID)

	require.Error(t, err)
	assert.Nil(t, assignments)

	deps.roleRepo.AssertExpectations(t)
}

func TestCreateResourcePermission_PermissionRepoError(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	rp := &permission.ResourcePermission{
		RoleID:     roleID,
		Resource:   "shipment",
		Operations: []permission.Operation{permission.OpRead},
		DataScope:  permission.DataScopeOrganization,
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:       roleID,
		IsSystem: false,
	}, nil)
	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(true, nil)
	deps.roleRepo.On("CreateResourcePermission", ctx, rp).Return(errors.New("db error"))

	err := deps.svc.CreateResourcePermission(ctx, actorID, orgID, rp)

	require.Error(t, err)
	assert.Equal(t, "db error", err.Error())

	deps.roleRepo.AssertExpectations(t)
	deps.userRepo.AssertExpectations(t)
}

func TestCreateResourcePermission_CacheInvalidationError(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	rp := &permission.ResourcePermission{
		RoleID:     roleID,
		Resource:   "shipment",
		Operations: []permission.Operation{permission.OpRead},
		DataScope:  permission.DataScopeOrganization,
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:       roleID,
		IsSystem: false,
	}, nil)
	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(true, nil)
	deps.roleRepo.On("CreateResourcePermission", ctx, rp).Return(nil)
	deps.permCache.On("InvalidateByRole", ctx, roleID, deps.roleRepo).
		Return(errors.New("cache error"))

	err := deps.svc.CreateResourcePermission(ctx, actorID, orgID, rp)

	require.NoError(t, err)

	deps.roleRepo.AssertExpectations(t)
	deps.userRepo.AssertExpectations(t)
	deps.permCache.AssertExpectations(t)
}

func TestUpdateResourcePermission_PermissionRepoError(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")
	permID := pulid.MustNew("rp_")

	rp := &permission.ResourcePermission{
		ID:         permID,
		RoleID:     roleID,
		Resource:   "shipment",
		Operations: []permission.Operation{permission.OpRead},
		DataScope:  permission.DataScopeOrganization,
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:       roleID,
		IsSystem: false,
	}, nil)
	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(true, nil)
	deps.roleRepo.On("UpdateResourcePermission", ctx, rp).Return(errors.New("db error"))

	err := deps.svc.UpdateResourcePermission(ctx, actorID, orgID, rp)

	require.Error(t, err)
	assert.Equal(t, "db error", err.Error())

	deps.roleRepo.AssertExpectations(t)
	deps.userRepo.AssertExpectations(t)
}

func TestUpdateResourcePermission_CacheInvalidationError(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")
	permID := pulid.MustNew("rp_")

	rp := &permission.ResourcePermission{
		ID:         permID,
		RoleID:     roleID,
		Resource:   "shipment",
		Operations: []permission.Operation{permission.OpRead},
		DataScope:  permission.DataScopeOrganization,
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:       roleID,
		IsSystem: false,
	}, nil)
	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(true, nil)
	deps.roleRepo.On("UpdateResourcePermission", ctx, rp).Return(nil)
	deps.permCache.On("InvalidateByRole", ctx, roleID, deps.roleRepo).
		Return(errors.New("cache error"))

	err := deps.svc.UpdateResourcePermission(ctx, actorID, orgID, rp)

	require.NoError(t, err)

	deps.roleRepo.AssertExpectations(t)
	deps.userRepo.AssertExpectations(t)
	deps.permCache.AssertExpectations(t)
}

func TestDeleteResourcePermission_NotFoundError(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")
	permID := pulid.MustNew("rp_")

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:       roleID,
		IsSystem: false,
	}, nil)
	deps.roleRepo.On("DeleteResourcePermission", ctx, permID).Return(errors.New("not found"))

	err := deps.svc.DeleteResourcePermission(ctx, orgID, permID, roleID)

	require.Error(t, err)
	assert.Equal(t, "not found", err.Error())

	deps.roleRepo.AssertExpectations(t)
}

func TestUpdateRole_IsPlatformAdminError(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	role := &permission.Role{
		ID:   roleID,
		Name: "Updated Role",
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:       roleID,
		IsSystem: false,
	}, nil)
	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(false, errors.New("db error"))

	err := deps.svc.UpdateRole(ctx, UpdateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.Error(t, err)

	deps.roleRepo.AssertExpectations(t)
	deps.userRepo.AssertExpectations(t)
}

func TestUpdateRole_OrgAdminWithoutPlatformAdmin_Fails(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	role := &permission.Role{
		ID:         roleID,
		Name:       "Admin Role",
		IsOrgAdmin: true,
	}

	deps.roleRepo.On("GetByID", ctx, repositories.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:       roleID,
		IsSystem: false,
	}, nil)
	deps.userRepo.On("IsPlatformAdmin", ctx, actorID).Return(false, nil)

	err := deps.svc.UpdateRole(ctx, UpdateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           role,
	})

	require.Error(t, err)
	assert.Equal(t, ErrCannotCreateOrgAdmin, err)

	deps.roleRepo.AssertExpectations(t)
	deps.userRepo.AssertExpectations(t)
}
