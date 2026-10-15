package roleservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func limitedEffectivePermissions() *permissionports.EffectivePermissions {
	return &permissionports.EffectivePermissions{
		MaxSensitivity: permission.SensitivityInternal,
		Resources: map[string]permissionports.EffectiveResourcePermission{
			permission.ResourceRole.String(): {
				Operations: []permission.Operation{
					permission.OpRead,
					permission.OpCreate,
					permission.OpUpdate,
				},
				DataScope: permission.DataScopeOrganization,
			},
		},
	}
}

func adminRole(orgID pulid.ID) *permission.Role {
	return &permission.Role{
		ID:             pulid.MustNew("rol_"),
		OrganizationID: orgID,
		Name:           "Organization Administrator",
		MaxSensitivity: permission.SensitivityConfidential,
		IsSystem:       true,
		Permissions: []*permission.ResourcePermission{{
			Resource:   permission.ResourceUser.String(),
			Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
			DataScope:  permission.DataScopeAll,
		}},
	}
}

func roleLookup(id pulid.ID) any {
	return mock.MatchedBy(func(req permissionports.GetRoleByIDRequest) bool {
		return req.ID == id
	})
}

func requireForbidden(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.True(t, errors.As(err, &multiErr), "expected a field error, got %v", err)
}

func TestUpsertRoleHierarchyEdge_RefusesInheritingMoreThanTheActorHolds(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	senior := &permission.Role{
		ID:             pulid.MustNew("rol_"),
		OrganizationID: orgID,
		Name:           "Manager",
		MaxSensitivity: permission.SensitivityInternal,
	}
	junior := adminRole(orgID)

	deps.roleRepo.On("GetByID", ctx, permissionports.GetRoleByIDRequest{
		ID:         senior.ID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(senior, nil)
	deps.roleRepo.On("GetByID", ctx, permissionports.GetRoleByIDRequest{
		ID:         junior.ID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(junior, nil)
	deps.permEngine.On("GetEffectivePermissions", ctx, actorID, orgID).
		Return(limitedEffectivePermissions(), nil)

	err := deps.svc.UpsertRoleHierarchyEdge(ctx, &UpsertHierarchyEdgeRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		SeniorRoleID:   senior.ID,
		JuniorRoleID:   junior.ID,
	})

	requireForbidden(t, err)
	refused := deps.audit.Only(t)
	assert.Equal(t, "Refused a grant beyond the actor's own permissions", refused.Comment)
	assert.Equal(t, actorID, refused.Actor.UserID)
	assert.Equal(t, "maxSensitivity", refused.Metadata["refused"])
	assert.Empty(t, deps.notifier.Organizations())
}

func TestUpsertRoleHierarchyEdge_RefusesTheJuniorsInheritedGrants(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	admin := adminRole(orgID)
	senior := &permission.Role{
		ID:             pulid.MustNew("rol_"),
		OrganizationID: orgID,
		MaxSensitivity: permission.SensitivityInternal,
	}
	wrapper := &permission.Role{
		ID:             pulid.MustNew("rol_"),
		OrganizationID: orgID,
		MaxSensitivity: permission.SensitivityInternal,
		ParentRoleIDs:  []pulid.ID{admin.ID},
	}

	deps.roleRepo.On("GetByID", ctx, permissionports.GetRoleByIDRequest{
		ID:         senior.ID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(senior, nil)
	deps.roleRepo.On("GetByID", ctx, permissionports.GetRoleByIDRequest{
		ID:         wrapper.ID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(wrapper, nil)
	deps.roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{admin.ID}).
		Return([]*permission.Role{admin}, nil)
	deps.permEngine.On("GetEffectivePermissions", ctx, actorID, orgID).
		Return(limitedEffectivePermissions(), nil)

	err := deps.svc.UpsertRoleHierarchyEdge(ctx, &UpsertHierarchyEdgeRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		SeniorRoleID:   senior.ID,
		JuniorRoleID:   wrapper.ID,
	})

	requireForbidden(t, err)
}

func TestUpsertRoleHierarchyEdge_RefusesASystemSenior(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	orgID := pulid.MustNew("org_")
	senior := adminRole(orgID)

	deps.roleRepo.On("GetByID", ctx, permissionports.GetRoleByIDRequest{
		ID:         senior.ID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(senior, nil)

	err := deps.svc.UpsertRoleHierarchyEdge(ctx, &UpsertHierarchyEdgeRequest{
		ActorID:        pulid.MustNew("usr_"),
		OrganizationID: orgID,
		SeniorRoleID:   senior.ID,
		JuniorRoleID:   pulid.MustNew("rol_"),
	})

	require.ErrorIs(t, err, ErrCannotModifySystemRole)
}

func TestUpsertRoleHierarchyEdge_AllowsWhatTheActorHolds(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	deps.expectFullEffectivePermissions()
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	senior := &permission.Role{ID: pulid.MustNew("rol_"), OrganizationID: orgID}
	junior := &permission.Role{
		ID:             pulid.MustNew("rol_"),
		OrganizationID: orgID,
		MaxSensitivity: permission.SensitivityInternal,
	}

	deps.roleRepo.On("GetByID", ctx, roleLookup(senior.ID)).Return(senior, nil)
	deps.roleRepo.On("GetByID", ctx, roleLookup(junior.ID)).Return(junior, nil)

	err := deps.svc.UpsertRoleHierarchyEdge(ctx, &UpsertHierarchyEdgeRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		SeniorRoleID:   senior.ID,
		JuniorRoleID:   junior.ID,
	})

	require.NoError(t, err)
	assert.Len(t, deps.audit.Changes(), 1)
}

func TestAssignRole_RefusesARoleWhoseParentsExceedTheActor(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	admin := adminRole(orgID)
	roleID := pulid.MustNew("rol_")

	deps.roleRepo.On("GetByID", ctx, permissionports.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{
		ID:             roleID,
		OrganizationID: orgID,
		MaxSensitivity: permission.SensitivityInternal,
		ParentRoleIDs:  []pulid.ID{admin.ID},
	}, nil)
	deps.roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{admin.ID}).
		Return([]*permission.Role{admin}, nil)
	deps.permEngine.On("GetEffectivePermissions", ctx, actorID, orgID).
		Return(limitedEffectivePermissions(), nil)

	err := deps.svc.AssignRole(ctx, AssignRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Assignment: &permission.UserRoleAssignment{
			UserID: pulid.MustNew("usr_"),
			RoleID: roleID,
		},
	})

	requireForbidden(t, err)
}

func TestCreateRole_RefusesParentsInAnotherOrganization(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	orgID := pulid.MustNew("org_")
	foreign := adminRole(pulid.MustNew("org_"))
	foreign.IsSystem = false

	deps.roleRepo.On("GetByID", ctx, mock.Anything).
		Return(&permission.Role{ID: foreign.ID}, nil)
	deps.roleRepo.On("GetRolesWithInheritance", ctx, []pulid.ID{foreign.ID}).
		Return([]*permission.Role{foreign}, nil)

	err := deps.svc.CreateRole(ctx, CreateRoleRequest{
		ActorID:        pulid.MustNew("usr_"),
		OrganizationID: orgID,
		Role: &permission.Role{
			Name:           "Borrowed",
			MaxSensitivity: permission.SensitivityInternal,
			ParentRoleIDs:  []pulid.ID{foreign.ID},
		},
	})

	require.Error(t, err)
	var validationErr *errortypes.Error
	require.True(t, errors.As(err, &validationErr))
	assert.Equal(t, "parentRoleIds", validationErr.Field)
}

func TestCreateRole_CannotCreateASystemRole(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	deps.expectFullEffectivePermissions()
	ctx := t.Context()
	role := &permission.Role{
		Name:           "Pretend system",
		MaxSensitivity: permission.SensitivityInternal,
		IsSystem:       true,
	}

	deps.roleRepo.On("Create", ctx, mock.MatchedBy(func(created *permission.Role) bool {
		return !created.IsSystem
	})).Return(nil)

	require.NoError(t, deps.svc.CreateRole(ctx, CreateRoleRequest{
		ActorID:        pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		Role:           role,
	}))
}

func TestUpdateRole_IgnoresFieldsTheEditorDoesNotOwn(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	deps.expectFullEffectivePermissions()
	ctx := t.Context()
	actorID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	existing := &permission.Role{
		ID:             pulid.MustNew("rol_"),
		OrganizationID: orgID,
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Dispatcher",
		MaxSensitivity: permission.SensitivityInternal,
		ParentRoleIDs:  []pulid.ID{},
		CreatedBy:      pulid.MustNew("usr_"),
	}
	grant := &permission.ResourcePermission{
		Resource:   permission.ResourceShipment.String(),
		Operations: []permission.Operation{permission.OpRead},
		DataScope:  permission.DataScopeOrganization,
	}
	request := &permission.Role{
		ID:             existing.ID,
		OrganizationID: pulid.MustNew("org_"),
		Name:           "Senior dispatcher",
		IsSystem:       true,
		ParentRoleIDs:  []pulid.ID{pulid.MustNew("rol_")},
		Permissions:    []*permission.ResourcePermission{grant, nil},
	}

	deps.roleRepo.On("GetByID", ctx, permissionports.GetRoleByIDRequest{
		ID:         existing.ID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(existing, nil)

	var written *permission.Role
	deps.roleRepo.On("Update", ctx, mock.Anything).Run(func(args mock.Arguments) {
		written = args.Get(1).(*permission.Role)
	}).Return(nil)

	require.NoError(t, deps.svc.UpdateRole(ctx, UpdateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           request,
	}))

	require.NotNil(t, written)
	assert.Equal(t, "Senior dispatcher", written.Name)
	assert.Equal(t, orgID, written.OrganizationID)
	assert.Equal(t, existing.BusinessUnitID, written.BusinessUnitID)
	assert.False(t, written.IsSystem)
	assert.Empty(t, written.ParentRoleIDs)
	assert.Equal(t, existing.CreatedBy, written.CreatedBy)
	assert.Equal(t, permission.SensitivityInternal, written.MaxSensitivity)
	assert.Equal(t, []*permission.ResourcePermission{grant}, written.Permissions)
	assert.Equal(t, "Senior dispatcher", request.Name)
	assert.False(t, request.IsSystem)
}

func TestUpdateRole_RefusesARoleOfAnotherOrganization(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	ctx := t.Context()
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	deps.roleRepo.On("GetByID", ctx, permissionports.GetRoleByIDRequest{
		ID:         roleID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(&permission.Role{ID: roleID}, nil)

	err := deps.svc.UpdateRole(ctx, UpdateRoleRequest{
		ActorID:        pulid.MustNew("usr_"),
		OrganizationID: orgID,
		Role:           &permission.Role{ID: roleID, Name: "Shared"},
	})

	require.ErrorIs(t, err, ErrCannotModifySystemRole)
}

func TestDeleteRole_RefusesARoleSomeoneHoldsUnlessForced(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	tenant := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	role := &permission.Role{
		ID:             pulid.MustNew("rol_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Name:           "Dispatcher",
	}
	deps.roleRepo.On("GetByID", mock.Anything, mock.Anything).Return(role, nil)
	deps.roleRepo.On("GetUsersWithRole", mock.Anything, role.ID).
		Return([]permissionports.ImpactedUser{{UserID: pulid.MustNew("usr_")}}, nil)

	err := deps.svc.DeleteRole(t.Context(), DeleteRoleRequest{Tenant: tenant, RoleID: role.ID})

	require.Error(t, err)
	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	deps.roleRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestDeleteRole_ForcedTakesItFromItsHolders(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	deps.expectFullEffectivePermissions()
	tenant := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	role := &permission.Role{
		ID:             pulid.MustNew("rol_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Name:           "Dispatcher",
	}
	holder := pulid.MustNew("usr_")
	deps.roleRepo.On("GetByID", mock.Anything, mock.Anything).Return(role, nil)
	deps.roleRepo.On("GetUsersWithRole", mock.Anything, role.ID).
		Return([]permissionports.ImpactedUser{{UserID: holder}}, nil)
	deps.roleRepo.On("Delete", mock.Anything, permissionports.DeleteRoleRequest{
		RoleID:         role.ID,
		OrganizationID: tenant.OrgID,
	}).Return(nil).Once()

	require.NoError(t, deps.svc.DeleteRole(t.Context(), DeleteRoleRequest{
		Tenant: tenant,
		RoleID: role.ID,
		Force:  true,
	}))

	changes := deps.audit.Changes()
	require.Len(t, changes, 2)
	assert.Equal(t, permission.OpUnassign, changes[0].Operation)
	assert.Equal(t, holder.String(), changes[0].Metadata[metadataUserID])
	assert.Equal(t, permission.OpDelete, changes[1].Operation)
	assert.Len(t, deps.notifier.Organizations(), 1)
}

func TestDeleteRole_ForcingNeedsPermissionToUnassign(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), UserID: pulid.MustNew("usr_")}
	role := &permission.Role{ID: pulid.MustNew("rol_"), OrganizationID: tenant.OrgID}
	deps.roleRepo.On("GetByID", mock.Anything, mock.Anything).Return(role, nil)
	deps.roleRepo.On("GetUsersWithRole", mock.Anything, role.ID).
		Return([]permissionports.ImpactedUser{{UserID: pulid.MustNew("usr_")}}, nil)
	deps.permEngine.On("GetEffectivePermissions", mock.Anything, tenant.UserID, tenant.OrgID).
		Return(limitedEffectivePermissions(), nil)

	err := deps.svc.DeleteRole(t.Context(), DeleteRoleRequest{Tenant: tenant, RoleID: role.ID, Force: true})

	requireForbidden(t, err)
	deps.roleRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestDeleteRole_RefusesASystemRole(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_")}
	admin := adminRole(tenant.OrgID)
	deps.roleRepo.On("GetByID", mock.Anything, mock.Anything).Return(admin, nil)

	err := deps.svc.DeleteRole(t.Context(), DeleteRoleRequest{Tenant: tenant, RoleID: admin.ID, Force: true})

	require.ErrorIs(t, err, ErrCannotDeleteSystemRole)
}
