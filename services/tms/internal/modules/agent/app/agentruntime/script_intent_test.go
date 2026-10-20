package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scriptStep = "script-step"

// intentDispatch dispatches a write the way a script's broker does: as the
// person, with the turn's taint, under the script's ledger step.
func intentDispatch(
	t *testing.T,
	tool *agenttest.StubActionTool,
	mutate func(*dispatchParams),
) toolOutcome {
	t.Helper()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []agentports.AgentTool{tool},
	}, nil)
	definition := testDefinition(tool.ToolName)
	definition.AutonomyCeiling = agent.TierAutoExecute
	definition.ToolTiers = map[string]agent.AutonomyTier{tool.ToolName: agent.TierAutoExecute}
	p := dispatchParams{
		req: &agentports.RunRequest{
			Definition: definition,
			Actor:      testActor(),
			Input:      "post the drafts",
		},
		call: agentports.ToolCall{
			ID:        "script-step:1",
			Name:      tool.ToolName,
			Arguments: map[string]any{},
		},
		taint:         &agent.RunTaint{},
		stepKey:       "intent-step-1",
		scriptStepKey: scriptStep,
	}
	if mutate != nil {
		mutate(&p)
	}

	return rt.dispatch(t.Context(), p)
}

func TestDispatch_AScriptsWriteIsAnIntentThatNeverRuns(t *testing.T) {
	t.Parallel()

	tool := &agenttest.StubActionTool{ToolName: "dispatch_assign_move", Tier: agent.TierAutoExecute}
	outcome := intentDispatch(t, tool, nil)

	assert.Zero(t, tool.Calls, "the tool's Execute is never reached")
	require.NotNil(t, outcome.action)
	assert.Equal(t, aitrace.OutcomeProposed, outcome.verdict)
	assert.False(t, outcome.action.Executed)
	assert.Equal(t, agent.TierActWithApproval, outcome.action.Tier)
	assert.Equal(t, scriptStep, outcome.action.ScriptStepKey)
	assert.Contains(t, outcome.action.HeldBy, agenttoolpolicy.HeldByScriptIntent)
	assert.Contains(t, outcome.content, "Nothing has run")
	assert.NotContains(t, outcome.content, "Recorded a proposal")
}

func TestDispatch_AScriptsWriteToAToolThatEscalatesIsStillHeld(t *testing.T) {
	t.Parallel()

	tool := &agenttest.StubActionTool{
		ToolName:  "agent_raise_exception",
		Tier:      agent.TierAutoExecute,
		Escalates: true,
	}
	outcome := intentDispatch(t, tool, nil)

	assert.Zero(t, tool.Calls)
	require.NotNil(t, outcome.action)
	assert.Equal(t, agent.TierActWithApproval, outcome.action.Tier)
}

func TestDispatch_AScriptsWriteAfterOutsideContentIsCappedAtPropose(t *testing.T) {
	t.Parallel()

	tool := &agenttest.StubActionTool{ToolName: "dispatch_assign_move", Tier: agent.TierAutoExecute}
	outcome := intentDispatch(t, tool, func(p *dispatchParams) { p.afterExternal = true })

	assert.Zero(t, tool.Calls)
	require.NotNil(t, outcome.action)
	assert.Equal(t, agent.TierPropose, outcome.action.Tier)
	assert.True(t, outcome.action.Tainted)
	assert.Contains(t, outcome.action.HeldBy, agenttoolpolicy.HeldByTainted)
}

func TestDispatch_AScriptsWriteTheActorMayNotMakeIsRefused(t *testing.T) {
	t.Parallel()

	tool := &agenttest.StubActionTool{ToolName: "dispatch_assign_move", Tier: agent.TierAutoExecute}
	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []agentports.AgentTool{tool},
	}, &stubPermissions{Denied: map[string]bool{"shipment_move:update": true}})
	outcome := rt.dispatch(t.Context(), dispatchParams{
		req: &agentports.RunRequest{
			Definition: testDefinition(tool.ToolName),
			Actor:      testActor(),
		},
		call:          agentports.ToolCall{ID: "script-step:1", Name: tool.ToolName},
		taint:         &agent.RunTaint{},
		scriptStepKey: scriptStep,
	})

	assert.Zero(t, tool.Calls)
	assert.Nil(t, outcome.action, "no intent is kept for a write the person may not make")
	assert.True(t, outcome.failed)
}
