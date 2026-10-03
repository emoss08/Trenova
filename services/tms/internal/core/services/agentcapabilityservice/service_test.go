package agentcapabilityservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeDefinitions struct {
	repositories.AgentDefinitionRepository

	byID map[pulid.ID]*agentdefinition.Definition
}

func (f *fakeDefinitions) GetByID(
	_ context.Context,
	req repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	definition, ok := f.byID[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Agent not found")
	}
	copied := *definition

	return &copied, nil
}

func (f *fakeDefinitions) ListByIDs(
	_ context.Context,
	req repositories.ListAgentDefinitionsByIDsRequest,
) ([]*agentdefinition.Definition, error) {
	out := make([]*agentdefinition.Definition, 0, len(req.IDs))
	for _, id := range req.IDs {
		if definition, ok := f.byID[id]; ok {
			out = append(out, definition)
		}
	}

	return out, nil
}

// fakeAgents saves a patch the way the definition service does, minus the
// database: the edit on a copy, validated, then stored.
type fakeAgents struct {
	services.AgentDefinitionService

	definitions *fakeDefinitions
	patches     int
}

func (f *fakeAgents) Patch(
	_ context.Context,
	req *services.PatchAgentDefinitionRequest,
	_ *services.RequestActor,
) (*agentdefinition.Definition, error) {
	f.patches++
	updated := *f.definitions.byID[req.ID]
	if err := req.Edit(&updated); err != nil {
		return nil, err
	}
	updated.ForgetReheldTools()
	updated.PruneDelegateTopics()
	multiErr := errortypes.NewMultiError()
	updated.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	updated.Version++
	f.definitions.byID[req.ID] = &updated

	return &updated, nil
}

type irreversibleTool struct {
	*agentruntimetest.StubActionTool
}

func (t irreversibleTool) Policy() services.ToolPolicy {
	policy := t.StubActionTool.Policy()
	policy.Reversible = false

	return policy
}

type fixture struct {
	svc         *Service
	agent       *agentdefinition.Definition
	delegate    *agentdefinition.Definition
	definitions *fakeDefinitions
	agents      *fakeAgents
	permissions *agentruntimetest.StubPermissions
}

func newFixture() *fixture {
	tenantOrg := pulid.MustNew("org_")
	tenantBU := pulid.MustNew("bu_")
	delegate := &agentdefinition.Definition{
		ID:             pulid.MustNew(agentdefinition.IDPrefix),
		OrganizationID: tenantOrg,
		BusinessUnitID: tenantBU,
		Name:           "Workforce Coordinator",
		Description:    "People records and pay",
		Enabled:        true,
	}
	delegate.ApplyDefaults()
	agentDef := &agentdefinition.Definition{
		ID:              pulid.MustNew(agentdefinition.IDPrefix),
		OrganizationID:  tenantOrg,
		BusinessUnitID:  tenantBU,
		Name:            "Billing Specialist",
		Description:     "Billing",
		Enabled:         true,
		AutonomyCeiling: agent.TierAutoExecute,
		ToolNames:       []string{"list_invoices", "assign_biller", "post_invoices"},
		ToolTiers: map[string]agent.AutonomyTier{
			"assign_biller": agent.TierAutoExecute,
		},
		DisabledToolNames: []string{"void_invoice"},
		DelegateIDs:       []pulid.ID{delegate.ID},
		Version:           3,
	}
	agentDef.ApplyDefaults()

	definitions := &fakeDefinitions{byID: map[pulid.ID]*agentdefinition.Definition{
		agentDef.ID: agentDef, delegate.ID: delegate,
	}}
	agents := &fakeAgents{definitions: definitions}
	permissions := &agentruntimetest.StubPermissions{Denied: map[string]bool{}}

	svc := New(Params{
		Logger:      zap.NewNop(),
		Definitions: definitions,
		Agents:      agents,
		Tools: &agentruntimetest.StubActionRegistry{Tools: []services.AgentTool{
			&agentruntimetest.StubActionTool{ToolName: "assign_biller", Tier: agent.TierActWithApproval},
			irreversibleTool{&agentruntimetest.StubActionTool{
				ToolName: "post_invoices", Tier: agent.TierActWithApproval,
			}},
			&agentruntimetest.StubActionTool{ToolName: "void_invoice", Tier: agent.TierActWithApproval},
		}},
		QueryTools: &agentruntimetest.StubQueryRegistry{Tools: []services.AgentQueryTool{
			&agentruntimetest.StubQueryTool{ToolName: "list_invoices"},
		}},
		Permissions: permissions,
	}).(*Service)

	return &fixture{
		svc: svc, agent: agentDef, delegate: delegate,
		definitions: definitions, agents: agents, permissions: permissions,
	}
}

