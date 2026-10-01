package permission

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func unattendedSystemActor() *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeAgent,
		PrincipalID:    services.AgentPrincipalID,
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func TestCheck_AnUnattendedActorWithTheSystemUserIsJudgedByTheAgentTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		resource  permission.Resource
		operation permission.Operation
	}{
		{
			name:      "a grant the agent table holds",
			resource:  permission.ResourceShipment,
			operation: permission.OpCreate,
		},
		{
			name:      "an operation the agent table withholds",
			resource:  permission.ResourceWorker,
			operation: permission.OpUpdate,
		},
		{
			name:      "approving is never an agent's",
			resource:  permission.ResourceBillingQueue,
			operation: permission.OpApprove,
		},
		{
			name:      "a resource the agent table does not name",
			resource:  permission.ResourceRole,
			operation: permission.OpRead,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			eng, roleRepo, cacheRepo, userRepo := setupTestEngine(t)
			actor := unattendedSystemActor()

			result, err := eng.Check(t.Context(), actor.PermissionCheck(tt.resource, tt.operation))
			require.NoError(t, err)

			assert.Equal(t, permission.IsAgentAllowed(tt.resource, tt.operation), result.Allowed)
			assert.Equal(t, permission.DataScopeOrganization, result.DataScope)
			roleRepo.AssertNotCalled(t, "GetUserRoleAssignments")
			cacheRepo.AssertNotCalled(t, "Get")
			userRepo.AssertExpectations(t)
		})
	}
}

func TestCheck_AnUnattendedActorCannotApprove(t *testing.T) {
	t.Parallel()

	eng, _, _, _ := setupTestEngine(t)
	actor := unattendedSystemActor()

	for resource := range registeredResources() {
		result, err := eng.Check(t.Context(), actor.PermissionCheck(resource, permission.OpApprove))
		require.NoError(t, err)
		assert.False(t, result.Allowed, "%s:approve", resource)
	}
}

func TestCheckBatch_AnUnattendedActorWithTheSystemUserIsJudgedByTheAgentTable(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, _ := setupTestEngine(t)
	actor := unattendedSystemActor()

	result, err := eng.CheckBatch(t.Context(), &services.BatchPermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		Checks: []services.ResourceOperationCheck{
			{Resource: permission.ResourceShipment.String(), Operation: permission.OpCreate},
			{Resource: permission.ResourceRole.String(), Operation: permission.OpRead},
			{Resource: permission.ResourceShipment.String(), Operation: permission.OpApprove},
		},
	})
	require.NoError(t, err)

	require.Len(t, result.Results, 3)
	assert.True(t, result.Results[0].Allowed)
	assert.False(t, result.Results[1].Allowed)
	assert.False(t, result.Results[2].Allowed)
	roleRepo.AssertNotCalled(t, "GetUserRoleAssignments")
	cacheRepo.AssertNotCalled(t, "Get")
}

func TestAgentsUsable_AnUnattendedActorIsNotReadAsItsSystemUser(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, _ := setupTestEngine(t)

	usable, err := eng.AgentsUsable(t.Context(), unattendedSystemActor(), permission.OpCreate)
	require.NoError(t, err)

	assert.Equal(
		t,
		permission.IsAgentAllowed(permission.ResourceAssistant, permission.OpCreate),
		usable.Assistant,
	)
	assert.Empty(t, usable.GrantedIDs)
	roleRepo.AssertNotCalled(t, "GetUserRoleAssignments")
	cacheRepo.AssertNotCalled(t, "Get")
}

func registeredResources() map[permission.Resource]struct{} {
	all := permission.NewRegistry().All()
	resources := make(map[permission.Resource]struct{}, len(all))
	for _, resource := range all {
		resources[permission.Resource(resource.Resource)] = struct{}{}
	}

	return resources
}
