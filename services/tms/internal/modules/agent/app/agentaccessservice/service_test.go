package agentaccessservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/internal/testutil/permchangetest"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type txDB struct {
	transactions int
}

func (d *txDB) Transact(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	d.transactions++
	return fn(ctx)
}

type fakeDefinitions struct {
	agentdefinitionports.AgentDefinitionRepository
	byID     map[pulid.ID]*agentdefinition.Definition
	modeSets []agentdefinitionports.SetAgentDefinitionAccessModeRequest
}

func (f *fakeDefinitions) GetByID(
	_ context.Context,
	req agentdefinitionports.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	definition, ok := f.byID[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError(
			"AgentDefinition not found within your organization",
		)
	}
	copied := *definition

	return &copied, nil
}

func (f *fakeDefinitions) ListByIDs(
	_ context.Context,
	req agentdefinitionports.ListAgentDefinitionsByIDsRequest,
) ([]*agentdefinition.Definition, error) {
	out := make([]*agentdefinition.Definition, 0, len(req.IDs))
	for _, id := range req.IDs {
		if definition, ok := f.byID[id]; ok {
			out = append(out, definition)
		}
	}

	return out, nil
}

func (f *fakeDefinitions) SetAccessMode(
	_ context.Context,
	req agentdefinitionports.SetAgentDefinitionAccessModeRequest,
) error {
	f.modeSets = append(f.modeSets, req)
	return nil
}

type fakeGrants struct {
	permissionports.RoleAgentGrantRepository
	roles        map[pulid.ID]*permission.Role
	agentRoles   map[pulid.ID][]pulid.ID
	agentReplace []permissionports.ReplaceAgentGrantsRequest
	roleReplace  []permissionports.ReplaceRoleGrantsRequest
	roleAgents   map[pulid.ID][]pulid.ID
}

func (f *fakeGrants) ListGrantableRoles(
	_ context.Context,
	req permissionports.ListGrantableRolesRequest,
) ([]*permission.Role, error) {
	if len(req.RoleIDs) == 0 {
		out := make([]*permission.Role, 0, len(f.roles))
		for _, role := range f.roles {
			out = append(out, role)
		}

		return out, nil
	}

	out := make([]*permission.Role, 0, len(req.RoleIDs))
	for _, id := range req.RoleIDs {
		if role, ok := f.roles[id]; ok {
			out = append(out, role)
		}
	}

	return out, nil
}

func (f *fakeGrants) ReplaceForAgent(
	_ context.Context,
	req permissionports.ReplaceAgentGrantsRequest,
) (permissionports.GrantChange, error) {
	f.agentReplace = append(f.agentReplace, req)
	previous := f.agentRoles[req.AgentID]
	f.agentRoles[req.AgentID] = req.RoleIDs

	return diff(previous, req.RoleIDs), nil
}

func (f *fakeGrants) ReplaceForRole(
	_ context.Context,
	req permissionports.ReplaceRoleGrantsRequest,
) (permissionports.GrantChange, error) {
	f.roleReplace = append(f.roleReplace, req)
	previous := f.roleAgents[req.RoleID]
	f.roleAgents[req.RoleID] = req.AgentIDs

	return diff(previous, req.AgentIDs), nil
}

func diff(previous, wanted []pulid.ID) permissionports.GrantChange {
	change := permissionports.GrantChange{Previous: previous}
	for _, id := range wanted {
		if !contains(previous, id) {
			change.Added = append(change.Added, id)
		}
	}
	for _, id := range previous {
		if !contains(wanted, id) {
			change.Removed = append(change.Removed, id)
		}
	}

	return change
}

func contains(ids []pulid.ID, id pulid.ID) bool {
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}

	return false
}

type checkingPermissions struct {
	permissionports.PermissionEngine
	denied map[string]bool
}

func (p *checkingPermissions) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	return &permissionports.PermissionCheckResult{
		Allowed: !p.denied[req.Resource+":"+string(req.Operation)],
	}, nil
}

type recordingAudit struct {
	auditports.AuditService
	logged []*auditports.LogActionParams
}

func (a *recordingAudit) LogAction(
	params *auditports.LogActionParams,
	_ ...auditports.LogOption,
) error {
	a.logged = append(a.logged, params)
	return nil
}

