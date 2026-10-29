package agentaccessservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreach"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// heldRuntime says an agent holds exactly the tools its definition names, so
// a preview is seen to read the form's tools rather than the saved ones.
type heldRuntime struct {
	agentruntimeports.AgentRuntime
}

func (heldRuntime) ToolSummaries(
	definition *agentdefinition.Definition,
) []agentturn.ToolSummary {
	out := make([]agentturn.ToolSummary, 0, len(definition.ToolNames))
	for _, name := range definition.ToolNames {
		out = append(out, agentturn.ToolSummary{Name: name})
	}

	return out
}

type coveringPermissions struct {
	*checkingPermissions
	required [][]permissionports.RequiredGrant
}

func (p *coveringPermissions) RoleCoverage(
	_ context.Context,
	req *permissionports.RoleCoverageRequest,
) ([]permissionports.RoleCoverage, error) {
	p.required = append(p.required, req.Required)
	out := make([]permissionports.RoleCoverage, 0, len(req.RoleIDs))
	for _, id := range req.RoleIDs {
		out = append(out, permissionports.RoleCoverage{
			RoleID:           id,
			Coverage:         agentreach.CoverageFull,
			MissingResources: []permission.Resource{},
		})
	}

	return out, nil
}

type grantsByAgent struct {
	*fakeGrants
	reads int
}

func (g *grantsByAgent) ListRolesByAgents(
	_ context.Context,
	req permissionports.ListGrantsByAgentsRequest,
) (map[pulid.ID][]*rbac.Role, error) {
	g.reads++
	out := make(map[pulid.ID][]*rbac.Role, len(req.AgentIDs))
	for _, agentID := range req.AgentIDs {
		for _, roleID := range g.agentRoles[agentID] {
			out[agentID] = append(out[agentID], g.roles[roleID])
		}
	}

	return out, nil
}

type previewFixture struct {
	*fixture
	coverage *coveringPermissions
	grants   *grantsByAgent
}

func newPreviewFixture(t *testing.T) *previewFixture {
	t.Helper()

	f := newFixture(t)
	coverage := &coveringPermissions{checkingPermissions: f.permissions}
	grants := &grantsByAgent{fakeGrants: f.grants}
	f.service.permissions = coverage
	f.service.grants = grants
	f.service.runtime = heldRuntime{}
	f.service.policies = policyTable{
		"shipment_get": {Name: "shipment_get", Resource: "shipment"},
		"customer_send_email": {
			Name:     "customer_send_email",
			Resource: "shipment",
			Egress:   []agent.EgressClass{agent.EgressExternalRecipient},
		},
	}
	f.agent.ToolNames = []string{"shipment_get"}
	f.agent.AccessMode = agentdefinition.AccessRoles

	return &previewFixture{fixture: f, coverage: coverage, grants: grants}
}