func (f *fixture) actor() *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: f.agent.OrganizationID,
		BusinessUnitID: f.agent.BusinessUnitID,
	}
}

func (f *fixture) get(t *testing.T) *services.AgentCapabilities {
	t.Helper()

	caps, err := f.svc.Get(t.Context(), &services.GetAgentCapabilitiesRequest{
		AgentID: f.agent.ID,
	}, f.actor())
	require.NoError(t, err)

	return caps
}

func toolByKey(tools []services.AgentCapabilityTool, key string) services.AgentCapabilityTool {
	for _, tool := range tools {
		if tool.Key == key {
			return tool
		}
	}

	return services.AgentCapabilityTool{}
}

func TestGet_MapsTiersToTheModesAPersonReads(t *testing.T) {
	t.Parallel()

	caps := newFixture().get(t)

	require.Len(t, caps.ReadTools, 1)
	read := caps.ReadTools[0]
	assert.Equal(t, "List invoices", read.Label)
	assert.Equal(t, services.AgentCapabilityAllowed, read.Mode)
	assert.Equal(t, []services.AgentCapabilityMode{
		services.AgentCapabilityAllowed, services.AgentCapabilityOff,
	}, read.AllowedModes, "a read never asks first")

	assign := toolByKey(caps.WriteTools, "assign_biller")
	assert.Equal(t, services.AgentCapabilityAllowed, assign.Mode, "AutoExecute is Allowed")
	assert.Len(t, assign.AllowedModes, 3)
	assert.Empty(t, assign.LockReason)

	post := toolByKey(caps.WriteTools, "post_invoices")
	assert.Equal(t, services.AgentCapabilityAskFirst, post.Mode, "ActWithApproval is Ask first")
	assert.Equal(t, "Always asks · this can't be undone", post.LockReason)
	assert.NotContains(t, post.AllowedModes, services.AgentCapabilityAllowed)

	void := toolByKey(caps.WriteTools, "void_invoice")
	assert.Equal(t, services.AgentCapabilityOff, void.Mode, "a switched-off tool is still offered")

	require.Len(t, caps.Handoffs, 1)
	assert.Equal(t, "People records and pay", caps.Handoffs[0].Topic,
		"without a topic the delegate's description says what goes to it")
	assert.True(t, caps.CanEdit)
	assert.Equal(t, agentdefinition.DefaultMaxChangeItems, caps.Limits.MaxChangeItems)
}

func TestGet_AnAgentThatAlwaysAsksLocksAllowed(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.agent.AutonomyCeiling = agent.TierPropose

	assign := toolByKey(f.get(t).WriteTools, "assign_biller")
	assert.Equal(t, services.AgentCapabilityAskFirst, assign.Mode,
		"the agent's ceiling holds the tool whatever its own tier says")
	assert.Equal(t, "Always asks · this agent asks before every change", assign.LockReason)
}

func TestGet_RefusedToSomeoneWhoMayNotUseTheAgent(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.permissions.Denied[permission.ResourceAssistant.String()+":"+string(permission.OpCreate)] = true

	_, err := f.svc.Get(t.Context(), &services.GetAgentCapabilitiesRequest{
		AgentID: f.agent.ID,
	}, f.actor())
	require.Error(t, err)
}

