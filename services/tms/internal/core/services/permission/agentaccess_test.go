package permission

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeAgentGrants struct {
	repositories.RoleAgentGrantRepository

	byRole map[pulid.ID][]pulid.ID
	asked  []repositories.ListGrantedAgentIDsRequest
}

func (f *fakeAgentGrants) ListGrantedAgentIDs(
	_ context.Context,
	req repositories.ListGrantedAgentIDsRequest,
) ([]pulid.ID, error) {
	f.asked = append(f.asked, req)

	ids := make([]pulid.ID, 0)
	for _, roleID := range req.RoleIDs {
		ids = append(ids, f.byRole[roleID]...)
	}

	return ids, nil
}

type agentAccessFixture struct {
	engine *engine
	grants *fakeAgentGrants
	actor  *services.RequestActor
	child  pulid.ID
	parent pulid.ID
}

// newAgentAccessFixture is a person holding a role that inherits another. The
// parent holds the assistant; the child holds nothing of its own.
func newAgentAccessFixture(t *testing.T, parentOps ...permission.Operation) *agentAccessFixture {
	t.Helper()

	eng, roleRepo, cacheRepo, _ := setupTestEngine(t)
	grants := &fakeAgentGrants{byRole: map[pulid.ID][]pulid.ID{}}
	eng.agentGrants = grants

	orgID := pulid.MustNew("org_")
	userID := pulid.MustNew("usr_")
	child := pulid.MustNew("rol_")
	parent := pulid.MustNew("rol_")
	if len(parentOps) == 0 {
		parentOps = []permission.Operation{permission.OpRead, permission.OpCreate}
	}

	cacheRepo.On("Get", mock.Anything, allRolesKey(userID, orgID)).Return(nil, nil)
	cacheRepo.On("Set", mock.Anything, allRolesKey(userID, orgID),
		mock.AnythingOfType("*repositories.CachedPermissions"), cacheTTL).Return(nil)
	roleRepo.On("GetUserRoleAssignments", mock.Anything, userID, orgID).
		Return([]*permission.UserRoleAssignment{
			{ID: pulid.MustNew("ura_"), RoleID: child, UserID: userID, OrganizationID: orgID},
		}, nil)
	roleRepo.On("GetRolesWithInheritance", mock.Anything, []pulid.ID{child}).
		Return([]*permission.Role{
			{ID: child, Name: "Dispatcher", OrganizationID: orgID, ParentRoleIDs: []pulid.ID{parent}},
			{
				ID:             parent,
				Name:           "Assistant user",
				OrganizationID: orgID,
				Permissions: []*permission.ResourcePermission{{
					Resource:   permission.ResourceAssistant.String(),
					Operations: parentOps,
					DataScope:  permission.DataScopeOrganization,
				}},
			},
		}, nil)

	return &agentAccessFixture{
		engine: eng,
		grants: grants,
		child:  child,
		parent: parent,
		actor: &services.RequestActor{
			PrincipalType:  services.PrincipalTypeUser,
			PrincipalID:    userID,
			UserID:         userID,
			OrganizationID: orgID,
			BusinessUnitID: pulid.MustNew("bu_"),
		},
	}
}

func restrictedAgent() *agentdefinition.Definition {
	return &agentdefinition.Definition{
		ID:         pulid.MustNew("agdef_"),
		Name:       "Payroll helper",
		AccessMode: agentdefinition.AccessRoles,
	}
}

/*
A grant on a role reaches everyone holding a role that inherits it, the way a
resource permission does: the grants are read for the whole role closure.
*/
func TestAgentsUsable_ResolvesAGrantThroughAnInheritedRole(t *testing.T) {
	t.Parallel()

	f := newAgentAccessFixture(t)
	agent := restrictedAgent()
	f.grants.byRole[f.parent] = []pulid.ID{agent.ID}

	usable, err := f.engine.AgentsUsable(t.Context(), f.actor, permission.OpCreate)

	require.NoError(t, err)
	assert.True(t, usable.Assistant)
	assert.Equal(t, []pulid.ID{agent.ID}, usable.GrantedIDs)
	require.Len(t, f.grants.asked, 1)
	assert.ElementsMatch(t, []pulid.ID{f.child, f.parent}, f.grants.asked[0].RoleIDs,
		"the grants are read for the role and every role it inherits")
	assert.Equal(t, f.actor.OrganizationID, f.grants.asked[0].OrganizationID)
}

