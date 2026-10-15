package agentaccessservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type unattendedRoleSets struct {
	*fakeDefinitions
	sets []agentdefinitionports.SetAgentDefinitionUnattendedRoleRequest
}

func (f *unattendedRoleSets) SetUnattendedRole(
	_ context.Context,
	req agentdefinitionports.SetAgentDefinitionUnattendedRoleRequest,
) error {
	f.sets = append(f.sets, req)
	return nil
}

type heldPermissions struct {
	*checkingPermissions
	held *permissionports.EffectivePermissions
}

func (p *heldPermissions) GetEffectivePermissions(
	context.Context,
	pulid.ID,
	pulid.ID,
) (*permissionports.EffectivePermissions, error) {
	return p.held, nil
}

func unattendedFixture(
	t *testing.T,
	held map[string]permissionports.EffectiveResourcePermission,
	roleGrants []*permission.ResourcePermission,
) (*fixture, *unattendedRoleSets) {
	t.Helper()

	f := newFixture(t)
	sets := &unattendedRoleSets{fakeDefinitions: f.definitions}
	f.service.definitions = sets
	f.service.permissions = &heldPermissions{
		checkingPermissions: f.permissions,
		held: &permissionports.EffectivePermissions{
			MaxSensitivity: permission.SensitivityRestricted,
			Resources:      held,
		},
	}
	roles, ok := f.service.roles.(*permissionportsmock.MockRoleRepository)
	require.True(t, ok)
	roles.On("GetRolesWithInheritance", mock.Anything, []pulid.ID{f.roleA.ID}).
		Return([]*permission.Role{{
			ID:             f.roleA.ID,
			MaxSensitivity: permission.SensitivityInternal,
			Permissions:    roleGrants,
		}}, nil).
		Maybe()

	return f, sets
}

func TestSetUnattendedRole_GivesTheAgentARoleThePersonCouldGrant(t *testing.T) {
	t.Parallel()

	f, sets := unattendedFixture(t,
		map[string]permissionports.EffectiveResourcePermission{
			"shipment": {
				Operations: []permission.Operation{permission.OpRead, permission.OpCreate},
				DataScope:  permission.DataScopeOrganization,
			},
		},
		[]*permission.ResourcePermission{{
			Resource:   "shipment",
			Operations: []permission.Operation{permission.OpRead},
			DataScope:  permission.DataScopeOrganization,
		}},
	)

	agent, err := f.service.SetUnattendedRole(t.Context(), &agentdefinitionports.SetAgentUnattendedRoleRequest{
		TenantInfo: f.tenant,
		AgentID:    f.agent.ID,
		RoleID:     f.roleA.ID,
	}, f.actor)

	require.NoError(t, err)
	assert.Equal(t, f.roleA.ID, agent.UnattendedRoleID)
	require.Len(t, sets.sets, 1)
	assert.Equal(t, f.roleA.ID, sets.sets[0].RoleID)
	assert.NotEmpty(t, f.notifier.Organizations(), "the agent's cached permissions must not outlive the change")
	require.Len(t, f.audit.logged, 1)
	assert.True(t, f.audit.logged[0].Critical)
}

func TestSetUnattendedRole_RefusesARoleBeyondThePerson(t *testing.T) {
	t.Parallel()

	f, sets := unattendedFixture(t,
		map[string]permissionports.EffectiveResourcePermission{
			"shipment": {
				Operations: []permission.Operation{permission.OpRead},
				DataScope:  permission.DataScopeOrganization,
			},
		},
		[]*permission.ResourcePermission{{
			Resource:   "shipment",
			Operations: []permission.Operation{permission.OpCreate},
			DataScope:  permission.DataScopeOrganization,
		}},
	)

	_, err := f.service.SetUnattendedRole(t.Context(), &agentdefinitionports.SetAgentUnattendedRoleRequest{
		TenantInfo: f.tenant,
		AgentID:    f.agent.ID,
		RoleID:     f.roleA.ID,
	}, f.actor)

	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.True(t, errors.As(err, &multiErr))
	assert.Empty(t, sets.sets)
}

func TestSetUnattendedRole_RefusesARoleOfAnotherOrganization(t *testing.T) {
	t.Parallel()

	f, sets := unattendedFixture(t, nil, nil)

	_, err := f.service.SetUnattendedRole(t.Context(), &agentdefinitionports.SetAgentUnattendedRoleRequest{
		TenantInfo: f.tenant,
		AgentID:    f.agent.ID,
		RoleID:     pulid.MustNew("rol_"),
	}, f.actor)

	require.Error(t, err)
	assert.Empty(t, sets.sets)
}

func TestSetUnattendedRole_NeedsPermissionToUpdateRoles(t *testing.T) {
	t.Parallel()

	f, sets := unattendedFixture(t, nil, nil)
	f.permissions.denied["role:update"] = true

	_, err := f.service.SetUnattendedRole(t.Context(), &agentdefinitionports.SetAgentUnattendedRoleRequest{
		TenantInfo: f.tenant,
		AgentID:    f.agent.ID,
		RoleID:     f.roleA.ID,
	}, f.actor)

	require.Error(t, err)
	assert.Empty(t, sets.sets)
}

func TestSetUnattendedRole_ClearsTheRole(t *testing.T) {
	t.Parallel()

	f, sets := unattendedFixture(t, nil, nil)
	f.definitions.byID[f.agent.ID].UnattendedRoleID = f.roleA.ID

	agent, err := f.service.SetUnattendedRole(t.Context(), &agentdefinitionports.SetAgentUnattendedRoleRequest{
		TenantInfo: f.tenant,
		AgentID:    f.agent.ID,
	}, f.actor)

	require.NoError(t, err)
	assert.True(t, agent.UnattendedRoleID.IsNil())
	require.Len(t, sets.sets, 1)
	assert.True(t, sets.sets[0].RoleID.IsNil())
}
