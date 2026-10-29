package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func profileRuntime(t *testing.T, agents ...*agentdefinition.Definition) *Service {
	t.Helper()

	query := &stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
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
	action := &stubActionRegistry{Tools: []agenttoolports.AgentTool{
		&agenttest.StubActionTool{ToolName: "invoice_post", Tier: agent.TierAutoExecute},
		&agenttest.StubActionTool{ToolName: "dispatch_assign_move", Tier: agent.TierAutoExecute},
	}}
	byID := make(map[pulid.ID]*agentdefinition.Definition, len(agents))
	for _, definition := range agents {
		byID[definition.ID] = definition
	}

	return buildRuntime(Params{
		Logger:      zap.NewNop(),
		QueryTools:  query,
		ActionTools: action,
		Permissions: &agenttest.StubPermissions{},
		Definitions: &runtimetest.ListedDefinitions{ByID: byID},
		Catalog: agenttoolcatalog.NewFromRegistries(agenttoolcatalog.Params{
			QueryTools:  query,
			ActionTools: action,
		}),
	})
}

func conversationRequest(definition *agentdefinition.Definition) *agentruntimeports.RunRequest {
	return &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		ThreadID:   pulid.MustNew("athr_"),
		Input:      "hello",
	}
}

func TestProfile_AConversationReachesTheToolsThePersonsOtherAgentsHold(t *testing.T) {
	t.Parallel()

	dispatch := runtimetest.AgentHolding("Dispatch helper", "shipment_list", "dispatch_assign_move")
	billing := runtimetest.AgentHolding("Billing helper", "invoice_list", "invoice_post")
	billing.AutonomyCeiling = agent.TierActWithApproval
	service := profileRuntime(t, dispatch, billing)

	profile := service.access.ProfileFor(t.Context(), conversationRequest(dispatch), nil)

	assert.Equal(t, toolaccess.ProfileConversation, profile.Mode)
	assert.Equal(t, agent.TierAutoExecute, profile.Beyond["invoice_list"])
	assert.Equal(t, agent.TierActWithApproval, profile.Beyond["invoice_post"],
		"a reached write runs no higher than the agent holding it would run it")
	assert.NotContains(t, profile.Beyond, "shipment_list", "its own tools are held, not reached")
	assert.NotContains(t, profile.Beyond, "dispatch_assign_move")
}

func TestProfile_ReachTakesTheMostPermissiveHolder(t *testing.T) {
	t.Parallel()

	dispatch := runtimetest.AgentHolding("Dispatch helper", "shipment_list")
	cautious := runtimetest.AgentHolding("Billing reviewer", "invoice_post")
	cautious.AutonomyCeiling = agent.TierPropose
	trusted := runtimetest.AgentHolding("Billing clerk", "invoice_post")
	trusted.AutonomyCeiling = agent.TierActWithApproval
	service := profileRuntime(t, dispatch, cautious, trusted)

	profile := service.access.ProfileFor(t.Context(), conversationRequest(dispatch), nil)

	assert.Equal(t, agent.TierActWithApproval, profile.Beyond["invoice_post"])
}

func TestProfile_ReachLeavesOutTheWritesOfAnAgentInSimulation(t *testing.T) {
	t.Parallel()

	dispatch := runtimetest.AgentHolding("Dispatch helper", "shipment_list")
	billing := runtimetest.AgentHolding("Billing helper", "invoice_list", "invoice_post")
	billing.SimulationMode = true
	service := profileRuntime(t, dispatch, billing)

	profile := service.access.ProfileFor(t.Context(), conversationRequest(dispatch), nil)

	assert.Contains(t, profile.Beyond, "invoice_list", "a read changes nothing, simulated or not")
	assert.NotContains(t, profile.Beyond, "invoice_post",
		"a write held only by an agent in simulation would otherwise run for real")
}

func TestProfile_ReachLeavesOutWhatTheAgentSwitchedOff(t *testing.T) {
	t.Parallel()

	dispatch := runtimetest.AgentHolding("Dispatch helper", "shipment_list")
	dispatch.DisabledToolNames = []string{"invoice_post"}
	billing := runtimetest.AgentHolding("Billing helper", "invoice_list", "invoice_post")
	service := profileRuntime(t, dispatch, billing)

	profile := service.access.ProfileFor(t.Context(), conversationRequest(dispatch), nil)

	assert.Contains(t, profile.Beyond, "invoice_list")
	assert.NotContains(t, profile.Beyond, "invoice_post")
}

func TestProfile_OnlyAConversationWithAPersonReachesBeyondItsAgent(t *testing.T) {
	t.Parallel()

	dispatch := runtimetest.AgentHolding("Dispatch helper", "shipment_list")
	billing := runtimetest.AgentHolding("Billing helper", "invoice_list")
	service := profileRuntime(t, dispatch, billing)

	cases := map[string]func(*agentruntimeports.RunRequest){
		"unattended": func(req *agentruntimeports.RunRequest) { req.Unattended = true },
		"delegated": func(req *agentruntimeports.RunRequest) {
			req.Delegation = &agentruntimeports.Delegation{ParentAgentID: billing.ID}
		},
		"inline":    func(req *agentruntimeports.RunRequest) { req.Inline = true },
		"no thread": func(req *agentruntimeports.RunRequest) { req.ThreadID = pulid.Nil },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := conversationRequest(dispatch)
			change(req)

			assert.Empty(t, service.access.ProfileFor(t.Context(), req, nil).Beyond)
		})
	}
}