func TestGet_ReadOnlyWithoutPermissionToUpdateAgents(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.permissions.Denied[permission.ResourceAgentDefinition.String()+":"+string(permission.OpUpdate)] = true

	assert.False(t, f.get(t).CanEdit)

	_, err := f.svc.Update(t.Context(), &services.UpdateAgentCapabilitiesRequest{
		AgentID: f.agent.ID,
		Tools:   []services.AgentCapabilityToolChange{{Key: "assign_biller", Mode: services.AgentCapabilityOff}},
	}, f.actor())
	require.Error(t, err)
	assert.Zero(t, f.agents.patches, "nothing is saved for a reader")
}

func TestUpdate_ChangesModesAndLimits(t *testing.T) {
	t.Parallel()

	f := newFixture()
	off := false
	limit := 150
	items := 40
	hours := true
	caps, err := f.svc.Update(t.Context(), &services.UpdateAgentCapabilitiesRequest{
		AgentID: f.agent.ID,
		Version: 3,
		Enabled: &off,
		Tools: []services.AgentCapabilityToolChange{
			{Key: "assign_biller", Mode: services.AgentCapabilityAskFirst},
			{Key: "list_invoices", Mode: services.AgentCapabilityOff},
			{Key: "void_invoice", Mode: services.AgentCapabilityAllowed},
		},
		DailyRequestLimit: &limit,
		MaxChangeItems:    &items,
		BusinessHoursOnly: &hours,
		DelegateTopics:    map[pulid.ID]string{f.delegate.ID: "Pay rates, people records"},
	}, f.actor())
	require.NoError(t, err)

	saved := f.definitions.byID[f.agent.ID]
	assert.False(t, saved.Enabled)
	assert.Equal(t, agent.TierActWithApproval, saved.ToolTiers["assign_biller"])
	assert.Equal(t, agent.TierAutoExecute, saved.ToolTiers["void_invoice"])
	assert.NotContains(t, saved.ToolNames, "list_invoices", "Off takes the tool from the grant")
	assert.Contains(t, saved.ToolNames, "void_invoice")
	assert.Equal(t, []string{"list_invoices"}, saved.DisabledToolNames)
	assert.Equal(t, 150, saved.DailyRunLimit)
	assert.Equal(t, 40, saved.MaxChangeItems)
	assert.True(t, saved.BusinessHoursOnly)

	assert.Equal(t, services.AgentCapabilityOff, toolByKey(caps.ReadTools, "list_invoices").Mode)
	assert.Equal(t, "Pay rates, people records", caps.Handoffs[0].Topic)
}

func TestUpdate_RefusesAModeTheRowDoesNotOffer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		change services.AgentCapabilityToolChange
	}{
		{"a locked write cannot be allowed",
			services.AgentCapabilityToolChange{Key: "post_invoices", Mode: services.AgentCapabilityAllowed}},
		{"a read cannot ask first",
			services.AgentCapabilityToolChange{Key: "list_invoices", Mode: services.AgentCapabilityAskFirst}},
		{"a tool the agent never had",
			services.AgentCapabilityToolChange{Key: "delete_everything", Mode: services.AgentCapabilityAllowed}},
		{"a mode that does not exist",
			services.AgentCapabilityToolChange{Key: "assign_biller", Mode: "Sometimes"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newFixture()
			_, err := f.svc.Update(t.Context(), &services.UpdateAgentCapabilitiesRequest{
				AgentID: f.agent.ID,
				Tools:   []services.AgentCapabilityToolChange{tt.change},
			}, f.actor())
			require.Error(t, err)
			assert.Zero(t, f.agents.patches)
		})
	}
}

func TestLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Post invoices", Label("post_invoices"))
	assert.Equal(t, "Billing queue list", Label("billing.queue.list"))
	assert.Equal(t, "Ab", Label("AB"))
}
