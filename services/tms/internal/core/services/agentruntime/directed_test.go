package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const directedStep = "Mark shipment SEED-PAY-001 ready to invoice."

// directedRequest is a turn in a conversation with an agent that lists no
// other agent, on which the person handed a step to the Report Builder.
func directedRequest(delegate agentdefinition.RuntimeDelegate) *serviceports.RunRequest {
	req := delegatingRequest()
	req.Input = directedStep
	req.Directed = &serviceports.DirectedTask{Delegate: delegate, Task: directedStep}

	return req
}

/*
A case step the conversation's agent cannot take, which the person handed to
an agent that can with the step's own button. The agent they chose takes it
inside the same conversation, though the conversation's agent lists no other
agent: the turn reads as though the conversation's agent had handed the task
over, and the conversation's own model is never asked.
*/
func TestDriveDirected_RunsTheChosenAgentWithoutAskingTheConversationsModel(t *testing.T) {
	t.Parallel()

	delegate := reportBuilder()
	completion := &scriptedCompletion{}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: scriptedDelegateRun(delegate)}

	result := driveWith(t, rt, directedRequest(delegate), fx)

	assert.Empty(t, completion.Requests, "the conversation's model is not asked")

	require.Len(t, fx.calls, 1)
	call := fx.calls[0]
	assert.True(t, call.Directed, "the opening skips only the allowlist")
	assert.Equal(t, delegate.ID, call.Delegate.ID)
	assert.Equal(t, directedStep, call.Task)
	assert.NotEmpty(t, call.StepScope)
	assert.Nil(t, call.Context, "nothing is handed over but the person's words")

	require.Len(t, result.Messages, 9)
	assert.Equal(t, conversation.RoleUser, result.Messages[0].Role)
	opening := result.Messages[1]
	assert.Equal(t, conversation.RoleAssistant, opening.Role)
	assert.Empty(t, opening.Content)
	require.Len(t, opening.ToolCalls, 1)
	callID := opening.ToolCalls[0].ID
	assert.Equal(t, call.Call.ID, callID)
	assert.Equal(t, delegateTaskName, opening.ToolCalls[0].Name)
	assert.Equal(t, delegate.ID.String(), opening.ToolCalls[0].Arguments["agentId"],
		"a later turn and the thread both read who was asked")
	assert.Equal(t, directedStep, opening.ToolCalls[0].Arguments["task"])

	for idx, message := range result.Messages[2:7] {
		assert.True(t, message.Delegated(), "step %d is the chosen agent's", idx)
		assert.Equal(t, delegate.ID, message.AgentDefinitionID)
		assert.Equal(t, callID, message.DelegateCallID)
	}
	answer := result.Messages[7]
	assert.Equal(t, conversation.RoleTool, answer.Role)
	assert.Equal(t, callID, answer.ToolCallID)
	assert.False(t, answer.ToolFailed)
	require.NotNil(t, answer.DelegateReport)
	assert.Equal(t, conversation.DelegateStatusCompleted, answer.DelegateReport.Status)

	closing := result.Messages[8]
	assert.Equal(t, conversation.RoleAssistant, closing.Role)
	assert.Equal(t, "I saved the report \"On-time this month\".", closing.Content,
		"the reply is the chosen agent's answer")
	assert.Empty(t, closing.ToolCalls)
	assert.Equal(t, closing.Content, result.Reply)

	require.Len(t, result.Delegations, 1)
	assert.Equal(t, delegate.ID, result.Delegations[0].Definition.ID)
	assert.Equal(t, callID, result.Delegations[0].CallID)
	assert.Len(t, result.Delegations[0].Actions, 2, "its writes are recorded as its own")
	assert.Empty(t, result.Actions)
	assert.Equal(t, 1, result.ToolCallsUsed)

	assert.Equal(t, []string{
		serviceports.AssistantEventMessage,
		serviceports.AssistantEventToolStarted,
		serviceports.AssistantEventDelegateStarted,
		serviceports.AssistantEventDelegateFinished,
		serviceports.AssistantEventToolFinished,
		serviceports.AssistantEventDelta,
	}, fx.eventNames(), "the stream shows it as any hand-off, then the answer")
	started, ok := fx.events[2].Data.(serviceports.AssistantDelegateStartedEvent)
	require.True(t, ok)
	assert.Equal(t, callID, started.DelegateCallID)
	assert.Equal(t, directedStep, started.Task)
}