type fixture struct {
	service     *Service
	db          *txDB
	definitions *fakeDefinitions
	grants      *fakeGrants
	notifier    *permchangetest.Recorder
	permissions *checkingPermissions
	audit       *recordingAudit
	actor       *permissionports.RequestActor
	tenant      pagination.TenantInfo
	agent       *agentdefinition.Definition
	system      *agentdefinition.Definition
	roleA       *permission.Role
	roleB       *permission.Role
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	payroll := &agentdefinition.Definition{
		ID:             pulid.MustNew("agdef_"),
		Name:           "Payroll helper",
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		AccessMode:     agentdefinition.AccessEveryone,
	}
	system := &agentdefinition.Definition{
		ID:             pulid.MustNew("agdef_"),
		Name:           "Briefing",
		SystemKey:      "briefing",
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		AccessMode:     agentdefinition.AccessEveryone,
	}
	roleA := &permission.Role{
		ID:             pulid.MustNew("rol_"),
		Name:           "Payroll",
		OrganizationID: tenant.OrgID,
	}
	roleB := &permission.Role{
		ID:             pulid.MustNew("rol_"),
		Name:           "Finance",
		OrganizationID: tenant.OrgID,
	}

	f := &fixture{
		db: &txDB{},
		definitions: &fakeDefinitions{byID: map[pulid.ID]*agentdefinition.Definition{
			payroll.ID: payroll,
			system.ID:  system,
		}},
		grants: &fakeGrants{
			roles:      map[pulid.ID]*permission.Role{roleA.ID: roleA, roleB.ID: roleB},
			agentRoles: map[pulid.ID][]pulid.ID{},
			roleAgents: map[pulid.ID][]pulid.ID{},
		},
		notifier:    &permchangetest.Recorder{},
		permissions: &checkingPermissions{denied: map[string]bool{}},
		audit:       &recordingAudit{},
		actor: &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeUser,
			PrincipalID:    pulid.MustNew("usr_"),
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
		},
		tenant: tenant,
		agent:  payroll,
		system: system,
		roleA:  roleA,
		roleB:  roleB,
	}
	f.service = &Service{
		l:           zap.NewNop(),
		db:          f.db,
		definitions: f.definitions,
		grants:      f.grants,
		roles:       permissionportsmock.NewMockRoleRepository(t),
		notifier:    f.notifier,
		permissions: f.permissions,
		audit:       f.audit,
		sensitive:   DefaultSensitiveRule(permission.NewRegistry()),
	}

	return f
}

/*
A grant change reaches the people holding the role at once: the permission
cache of every role gained or lost is invalidated, exactly as a change to the
role's resource permissions is.
*/
func TestSetAgentAccess_RestrictsGrantsAndInvalidatesTheChangedRoles(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.grants.agentRoles[f.agent.ID] = []pulid.ID{f.roleB.ID}

	access, err := f.service.SetAgentAccess(t.Context(), &agentdefinitionports.SetAgentAccessRequest{
		TenantInfo: f.tenant,
		AgentID:    f.agent.ID,
		Mode:       agentdefinition.AccessRoles,
		RoleIDs:    []pulid.ID{f.roleA.ID},
	}, f.actor)

	require.NoError(t, err)
	assert.Equal(t, agentdefinition.AccessRoles, access.Agent.AccessMode)
	assert.Equal(t, []*permission.Role{f.roleA}, access.Roles)
	assert.Equal(t, 1, f.db.transactions, "the mode and the grants change together")
	require.Len(t, f.definitions.modeSets, 1)
	assert.Equal(t, agentdefinition.AccessRoles, f.definitions.modeSets[0].Mode)
	require.Len(t, f.grants.agentReplace, 1)
	assert.Equal(t, f.actor.UserID, f.grants.agentReplace[0].GrantedBy)

	resources := make([]permission.Resource, 0, len(f.audit.logged))
	for _, entry := range f.audit.logged {
		resources = append(resources, entry.Resource)
	}
	assert.ElementsMatch(t, []permission.Resource{
		permission.ResourceAgentDefinition,
		permission.ResourceRole,
		permission.ResourceRole,
	}, resources, "the agent and each role gained or lost are audited")
}

