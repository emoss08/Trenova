package agentruntime

import (
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

	tool := queryTool("search_worker", map[string]any{}, nil)
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("search_worker", map[string]any{"query": "Maria"}),
		toolTurn("get_shipment", map[string]any{"shipmentId": "shp_1"}),
		textTurn("I cannot look that up."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{Tools: []serviceports.AgentQueryTool{
		tool, queryTool("get_shipment", map[string]any{"id": "shp_1"}, nil),
	}}, &stubActionRegistry{}, nil)

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("get_shipment"),
		Actor:      testActor(),
		Input:      "Who is Maria?",
	})
	require.NoError(t, err)

	require.Len(t, result.LoopRefusals, 1, "a dispatched call writes its own step")
	refusal := result.LoopRefusals[0]
	assert.Equal(t, "search_worker", refusal.ToolName)
	assert.Equal(t, aitrace.OutcomeDenied, refusal.Verdict)
	assert.NotEmpty(t, refusal.CallID)
	assert.Contains(t, refusal.Content, "not enabled for this agent")
}

func TestRecordLoopRefusals_WritesEachOnceAsAFailedToolStep(t *testing.T) {
	t.Parallel()

	ledger := newKeyedLedger()
	definition := pulid.MustNew("agdef_")
	req := &RecordLoopRefusalsRequest{
		Ledger: ledger,
		Tenant: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		Owner: serviceports.RunStepOwner{
			Kind: serviceports.RunStepOwnerAssistantTurn,
			ID:   pulid.MustNew("atrn_"),
		},
		DefinitionID: definition,
		Attempt:      1,
		Refusals: []serviceports.LoopRefusal{
			{
				CallID:   "call_1",
				ToolName: "search_worker",
				Verdict:  aitrace.OutcomeDenied,
				Reason:   "the agent does not hold the tool",
				Content:  `"search_worker" is not enabled for this agent`,
			},
			{CallID: "", ToolName: "nameless", Verdict: aitrace.OutcomeInvalid},
		},
	}

	assert.Zero(t, RecordLoopRefusals(t.Context(), req))
	require.Len(t, ledger.steps, 1, "a refusal with no call id has nothing to key it by")

	step := ledger.steps[loopRefusalKeyPrefix+"call_1"]
	assert.Equal(t, serviceports.RunStepTool, step.Kind)
	assert.Equal(t, serviceports.RunStepFailed, step.Status)
	assert.Equal(t, definition, step.DefinitionID, "the scorecard counts it toward its agent")
	assert.Equal(t, aitrace.OutcomeDenied, step.Outcome.Verdict)
	assert.Equal(t, "the agent does not hold the tool", step.Outcome.Reason)

	req.Attempt = 2
	req.Refusals[0].Reason = "written again"
	assert.Zero(t, RecordLoopRefusals(t.Context(), req))
	assert.Equal(t, "the agent does not hold the tool",
		ledger.steps[loopRefusalKeyPrefix+"call_1"].Outcome.Reason,
		"a retried finish does not write the refusal twice")
}