// The made-up call is a call like any other: its id is not one the
// conversation already holds, so a provider never sees two calls with one id.
func TestDriveDirected_TakesACallIDTheConversationDoesNotHold(t *testing.T) {
	t.Parallel()

	delegate := reportBuilder()
	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: scriptedDelegateRun(delegate)}
	req := directedRequest(delegate)
	req.History = []conversation.Message{
		{Role: conversation.RoleUser, Content: "Where is it?"},
		{
			Role:      conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{ID: "call_old", Name: "get_shipment"}},
		},
		{Role: conversation.RoleTool, ToolCallID: "call_old", ToolName: "get_shipment"},
		{Role: conversation.RoleAssistant, Content: "It delivered."},
	}

	result := driveWith(t, rt, req, fx)

	require.Len(t, fx.calls, 1)
	assert.NotEqual(t, "call_old", fx.calls[0].Call.ID)
	assert.Contains(t, fx.calls[0].CallIDs, "call_old")
	assert.Contains(t, fx.calls[0].CallIDs, fx.calls[0].Call.ID,
		"the delegate is told the new call's id is taken too")
	assert.Equal(t, fx.calls[0].Call.ID, result.Messages[1].ToolCalls[0].ID)
}

// An agent that cannot take the task says why, and that is the reply: the
// conversation never ends on a tool result, and nothing claims the step was
// done.
func TestDriveDirected_ADeclinedTaskRepliesWithTheReason(t *testing.T) {
	t.Parallel()

	delegate := reportBuilder()
	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: DelegateRun{
		Declined: "The person may not use Report Builder, so it cannot be asked.",
	}}

	result := driveWith(t, rt, directedRequest(delegate), fx)

	require.Len(t, result.Messages, 4)
	assert.True(t, result.Messages[2].ToolFailed)
	assert.Empty(t, result.Delegations)
	assert.Equal(t, "The person may not use Report Builder, so it cannot be asked.",
		result.Reply)
	assert.Equal(t, conversation.RoleAssistant, result.Messages[3].Role)
	assert.Equal(t, result.Reply, result.Messages[3].Content)
}

// Only a turn the person reads, in a conversation, and working for them
// directly runs a directed task. A delegate's own turn, and a run nobody
// reads, answer with their own model as they always did.
func TestDriveDirected_IsIgnoredWhereNoPersonHandsOverATask(t *testing.T) {
	t.Parallel()

	tests := map[string]func(*serviceports.RunRequest){
		"a delegate's turn": func(req *serviceports.RunRequest) {
			req.Delegation = &serviceports.Delegation{
				ParentAgentID: pulid.MustNew("agdef_"),
				CallID:        "call_parent",
				StepScope:     "scope",
			}
		},
		"an unattended run": func(req *serviceports.RunRequest) { req.Unattended = true },
		"no conversation":   func(req *serviceports.RunRequest) { req.ThreadID = pulid.Nil },
	}

	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			delegate := reportBuilder()
			completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
				textTurn("Done."),
			}}
			rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
			fx := &delegateRecorder{run: scriptedDelegateRun(delegate)}
			req := directedRequest(delegate)
			change(req)

			result := driveWith(t, rt, req, fx)

			assert.Empty(t, fx.calls)
			assert.Len(t, completion.Requests, 1)
			assert.Equal(t, "Done.", result.Reply)
		})
	}
}

// The task travels in the turn's state, which a durable turn is restored
// from on every worker, so a replay runs the same task.
func TestDriveDirected_TheTaskSurvivesTheTurnsState(t *testing.T) {
	t.Parallel()

	delegate := reportBuilder()
	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	req := directedRequest(delegate)

	state := rt.OpenTurn(t.Context(), req).State()
	require.NotNil(t, state.Directed)
	assert.Equal(t, delegate.ID, state.Directed.Delegate.ID)
	assert.Equal(t, directedStep, state.Directed.Task)

	restored := rt.RestoreTurn(req, state)
	require.NotNil(t, restored.directed)
	assert.Equal(t, delegate.ID, restored.directed.Delegate.ID)

	req.Directed = nil
	assert.Nil(t, rt.OpenTurn(t.Context(), req).State().Directed)
}