/*
The warning follows the form, not the saved agent. An agent saved restricted
and switched back to Everyone on screen is warned about at once, for the tools
the form holds now, including one added since the save; restricting it again on
screen clears the warning.
*/
func TestPreviewAudience_ReadsTheFormNotTheSavedAgent(t *testing.T) {
	t.Parallel()

	f := newPreviewFixture(t)
	f.grants.agentRoles[f.agent.ID] = []pulid.ID{f.roleA.ID}

	open, err := f.service.PreviewAudience(
		t.Context(),
		&agentdefinitionports.PreviewAgentAudienceRequest{
			TenantInfo: f.tenant,
			AgentID:    f.agent.ID,
			ToolNames:  []string{"shipment_get", "customer_send_email"},
			Mode:       agentdefinition.AccessEveryone,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, agentdefinition.AccessEveryone, open.Agent.EffectiveAccessMode())
	assert.Equal(t, []string{"customer_send_email"}, open.SensitiveTools)
	assert.Equal(t, 1, f.grants.reads, "an agent being edited says which roles hold it now")
	for _, role := range open.Roles {
		assert.Equal(t, role.Role.ID == f.roleA.ID, role.Granted, role.Role.Name)
	}

	stored, err := f.definitions.GetByID(
		t.Context(),
		agentdefinitionports.GetAgentDefinitionByIDRequest{
			ID: f.agent.ID, TenantInfo: f.tenant,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, agentdefinition.AccessRoles, stored.AccessMode, "a preview changes nothing")
	assert.Equal(t, []string{"shipment_get"}, stored.ToolNames)
	assert.Empty(t, f.definitions.modeSets)

	restricted, err := f.service.PreviewAudience(
		t.Context(),
		&agentdefinitionports.PreviewAgentAudienceRequest{
			TenantInfo: f.tenant,
			AgentID:    f.agent.ID,
			ToolNames:  []string{"shipment_get", "customer_send_email"},
			Mode:       agentdefinition.AccessRoles,
		},
	)
	require.NoError(t, err)
	assert.Empty(t, restricted.SensitiveTools)
}

// A new agent has nothing saved to read: the preview holds only the form's
// tools, and no role is granted it yet.
func TestPreviewAudience_WorksForAnAgentNotYetSaved(t *testing.T) {
	t.Parallel()

	f := newPreviewFixture(t)

	preview, err := f.service.PreviewAudience(
		t.Context(),
		&agentdefinitionports.PreviewAgentAudienceRequest{
			TenantInfo: f.tenant,
			ToolNames:  []string{" customer_send_email ", "customer_send_email", "", "not_a_tool"},
			Mode:       agentdefinition.AccessEveryone,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"customer_send_email"}, preview.SensitiveTools)
	assert.Zero(t, f.grants.reads, "a new agent has no grants to read")
	require.Len(t, preview.Roles, 2)
	for _, role := range preview.Roles {
		assert.False(t, role.Granted)
	}
	require.Len(t, f.coverage.required, 1)
	assert.Equal(t, []permissionports.RequiredGrant{
		{Tool: "customer_send_email", Resource: "shipment", Operation: permission.OpRead},
	}, f.coverage.required[0])
}

// A system agent is open to everyone whatever the form says.
func TestPreviewAudience_ASystemAgentStaysOpen(t *testing.T) {
	t.Parallel()

	f := newPreviewFixture(t)

	preview, err := f.service.PreviewAudience(
		t.Context(),
		&agentdefinitionports.PreviewAgentAudienceRequest{
			TenantInfo: f.tenant,
			AgentID:    f.system.ID,
			ToolNames:  []string{"customer_send_email"},
			Mode:       agentdefinition.AccessRoles,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, agentdefinition.AccessEveryone, preview.Agent.EffectiveAccessMode())
	assert.Equal(t, []string{"customer_send_email"}, preview.SensitiveTools)
}

func TestPreviewAudience_RefusesABadRequest(t *testing.T) {
	t.Parallel()

	f := newPreviewFixture(t)

	_, err := f.service.PreviewAudience(
		t.Context(),
		&agentdefinitionports.PreviewAgentAudienceRequest{
			TenantInfo: f.tenant,
			Mode:       "Somebody",
		},
	)
	var invalid *errortypes.Error
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "accessMode", invalid.Field)

	tooMany := make([]string, maxPreviewTools+1)
	for i := range tooMany {
		tooMany[i] = "shipment_get"
	}
	_, err = f.service.PreviewAudience(
		t.Context(),
		&agentdefinitionports.PreviewAgentAudienceRequest{
			TenantInfo: f.tenant,
			ToolNames:  tooMany,
			Mode:       agentdefinition.AccessEveryone,
		},
	)
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "toolNames", invalid.Field)

	_, err = f.service.PreviewAudience(
		t.Context(),
		&agentdefinitionports.PreviewAgentAudienceRequest{
			TenantInfo: f.tenant,
			AgentID:    pulid.MustNew("agdef_"),
			Mode:       agentdefinition.AccessEveryone,
		},
	)
	assert.True(t, errortypes.IsNotFoundError(err), "another tenant's agent is not found")
}
