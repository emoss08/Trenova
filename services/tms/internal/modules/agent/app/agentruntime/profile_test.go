package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func profileRuntime(t *testing.T, agents ...*agentdefinition.Definition) *Service {
	t.Helper()

	query := &stubQueryRegistry{Tools: []agentports.AgentQueryTool{
		&agenttest.StubQueryTool{
			ToolName: "shipment_list",
			Desc:     "List shipments by status or dates.",
			Resource: permission.ResourceShipment,
		},
		&agenttest.StubQueryTool{
			ToolName: "invoice_list",
			Desc:     "List invoices by status or due date.",
			Resource: permission.ResourceInvoice,
		},
	}}
	action := &stubActionRegistry{Tools: []agentports.AgentTool{
		&agenttest.StubActionTool{ToolName: "invoice_post", Tier: agent.TierAutoExecute},
		&agenttest.StubActionTool{ToolName: "assign_move", Tier: agent.TierAutoExecute},
	}}
	byID := make(map[pulid.ID]*agentdefinition.Definition, len(agents))
	for _, definition := range agents {
		byID[definition.ID] = definition
	}

	return &Service{
		logger:      zap.NewNop(),
		queryTools:  query,
		actionTools: action,
		permissions: &agenttest.StubPermissions{},
		definitions: &listedDefinitions{byID: byID},
		catalog: agenttoolcatalog.NewFromRegistries(agenttoolcatalog.Params{
			QueryTools:  query,
			ActionTools: action,
		}),
	}
}

func conversationRequest(definition *agentdefinition.Definition) *agentports.RunRequest {
	return &agentports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		ThreadID:   pulid.MustNew("athr_"),
		Input:      "hello",
	}
}

func TestProfile_AConversationReachesTheToolsThePersonsOtherAgentsHold(t *testing.T) {
	t.Parallel()

	dispatch := agentHolding("Dispatch helper", "shipment_list", "assign_move")
	billing := agentHolding("Billing helper", "invoice_list", "invoice_post")
	billing.AutonomyCeiling = agent.TierActWithApproval
	service := profileRuntime(t, dispatch, billing)

	profile := service.profileFor(t.Context(), conversationRequest(dispatch), nil)

	assert.Equal(t, ProfileConversation, profile.mode)
	assert.Equal(t, agent.TierAutoExecute, profile.beyond["invoice_list"])
	assert.Equal(t, agent.TierActWithApproval, profile.beyond["invoice_post"],
		"a reached write runs no higher than the agent holding it would run it")
	assert.NotContains(t, profile.beyond, "shipment_list", "its own tools are held, not reached")
	assert.NotContains(t, profile.beyond, "assign_move")
}

func TestProfile_ReachTakesTheMostPermissiveHolder(t *testing.T) {
	t.Parallel()

	dispatch := agentHolding("Dispatch helper", "shipment_list")
	cautious := agentHolding("Billing reviewer", "invoice_post")
	cautious.AutonomyCeiling = agent.TierPropose
	trusted := agentHolding("Billing clerk", "invoice_post")
	trusted.AutonomyCeiling = agent.TierActWithApproval
	service := profileRuntime(t, dispatch, cautious, trusted)

	profile := service.profileFor(t.Context(), conversationRequest(dispatch), nil)

	assert.Equal(t, agent.TierActWithApproval, profile.beyond["invoice_post"])
}

func TestProfile_ReachLeavesOutTheWritesOfAnAgentInSimulation(t *testing.T) {
	t.Parallel()

	dispatch := agentHolding("Dispatch helper", "shipment_list")
	billing := agentHolding("Billing helper", "invoice_list", "invoice_post")
	billing.SimulationMode = true
	service := profileRuntime(t, dispatch, billing)

	profile := service.profileFor(t.Context(), conversationRequest(dispatch), nil)

	assert.Contains(t, profile.beyond, "invoice_list", "a read changes nothing, simulated or not")
	assert.NotContains(t, profile.beyond, "invoice_post",
		"a write held only by an agent in simulation would otherwise run for real")
}

func TestProfile_ReachLeavesOutWhatTheAgentSwitchedOff(t *testing.T) {
	t.Parallel()

	dispatch := agentHolding("Dispatch helper", "shipment_list")
	dispatch.DisabledToolNames = []string{"invoice_post"}
	billing := agentHolding("Billing helper", "invoice_list", "invoice_post")
	service := profileRuntime(t, dispatch, billing)

	profile := service.profileFor(t.Context(), conversationRequest(dispatch), nil)

	assert.Contains(t, profile.beyond, "invoice_list")
	assert.NotContains(t, profile.beyond, "invoice_post")
}

