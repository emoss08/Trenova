package agentsafetyservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentaccessservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePolicies struct {
	byName map[string]agentports.ToolPolicy
	order  []string
}

func newFakePolicies(policies ...agentports.ToolPolicy) *fakePolicies {
	out := &fakePolicies{byName: make(map[string]agentports.ToolPolicy, len(policies))}
	for _, policy := range policies {
		out.byName[policy.Name] = policy
		out.order = append(out.order, policy.Name)
	}

	return out
}

func (f *fakePolicies) Get(name string) (agentports.ToolPolicy, bool) {
	policy, ok := f.byName[name]

	return policy, ok
}

func (f *fakePolicies) All() []agentports.ToolPolicy {
	out := make([]agentports.ToolPolicy, 0, len(f.order))
	for _, name := range f.order {
		out = append(out, f.byName[name])
	}

	return out
}

type fakeHolder struct{}

func (fakeHolder) ToolSummaries(
	definition *agentdefinition.Definition,
) []agentdefinition.ToolSummary {
	out := make([]agentdefinition.ToolSummary, 0, len(definition.ToolNames)+1)
	for _, name := range append(definition.ToolNames, definition.ToolNames...) {
		out = append(out, agentdefinition.ToolSummary{Name: name})
	}

	return append(out, agentdefinition.ToolSummary{Name: "unregistered_tool"})
}

type fakeDefinitions struct {
	agentdefinitionports.AgentDefinitionRepository

	all   []*agentdefinition.Definition
	pages int
}

func (f *fakeDefinitions) List(
	_ context.Context,
	req *agentdefinitionports.ListAgentDefinitionRequest,
) (*pagination.ListResult[*agentdefinition.Definition], error) {
	f.pages++
	offset := req.Filter.Pagination.Offset
	end := min(offset+req.Filter.Pagination.Limit, len(f.all))
	if offset >= len(f.all) {
		return &pagination.ListResult[*agentdefinition.Definition]{Total: len(f.all)}, nil
	}

	return &pagination.ListResult[*agentdefinition.Definition]{
		Items: f.all[offset:end],
		Total: len(f.all),
	}, nil
}

func (f *fakeDefinitions) ListByIDs(
	_ context.Context,
	req agentdefinitionports.ListAgentDefinitionsByIDsRequest,
) ([]*agentdefinition.Definition, error) {
	out := make([]*agentdefinition.Definition, 0, len(req.IDs))
	for _, definition := range f.all {
		for _, id := range req.IDs {
			if definition.ID == id {
				out = append(out, definition)
			}
		}
	}

	return out, nil
}

type fakeControls struct {
	tenantports.AgentControlRepository

	control *tenant.AgentControl
}

func (f *fakeControls) GetOrCreate(
	context.Context,
	pagination.TenantInfo,
) (*tenant.AgentControl, error) {
	return f.control, nil
}

func policy(name string, kind agent.ToolKind, egress ...agent.EgressClass) agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          name,
		Kind:          kind,
		Scope:         agent.ToolScopeTenant,
		Resource:      permission.ResourceShipment,
		Operation:     permission.OpUpdate,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		Egress:        egress,
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     name + " rationale",
	}
}

func testService(defs *fakeDefinitions, control *tenant.AgentControl) *Service {
	personal := policy("home_add_widget", agent.ToolKindAction, agent.EgressPersonal)
	personal.Scope = agent.ToolScopeSelf

	return newService(serviceDeps{
		policies: newFakePolicies(
			policy("shipment_get", agent.ToolKindQuery, agent.EgressNone),
			policy("dispatch_assign_move", agent.ToolKindAction, agent.EgressInternal),
			policy("customer_send_email", agent.ToolKindAction, agent.EgressExternalRecipient),
			personal,
		),
		holder:      fakeHolder{},
		definitions: defs,
		controls:    &fakeControls{control: control},
		sensitive:   agentaccessservice.LeavesOrganizationRule(),
	})
}

func agentWith(name string, tools ...string) *agentdefinition.Definition {
	return &agentdefinition.Definition{
		ID:              pulid.MustNew("agdef_"),
		Name:            name,
		ToolNames:       tools,
		AutonomyCeiling: agent.TierAutoExecute,
		TriggerMode:     agentdefinition.TriggerChat,
		AccessMode:      agentdefinition.AccessEveryone,
	}
}