func TestProfile_ALookupHandedOutHoldsOnlyWhatReads(t *testing.T) {
	t.Parallel()

	billing := runtimetest.AgentHolding("Billing helper", "invoice_list", "invoice_post")
	service := profileRuntime(t, billing)
	req := conversationRequest(billing)
	req.Delegation = &agentruntimeports.Delegation{
		ParentAgentID: pulid.MustNew("agdef_"),
		ReadOnly:      true,
	}

	profile := service.access.ProfileFor(t.Context(), req, nil)

	assert.Equal(t, toolaccess.ProfileFanOut, profile.Mode)
	assert.Contains(t, profile.Held, "invoice_list")
	assert.NotContains(t, profile.Held, "invoice_post")
	assert.NotContains(t, profile.Held, agentdefinition.CoreToolRemember)
	assert.NotContains(t, profile.Held, runtimetools.DelegateTaskName)
}

func TestProfile_ATaskThePersonDirectedKeepsItsWrites(t *testing.T) {
	t.Parallel()

	billing := runtimetest.AgentHolding("Billing helper", "invoice_list", "invoice_post")
	service := profileRuntime(t, billing)
	req := conversationRequest(billing)
	req.Delegation = &agentruntimeports.Delegation{ParentAgentID: pulid.MustNew("agdef_")}

	profile := service.access.ProfileFor(t.Context(), req, nil)

	assert.Equal(t, toolaccess.ProfileDelegated, profile.Mode)
	assert.Contains(t, profile.Held, "invoice_post")
	assert.NotContains(t, profile.Held, agentdefinition.CoreToolOpenPage)
}

func TestOpenTurn_AReachedWriteIsDecidedUnderItsHoldersCeiling(t *testing.T) {
	t.Parallel()

	dispatch := runtimetest.AgentHolding("Dispatch helper", "shipment_list")
	dispatch.AutonomyCeiling = agent.TierAutoExecute
	billing := runtimetest.AgentHolding("Billing helper", "invoice_post")
	billing.AutonomyCeiling = agent.TierPropose
	service := profileRuntime(t, dispatch, billing)

	turn := service.OpenTurn(t.Context(), conversationRequest(dispatch))

	require.True(t, turn.Holds("invoice_post"))
	call := turn.dispatchCall(&llmports.ToolCall{ID: "call_1", Name: "invoice_post"}, "")
	assert.Equal(t, agent.TierPropose, call.Beyond)
	assert.Empty(t, turn.dispatchCall(
		&llmports.ToolCall{ID: "call_2", Name: "shipment_list"}, "",
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

	dispatch := runtimetest.AgentHolding("Dispatch helper", "shipment_list")
	billing := runtimetest.AgentHolding("Billing helper", "invoice_list")
	service := profileRuntime(t, dispatch, billing)
	req := conversationRequest(dispatch)

	restored := service.RestoreTurn(req, service.OpenTurn(t.Context(), req).State())

	assert.True(t, restored.Holds("invoice_list"))
	assert.Equal(t, toolaccess.ProfileConversation, restored.Mode())
	assert.Contains(t, restored.ToolsState().Beyond, "invoice_list")
}

func TestDispatchStep_RunsAReachedWriteUnderItsHoldersCeiling(t *testing.T) {
	t.Parallel()

	dispatch := runtimetest.AgentHolding("Dispatch helper", "shipment_list")
	dispatch.AutonomyCeiling = agent.TierAutoExecute
	service := profileRuntime(t)
	req := conversationRequest(dispatch)
	call := llmports.ToolCall{ID: "call_1", Name: "invoice_post", Arguments: map[string]any{}}

	refused := service.DispatchStep(t.Context(), req, toolcall.Call{Call: call})
	reached := service.DispatchStep(t.Context(), req, toolcall.Call{
		Call:   call,
		Beyond: agent.TierPropose,
	})

	assert.True(t, refused.Failed, "a write the agent neither holds nor reaches is refused")
	require.False(t, reached.Failed, reached.Content)
	require.NotNil(t, reached.Action)
	assert.Equal(t, agent.TierPropose, reached.Action.Tier)
	assert.False(t, reached.Action.Executed)
}

/*
Scheduling a request repeats it in the person's own conversation, so only a
conversation with a person in it holds the tool: not a run nobody watches,
not a task another agent handed over, and not a turn with no conversation to
post into.
*/
func TestProfile_OnlyAPersonsConversationHoldsTheScheduleTool(t *testing.T) {
	t.Parallel()

	billing := runtimetest.AgentHolding("Billing helper", "invoice_list")
	service := profileRuntime(t, billing)

	held := service.access.ProfileFor(t.Context(), conversationRequest(billing), nil).Held
	assert.Contains(t, held, agentdefinition.CoreToolSchedule)

	cases := map[string]func(*agentruntimeports.RunRequest){
		"unattended": func(req *agentruntimeports.RunRequest) { req.Unattended = true },
		"delegated": func(req *agentruntimeports.RunRequest) {
			req.Delegation = &agentruntimeports.Delegation{ParentAgentID: pulid.MustNew("agdef_")}
		},
		"no thread": func(req *agentruntimeports.RunRequest) { req.ThreadID = pulid.Nil },
		"no person": func(req *agentruntimeports.RunRequest) {
			req.Actor.PrincipalType = permissionports.PrincipalTypeAgent
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := conversationRequest(billing)
			change(req)

			assert.NotContains(t,
				service.access.ProfileFor(t.Context(), req, nil).Held,
				agentdefinition.CoreToolSchedule,
			)
		})
	}
}
