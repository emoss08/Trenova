package agentruntime

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hoursBudgets answers only the business-hours question; the runtime asks
// nothing else of it in these tests.
type hoursBudgets struct {
	agentports.AgentBudgetService

	within bool
}

func (b hoursBudgets) WithinBusinessHours(context.Context, *agentdefinition.Definition) bool {
	return b.within
}

func (b hoursBudgets) CheckTool(
	context.Context,
	agentports.CheckToolBudgetRequest,
) (agentports.BudgetRefusal, error) {
	return agentports.BudgetRefusal{}, nil
}

func autoChangeDefinition(tool string) *agentdefinition.Definition {
	definition := runtimetest.Definition(tool)
	definition.AutonomyCeiling = agent.TierAutoExecute
	definition.ToolTiers = map[string]agent.AutonomyTier{tool: agent.TierAutoExecute}

	return definition
}

func runOneCall(
	t *testing.T,
	rt *Service,
	definition *agentdefinition.Definition,
) *agentruntimeports.RunResult {
	t.Helper()

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
		Input:      "approve these",
	})
	require.NoError(t, err)

	return result
}

// A batch past the agent's largest single change is refused before it is
// proposed or run, and the model is told to split it so each part is
// approved on its own.
func TestRun_RefusesAChangeBiggerThanTheAgentsLargest(t *testing.T) {
	t.Parallel()

	tool := actionTool("approve_billing_items", agent.TierAutoExecute, nil)
	tool.Schema = runtimetest.BatchSchema()
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn(
			"approve_billing_items",
			map[string]any{"billingQueueItemIds": runtimetest.IDs(12)},
		),
		runtimetest.TextTurn("I will split it."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{tool}}, nil)
	definition := autoChangeDefinition("approve_billing_items")
	definition.MaxChangeItems = 10

	result := runOneCall(t, rt, definition)

	assert.Zero(t, tool.Calls, "an oversized change never runs")
	assert.Empty(t, result.Actions, "nor is it proposed")
	require.Len(t, result.Messages, 4)
	assert.Contains(t, result.Messages[2].Content, "would change 12 records")
	assert.Contains(t, result.Messages[2].Content, "at most 10 records each")
}

func TestRun_AChangeWithinTheLimitRuns(t *testing.T) {
	t.Parallel()

	tool := actionTool("approve_billing_items", agent.TierAutoExecute, nil)
	tool.Schema = runtimetest.BatchSchema()
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn(
			"approve_billing_items",
			map[string]any{"billingQueueItemIds": runtimetest.IDs(10)},
		),
		runtimetest.TextTurn("Done."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{tool}}, nil)
	definition := autoChangeDefinition("approve_billing_items")
	definition.MaxChangeItems = 10

	result := runOneCall(t, rt, definition)

	assert.Equal(t, 1, tool.Calls)
	require.Len(t, result.Actions, 1)
	assert.True(t, result.Actions[0].Executed)
}

// Outside business hours a change that would run on its own waits for a
// person instead, and says why.
func TestRun_HoldsAnAutomaticChangeOutsideBusinessHours(t *testing.T) {
	t.Parallel()

	tool := actionTool("dispatch_assign_move", agent.TierAutoExecute, nil)
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("dispatch_assign_move", map[string]any{"moveId": "mv_1"}),
		runtimetest.TextTurn("It is waiting for you."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{tool}}, nil)
	rewire(rt, func(p *Params) { p.Budgets = hoursBudgets{within: false} })
	definition := autoChangeDefinition("dispatch_assign_move")
	definition.BusinessHoursOnly = true

	result := runOneCall(t, rt, definition)

	assert.Zero(t, tool.Calls, "nothing runs outside the window")
	require.Len(t, result.Actions, 1)
	assert.Equal(t, agent.TierActWithApproval, result.Actions[0].Tier)
	assert.False(t, result.Actions[0].Executed)
	assert.Contains(t, result.Actions[0].HeldBy, agenttoolpolicy.HeldByBusinessHours)
	assert.Contains(t, result.Messages[2].Content, "business hours")
}

func TestRun_RunsAnAutomaticChangeInsideBusinessHours(t *testing.T) {
	t.Parallel()

	tool := actionTool("dispatch_assign_move", agent.TierAutoExecute, nil)
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("dispatch_assign_move", map[string]any{"moveId": "mv_1"}),
		runtimetest.TextTurn("Assigned."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{tool}}, nil)
	rewire(rt, func(p *Params) { p.Budgets = hoursBudgets{within: true} })
	definition := autoChangeDefinition("dispatch_assign_move")
	definition.BusinessHoursOnly = true

	result := runOneCall(t, rt, definition)

	assert.Equal(t, 1, tool.Calls)
	require.Len(t, result.Actions, 1)
	assert.True(t, result.Actions[0].Executed)
}
