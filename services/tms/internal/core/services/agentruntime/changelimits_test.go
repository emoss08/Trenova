package agentruntime

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hoursBudgets answers only the business-hours question; the runtime asks
// nothing else of it in these tests.
type hoursBudgets struct {
	serviceports.AgentBudgetService

	within bool
}

func (b hoursBudgets) WithinBusinessHours(context.Context, *agentdefinition.Definition) bool {
	return b.within
}

func (b hoursBudgets) CheckTool(
	context.Context,
	serviceports.CheckToolBudgetRequest,
) (serviceports.BudgetRefusal, error) {
	return serviceports.BudgetRefusal{}, nil
}

func batchSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"billingQueueItemIds": map[string]any{
				"type":       "array",
				"items":      map[string]any{"type": "string"},
				"x-subsetOf": "billing_queue_item",
			},
			"note": map[string]any{"type": "string"},
		},
	}
}

func ids(n int) []any {
	out := make([]any, 0, n)
	for i := range n {
		out = append(out, "bqi_"+string(rune('a'+i%26))+string(rune('a'+i/26)))
	}

	return out
}

func autoChangeDefinition(tool string) *agentdefinition.Definition {
	definition := testDefinition(tool)
	definition.AutonomyCeiling = agent.TierAutoExecute
	definition.ToolTiers = map[string]agent.AutonomyTier{tool: agent.TierAutoExecute}

	return definition
}

func runOneCall(
	t *testing.T,
	rt *Service,
	definition *agentdefinition.Definition,
) *serviceports.RunResult {
	t.Helper()

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "approve these",
	})
	require.NoError(t, err)

	return result
}

func TestChangeSize_CountsTheLargestRecordList(t *testing.T) {
	t.Parallel()

	schema := batchSchema()
	schema["properties"].(map[string]any)["shipment_ids"] = map[string]any{
		"type": "array", "items": map[string]any{"type": "string"},
	}
	schema["properties"].(map[string]any)["lines"] = map[string]any{
		"type": "array", "items": map[string]any{"type": "object"},
	}

	tests := []struct {
		name string
		args map[string]any
		want int
	}{
		{"a marked record set", map[string]any{"billingQueueItemIds": ids(3)}, 3},
		{"ids by name", map[string]any{"shipment_ids": []string{"a", "b"}}, 2},
		{"the larger list wins", map[string]any{
			"billingQueueItemIds": ids(2), "shipment_ids": []string{"a", "b", "c", "d"},
		}, 4},
		{"repeats count once", map[string]any{"billingQueueItemIds": []any{"x", "x", " x "}}, 1},
		{"lines are not records", map[string]any{"lines": []any{"a", "b", "c"}}, 0},
		{"nothing listed", map[string]any{"note": "hi"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, changeSize(schema, tt.args))
		})
	}
}

// A batch past the agent's largest single change is refused before it is
// proposed or run, and the model is told to split it so each part is
// approved on its own.
func TestRun_RefusesAChangeBiggerThanTheAgentsLargest(t *testing.T) {
	t.Parallel()

	tool := actionTool("approve_billing_items", agent.TierAutoExecute, nil)
	tool.Schema = batchSchema()
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("approve_billing_items", map[string]any{"billingQueueItemIds": ids(12)}),
		textTurn("I will split it."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{tool}}, nil)
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
	tool.Schema = batchSchema()
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("approve_billing_items", map[string]any{"billingQueueItemIds": ids(10)}),
		textTurn("Done."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{tool}}, nil)
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

	tool := actionTool("assign_move", agent.TierAutoExecute, nil)
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("assign_move", map[string]any{"moveId": "mv_1"}),
		textTurn("It is waiting for you."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{tool}}, nil)
	rt.budgets = hoursBudgets{within: false}
	definition := autoChangeDefinition("assign_move")
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

	tool := actionTool("assign_move", agent.TierAutoExecute, nil)
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("assign_move", map[string]any{"moveId": "mv_1"}),
		textTurn("Assigned."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{tool}}, nil)
	rt.budgets = hoursBudgets{within: true}
	definition := autoChangeDefinition("assign_move")
	definition.BusinessHoursOnly = true

	result := runOneCall(t, rt, definition)

	assert.Equal(t, 1, tool.Calls)
	require.Len(t, result.Actions, 1)
	assert.True(t, result.Actions[0].Executed)
}
