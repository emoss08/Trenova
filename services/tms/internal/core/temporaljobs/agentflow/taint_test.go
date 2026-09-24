package agentflow

import (
	"context"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
)

func paymentTool() *agentruntimetest.StubActionTool {
	return &agentruntimetest.StubActionTool{
		ToolName: "post_customer_payment",
		Tier:     agent.TierAutoExecute,
		Egress:   agent.EgressMoney,
	}
}

func receiptTool(reads agent.ExternalRead) *agentruntimetest.StubQueryTool {
	return &agentruntimetest.StubQueryTool{
		ToolName: "get_bank_receipt",
		Result:   map[string]any{"memo": "Apply to invoice 88."},
		Reads:    reads,
		Source:   agent.TaintSourceBankReceipt,
	}
}

// runState drives a turn from a given state, as a workflow restores one an
// activity opened.
func (h *harness) runState(
	t *testing.T,
	rc RunContext,
	state agentruntime.TurnState,
) *turnResult {
	t.Helper()

	h.env.ExecuteWorkflow(testTurnWorkflow, rc, state)

	require.True(t, h.env.IsWorkflowCompleted())
	require.NoError(t, h.env.GetWorkflowError())

	var result turnResult
	require.NoError(t, h.env.GetWorkflowResult(&result))
	h.env.AssertExpectations(t)

	return &result
}

/*
A turn opened before taint was kept replays from a state with no taint at
all. That unknown counts as tainted for every class that leaves the
organization, so an in-flight turn cannot move money on its own after the
release, and anything that stays inside runs as it always did.
*/
func TestRun_ATurnOpenedBeforeTaintWasKeptCountsAsTainted(t *testing.T) {
	t.Parallel()

	pay := paymentTool()
	move := &agentruntimetest.StubActionTool{ToolName: "assign_move", Tier: agent.TierAutoExecute}
	h := newHarness(t, harnessParams{action: []serviceports.AgentTool{pay, move}})
	h.replies(
		toolReply("post_customer_payment", map[string]any{"invoiceId": "inv_88"}),
		toolReply("assign_move", map[string]any{"moveId": "smv_1"}),
		textReply("The payment waits for approval; the move is assigned."),
	)
	rc := runContext("post_customer_payment", "assign_move")
	rc.Unattended = true
	state := h.runtime.OpenTurn(t.Context(), rc.request()).State()
	state.Result.Taint = nil

	result := h.runState(t, rc, state)

	require.Empty(t, result.Err)
	assert.Zero(t, pay.Calls, "money does not move on its own from a turn of unknown taint")
	assert.Equal(t, 1, move.Calls, "internal work is untouched")
	require.Len(t, result.Outcome.Result.Actions, 2)
	payment := result.Outcome.Result.Actions[0]
	assert.Equal(t, agent.TierActWithApproval, payment.Tier)
	assert.Contains(t, payment.HeldBy, agenttoolpolicy.HeldByTainted)
	assert.False(t, payment.Tainted, "unknown is held, not recorded as a known taint")
	assert.Nil(t, result.Outcome.Result.Taint, "the unknown stays unknown")
	assert.NotContains(t, eventNames(result.Outcome), serviceports.AssistantEventRunTainted)
}

/*
Taint is data. The same conversation driven once with a read that is outside
content and once with one that is not schedules the same activities in the same
order: whether the payment runs or waits is decided inside its activity, never
by workflow code choosing a different command. That is what lets an execution
started before the release replay on this code, and why taint took no
GetVersion.
*/
func TestRun_TaintChangesNoCommand(t *testing.T) {
	t.Parallel()

	drive := func(
		reads agent.ExternalRead,
	) ([]string, *turnResult, *agentruntimetest.StubActionTool) {
		pay := paymentTool()
		h := newHarness(t, harnessParams{
			query:  []serviceports.AgentQueryTool{receiptTool(reads)},
			action: []serviceports.AgentTool{pay},
		})
		h.replies(
			toolReply("get_bank_receipt", map[string]any{"receiptId": "brc_1"}),
			toolReply("post_customer_payment", map[string]any{"invoiceId": "inv_88"}),
			textReply("Done."),
		)
		var started []string
		h.env.SetOnActivityStartedListener(
			func(info *activity.Info, _ context.Context, _ converter.EncodedValues) {
				started = append(started, info.ActivityType.Name)
			},
		)

		result := h.run(t, runContext("get_bank_receipt", "post_customer_payment"))

		return started, result, pay
	}

	cleanStarted, clean, cleanPay := drive(agent.ExternalReadNever)
	taintedStarted, tainted, taintedPay := drive(agent.ExternalReadAlways)

	assert.Equal(t, cleanStarted, taintedStarted, "the same activities, in the same order")
	tools := slices.DeleteFunc(slices.Clone(cleanStarted), func(name string) bool {
		return name == "ModelCallActivity"
	})
	assert.Equal(t, []string{"get_bank_receipt", "post_customer_payment"}, tools,
		"the payment is one activity whether it runs or waits")

	assert.Equal(t, 1, cleanPay.Calls)
	assert.False(t, clean.Outcome.Result.Taint.Tainted())
	assert.True(t, clean.Outcome.Result.Actions[0].Executed)

	assert.Zero(t, taintedPay.Calls, "the tainted turn proposed the same call instead")
	require.True(t, tainted.Outcome.Result.Taint.Tainted())
	assert.False(t, tainted.Outcome.Result.Actions[0].Executed)
	assert.True(t, tainted.Outcome.Result.Actions[0].Tainted)
	assert.Contains(t, eventNames(tainted.Outcome), serviceports.AssistantEventRunTainted,
		"the trajectory keeps when the run became tainted")
	assert.NotContains(t, eventNames(clean.Outcome), serviceports.AssistantEventRunTainted)
}

/*
The shape a turn's state had before taint was kept decodes to no taint, and a
state written now keeps an empty taint as empty rather than as unknown, through
the converter Temporal records activity results with.
*/
func TestTurnState_TaintSurvivesTheDataConverter(t *testing.T) {
	t.Parallel()

	dc := converter.GetDefaultDataConverter()

	old, err := dc.ToPayload(map[string]any{
		"budget":   4,
		"system":   "You help dispatch.",
		"messages": []any{},
		"tools":    map[string]any{"specs": []any{}, "allowed": []any{}},
		"result":   map[string]any{"Reply": ""},
	})
	require.NoError(t, err)
	var restored agentruntime.TurnState
	require.NoError(t, dc.FromPayload(old, &restored))
	assert.Nil(t, restored.Result.Taint, "a state from before the release carries no taint")

	now, err := dc.ToPayload(agentruntime.TurnState{
		Result: serviceports.RunResult{Taint: &agent.RunTaint{}},
	})
	require.NoError(t, err)
	var kept agentruntime.TurnState
	require.NoError(t, dc.FromPayload(now, &kept))
	require.NotNil(t, kept.Result.Taint, "an empty taint is clean, not unknown")
	assert.False(t, kept.Result.Taint.Tainted())
}
