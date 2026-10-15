package roleservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func scopedTo(orgID, buID pulid.ID) any {
	return mock.MatchedBy(func(ctx context.Context) bool {
		tenant, ok := dbscope.TenantFrom(ctx)
		return ok && tenant.OrganizationID == orgID && tenant.BusinessUnitID == buID
	})
}

func TestRevokeOrganizationRoles_DeletesInThatOrganizationsScopeAndAudits(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	actorID, userID := pulid.MustNew("usr_"), pulid.MustNew("usr_")
	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	deleted := []*permission.UserRoleAssignment{
		{ID: pulid.MustNew("ura_"), UserID: userID, OrganizationID: orgID, RoleID: pulid.MustNew("rol_")},
		{ID: pulid.MustNew("ura_"), UserID: userID, OrganizationID: orgID, RoleID: pulid.MustNew("rol_")},
	}

	deps.roleRepo.On("DeleteUserAssignments", scopedTo(orgID, buID), userID, orgID).
		Return(deleted, nil).Once()
	deps.permEngine.On("InvalidateUser", scopedTo(orgID, buID), userID, orgID).Return(nil).Once()

	require.NoError(t, deps.svc.RevokeOrganizationRoles(t.Context(), RevokeOrganizationRolesRequest{
		ActorID:        actorID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}))

	changes := deps.audit.Changes()
	require.Len(t, changes, 2)
	for i, change := range changes {
		assert.Equal(t, permission.OpUnassign, change.Operation)
		assert.Equal(t, deleted[i].RoleID.String(), change.ResourceID)
		assert.Equal(t, orgID, change.OrganizationID)
		assert.Equal(t, actorID, change.Actor.UserID)
		assert.Equal(t, userID.String(), change.Metadata[metadataUserID])
	}
}

func TestRevokeOrganizationRoles_ReportsAFailedDelete(t *testing.T) {
	t.Parallel()

	deps := setupTestService(t)
	userID, orgID, buID := pulid.MustNew("usr_"), pulid.MustNew("org_"), pulid.MustNew("bu_")

	deps.roleRepo.On("DeleteUserAssignments", mock.Anything, userID, orgID).
		Return(nil, errors.New("database unavailable")).Once()

	err := deps.svc.RevokeOrganizationRoles(t.Context(), RevokeOrganizationRolesRequest{
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})

	require.Error(t, err)
	assert.Empty(t, deps.audit.Changes())
}
