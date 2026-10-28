package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const preparedScriptStep = "script-step"

// intentDispatch dispatches a write the way a script's broker does: as the
// person, with the turn's taint, under the script's ledger step.
func intentDispatch(
	t *testing.T,
	tool *agenttest.StubActionTool,
	mutate func(*dispatchParams),
) toolOutcome {
	t.Helper()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []agenttoolports.AgentTool{tool},
	}, nil)
	definition := testDefinition(tool.ToolName)
	definition.AutonomyCeiling = agent.TierAutoExecute
	definition.ToolTiers = map[string]agent.AutonomyTier{tool.ToolName: agent.TierAutoExecute}
	p := dispatchParams{
		req: &agentruntimeports.RunRequest{
			Definition: definition,
			Actor:      testActor(),
			Input:      "post the drafts",
		},
		call: llmports.ToolCall{
			ID:        "script-step:1",
			Name:      tool.ToolName,
			Arguments: map[string]any{},
		},
		taint:         &agent.RunTaint{},
		stepKey:       "intent-step-1",
		scriptStepKey: preparedScriptStep,
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
	assert.Equal(t, preparedScriptStep, outcome.action.ScriptStepKey)
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
		Tools: []agenttoolports.AgentTool{tool},
	}, &stubPermissions{Denied: map[string]bool{"shipment_move:update": true}})
	outcome := rt.dispatch(t.Context(), dispatchParams{
		req: &agentruntimeports.RunRequest{
			Definition: testDefinition(tool.ToolName),
			Actor:      testActor(),
		},
		call:          llmports.ToolCall{ID: "script-step:1", Name: tool.ToolName},
		taint:         &agent.RunTaint{},
		scriptStepKey: preparedScriptStep,
	})

	assert.Zero(t, tool.Calls)
	assert.Nil(t, outcome.action, "no intent is kept for a write the person may not make")
	assert.True(t, outcome.failed)
}

type writePath struct {
	name string
	tool func() *agenttest.StubActionTool
	args map[string]any
}

func writePaths() []writePath {
	return []writePath{
		{
			name: "an internal write at AutoExecute",
			tool: func() *agenttest.StubActionTool {
				return &agenttest.StubActionTool{
					ToolName: "dispatch_assign_move",
					Tier:     agent.TierAutoExecute,
				}
			},
			args: map[string]any{"moveId": "mv_1"},
		},
		{
			name: "a money write within its bounds",
			tool: moneyTool,
			args: map[string]any{"invoiceId": "inv_1"},
		},
		{
			name: "a reversible internal write the agent may make alone",
			tool: func() *agenttest.StubActionTool {
				return &agenttest.StubActionTool{
					ToolName: "shipment_add_comment",
					Tier:     agent.TierAutoExecute,
					Resource: "shipment",
				}
			},
			args: map[string]any{"shipmentId": "shp_1", "body": "Checked."},
		},
	}
}

// guardedWrite dispatches one call under the run's ledger, as an attempt of
// the run does, and as a script's broker does when scriptStepKey is set.
func guardedWrite(
	t *testing.T,
	tool *agenttest.StubActionTool,
	args map[string]any,
	ledger *keyedLedger,
	runID pulid.ID,
	scriptStepKey string,
) toolOutcome {
	t.Helper()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{tool}}, nil)
	rt.budgets = &moneyBudgets{Limits: roomyMoneyLimits()}

	return rt.guardedDispatch(t.Context(), &guardedDispatchParams{
		req: &agentruntimeports.RunRequest{
			Definition: autoDefinition(tool.ToolName),
			Actor:      testActor(),
			Input:      "do it",
			RunID:      runID,
			Steps:      ledger,
			StepOwner: agentruntimeports.RunStepOwner{
				Kind: agentruntimeports.RunStepOwnerAgentRun,
				ID:   runID,
			},
			Attempt: 1,
		},
		call:          llmports.ToolCall{ID: "call_1", Name: tool.ToolName, Arguments: args},
		ordinal:       1,
		taint:         &agent.RunTaint{},
		scriptStepKey: scriptStepKey,
	})
}

/*
No path runs a script's write: not a tool the agent may run alone, not a
money write inside its bounds, not a reversible internal one, and not a
retried activity replaying the step. Each case first shows that the same call
outside a script does run, so the test would see a path that ran.
*/
func TestDispatch_NoPathRunsAScriptsWrite(t *testing.T) {
	t.Parallel()

	for _, path := range writePaths() {
		t.Run(path.name, func(t *testing.T) {
			t.Parallel()

			control := path.tool()
			ran := guardedWrite(t, control, path.args, newKeyedLedger(), pulid.MustNew("ar_"), "")
			require.Equal(t, 1, control.Calls, "outside a script the call runs: %s", ran.content)

			tool := path.tool()
			ledger := newKeyedLedger()
			runID := pulid.MustNew("ar_")
			for attempt := range 3 {
				outcome := guardedWrite(t, tool, path.args, ledger, runID, preparedScriptStep)
				require.NotNil(t, outcome.action, "attempt %d: %s", attempt+1, outcome.content)
				assert.False(t, outcome.action.Executed, "attempt %d", attempt+1)
				assert.NotEqual(t, agent.TierAutoExecute, outcome.action.Tier)
			}

			assert.Zero(t, tool.Calls, "no attempt reaches the tool's Execute")
			require.Len(t, ledger.steps, 1, "every attempt reads the one step")
			for _, step := range ledger.steps {
				require.NotNil(t, step.Outcome.Action)
				assert.False(
					t,
					step.Outcome.Action.Executed,
					"the ledger records no executed write",
				)
				assert.Equal(t, preparedScriptStep, step.Outcome.Action.ScriptStepKey)
			}
		})
	}
}
