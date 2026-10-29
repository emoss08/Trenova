package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
The scorecard's failure breakdown reads the step ledger, and dispatch writes
a step only for the calls it handles. A model that called a tool its agent
did not hold, sent arguments that did not parse or repeated a failed call was
answered by the loop itself, and the agent looked as if nothing had failed.
The loop now keeps those refusals on the result, with their verdicts.
*/
func TestRun_KeepsTheCallsTheLoopRefusedWithoutDispatching(t *testing.T) {
	t.Parallel()

	tool := queryTool("worker_list", map[string]any{}, nil)
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("worker_list", map[string]any{"query": "Maria"}),
		runtimetest.ToolTurn("shipment_get", map[string]any{"shipmentId": "shp_1"}),
		runtimetest.TextTurn("I cannot look that up."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
		tool, queryTool("shipment_get", map[string]any{"id": "shp_1"}, nil),
	}}, &stubActionRegistry{}, nil)

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("shipment_get"),
		Actor:      runtimetest.Actor(),
		Input:      "Who is Maria?",
	})
	require.NoError(t, err)

	require.Len(t, result.LoopRefusals, 1, "a dispatched call writes its own step")
	refusal := result.LoopRefusals[0]
	assert.Equal(t, "worker_list", refusal.ToolName)
	assert.Equal(t, aitrace.OutcomeDenied, refusal.Verdict)
	assert.NotEmpty(t, refusal.CallID)
	assert.Contains(t, refusal.Content, "not enabled for this agent")
}
