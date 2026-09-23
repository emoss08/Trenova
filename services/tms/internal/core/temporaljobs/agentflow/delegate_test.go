package agentflow

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

// scriptedOpener opens the delegate's turn the way the assistant does, as its
// own agent and marked as working for the run's agent, or declines.
type scriptedOpener struct {
	runtime    *agentruntime.Service
	definition *agentdefinition.Definition
	declined   string

	mu     sync.Mutex
	opened int
}

func (o *scriptedOpener) OpenDelegate(
	ctx context.Context,
	run RunContext,
	call agentruntime.DelegateCall,
) (*DelegateOpening, error) {
	o.mu.Lock()
	o.opened++
	o.mu.Unlock()

	if o.declined != "" {
		return nil, &DelegateDeclinedError{Reason: o.declined}
	}

	req := &serviceports.RunRequest{
		Definition: o.definition,
		Actor:      run.Actor,
		Input:      call.Task,
		ThreadID:   run.ThreadID,
		StepOwner:  run.StepOwner,
		Delegation: &serviceports.Delegation{
			ParentAgentID:   run.Definition.ID,
			ParentAgentName: run.Definition.Name,
			CallID:          call.Call.ID,
			StepScope:       call.StepScope,
		},
	}

	return &DelegateOpening{
		Run:  NewRunContext(req, run.PriorityKey),
		Turn: o.runtime.OpenTurn(ctx, req).State(),
	}, nil
}

func delegateDefinition(tools ...string) *agentdefinition.Definition {
	definition := &agentdefinition.Definition{
		ID:              pulid.MustNew("agdef_"),
		Name:            "Shipment Desk",
		Instructions:    "Look shipments up.",
		AutonomyCeiling: agent.TierAutoExecute,
		ToolNames:       tools,
		Enabled:         true,
	}
	definition.ApplyDefaults()

	return definition
}

// delegatingRun is a conversation turn whose agent may ask the delegate, as
// the prepare activity opens it.
func (h *harness) delegatingRun(
	t *testing.T,
	delegate *agentdefinition.Definition,
) (RunContext, agentruntime.TurnState) {
	t.Helper()

	rc := runContext()
	rc.Definition.ID = pulid.MustNew("agdef_")
	rc.ThreadID = pulid.MustNew("athr_")
	rc.StepOwner = serviceports.RunStepOwner{
		Kind: serviceports.RunStepOwnerAssistantTurn,
		ID:   pulid.MustNew("atrn_"),
	}
	req := rc.request()
	req.Context.Delegates = []agentdefinition.RuntimeDelegate{{
		ID:    delegate.ID,
		Name:  delegate.Name,
		Tools: delegate.ToolNames,
	}}

	return rc, h.runtime.OpenTurn(t.Context(), req).State()
}

// scopedReplies scripts the model for the calls in one scope, in order.
func (h *harness) scopedReplies(
	scope func(*ModelCallInput) bool,
	replies ...*serviceports.ChatCompletionResult,
) {
	for _, reply := range replies {
		h.env.OnActivity(h.activities.ModelCallActivity, mock.Anything, mock.MatchedBy(scope)).
			Return(&agentruntime.ModelReply{Completion: reply}, nil).Once()
	}
}