func TestPolicyViewsSayWhatEachToolNeeds(t *testing.T) {
	t.Parallel()

	svc := testService(&fakeDefinitions{}, &tenant.AgentControl{})

	email, ok := svc.ToolPolicy("customer_send_email")
	require.True(t, ok)
	assert.Equal(t, "Send customer email", email.Title)
	assert.True(t, email.Leaves)
	assert.Equal(t, agent.TierActWithApproval, email.Promotable)
	require.NotNil(t, email.Needs)
	assert.Equal(t, permission.ResourceShipment, email.Needs.Resource)
	assert.NotEmpty(t, email.Explanation)

	widget, ok := svc.ToolPolicy("home_add_widget")
	require.True(t, ok)
	assert.Nil(t, widget.Needs, "a tool on the caller's own records needs no grant")
	assert.False(t, widget.Leaves)
}

/*
Three representative agents: one that may do everything, one in simulation,
and one whose trust ledger earned a tier. Each held tool is answered twice,
before and after the run read outside text, and a tool the registry does not
know is left out rather than guessed at.
*/
func TestAssessAnswersEachHeldToolCleanAndTainted(t *testing.T) {
	t.Parallel()

	svc := testService(&fakeDefinitions{}, &tenant.AgentControl{})
	open := agentWith("Dispatcher", "shipment_get", "dispatch_assign_move", "customer_send_email")

	answers := svc.Assess(t.Context(), &agentports.AssessAgentSafetyRequest{
		Subject: &agentports.AgentSafetySubject{Agent: open, Control: &tenant.AgentControl{}},
	})
	require.Len(t, answers, 3)

	byName := make(map[string]agentports.AgentToolSafety, len(answers))
	for _, answer := range answers {
		byName[answer.PolicyName] = answer
	}

	assert.Equal(t, agent.AutonomyRunsOnItsOwn, byName["shipment_get"].Clean.Answer)
	assert.Equal(t, agent.AutonomyRunsOnItsOwn, byName["shipment_get"].Tainted.Answer)
	assert.Equal(t, agent.AutonomyRunsOnItsOwn, byName["dispatch_assign_move"].Clean.Answer)
	assert.Equal(t, agent.AutonomyRunsOnItsOwn, byName["dispatch_assign_move"].Tainted.Answer)
	assert.Equal(t, agent.AutonomyNeedsApproval, byName["customer_send_email"].Clean.Answer)
	assert.Equal(t, agent.AutonomyNeedsApproval, byName["customer_send_email"].Tainted.Answer)

	simulated := agentWith("Rehearsal", "dispatch_assign_move", "shipment_get")
	simulated.SimulationMode = true
	rehearsal := svc.Assess(t.Context(), &agentports.AssessAgentSafetyRequest{
		Subject: &agentports.AgentSafetySubject{Agent: simulated, Control: &tenant.AgentControl{}},
	})
	require.Len(t, rehearsal, 2)
	assert.Equal(t, agent.AutonomySimulated, rehearsal[0].Clean.Answer)
	assert.Equal(t, agent.AutonomyRunsOnItsOwn, rehearsal[1].Clean.Answer)

	earner := agentWith("Earner", "dispatch_assign_move")
	earner.ToolTiers = map[string]agent.AutonomyTier{"dispatch_assign_move": agent.TierActWithApproval}
	earned := svc.Assess(t.Context(), &agentports.AssessAgentSafetyRequest{
		Subject: &agentports.AgentSafetySubject{
			Agent:   earner,
			Control: &tenant.AgentControl{EarnedAutonomy: true, PromotionThreshold: 4},
		},
		Trust: []*agent.ToolTrust{{
			AgentDefinitionID: earner.ID,
			ToolName:          "dispatch_assign_move",
			Streak:            1,
			EarnedTier:        agent.TierActWithApproval,
		}},
	})
	require.Len(t, earned, 1)
	assert.Equal(t, agent.AutonomyNeedsApproval, earned[0].Clean.Answer)
	assert.True(t, earned[0].Clean.Earned)
	require.NotNil(t, earned[0].Clean.ApprovalsToNext)
	assert.Equal(t, 3, *earned[0].Clean.ApprovalsToNext)
}

