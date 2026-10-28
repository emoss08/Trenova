package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func taintOf(marks ...agent.TaintMark) *agent.RunTaint {
	taint := &agent.RunTaint{}
	for _, mark := range marks {
		taint.Add(mark)
	}

	return taint
}

func delegatingTurnScript(delegateID string) *scriptedCompletion {
	return &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		toolTurn(delegateTaskName, map[string]any{
			"agentId": delegateID,
			"task":    "Create the report and return its id.",
		}),
		textTurn("Done."),
	}}
}

func TestDelegate_HandsTheTurnsTaintDown(t *testing.T) {
	t.Parallel()

	delegate := reportBuilder()
	email := agent.TaintMark{
		Source:   agent.TaintSourceInboundMessage,
		ToolName: "inbox_get_inbound_message",
		CallID:   "call_earlier",
	}
	rt := newRuntime(delegatingTurnScript(delegate.ID.String()),
		&stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: scriptedDelegateRun(delegate)}
	req := delegatingRequest(delegate)
	req.Taint = taintOf(email)

	driveWith(t, rt, req, fx)

	require.Len(t, fx.calls, 1)
	require.NotNil(t, fx.calls[0].Taint)
	handed := email
	handed.Delegated = true
	assert.Equal(t, []agent.TaintMark{handed}, fx.calls[0].Taint.Marks,
		"the delegate opens with what the turn that asked had read, marked as handed on")
	assert.False(t, req.Taint.Marks[0].Delegated, "the turn's own mark is left as it was")
}

func TestDelegate_FoldsTheDelegatesTaintBackUp(t *testing.T) {
	t.Parallel()

	delegate := reportBuilder()
	document := agent.TaintMark{
		Source:   agent.TaintSourceDocument,
		ToolName: "document_get_summary",
		CallID:   "call_d0",
		Ref:      &agent.RecordRef{EntityType: agent.TaintEntityDocument, ID: "doc_1"},
	}
	run := scriptedDelegateRun(delegate)
	run.Result.Taint = taintOf(document)
	rt := newRuntime(delegatingTurnScript(delegate.ID.String()),
		&stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: run}

	result := driveWith(t, rt, delegatingRequest(delegate), fx)

	require.True(t, result.Taint.Tainted(), "the delegate's reply entered this turn")
	inherited := document
	inherited.Delegated = true
	assert.Equal(t, []agent.TaintMark{inherited}, result.Taint.Marks,
		"what the delegate read is marked as read across the delegation")
	require.Len(t, result.Delegations, 1)
	require.NotNil(t, result.Delegations[0].Taint)
	assert.Equal(t, []agent.TaintMark{document}, result.Delegations[0].Taint.Marks,
		"the delegate's proposals carry its own turn's taint")

	var announced []assistantports.AssistantRunTaintedEvent
	for _, event := range fx.events {
		if data, ok := event.Data.(assistantports.AssistantRunTaintedEvent); ok {
			announced = append(announced, data)
		}
	}
	require.Len(t, announced, 1)
	assert.Equal(t, inherited, announced[0].Mark)
}

func TestDelegate_AnOverflowedDelegateTaintOverflowsTheTurn(t *testing.T) {
	t.Parallel()

	delegate := reportBuilder()
	run := scriptedDelegateRun(delegate)
	run.Result.Taint = &agent.RunTaint{
		Seen:       []agent.TaintSource{agent.TaintSourceWeb},
		Overflowed: true,
	}
	rt := newRuntime(delegatingTurnScript(delegate.ID.String()),
		&stubQueryRegistry{}, &stubActionRegistry{}, nil)

	result := driveWith(t, rt, delegatingRequest(delegate), &delegateRecorder{run: run})

	assert.True(t, result.Taint.Overflowed)
	assert.True(
		t,
		result.Taint.Has(agent.TaintSourceWeb),
		"a source past the delegate's marks is kept",
	)
}

func TestDelegate_AMarkTheTurnAlreadyHadIsNotAnnouncedAgain(t *testing.T) {
	t.Parallel()

	delegate := reportBuilder()
	email := agent.TaintMark{
		Source:   agent.TaintSourceInboundMessage,
		ToolName: "inbox_get_inbound_message",
		CallID:   "call_earlier",
	}
	run := scriptedDelegateRun(delegate)
	run.Result.Taint = taintOf(email)
	rt := newRuntime(delegatingTurnScript(delegate.ID.String()),
		&stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: run}
	req := delegatingRequest(delegate)
	req.Taint = taintOf(email)

	result := driveWith(t, rt, req, fx)

	assert.Len(t, result.Taint.Marks, 1)
	assert.NotContains(t, fx.eventNames(), assistantports.AssistantEventRunTainted)
}

func TestDelegate_ADelegateOfUnknownTaintLeavesTheTurnUnknown(t *testing.T) {
	t.Parallel()

	delegate := reportBuilder()
	run := scriptedDelegateRun(delegate)
	run.Result.Taint = nil
	rt := newRuntime(delegatingTurnScript(delegate.ID.String()),
		&stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: run}

	result := driveWith(t, rt, delegatingRequest(delegate), fx)

	assert.Nil(t, result.Taint, "what the delegate read is unknown, so this turn's is too")
}

func TestDelegate_ADeclinedTaskTaintsNothing(t *testing.T) {
	t.Parallel()

	delegate := reportBuilder()
	rt := newRuntime(delegatingTurnScript(delegate.ID.String()),
		&stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: DelegateRun{Declined: "Report Builder is disabled."}}

	result := driveWith(t, rt, delegatingRequest(delegate), fx)

	require.NotNil(t, result.Taint)
	assert.False(t, result.Taint.Tainted())
}