func TestMayUseAgent_UnderEachMode(t *testing.T) {
	t.Parallel()

	system := &agentdefinition.Definition{
		ID:         pulid.MustNew("agdef_"),
		Name:       "Briefing",
		SystemKey:  "briefing",
		AccessMode: agentdefinition.AccessRoles,
	}
	open := &agentdefinition.Definition{
		ID:         pulid.MustNew("agdef_"),
		Name:       "Dispatch",
		AccessMode: agentdefinition.AccessEveryone,
	}

	tests := map[string]struct {
		definition *agentdefinition.Definition
		grant      bool
		parentOps  []permission.Operation
		want       bool
	}{
		"open to everyone":              {definition: open, want: true},
		"restricted and granted":        {definition: restrictedAgent(), grant: true, want: true},
		"restricted and not granted":    {definition: restrictedAgent(), want: false},
		"a system agent is always open": {definition: system, want: true},
		"granted but no assistant create": {
			definition: restrictedAgent(),
			grant:      true,
			parentOps:  []permission.Operation{permission.OpRead},
			want:       false,
		},
		"open but no assistant create": {
			definition: open,
			parentOps:  []permission.Operation{permission.OpRead},
			want:       false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newAgentAccessFixture(t, tt.parentOps...)
			if tt.grant {
				f.grants.byRole[f.child] = []pulid.ID{tt.definition.ID}
			}

			allowed, err := f.engine.MayUseAgent(t.Context(), f.actor, tt.definition)

			require.NoError(t, err)
			assert.Equal(t, tt.want, allowed)
		})
	}
}

func TestMayUseAgent_NobodyWithoutAnActor(t *testing.T) {
	t.Parallel()

	eng, _, _, _ := setupTestEngine(t)

	allowed, err := eng.MayUseAgent(t.Context(), nil, restrictedAgent())

	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestRoleCoverage_FullPartialAndNone(t *testing.T) {
	t.Parallel()

	eng, roleRepo, _, _ := setupTestEngine(t)
	orgID := pulid.MustNew("org_")
	base := pulid.MustNew("rol_")
	full := pulid.MustNew("rol_")
	partial := pulid.MustNew("rol_")
	none := pulid.MustNew("rol_")
	elsewhere := pulid.MustNew("rol_")

	grant := func(resource permission.Resource, ops ...permission.Operation) *permission.ResourcePermission {
		return &permission.ResourcePermission{
			Resource:   resource.String(),
			Operations: ops,
			DataScope:  permission.DataScopeOrganization,
		}
	}

	requested := []pulid.ID{full, partial, none, elsewhere}
	roleRepo.On("GetRolesWithInheritance", mock.Anything, requested).Return([]*permission.Role{
		{
			ID: base, OrganizationID: orgID,
			Permissions: []*permission.ResourcePermission{
				grant(permission.ResourceAssistant, permission.OpRead, permission.OpCreate),
			},
		},
		{
			ID: full, OrganizationID: orgID, ParentRoleIDs: []pulid.ID{base},
			Permissions: []*permission.ResourcePermission{
				grant(permission.ResourceShipment, permission.OpRead, permission.OpUpdate),
				grant(permission.ResourceWorker, permission.OpRead),
			},
		},
		{
			ID: partial, OrganizationID: orgID, ParentRoleIDs: []pulid.ID{base},
			Permissions: []*permission.ResourcePermission{
				grant(permission.ResourceShipment, permission.OpRead),
			},
		},
		{
			ID: none, OrganizationID: orgID,
			Permissions: []*permission.ResourcePermission{
				grant(permission.ResourceShipment, permission.OpRead, permission.OpUpdate),
			},
		},
		{ID: elsewhere, OrganizationID: pulid.MustNew("org_"), ParentRoleIDs: []pulid.ID{base}},
	}, nil)

	coverage, err := eng.RoleCoverage(t.Context(), &services.RoleCoverageRequest{
		OrganizationID: orgID,
		RoleIDs:        requested,
		Required: []services.RequiredGrant{
			{Tool: "get_shipment", Resource: permission.ResourceShipment, Operation: permission.OpRead},
			{Tool: "hold_shipment", Resource: permission.ResourceShipment, Operation: permission.OpUpdate},
			{Tool: "get_worker", Resource: permission.ResourceWorker, Operation: permission.OpRead},
		},
	})
	require.NoError(t, err)
	require.Len(t, coverage, 3, "a role of another organization is not covered")

	byRole := make(map[pulid.ID]services.RoleCoverage, len(coverage))
	for _, entry := range coverage {
		byRole[entry.RoleID] = entry
	}

	assert.Equal(t, agentdefinition.CoverageFull, byRole[full].Coverage,
		"the assistant is inherited from the base role")
	assert.Empty(t, byRole[full].MissingResources)
	assert.Equal(t, agentdefinition.CoveragePartial, byRole[partial].Coverage)
	assert.Equal(t,
		[]permission.Resource{permission.ResourceShipment, permission.ResourceWorker},
		byRole[partial].MissingResources,
		"each missing resource is named once")
	assert.Equal(t, agentdefinition.CoverageNone, byRole[none].Coverage,
		"a role that cannot use the assistant covers nothing")
}
