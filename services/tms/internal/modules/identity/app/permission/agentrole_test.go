package permission

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/iamportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type agentRoles map[pulid.ID]pulid.ID

func (r agentRoles) UnattendedRoleID(
	_ context.Context,
	_ pagination.TenantInfo,
	agentID pulid.ID,
) (pulid.ID, error) {
	return r[agentID], nil
}

func agentCheck(agentID, orgID pulid.ID, resource string, op permission.Operation) *permissionports.PermissionCheckRequest {
	return &permissionports.PermissionCheckRequest{
		PrincipalType:  permissionports.PrincipalTypeAgent,
		PrincipalID:    agentID,
		OrganizationID: orgID,
		BusinessUnitID: pulid.MustNew("bu_"),
		Resource:       resource,
		Operation:      op,
	}
}

func TestCheck_AnUnattendedAgentIsHeldToItsRole(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, _ := setupTestEngine(t)
	agentID, roleID, orgID := pulid.MustNew("agdef_"), pulid.MustNew("rol_"), pulid.MustNew("org_")
	eng.agentRoles = agentRoles{agentID: roleID}

	cacheRepo.On("Get", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	cacheRepo.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	roleRepo.On("GetRolesWithInheritance", mock.Anything, []pulid.ID{roleID}).Return([]*permission.Role{
		{
			ID:             roleID,
			MaxSensitivity: permission.SensitivityInternal,
			Permissions: []*permission.ResourcePermission{
				{Resource: "shipment", Operations: []permission.Operation{permission.OpRead}},
			},
		},
	}, nil)

	read, err := eng.Check(t.Context(), agentCheck(agentID, orgID, "shipment", permission.OpRead))
	require.NoError(t, err)
	assert.True(t, read.Allowed)

	create, err := eng.Check(t.Context(), agentCheck(agentID, orgID, "shipment", permission.OpCreate))
	require.NoError(t, err)
	assert.False(t, create.Allowed, "the agent ceiling allows it, the agent's role does not")
	assert.Equal(t, "agent_role_denied", create.Reason)
}

func TestCheck_AnAgentWithoutARoleKeepsTheCeiling(t *testing.T) {
	t.Parallel()

	eng, _, cacheRepo, _ := setupTestEngine(t)
	agentID, orgID := pulid.MustNew("agdef_"), pulid.MustNew("org_")
	eng.agentRoles = agentRoles{}

	cacheRepo.On("Get", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	cacheRepo.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	result, err := eng.Check(t.Context(), agentCheck(agentID, orgID, "shipment", permission.OpRead))

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, "agent_allowed", result.Reason)
}

func TestCheck_AnAgentRoleNeverWidensTheCeiling(t *testing.T) {
	t.Parallel()

	eng, roleRepo, cacheRepo, _ := setupTestEngine(t)
	agentID, roleID, orgID := pulid.MustNew("agdef_"), pulid.MustNew("rol_"), pulid.MustNew("org_")
	eng.agentRoles = agentRoles{agentID: roleID}

	cacheRepo.On("Get", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	cacheRepo.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	roleRepo.On("GetRolesWithInheritance", mock.Anything, []pulid.ID{roleID}).Return([]*permission.Role{
		{
			ID: roleID,
			Permissions: []*permission.ResourcePermission{
				{Resource: "role", Operations: []permission.Operation{permission.OpDelete}},
			},
		},
	}, nil).Maybe()

	result, err := eng.Check(t.Context(), agentCheck(agentID, orgID, "role", permission.OpDelete))

	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, "agent_denied", result.Reason)
}

func TestCheck_AccessPoliciesBindAgents(t *testing.T) {
	t.Parallel()

	eng, _, _, _ := setupTestEngine(t)
	iamRepo := iamportsmock.NewMockIAMRepository(t)
	eng.iamRepo = iamRepo
	orgID := pulid.MustNew("org_")
	req := agentCheck(permissionports.AgentPrincipalID, orgID, "shipment", permission.OpRead)

	iamRepo.EXPECT().
		ListEnabledTenantAccessPolicies(mock.Anything, iamports.IAMTenantPolicyLookupRequest{
			OrganizationID: orgID,
			BusinessUnitID: req.BusinessUnitID,
		}).
		Return([]*iam.AccessPolicy{
			{
				ID:         pulid.MustNew("ap_"),
				Resource:   "shipment",
				Operation:  string(permission.OpRead),
				Effect:     iam.PolicyEffectDeny,
				Enabled:    true,
				Conditions: map[string]string{"principalType": string(permissionports.PrincipalTypeAgent)},
			},
		}, nil).
		Once()

	result, err := eng.Check(t.Context(), req)

	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, "iam_policy_denied", result.Reason)
}
