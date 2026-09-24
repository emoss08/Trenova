package agentaccessservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// savedIn is the caller's save: it counts its calls and hands back the agent
// as the repository would, or fails.
type savedIn struct {
	calls int
	agent *agentdefinition.Definition
	err   error
}

func (s *savedIn) save(context.Context) (*agentdefinition.Definition, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	copied := *s.agent

	return &copied, nil
}

func newAgent(f *fixture) *agentdefinition.Definition {
	agent := &agentdefinition.Definition{
		ID:             pulid.MustNew("agdef_"),
		Name:           "Payroll desk",
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
		AccessMode:     agentdefinition.AccessEveryone,
		Enabled:        true,
	}

	return agent
}

/*
A new agent meant for some roles is created restricted in one transaction: the
save and the access land together, the mode is set before anything commits,
the roles are granted, and the grant reaches the role holders at once.
*/
func TestSaveWithAccess_CreatesARestrictedAgentInOneStep(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	saved := &savedIn{agent: newAgent(f)}
	f.cache.On("InvalidateByRole", mock.Anything, f.roleA.ID, mock.Anything).Return(nil).Once()

	agent, err := f.service.SaveWithAccess(t.Context(), &services.SaveAgentWithAccessRequest{
		TenantInfo: f.tenant,
		Access: services.AgentAccessWrite{
			Mode:    agentdefinition.AccessRoles,
			RoleIDs: []pulid.ID{f.roleA.ID},
		},
		Save: saved.save,
	}, f.actor)

	require.NoError(t, err)
	assert.Equal(t, 1, saved.calls)
	assert.Equal(t, 1, f.db.transactions, "the agent and who may use it are one change")
	assert.Equal(t, agentdefinition.AccessRoles, agent.AccessMode)
	assert.True(t, agent.Enabled, "restricted from the start, so it need not be held back")
	require.Len(t, f.definitions.modeSets, 1)
	assert.Equal(t, saved.agent.ID, f.definitions.modeSets[0].ID)
	require.Len(t, f.grants.agentReplace, 1)
	assert.Equal(t, []pulid.ID{f.roleA.ID}, f.grants.agentReplace[0].RoleIDs)

	resources := make([]permission.Resource, 0, len(f.audit.logged))
	for _, entry := range f.audit.logged {
		resources = append(resources, entry.Resource)
	}
	assert.ElementsMatch(t, []permission.Resource{
		permission.ResourceAgentDefinition,
		permission.ResourceRole,
	}, resources)
}

/*
Changing who may use an agent needs role:update, whichever way it is saved. A
person without it is refused clearly, and the save is undone with the access:
nothing is left half-written, nobody's cache is touched, nothing is audited.
*/
func TestSaveWithAccess_RefusesAnAccessChangeWithoutRoleUpdate(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.permissions.denied["role:update"] = true
	saved := &savedIn{agent: newAgent(f)}

	_, err := f.service.SaveWithAccess(t.Context(), &services.SaveAgentWithAccessRequest{
		TenantInfo: f.tenant,
		Access: services.AgentAccessWrite{
			Mode:    agentdefinition.AccessRoles,
			RoleIDs: []pulid.ID{f.roleA.ID},
		},
		Save: saved.save,
	}, f.actor)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Contains(t, err.Error(), "update roles")
	assert.Empty(t, f.audit.logged)
	f.cache.AssertNotCalled(t, "InvalidateByRole", mock.Anything, mock.Anything, mock.Anything)
}

// Access that already reads as asked is no change, so a person who may not
// update roles can still save the rest of an agent.
func TestSaveWithAccess_UnchangedAccessNeedsNoRoleUpdate(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.permissions.denied["role:update"] = true
	agent := newAgent(f)
	agent.AccessMode = agentdefinition.AccessRoles
	f.grants.agentRoles[agent.ID] = []pulid.ID{f.roleA.ID}
	saved := &savedIn{agent: agent}

	result, err := f.service.SaveWithAccess(t.Context(), &services.SaveAgentWithAccessRequest{
		TenantInfo: f.tenant,
		Access: services.AgentAccessWrite{
			Mode:    agentdefinition.AccessRoles,
			RoleIDs: []pulid.ID{f.roleA.ID},
		},
		Save: saved.save,
	}, f.actor)

	require.NoError(t, err)
	assert.Equal(t, agentdefinition.AccessRoles, result.AccessMode)
	assert.Empty(t, f.definitions.modeSets)
	assert.Empty(t, f.audit.logged, "nothing changed, so nothing is audited")
}

// A new agent open to everyone with no roles is the default; saving it so
// needs no role permission either.
func TestSaveWithAccess_TheDefaultNeedsNoRoleUpdate(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.permissions.denied["role:update"] = true
	saved := &savedIn{agent: newAgent(f)}

	result, err := f.service.SaveWithAccess(t.Context(), &services.SaveAgentWithAccessRequest{
		TenantInfo: f.tenant,
		Access:     services.AgentAccessWrite{Mode: agentdefinition.AccessEveryone},
		Save:       saved.save,
	}, f.actor)

	require.NoError(t, err)
	assert.True(t, result.OpenToEveryone())
}

func TestSaveWithAccess_AFailedSaveSetsNoAccess(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	refusal := errors.New("version conflict")
	saved := &savedIn{agent: newAgent(f), err: refusal}

	_, err := f.service.SaveWithAccess(t.Context(), &services.SaveAgentWithAccessRequest{
		TenantInfo: f.tenant,
		Access: services.AgentAccessWrite{
			Mode:    agentdefinition.AccessRoles,
			RoleIDs: []pulid.ID{f.roleA.ID},
		},
		Save: saved.save,
	}, f.actor)

	require.ErrorIs(t, err, refusal)
	assert.Empty(t, f.definitions.modeSets)
	assert.Empty(t, f.grants.agentReplace)
}

func TestSaveWithAccess_RefusesBadAccessBeforeSaving(t *testing.T) {
	t.Parallel()

	for name, access := range map[string]func(f *fixture) services.AgentAccessWrite{
		"unknown mode": func(*fixture) services.AgentAccessWrite {
			return services.AgentAccessWrite{Mode: "Somebody"}
		},
		"duplicate role": func(f *fixture) services.AgentAccessWrite {
			return services.AgentAccessWrite{
				Mode:    agentdefinition.AccessRoles,
				RoleIDs: []pulid.ID{f.roleA.ID, f.roleA.ID},
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			saved := &savedIn{agent: newAgent(f)}

			_, err := f.service.SaveWithAccess(t.Context(), &services.SaveAgentWithAccessRequest{
				TenantInfo: f.tenant,
				Access:     access(f),
				Save:       saved.save,
			}, f.actor)

			require.Error(t, err)
			assert.Zero(t, saved.calls, "nothing is saved for access that cannot be set")
		})
	}
}

// A role from another organization is refused inside the transaction, so
// the agent's save is undone with it.
func TestSaveWithAccess_RefusesAnotherTenantsRole(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	saved := &savedIn{agent: newAgent(f)}

	_, err := f.service.SaveWithAccess(t.Context(), &services.SaveAgentWithAccessRequest{
		TenantInfo: f.tenant,
		Access: services.AgentAccessWrite{
			Mode:    agentdefinition.AccessRoles,
			RoleIDs: []pulid.ID{pulid.MustNew("rol_")},
		},
		Save: saved.save,
	}, f.actor)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Empty(t, f.grants.agentReplace)
}