func TestReachWarnsAboutAnOpenAgentWithSensitiveToolsAndAnAgentNobodyCanUse(t *testing.T) {
	t.Parallel()

	svc := testService(&fakeDefinitions{}, &tenant.AgentControl{})

	open := agentWith("Customer desk", "dispatch_assign_move", "customer_send_email")
	warnings := svc.ReachWarnings(&agentports.AgentReachRequest{
		Subject: &agentports.AgentSafetySubject{Agent: open},
	})
	require.Len(t, warnings, 1)
	assert.Equal(t, agentdefinition.ReachOpenWithSensitiveTools, warnings[0].Kind)
	assert.Equal(t, []string{"customer_send_email"}, warnings[0].Tools)

	restricted := agentWith("Customer desk", "customer_send_email")
	restricted.AccessMode = agentdefinition.AccessRoles
	assert.Equal(t, []agentports.AgentReachWarning{{
		Kind:  agentdefinition.ReachNoAudience,
		Tools: []string{},
	}}, svc.ReachWarnings(&agentports.AgentReachRequest{
		Subject: &agentports.AgentSafetySubject{Agent: restricted},
	}))

	assert.Empty(t, svc.ReachWarnings(&agentports.AgentReachRequest{
		Subject:      &agentports.AgentSafetySubject{Agent: restricted},
		GrantedRoles: 2,
	}))
}

func TestListSubjectsReadsEveryAgentOrTheOnesNamed(t *testing.T) {
	t.Parallel()

	all := make([]*agentdefinition.Definition, 0, 150)
	for range 150 {
		all = append(all, agentWith("Agent"))
	}
	defs := &fakeDefinitions{all: all}
	control := &tenant.AgentControl{ShadowMode: true}
	svc := testService(defs, control)

	subjects, err := svc.ListSubjects(t.Context(), &agentports.ListAgentSafetyRequest{})
	require.NoError(t, err)
	assert.Len(t, subjects, 150)
	assert.Equal(t, 2, defs.pages)
	assert.Same(t, control, subjects[0].Control)

	named, err := svc.ListSubjects(t.Context(), &agentports.ListAgentSafetyRequest{
		AgentIDs: []pulid.ID{all[3].ID, all[3].ID},
	})
	require.NoError(t, err)
	require.Len(t, named, 1)
	assert.Equal(t, all[3].ID, named[0].Agent.ID)

	none, err := svc.ListSubjects(t.Context(), &agentports.ListAgentSafetyRequest{
		AgentIDs: []pulid.ID{},
	})
	require.NoError(t, err)
	assert.Empty(t, none)

	tooMany := make([]pulid.ID, 0, MaxAgentIDs+1)
	for range MaxAgentIDs + 1 {
		tooMany = append(tooMany, pulid.MustNew("agdef_"))
	}
	_, err = svc.ListSubjects(t.Context(), &agentports.ListAgentSafetyRequest{AgentIDs: tooMany})
	require.Error(t, err)

	_, err = svc.ListSubjects(t.Context(), &agentports.ListAgentSafetyRequest{
		AgentIDs: []pulid.ID{pulid.Nil},
	})
	require.Error(t, err)
}

func TestHeldByKeysAreTheOnesTheClientLabels(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(t, []string{
		"agent_ceiling", "tool_max", "egress_class", "condition", "tainted",
		"tool_tier", "personal_exemption", "shadow_mode", "simulation_mode",
	}, []string{
		agenttoolpolicy.HeldByAgentCeiling,
		agenttoolpolicy.HeldByToolMax,
		agenttoolpolicy.HeldByEgressClass,
		agenttoolpolicy.HeldByCondition,
		agenttoolpolicy.HeldByTainted,
		agenttoolpolicy.HeldByToolTier,
		agenttoolpolicy.HeldByPersonalExemption,
		agenttoolpolicy.HeldByShadowMode,
		agenttoolpolicy.HeldBySimulationMode,
	})
}