func TestProfile_OnlyAConversationWithAPersonReachesBeyondItsAgent(t *testing.T) {
	t.Parallel()

	dispatch := agentHolding("Dispatch helper", "shipment_list")
	billing := agentHolding("Billing helper", "invoice_list")
	service := profileRuntime(t, dispatch, billing)

	cases := map[string]func(*agentports.RunRequest){
		"unattended": func(req *agentports.RunRequest) { req.Unattended = true },
		"delegated": func(req *agentports.RunRequest) {
			req.Delegation = &agentports.Delegation{ParentAgentID: billing.ID}
		},
		"inline":    func(req *agentports.RunRequest) { req.Inline = true },
		"no thread": func(req *agentports.RunRequest) { req.ThreadID = pulid.Nil },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := conversationRequest(dispatch)
			change(req)

			assert.Empty(t, service.profileFor(t.Context(), req, nil).beyond)
		})
	}
}

func TestProfile_ALookupHandedOutHoldsOnlyWhatReads(t *testing.T) {
	t.Parallel()

	billing := agentHolding("Billing helper", "invoice_list", "invoice_post")
	service := profileRuntime(t, billing)
	req := conversationRequest(billing)
	req.Delegation = &agentports.Delegation{ParentAgentID: pulid.MustNew("agdef_"), ReadOnly: true}

	profile := service.profileFor(t.Context(), req, nil)

	assert.Equal(t, ProfileFanOut, profile.mode)
	assert.Contains(t, profile.held, "invoice_list")
	assert.NotContains(t, profile.held, "invoice_post")
	assert.NotContains(t, profile.held, agentdefinition.CoreToolRemember)
	assert.NotContains(t, profile.held, delegateTaskName)
}

func TestProfile_ATaskThePersonDirectedKeepsItsWrites(t *testing.T) {
	t.Parallel()

	billing := agentHolding("Billing helper", "invoice_list", "invoice_post")
	service := profileRuntime(t, billing)
	req := conversationRequest(billing)
	req.Delegation = &agentports.Delegation{ParentAgentID: pulid.MustNew("agdef_")}

	profile := service.profileFor(t.Context(), req, nil)

	assert.Equal(t, ProfileDelegated, profile.mode)
	assert.Contains(t, profile.held, "invoice_post")
	assert.NotContains(t, profile.held, agentdefinition.CoreToolOpenPage)
}

func TestOpenTurn_AReachedWriteIsDecidedUnderItsHoldersCeiling(t *testing.T) {
	t.Parallel()

	dispatch := agentHolding("Dispatch helper", "shipment_list")
	dispatch.AutonomyCeiling = agent.TierAutoExecute
	billing := agentHolding("Billing helper", "invoice_post")
	billing.AutonomyCeiling = agent.TierPropose
	service := profileRuntime(t, dispatch, billing)

	turn := service.OpenTurn(t.Context(), conversationRequest(dispatch))

	require.True(t, turn.holds("invoice_post"))
	call := turn.dispatchCall(&agentports.ToolCall{ID: "call_1", Name: "invoice_post"}, "")
	assert.Equal(t, agent.TierPropose, call.Beyond)
	assert.Empty(t, turn.dispatchCall(
		&agentports.ToolCall{ID: "call_2", Name: "shipment_list"}, "",
	).Beyond, "a tool the agent holds is decided as its own")

	decision := agenttoolpolicy.Decide(t.Context(), agenttoolpolicy.DecideInput{
		Policy:        (&agenttest.StubActionTool{ToolName: "invoice_post", Tier: agent.TierAutoExecute}).Policy(),
		Definition:    dispatch,
		HolderCeiling: call.Beyond,
	})
	assert.Equal(t, agent.TierPropose, decision.Tier)
	assert.Contains(t, decision.HeldBy, agenttoolpolicy.HeldByAgentCeiling)
}

func TestTurnState_KeepsWhatTheTurnReaches(t *testing.T) {
	t.Parallel()

	dispatch := agentHolding("Dispatch helper", "shipment_list")
	billing := agentHolding("Billing helper", "invoice_list")
	service := profileRuntime(t, dispatch, billing)
	req := conversationRequest(dispatch)

	restored := service.RestoreTurn(req, service.OpenTurn(t.Context(), req).State())

	assert.True(t, restored.holds("invoice_list"))
	assert.Equal(t, ProfileConversation, restored.Mode())
	assert.Contains(t, restored.ToolsState().Beyond, "invoice_list")
}

func TestDispatchStep_RunsAReachedWriteUnderItsHoldersCeiling(t *testing.T) {
	t.Parallel()

	dispatch := agentHolding("Dispatch helper", "shipment_list")
	dispatch.AutonomyCeiling = agent.TierAutoExecute
	service := profileRuntime(t)
	service.arguments = toolschema.NewValidator()
	req := conversationRequest(dispatch)
	call := agentports.ToolCall{ID: "call_1", Name: "invoice_post", Arguments: map[string]any{}}

	refused := service.DispatchStep(t.Context(), req, DispatchCall{Call: call})
	reached := service.DispatchStep(t.Context(), req, DispatchCall{
		Call:   call,
		Beyond: agent.TierPropose,
	})

	assert.True(t, refused.Failed, "a write the agent neither holds nor reaches is refused")
	require.False(t, reached.Failed, reached.Content)
	require.NotNil(t, reached.Action)
	assert.Equal(t, agent.TierPropose, reached.Action.Tier)
	assert.False(t, reached.Action.Executed)
}
