package agentcapabilityservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentcapabilityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeDefinitions struct {
	agentdefinitionports.AgentDefinitionRepository

	byID map[pulid.ID]*agentdefinition.Definition
}

func (f *fakeDefinitions) GetByID(
	_ context.Context,
	req agentdefinitionports.GetAgentDefinitionByIDRequest,
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

// fakeAgents saves a patch the way the definition service does, minus the
// database: the edit on a copy, validated, then stored.
type fakeAgents struct {
	agentdefinitionports.AgentDefinitionService

	definitions *fakeDefinitions
	patches     int
}

func (f *fakeAgents) Patch(
	_ context.Context,
	req *agentdefinitionports.PatchAgentDefinitionRequest,
	_ *permissionports.RequestActor,
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
	*agenttest.StubActionTool
}

func (t irreversibleTool) Policy() agenttoolports.ToolPolicy {
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
	permissions *agenttest.StubPermissions
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
		ToolNames:       []string{"invoice_list", "assign_biller", "invoice_post"},
		ToolTiers: map[string]agent.AutonomyTier{
			"assign_biller": agent.TierAutoExecute,
		},
		DisabledToolNames: []string{"invoice_void"},
		DelegateIDs:       []pulid.ID{delegate.ID},
		Version:           3,
	}
	agentDef.ApplyDefaults()

	definitions := &fakeDefinitions{byID: map[pulid.ID]*agentdefinition.Definition{
		agentDef.ID: agentDef, delegate.ID: delegate,
	}}
	agents := &fakeAgents{definitions: definitions}
	permissions := &agenttest.StubPermissions{Denied: map[string]bool{}}

	svc := New(Params{
		Logger:      zap.NewNop(),
		Definitions: definitions,
		Agents:      agents,
		Tools: &agenttest.StubActionRegistry{Tools: []agenttoolports.AgentTool{
			&agenttest.StubActionTool{ToolName: "assign_biller", Tier: agent.TierActWithApproval},
			irreversibleTool{&agenttest.StubActionTool{
				ToolName: "invoice_post", Tier: agent.TierActWithApproval,
			}},
			&agenttest.StubActionTool{ToolName: "invoice_void", Tier: agent.TierActWithApproval},
		}},
		QueryTools: &agenttest.StubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
			&agenttest.StubQueryTool{ToolName: "invoice_list"},
		}},
		Permissions: permissions,
	}).(*Service)

	return &fixture{
		svc: svc, agent: agentDef, delegate: delegate,
		definitions: definitions, agents: agents, permissions: permissions,
	}
}

func (f *fixture) actor() *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: f.agent.OrganizationID,
		BusinessUnitID: f.agent.BusinessUnitID,
	}
}

func (f *fixture) get(t *testing.T) *agentcapabilityports.AgentCapabilities {
	t.Helper()

	caps, err := f.svc.Get(t.Context(), &agentcapabilityports.GetAgentCapabilitiesRequest{
		AgentID: f.agent.ID,
	}, f.actor())
	require.NoError(t, err)

	return caps
}

func toolByKey(
	tools []agentcapabilityports.AgentCapabilityTool,
	key string,
) agentcapabilityports.AgentCapabilityTool {
	for _, tool := range tools {
		if tool.Key == key {
			return tool
		}
	}

	return agentcapabilityports.AgentCapabilityTool{}
}

func TestGet_MapsTiersToTheModesAPersonReads(t *testing.T) {
	t.Parallel()

	caps := newFixture().get(t)

	require.Len(t, caps.ReadTools, 1)
	read := caps.ReadTools[0]
	assert.Equal(t, "List invoices", read.Label)
	assert.Equal(t, agentcapabilityports.AgentCapabilityAllowed, read.Mode)
	assert.Equal(t, []agentcapabilityports.AgentCapabilityMode{
		agentcapabilityports.AgentCapabilityAllowed, agentcapabilityports.AgentCapabilityOff,
	}, read.AllowedModes, "a read never asks first")

	assign := toolByKey(caps.WriteTools, "assign_biller")
	assert.Equal(
		t,
		agentcapabilityports.AgentCapabilityAllowed,
		assign.Mode,
		"AutoExecute is Allowed",
	)
	assert.Len(t, assign.AllowedModes, 3)
	assert.Empty(t, assign.LockReason)

	post := toolByKey(caps.WriteTools, "invoice_post")
	assert.Equal(
		t,
		agentcapabilityports.AgentCapabilityAskFirst,
		post.Mode,
		"ActWithApproval is Ask first",
	)
	assert.Equal(t, "Always asks · this can't be undone", post.LockReason)
	assert.NotContains(t, post.AllowedModes, agentcapabilityports.AgentCapabilityAllowed)

	void := toolByKey(caps.WriteTools, "invoice_void")
	assert.Equal(
		t,
		agentcapabilityports.AgentCapabilityOff,
		void.Mode,
		"a switched-off tool is still offered",
	)

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
	assert.Equal(t, agentcapabilityports.AgentCapabilityAskFirst, assign.Mode,
		"the agent's ceiling holds the tool whatever its own tier says")
	assert.Equal(t, "Always asks · this agent asks before every change", assign.LockReason)
}

func TestGet_RefusedToSomeoneWhoMayNotUseTheAgent(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.permissions.Denied[permission.ResourceAssistant.String()+":"+string(permission.OpCreate)] = true

	_, err := f.svc.Get(t.Context(), &agentcapabilityports.GetAgentCapabilitiesRequest{
		AgentID: f.agent.ID,
	}, f.actor())
	require.Error(t, err)
}

