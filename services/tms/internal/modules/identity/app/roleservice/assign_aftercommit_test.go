package roleservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAssignRoleInsideATransactionAuditsOnlyAfterCommit(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	deps.expectFullEffectivePermissions()
	txCtx, hooks := kernelports.WithAfterCommitHooks(t.Context())
	targetUserID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")
	roleID := pulid.MustNew("rol_")

	deps.roleRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&permission.Role{ID: roleID}, nil)
	deps.roleRepo.On("GetUserRoleAssignments", mock.Anything, targetUserID, orgID).
		Return([]*permission.UserRoleAssignment{}, nil)
	deps.roleRepo.On("CreateAssignment", mock.Anything, mock.Anything).Return(nil)

	err := deps.svc.AssignRole(txCtx, AssignRoleRequest{
		ActorID:        pulid.MustNew("usr_"),
		OrganizationID: orgID,
		Assignment:     &permission.UserRoleAssignment{UserID: targetUserID, RoleID: roleID},
	})

	require.NoError(t, err)
	assert.Empty(t, deps.audit.Changes(), "nothing is audited before the transaction commits")
	assert.Empty(t, deps.notifier.Users())

	hooks.Run(t.Context())

	require.Len(t, deps.notifier.Users(), 1)
	assert.Equal(t, targetUserID, deps.notifier.Users()[0].UserID)
	change := deps.audit.Only(t)
	assert.Equal(t, permission.OpAssign, change.Operation)
	assert.Equal(t, roleID.String(), change.ResourceID)
}

func TestAssignRoleRolledBackLeavesNoAuditEntry(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	deps.expectFullEffectivePermissions()
	txCtx, _ := kernelports.WithAfterCommitHooks(t.Context())
	targetUserID := pulid.MustNew("usr_")
	orgID := pulid.MustNew("org_")

	deps.roleRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(&permission.Role{ID: pulid.MustNew("rol_")}, nil)
	deps.roleRepo.On("GetUserRoleAssignments", mock.Anything, targetUserID, orgID).
		Return([]*permission.UserRoleAssignment{}, nil)
	deps.roleRepo.On("CreateAssignment", mock.Anything, mock.Anything).Return(nil)

	require.NoError(t, deps.svc.AssignRole(txCtx, AssignRoleRequest{
		ActorID:        pulid.MustNew("usr_"),
		OrganizationID: orgID,
		Assignment:     &permission.UserRoleAssignment{UserID: targetUserID, RoleID: pulid.MustNew("rol_")},
	}))

	assert.Empty(t, deps.audit.Changes())
	assert.Empty(t, deps.notifier.Users())
}