func TestSetAgentAccess_RefusesToRestrictASystemAgent(t *testing.T) {
	t.Parallel()

	for name, req := range map[string]*agentdefinitionports.SetAgentAccessRequest{
		"restricted to roles": {Mode: agentdefinition.AccessRoles},
		"granted a role":      {Mode: agentdefinition.AccessEveryone},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			req.TenantInfo = f.tenant
			req.AgentID = f.system.ID
			if req.Mode == agentdefinition.AccessEveryone {
				req.RoleIDs = []pulid.ID{f.roleA.ID}
			}

			_, err := f.service.SetAgentAccess(t.Context(), req, f.actor)

			var multiErr *errortypes.MultiError
			require.ErrorAs(t, err, &multiErr)
			assert.Empty(t, f.definitions.modeSets)
			assert.Empty(t, f.grants.agentReplace)
			assert.Empty(t, f.audit.logged)
		})
	}
}

func TestSetAgentAccess_NeedsAgentAndRoleUpdate(t *testing.T) {
	t.Parallel()

	for _, denied := range []string{"agent_definition:update", "role:update"} {
		t.Run(denied, func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			f.permissions.denied[denied] = true

			_, err := f.service.SetAgentAccess(t.Context(), &agentdefinitionports.SetAgentAccessRequest{
				TenantInfo: f.tenant,
				AgentID:    f.agent.ID,
				Mode:       agentdefinition.AccessRoles,
			}, f.actor)

			assert.True(t, errortypes.IsAuthorizationError(err))
			assert.Zero(t, f.db.transactions)
		})
	}
}

func TestSetAgentAccess_RefusesARoleOfAnotherTenant(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	_, err := f.service.SetAgentAccess(t.Context(), &agentdefinitionports.SetAgentAccessRequest{
		TenantInfo: f.tenant,
		AgentID:    f.agent.ID,
		Mode:       agentdefinition.AccessRoles,
		RoleIDs:    []pulid.ID{pulid.MustNew("rol_")},
	}, f.actor)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Empty(t, f.grants.agentReplace)
}

func TestSetAgentAccess_RefusesAnotherTenantsRequest(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	_, err := f.service.SetAgentAccess(t.Context(), &agentdefinitionports.SetAgentAccessRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: f.tenant.BuID},
		AgentID:    f.agent.ID,
		Mode:       agentdefinition.AccessRoles,
	}, f.actor)

	assert.True(t, errortypes.IsAuthorizationError(err))
}

// Taking the last role off a restricted agent leaves it restricted, usable by
// nobody, never silently open.
func TestSetAgentAccess_RemovingTheLastRoleKeepsItRestricted(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.agent.AccessMode = agentdefinition.AccessRoles
	f.grants.agentRoles[f.agent.ID] = []pulid.ID{f.roleA.ID}

	access, err := f.service.SetAgentAccess(t.Context(), &agentdefinitionports.SetAgentAccessRequest{
		TenantInfo: f.tenant,
		AgentID:    f.agent.ID,
		Mode:       agentdefinition.AccessRoles,
	}, f.actor)

	require.NoError(t, err)
	assert.Equal(t, agentdefinition.AccessRoles, access.Agent.AccessMode)
	assert.False(t, access.Agent.OpenToEveryone())
	assert.Empty(t, f.definitions.modeSets, "the mode did not change")
	assert.Empty(t, f.grants.agentRoles[f.agent.ID])
}

func TestSetRoleAgents_ReplacesAndInvalidatesTheRole(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	result, err := f.service.SetRoleAgents(t.Context(), &agentdefinitionports.SetRoleAgentsRequest{
		TenantInfo: f.tenant,
		RoleID:     f.roleA.ID,
		AgentIDs:   []pulid.ID{f.agent.ID},
	}, f.actor)

	require.NoError(t, err)
	assert.Equal(t, f.roleA, result.Role)
	assert.Equal(t, []*agentdefinition.Definition{f.agent}, result.Agents)
	require.Len(t, f.grants.roleReplace, 1)
	assert.Len(t, f.audit.logged, 2, "the role and the agent it gained are audited")
}

