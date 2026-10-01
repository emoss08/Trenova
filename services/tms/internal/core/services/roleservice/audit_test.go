package roleservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUpdateRoleRecordsBeforeAndAfter(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	deps.expectFullEffectivePermissions()
	actorID, orgID, buID := pulid.MustNew("usr_"), pulid.MustNew("org_"), pulid.MustNew("bu_")
	existing := &permission.Role{ID: pulid.MustNew("rol_"), Name: "Dispatch", BusinessUnitID: buID}
	updated := &permission.Role{ID: existing.ID, Name: "Dispatch lead", MaxSensitivity: permission.SensitivityInternal}

	deps.roleRepo.On("GetByID", mock.Anything, repositories.GetRoleByIDRequest{
		ID:         existing.ID,
		TenantInfo: pagination.TenantInfo{OrgID: orgID},
	}).Return(existing, nil)
	deps.roleRepo.On("Update", mock.Anything, updated).Return(nil)
	deps.permCache.On("InvalidateByRole", mock.Anything, existing.ID, deps.roleRepo).Return(nil)

	require.NoError(t, deps.svc.UpdateRole(t.Context(), UpdateRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Role:           updated,
	}))

	change := deps.audit.Only(t)
	assert.Equal(t, permission.ResourceRole, change.Resource)
	assert.Equal(t, permission.OpUpdate, change.Operation)
	assert.Equal(t, existing.ID.String(), change.ResourceID)
	assert.Equal(t, actorID, change.Actor.UserID)
	assert.Equal(t, orgID, change.OrganizationID)
	assert.Equal(t, buID, change.BusinessUnitID)
	assert.Same(t, existing, change.Before)
	assert.Same(t, updated, change.After)
}

func TestAssignAndUnassignRoleRecordTheUser(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	deps.expectFullEffectivePermissions()
	actorID, orgID := pulid.MustNew("usr_"), pulid.MustNew("org_")
	target := pulid.MustNew("usr_")
	roleID := pulid.MustNew("rol_")
	assignment := &permission.UserRoleAssignment{ID: pulid.MustNew("ura_"), UserID: target, RoleID: roleID}

	deps.roleRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&permission.Role{ID: roleID}, nil)
	deps.roleRepo.On("GetUserRoleAssignments", mock.Anything, target, orgID).
		Return([]*permission.UserRoleAssignment{}, nil)
	deps.roleRepo.On("CreateAssignment", mock.Anything, assignment).Return(nil)
	deps.roleRepo.On("DeleteAssignment", mock.Anything, repositories.DeleteRoleAssignmentRequest{
		AssignmentID:   assignment.ID,
		OrganizationID: orgID,
	}).Return(assignment, nil)
	deps.permEngine.On("InvalidateUser", mock.Anything, target, orgID).Return(nil)

	require.NoError(t, deps.svc.AssignRole(t.Context(), AssignRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		Assignment:     assignment,
	}))
	require.NoError(t, deps.svc.UnassignRole(t.Context(), UnassignRoleRequest{
		ActorID:        actorID,
		OrganizationID: orgID,
		AssignmentID:   assignment.ID,
	}))

	changes := deps.audit.Changes()
	require.Len(t, changes, 2)
	assert.Equal(t, permission.OpAssign, changes[0].Operation)
	assert.Equal(t, permission.OpUnassign, changes[1].Operation)
	for _, change := range changes {
		assert.Equal(t, roleID.String(), change.ResourceID)
		assert.Equal(t, target.String(), change.Metadata["userId"])
		assert.Equal(t, assignment.ID.String(), change.Metadata["assignmentId"])
		assert.Equal(t, actorID, change.Actor.UserID)
	}
}

func TestUpdateResourcePermissionRecordsThePreviousGrant(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	deps.expectFullEffectivePermissions()
	actorID, orgID := pulid.MustNew("usr_"), pulid.MustNew("org_")
	roleID, permID := pulid.MustNew("rol_"), pulid.MustNew("rp_")
	previous := &permission.ResourcePermission{
		ID:         permID,
		RoleID:     roleID,
		Resource:   "shipment",
		Operations: []permission.Operation{permission.OpRead},
		DataScope:  permission.DataScopeOrganization,
	}
	updated := &permission.ResourcePermission{
		ID:         permID,
		RoleID:     roleID,
		Resource:   "shipment",
		Operations: []permission.Operation{permission.OpRead, permission.OpUpdate},
		DataScope:  permission.DataScopeOrganization,
	}

	deps.roleRepo.On("GetByID", mock.Anything, mock.Anything).Return(&permission.Role{
		ID:          roleID,
		Permissions: []*permission.ResourcePermission{previous},
	}, nil)
	deps.roleRepo.On("UpdateResourcePermission", mock.Anything, updated).Return(nil)
	deps.permCache.On("InvalidateByRole", mock.Anything, roleID, deps.roleRepo).Return(nil)

	require.NoError(t, deps.svc.UpdateResourcePermission(t.Context(), actorID, orgID, updated))

	change := deps.audit.Only(t)
	assert.Same(t, previous, change.Before)
	assert.Same(t, updated, change.After)
	assert.Equal(t, "shipment", change.Metadata["resource"])
}

func TestRefusedRoleChangesAreNotRecorded(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	deps.roleRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&permission.Role{ID: roleID, IsSystem: true}, nil)

	err := deps.svc.DeleteResourcePermission(t.Context(), orgID, pulid.MustNew("rp_"), roleID)
	require.ErrorIs(t, err, ErrCannotModifySystemRole)
	assert.Empty(t, deps.audit.Changes())
}