func TestDirectedReply_SaysWhatCameOfTheTask(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		report serviceports.AssistantDelegateFinishedEvent
		want   string
	}{
		"completed": {
			report: serviceports.AssistantDelegateFinishedEvent{
				Status: serviceports.DelegateStatusCompleted, Reply: "  Marked ready.  ",
			},
			want: "Marked ready.",
		},
		"completed in silence": {
			report: serviceports.AssistantDelegateFinishedEvent{
				Status: serviceports.DelegateStatusCompleted, AgentName: "Billing Assistant",
			},
			want: "Billing Assistant finished without saying anything more.",
		},
		"out of tool calls": {
			report: serviceports.AssistantDelegateFinishedEvent{
				Status: serviceports.DelegateStatusExhausted,
				Reply:  "I checked the paperwork.",
				Reason: "It used every tool call.",
			},
			want: "I checked the paperwork.\n\nIt used every tool call.",
		},
		"failed": {
			report: serviceports.AssistantDelegateFinishedEvent{
				Status: serviceports.DelegateStatusFailed, Reason: "The model stopped answering.",
			},
			want: "The model stopped answering.",
		},
		"failed without a reason": {
			report: serviceports.AssistantDelegateFinishedEvent{
				Status: serviceports.DelegateStatusFailed, AgentName: "Billing Assistant",
			},
			want: "Billing Assistant did not report back.",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, directedReply(&tt.report))
		})
	}
}

/*
"bill it", handed to the billing agent in a conversation about one load,
reached it as those two words alone, and it went looking for which load. The
conversation's subject and the records it has been working on go with the task.
*/
func TestDriveDirected_HandsOverTheRecordsInPlay(t *testing.T) {
	t.Parallel()

	delegate := reportBuilder()
	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: scriptedDelegateRun(delegate)}
	req := directedRequest(delegate)
	req.Records = []agent.EntityRef{{Type: "shipment", ID: "shp_01M4HYAB7FKYM27B5JC48Q68MB"}}
	req.Context.Anchors = []agentdefinition.RuntimeAnchor{
		{Kind: "invoice", ID: "inv_01M4HYAB7FKYM27B5JC48Q68MC"},
		{Kind: "worker", ID: "wrk_01M4HYAB7FKYM27B5JC48Q68MD", Note: "withheld"},
	}

	driveWith(t, rt, req, fx)

	require.Len(t, fx.calls, 1)
	require.NotNil(t, fx.calls[0].Context)
	assert.Equal(t, []agent.RecordRef{
		{EntityType: "shipment", ID: "shp_01M4HYAB7FKYM27B5JC48Q68MB"},
		{EntityType: "invoice", ID: "inv_01M4HYAB7FKYM27B5JC48Q68MC"},
	}, fx.calls[0].Context.Records, "a record the person may not read is not handed over")
	assert.Contains(t, DelegateInput(directedStep, fx.calls[0].Context),
		"shipment shp_01M4HYAB7FKYM27B5JC48Q68MB")
}

// The workflow runs a directed task from the turn's saved state and a request
// that no longer carries the anchors, so the records are kept on the task when
// the turn opens. Before, the billing agent got "bill it" alone in every real
// run, though a turn driven directly handed the records over.
func TestDriveDirected_TheRecordsInPlaySurviveTheTurnsState(t *testing.T) {
	t.Parallel()

	delegate := reportBuilder()
	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	req := directedRequest(delegate)
	req.Context.Anchors = []agentdefinition.RuntimeAnchor{
		{Kind: "shipment", ID: "shp_01M4J2YG6896P8QFJPH2DB8PBA"},
	}

	state := rt.OpenTurn(t.Context(), req).State()
	require.NotNil(t, state.Directed)
	assert.Equal(t, []agent.RecordRef{{EntityType: "shipment", ID: "shp_01M4J2YG6896P8QFJPH2DB8PBA"}},
		state.Directed.Records)

	bare := directedRequest(delegate)
	restored := rt.RestoreTurn(bare, state)
	require.NotNil(t, restored.directed)
	assert.Equal(t, state.Directed.Records, restored.directed.Records)
}