func TestGet_ReadOnlyWithoutPermissionToUpdateAgents(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.permissions.Denied[permission.ResourceAgentDefinition.String()+":"+string(permission.OpUpdate)] = true

	assert.False(t, f.get(t).CanEdit)

	_, err := f.svc.Update(t.Context(), &agentcapabilityports.UpdateAgentCapabilitiesRequest{
		AgentID: f.agent.ID,
		Tools: []agentcapabilityports.AgentCapabilityToolChange{
			{Key: "assign_biller", Mode: agentcapabilityports.AgentCapabilityOff},
		},
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
	caps, err := f.svc.Update(t.Context(), &agentcapabilityports.UpdateAgentCapabilitiesRequest{
		AgentID: f.agent.ID,
		Version: 3,
		Enabled: &off,
		Tools: []agentcapabilityports.AgentCapabilityToolChange{
			{Key: "assign_biller", Mode: agentcapabilityports.AgentCapabilityAskFirst},
			{Key: "invoice_list", Mode: agentcapabilityports.AgentCapabilityOff},
			{Key: "invoice_void", Mode: agentcapabilityports.AgentCapabilityAllowed},
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
	assert.Equal(t, agent.TierAutoExecute, saved.ToolTiers["invoice_void"])
	assert.NotContains(t, saved.ToolNames, "invoice_list", "Off takes the tool from the grant")
	assert.Contains(t, saved.ToolNames, "invoice_void")
	assert.Equal(t, []string{"invoice_list"}, saved.DisabledToolNames)
	assert.Equal(t, 150, saved.DailyRunLimit)
	assert.Equal(t, 40, saved.MaxChangeItems)
	assert.True(t, saved.BusinessHoursOnly)

	assert.Equal(
		t,
		agentcapabilityports.AgentCapabilityOff,
		toolByKey(caps.ReadTools, "invoice_list").Mode,
	)
	assert.Equal(t, "Pay rates, people records", caps.Handoffs[0].Topic)
}

func TestUpdate_RefusesAModeTheRowDoesNotOffer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		change agentcapabilityports.AgentCapabilityToolChange
	}{
		{"a locked write cannot be allowed",
			agentcapabilityports.AgentCapabilityToolChange{
				Key:  "invoice_post",
				Mode: agentcapabilityports.AgentCapabilityAllowed,
			}},
		{"a read cannot ask first",
			agentcapabilityports.AgentCapabilityToolChange{
				Key:  "invoice_list",
				Mode: agentcapabilityports.AgentCapabilityAskFirst,
			}},
		{"a tool the agent never had",
			agentcapabilityports.AgentCapabilityToolChange{
				Key:  "delete_everything",
				Mode: agentcapabilityports.AgentCapabilityAllowed,
			}},
		{"a mode that does not exist",
			agentcapabilityports.AgentCapabilityToolChange{
				Key:  "assign_biller",
				Mode: "Sometimes",
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newFixture()
			_, err := f.svc.Update(
				t.Context(),
				&agentcapabilityports.UpdateAgentCapabilitiesRequest{
					AgentID: f.agent.ID,
					Tools:   []agentcapabilityports.AgentCapabilityToolChange{tt.change},
				},
				f.actor(),
			)
			require.Error(t, err)
			assert.Zero(t, f.agents.patches)
		})
	}
}

func TestLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Post invoice", Label("invoice_post"))
	assert.Equal(t, "List shipments", Label("shipment_list"))
	assert.Equal(t, "Place shipment hold", Label("shipment_place_hold"))
	assert.Equal(t, "Get billing queue item", Label("billing_get_queue_item"))
	assert.Equal(t, "Billing queue list", Label("billing.queue.list"))
	assert.Equal(t, "Ab", Label("AB"))
}

type bringingTool struct {
	*agenttest.StubActionTool
	brings []string
}

func (t bringingTool) Companions() []string { return t.brings }

func TestGet_ListsTheWritesAHeldToolBrings(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.agent.ToolNames = []string{"shipment_create"}
	f.agent.DisabledToolNames = []string{"masterdata_create_commodity"}
	f.svc.tools = &agenttest.StubActionRegistry{Tools: []agenttoolports.AgentTool{
		bringingTool{
			StubActionTool: &agenttest.StubActionTool{
				ToolName: "shipment_create", Tier: agent.TierPropose,
			},
			brings: []string{"customer_create", "masterdata_create_commodity"},
		},
		&agenttest.StubActionTool{ToolName: "customer_create", Tier: agent.TierPropose},
		&agenttest.StubActionTool{ToolName: "masterdata_create_commodity", Tier: agent.TierPropose},
	}}

	writes := f.get(t).WriteTools

	customer := toolByKey(writes, "customer_create")
	assert.Equal(t, "customer_create", customer.Key, "a companion is listed with its tool")
	assert.Equal(t, agentcapabilityports.AgentCapabilityAskFirst, customer.Mode)
	assert.Contains(t, customer.AllowedModes, agentcapabilityports.AgentCapabilityOff,
		"a companion can be switched off")
	assert.Equal(
		t,
		agentcapabilityports.AgentCapabilityOff,
		toolByKey(writes, "masterdata_create_commodity").Mode,
	)
}