func TestSetRoleAgents_RefusesASystemAgentAndDuplicates(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	_, err := f.service.SetRoleAgents(t.Context(), &agentdefinitionports.SetRoleAgentsRequest{
		TenantInfo: f.tenant,
		RoleID:     f.roleA.ID,
		AgentIDs:   []pulid.ID{f.system.ID},
	}, f.actor)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)

	_, err = f.service.SetRoleAgents(t.Context(), &agentdefinitionports.SetRoleAgentsRequest{
		TenantInfo: f.tenant,
		RoleID:     f.roleA.ID,
		AgentIDs:   []pulid.ID{f.agent.ID, f.agent.ID},
	}, f.actor)
	require.ErrorAs(t, err, &multiErr)
	assert.Empty(t, f.grants.roleReplace)
}

func TestSetRoleAgents_NeedsRoleUpdate(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.permissions.denied["role:update"] = true

	_, err := f.service.SetRoleAgents(t.Context(), &agentdefinitionports.SetRoleAgentsRequest{
		TenantInfo: f.tenant,
		RoleID:     f.roleA.ID,
	}, f.actor)

	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestSetRoleAgents_UnknownRoleIsNotFound(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	_, err := f.service.SetRoleAgents(t.Context(), &agentdefinitionports.SetRoleAgentsRequest{
		TenantInfo: f.tenant,
		RoleID:     pulid.MustNew("rol_"),
	}, f.actor)

	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestInvalidate_AnnouncesOnceForAnyChangedRole(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	f.service.invalidate(t.Context(), f.tenant, []pulid.ID{f.roleA.ID, f.roleB.ID}, []pulid.ID{f.roleA.ID})

	require.Len(t, f.notifier.Organizations(), 1)
	assert.Equal(t, f.tenant.OrgID, f.notifier.Organizations()[0].OrgID)
}

func TestInvalidate_AnnouncesNothingWhenNoRoleChanged(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	f.service.invalidate(t.Context(), f.tenant, nil, []pulid.ID{})

	assert.Empty(t, f.notifier.Organizations())
}

type summarizingRuntime struct {
	agentports.AgentRuntime
	names []string
}

func (r summarizingRuntime) ToolSummaries(
	*agentdefinition.Definition,
) []agentdefinition.ToolSummary {
	out := make([]agentdefinition.ToolSummary, 0, len(r.names))
	for _, name := range r.names {
		out = append(out, agentdefinition.ToolSummary{Name: name})
	}

	return out
}

type policyTable map[string]agentports.ToolPolicy

func (p policyTable) Get(name string) (agentports.ToolPolicy, bool) {
	policy, ok := p[name]
	return policy, ok
}

/*
What an agent holds is read from each tool's declared policy: the resource and
operation it needs of the person, and where its work can go. A self-scoped tool
needs no grant but its egress still counts, and a name the catalog does not
know is left out.
*/
func TestHeldTools_ReadsEachToolsPolicy(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.service.runtime = summarizingRuntime{
		names: []string{"shipment_get", "customer_send_email", "agent_create_memory", "shipment_get", "gone"},
	}
	f.service.policies = policyTable{
		"shipment_get": {Name: "shipment_get", Resource: "shipment"},
		"customer_send_email": {
			Name:      "customer_send_email",
			Resource:  "shipment",
			Operation: permission.OpUpdate,
			Egress:    []agent.EgressClass{agent.EgressExternalRecipient},
		},
		"agent_create_memory": {
			Name:     "agent_create_memory",
			Resource: "agent_memory",
			Scope:    agent.ToolScopeSelf,
			Egress:   []agent.EgressClass{agent.EgressPersonal},
		},
	}

	held := f.service.HeldTools(f.agent)

	assert.Equal(t, []HeldTool{
		{Name: "shipment_get", Resource: "shipment", Operation: permission.OpRead},
		{
			Name:      "customer_send_email",
			Resource:  "shipment",
			Operation: permission.OpUpdate,
			Egress:    []agent.EgressClass{agent.EgressExternalRecipient},
		},
		{Name: "agent_create_memory", Egress: []agent.EgressClass{agent.EgressPersonal}},
	}, held)
	assert.Equal(t, []permissionports.RequiredGrant{
		{Tool: "shipment_get", Resource: "shipment", Operation: permission.OpRead},
		{Tool: "customer_send_email", Resource: "shipment", Operation: permission.OpUpdate},
	}, requiredGrants(held))
	assert.Equal(t, []string{"customer_send_email"}, SensitiveTools(held, f.service.sensitive))
}