func (h *harness) execute(
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
The delegate's turn runs inside the run's own workflow: its model calls and
tool calls are activities of this execution, its reply streams tagged with the
task, and its steps come back with the run, tagged, with its writes kept apart
as its own.
*/
func TestRunDrivesADelegatesTurnInTheSameWorkflow(t *testing.T) {
	t.Parallel()

	lookup := &agentruntimetest.StubQueryTool{
		ToolName: "get_shipment",
		Result:   map[string]any{"city": "Memphis"},
	}
	delegate := delegateDefinition("get_shipment")
	opener := &scriptedOpener{definition: delegate}
	h := newHarness(t, harnessParams{
		query:     []serviceports.AgentQueryTool{lookup},
		delegates: opener,
	})
	opener.runtime = h.runtime
	// The run's own model calls stream untagged and the delegate's stream
	// tagged with the task: each reply answers only a call in its own scope.
	ownScope := func(in *ModelCallInput) bool { return in.Scope.Empty() }
	delegateScope := func(in *ModelCallInput) bool {
		return in.Scope.AgentID == delegate.ID && in.Scope.DelegateCallID == "call_1"
	}
	h.scopedReplies(ownScope,
		toolReply("delegate_task", map[string]any{
			"agentId": delegate.ID.String(),
			"task":    "Find where load 12345 is and return the city.",
		}),
		textReply("The Shipment Desk found it in Memphis."),
	)
	h.scopedReplies(delegateScope,
		toolReply("get_shipment", map[string]any{"proNumber": "12345"}),
		textReply("Load 12345 is in Memphis."),
	)

	rc, state := h.delegatingRun(t, delegate)
	result := h.execute(t, rc, state)

	require.Empty(t, result.Err)
	assert.Equal(t, 1, opener.opened)
	assert.Equal(t, 1, lookup.Calls, "the delegate's tool ran as an activity of this run")
	assert.Equal(t, "The Shipment Desk found it in Memphis.", result.Outcome.Result.Reply)

	assert.Equal(t, []string{
		serviceports.AssistantEventMessage,
		serviceports.AssistantEventToolStarted,
		serviceports.AssistantEventDelegateStarted,
		serviceports.AssistantEventMessage,
		serviceports.AssistantEventToolStarted,
		serviceports.AssistantEventToolFinished,
		serviceports.AssistantEventDelegateFinished,
		serviceports.AssistantEventToolFinished,
	}, eventNames(result.Outcome))
	nested, ok := result.Outcome.Events[4].Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, delegate.ID.String(), nested["agentId"])
	assert.Equal(t, "call_1", nested["delegateCallId"])
	own, ok := result.Outcome.Events[1].Data.(map[string]any)
	require.True(t, ok)
	assert.NotContains(t, own, "delegateCallId", "the run's own call is not nested")

	messages := result.Outcome.Result.Messages
	delegated := 0
	for _, message := range messages {
		if message.Delegated() {
			delegated++
			assert.Equal(t, delegate.ID, message.AgentDefinitionID)
			assert.Equal(t, "call_1", message.DelegateCallID)
		}
	}
	assert.Equal(t, 4, delegated, "the task, the delegate's call, its result and its answer")

	var callIDs []string
	for _, message := range messages {
		for _, call := range message.ToolCalls {
			callIDs = append(callIDs, call.ID)
		}
	}
	assert.Len(t, callIDs, 2)
	assert.NotEqual(t, callIDs[0], callIDs[1],
		"the delegate's call is given its own id in the thread they share")

	answer := messages[len(messages)-2]
	assert.Equal(t, conversation.RoleTool, answer.Role)
	assert.Contains(t, answer.Content, "Load 12345 is in Memphis.")
	require.Len(t, result.Outcome.Result.Delegations, 1)
	assert.Equal(t, delegate.ID, result.Outcome.Result.Delegations[0].Definition.ID)
}

// A delegate that may not be asked is refused once, not retried, and the
// reason reaches the run's agent as a failed call.
func TestRunPassesOnADelegatesRefusal(t *testing.T) {
	t.Parallel()

	delegate := delegateDefinition()
	opener := &scriptedOpener{
		definition: delegate,
		declined:   "Shipment Desk is disabled, so it cannot take tasks.",
	}
	h := newHarness(t, harnessParams{delegates: opener})
	opener.runtime = h.runtime
	h.replies(
		toolReply("delegate_task", map[string]any{
			"agentId": delegate.ID.String(),
			"task":    "Find load 12345.",
		}),
		textReply("The Shipment Desk is disabled."),
	)

	rc, state := h.delegatingRun(t, delegate)
	result := h.execute(t, rc, state)

	require.Empty(t, result.Err)
	assert.Equal(t, 1, opener.opened, "a refusal is not retried")
	messages := result.Outcome.Result.Messages
	require.Len(t, messages, 4)
	assert.True(t, messages[2].ToolFailed)
	assert.Contains(t, messages[2].Content, "is disabled")
	assert.Empty(t, result.Outcome.Result.Delegations)
}

// A stop ends the delegate's turn as it ends the run's: it is reported as
// stopped, never as something that did not happen.
func TestDeclinedDelegate_AStopIsNotARefusal(t *testing.T) {
	t.Parallel()

	call := agentruntime.DelegateCall{
		Delegate: agentdefinition.RuntimeDelegate{Name: "Shipment Desk"},
	}

	stopped := declinedDelegate(call, temporal.NewCanceledError())
	assert.True(t, stopped.Stopped)
	assert.Empty(t, stopped.Declined)

	refused := declinedDelegate(call, temporal.NewNonRetryableApplicationError(
		"Shipment Desk is disabled.", ErrTypeDelegateDeclined, nil,
	))
	assert.Equal(t, "Shipment Desk is disabled.", refused.Declined)

	lost := declinedDelegate(call, temporal.NewApplicationError("worker lost", "Lost"))
	assert.Contains(t, lost.Declined, "could not be started")

	assert.Contains(t,
		delegateFailureReason("Shipment Desk", &modelcall.Failure{TimedOut: true}),
		"did not answer Shipment Desk in time")
	assert.Contains(t,
		delegateFailureReason("Shipment Desk", &modelcall.Failure{Status: 503}),
		"status 503")
}
