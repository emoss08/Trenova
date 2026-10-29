package agentruntime

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/askuser"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/delegation"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// delegateRecorder runs every effect in process except handing a task to
// another agent, which it records and answers with a scripted run.
type delegateRecorder struct {
	*localEffects

	run     delegation.Run
	calls   []delegation.Call
	batches []int
	events  []kernelports.StreamEvent
}

func (fx *delegateRecorder) Delegate(
	_ *Turn,
	calls []delegation.Call,
) []delegation.Run {
	fx.batches = append(fx.batches, len(calls))
	runs := make([]delegation.Run, 0, len(calls))
	for _, call := range calls {
		fx.calls = append(fx.calls, call)
		runs = append(runs, fx.run)
	}

	return runs
}

func (fx *delegateRecorder) Emit(event kernelports.StreamEvent) {
	fx.events = append(fx.events, event)
}

func (fx *delegateRecorder) eventNames() []string {
	names := make([]string, 0, len(fx.events))
	for _, event := range fx.events {
		names = append(names, event.Event)
	}

	return names
}

func delegatingRequest(delegates ...agentturn.RuntimeDelegate) *agentruntimeports.RunRequest {
	return &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.Actor(),
		Context:    agentturn.RuntimeContext{Delegates: delegates},
		Input:      "Build a dashboard of this month's on-time delivery.",
		ThreadID:   pulid.MustNew("athr_"),
		StepOwner: agentruntimeports.RunStepOwner{
			Kind: agentruntimeports.RunStepOwnerAssistantTurn,
			ID:   pulid.MustNew("atrn_"),
		},
	}
}

func delegateTurn(agentID pulid.ID, task string) *llmports.ChatCompletionResult {
	return runtimetest.ToolTurn(runtimetools.DelegateTaskName, map[string]any{
		"agentId": agentID.String(),
		"task":    task,
	})
}

func offered(req *llmports.ChatCompletionRequest) []string {
	names := make([]string, 0, len(req.Tools))
	for _, spec := range req.Tools {
		if !spec.Deferred {
			names = append(names, spec.Name)
		}
	}

	return names
}

func driveWith(
	t *testing.T,
	rt *Service,
	req *agentruntimeports.RunRequest,
	fx *delegateRecorder,
) *agentruntimeports.RunResult {
	t.Helper()

	// The model's streamed words go nowhere; what the loop itself emits is
	// recorded by the recorder's Emit.
	fx.localEffects = &localEffects{
		s:    rt,
		ctx:  t.Context(),
		emit: func(kernelports.StreamEvent) {},
	}
	result, err := rt.Drive(rt.OpenTurn(t.Context(), req), fx)
	require.NoError(t, err)

	return result
}

// scriptedDelegateRun is what the Report Builder did when asked: it saved a
// private report on its own and proposed sharing it, which waits on the person.
func scriptedDelegateRun(delegate agentturn.RuntimeDelegate) delegation.Run {
	definition := &agentdefinition.Definition{ID: delegate.ID, Name: delegate.Name}

	return delegation.Run{
		Definition: definition,
		Result: &agentruntimeports.RunResult{
			Reply: "I saved the report \"On-time this month\".",
			Model: "delegate-model",
			Messages: []conversation.Message{
				{Role: conversation.RoleUser, Content: "Create the report."},
				{
					Role: conversation.RoleAssistant,
					ToolCalls: []conversation.ToolCallRecord{
						{ID: "call_d1", Name: "report_create"},
						{ID: "call_d2", Name: "share_report"},
					},
				},
				{Role: conversation.RoleTool, ToolCallID: "call_d1", ToolName: "report_create"},
				{Role: conversation.RoleTool, ToolCallID: "call_d2", ToolName: "share_report"},
				{Role: conversation.RoleAssistant, Content: "I saved the report."},
			},
			Actions: []agentruntimeports.PendingAction{
				{
					ToolName:   "report_create",
					ToolCallID: "call_d1",
					Tier:       agent.TierAutoExecute,
					Executed:   true,
					Arguments:  map[string]any{"name": "On-time this month"},
					ExecutionResult: &agentrecord.ToolExecutionResult{
						Action: "created",
						Kind:   "report",
						Name:   "On-time this month",
						IDs:    map[string]string{"definitionId": "rd_123"},
						Record: &agentrecord.RecordRef{EntityType: "report", ID: "rd_123"},
					},
				},
				{
					ToolName:   "share_report",
					ToolCallID: "call_d2",
					Tier:       agent.TierPropose,
					Arguments:  map[string]any{"name": "On-time this month"},
				},
			},
			ToolCallsUsed: 2,
		},
		Documents: []assistantports.DelegateDocument{
			{ID: pulid.MustNew("aart_"), Kind: "document", Title: "Report notes"},
		},
	}
}

/*
The Homepage Widget Builder holds no report tools. Asked for a dashboard of
on-time delivery, it hands the report to the Report Builder, reads back the new
report's id and carries on. The Report Builder's steps are kept with the turn,
tagged with the agent and the call, and its writes are kept apart so they are
recorded as its own.
*/
func TestDelegate_HandsTheTaskAndFoldsBackWhatTheDelegateDid(t *testing.T) {
	t.Parallel()

	delegate := runtimetest.ReportBuilder()
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		delegateTurn(delegate.ID, "Create a report of this month's on-time deliveries and "+
			"return its id."),
		runtimetest.TextTurn("The report is saved and the tile is ready."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: scriptedDelegateRun(delegate)}

	result := driveWith(t, rt, delegatingRequest(delegate), fx)

	first := completion.Requests[0]
	assert.Contains(t, offered(first), runtimetools.DelegateTaskName)
	assert.Contains(t, first.System, "## Agents you can ask")
	assert.Contains(t, first.System, "Report Builder (agentId "+delegate.ID.String()+")")
	assert.Contains(t, first.System, "report_create, report_run")

	require.Len(t, fx.calls, 1)
	assert.Equal(t, delegate.ID, fx.calls[0].Delegate.ID)
	assert.Contains(t, fx.calls[0].Task, "on-time deliveries")
	assert.NotEmpty(t, fx.calls[0].StepScope, "the delegate's steps are namespaced in the ledger")

	require.Len(t, result.Messages, 9)
	for idx, message := range result.Messages[2:7] {
		assert.True(t, message.Delegated(), "step %d is the delegate's", idx)
		assert.Equal(t, delegate.ID, message.AgentDefinitionID)
		assert.Equal(t, "call_1", message.DelegateCallID)
	}
	answer := result.Messages[7]
	assert.Equal(t, conversation.RoleTool, answer.Role)
	assert.False(t, answer.Delegated(), "the call's own result is the turn's")
	assert.Equal(t, agenttooltrust.ToolEffectDelegate, answer.ToolEffect)
	assert.False(t, answer.ToolFailed)
	assert.Contains(t, answer.Content, "rd_123", "the new report's id comes back")
	assert.Contains(t, answer.Content, "share_report", "the waiting proposal is named")
	assert.Contains(t, answer.Content, "Report notes", "the published document is named")
	assert.Equal(t, "The report is saved and the tile is ready.", result.Reply)

	saved := answer.DelegateReport
	require.NotNil(t, saved, "the account is kept structured on the call's result")
	assert.Equal(t, conversation.DelegateStatusCompleted, saved.Status)
	assert.Equal(t, "call_1", saved.DelegateCallID)
	assert.Equal(t, delegate.ID, saved.AgentID)
	assert.Equal(t, "Report Builder", saved.AgentName)
	assert.Equal(t, "receipt", saved.Icon)
	assert.Equal(t, "teal", saved.Accent)
	assert.Equal(t, "I saved the report \"On-time this month\".", saved.Reply)
	require.Len(t, saved.Made, 1)
	assert.Equal(t, &agentrecord.RecordRef{EntityType: "report", ID: "rd_123"},
		saved.Made[0].Result.Record)
	require.Len(t, saved.Awaiting, 1)
	assert.Len(t, saved.Published, 1)
	assert.Equal(t, 2, saved.ToolCallsUsed)
	for _, message := range result.Messages[2:7] {
		assert.Nil(t, message.DelegateReport, "only the call's result carries the account")
	}

	require.Len(t, result.Delegations, 1)
	assert.Equal(t, delegate.ID, result.Delegations[0].Definition.ID)
	assert.Equal(t, "call_1", result.Delegations[0].CallID)
	assert.Len(t, result.Delegations[0].Actions, 2)
	assert.Empty(t, result.Actions, "the delegate's writes are not the turn's own")
	assert.Equal(t, 1, result.ToolCallsUsed, "the task costs the turn one call")

	assert.Equal(t, []string{
		assistantports.AssistantEventMessage,
		assistantports.AssistantEventToolStarted,
		assistantports.AssistantEventDelegateStarted,
		assistantports.AssistantEventDelegateFinished,
		assistantports.AssistantEventToolFinished,
	}, fx.eventNames())
	finished, ok := fx.events[3].Data.(assistantports.AssistantDelegateFinishedEvent)
	require.True(t, ok)
	assert.Equal(t, assistantports.DelegateStatusCompleted, finished.Status)
	assert.Equal(t, "Report Builder", finished.AgentName)
	assert.Equal(t, "receipt", finished.Icon)
	assert.Equal(t, "teal", finished.Accent)
	require.Len(t, finished.Made, 1)
	assert.Equal(t, "rd_123", finished.Made[0].Result.IDs["definitionId"])
	assert.Equal(t, "report", finished.Made[0].Result.Record.EntityType)
	require.Len(t, finished.Awaiting, 1)
	assert.Equal(t, "share_report", finished.Awaiting[0].ToolName)
	assert.Len(t, finished.Published, 1)
}

// One level only: a turn working on a task another agent handed it is never
// offered delegate_task, nor a question for a person it cannot reach, and a
// call to delegate_task anyway is refused without asking anyone.
func TestDelegate_ASubAgentTurnNeverHandsItsTaskOn(t *testing.T) {
	t.Parallel()

	delegate := runtimetest.ReportBuilder()
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		delegateTurn(delegate.ID, "Do it for me."),
		runtimetest.TextTurn("I could not finish the tile."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	req := delegatingRequest(delegate)
	req.Delegation = &agentruntimeports.Delegation{
		ParentAgentName: "Homepage Widget Builder",
		CallID:          "call_parent",
		StepScope:       "scope",
	}
	fx := &delegateRecorder{run: scriptedDelegateRun(delegate)}

	result := driveWith(t, rt, req, fx)

	first := completion.Requests[0]
	assert.NotContains(t, offered(first), runtimetools.DelegateTaskName)
	assert.NotContains(t, offered(first), askuser.Name)
	assert.NotContains(t, first.System, "## Agents you can ask")
	assert.Empty(t, fx.calls, "nobody was asked")
	require.Len(t, result.Messages, 4)
	assert.True(t, result.Messages[2].ToolFailed)
	assert.Equal(t, delegation.DelegatedRefusal, result.Messages[2].Content)
	assert.Empty(t, result.Delegations)
}

// A run nobody is watching, and a turn outside a conversation, never delegate:
// there is nowhere to keep what the other agent does.
func TestDelegate_IsOfferedOnlyInAConversationSomebodyIsReading(t *testing.T) {
	t.Parallel()

	delegate := runtimetest.ReportBuilder()
	for name, change := range map[string]func(*agentruntimeports.RunRequest){
		"unattended":     func(req *agentruntimeports.RunRequest) { req.Unattended = true },
		"no thread":      func(req *agentruntimeports.RunRequest) { req.ThreadID = pulid.Nil },
		"no delegates":   func(req *agentruntimeports.RunRequest) { req.Context.Delegates = nil },
		"a sub-agent":    func(req *agentruntimeports.RunRequest) { req.Delegation = &agentruntimeports.Delegation{} },
		"all conditions": func(*agentruntimeports.RunRequest) {},
	} {
		completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
			runtimetest.TextTurn("Done."),
		}}
		rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
		req := delegatingRequest(delegate)
		change(req)

		state := rt.OpenTurn(t.Context(), req).State()
		holds := strings.Contains(strings.Join(state.Held, ","), runtimetools.DelegateTaskName)
		assert.Equal(t, name == "all conditions", holds, name)
	}
}

// An agent the model names that is not on the list is refused with the names
// it may use, and nobody is asked.
func TestDelegate_RefusesAnAgentItWasNotOffered(t *testing.T) {
	t.Parallel()

	delegate := runtimetest.ReportBuilder()
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		delegateTurn(pulid.MustNew("agdef_"), "Build it."),
		runtimetest.TextTurn("I could not ask that agent."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: scriptedDelegateRun(delegate)}

	result := driveWith(t, rt, delegatingRequest(delegate), fx)

	assert.Empty(t, fx.calls)
	assert.True(t, result.Messages[2].ToolFailed)
	assert.Contains(t, result.Messages[2].Content, "is not an agent you can ask")
	assert.Contains(t, result.Messages[2].Content, "Report Builder (agentId "+delegate.ID.String())
	assert.Nil(t, result.Messages[2].DelegateReport, "nobody was asked, so there is no account")
}

// A turn hands out at most a few tasks. The one past the cap is refused and
// counted, so a model that only delegates still runs out of turn.
func TestDelegate_CapsTheTasksOneTurnHandsOut(t *testing.T) {
	t.Parallel()

	delegate := runtimetest.ReportBuilder()
	turns := make([]*llmports.ChatCompletionResult, 0, delegation.MaxPerTurn+2)
	for idx := 0; idx <= delegation.MaxPerTurn; idx++ {
		turns = append(turns, delegateTurn(delegate.ID, "Task "+string(rune('A'+idx))))
	}
	turns = append(turns, runtimetest.TextTurn("That is everything."))
	completion := &scriptedCompletion{Turns: turns}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: scriptedDelegateRun(delegate)}

	result := driveWith(t, rt, delegatingRequest(delegate), fx)

	assert.Len(t, fx.calls, delegation.MaxPerTurn)
	last := result.Messages[len(result.Messages)-2]
	assert.True(t, last.ToolFailed)
	assert.Contains(t, last.Content, "already handed out")
	assert.Len(t, result.Delegations, delegation.MaxPerTurn)
}

// A lookup that ended partway is a failed call that changed nothing, and the
// delegating model is told to work from what it did read.
func TestDelegate_AFailedLookupIsAFailedCallThatChangedNothing(t *testing.T) {
	t.Parallel()

	delegate := runtimetest.ReportBuilder()
	run := scriptedDelegateRun(delegate)
	run.Failure = "The model provider did not answer Report Builder in time."
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		delegateTurn(delegate.ID, "Build it."),
		runtimetest.TextTurn("The report was saved, but the rest is unconfirmed."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: run}

	result := driveWith(t, rt, delegatingRequest(delegate), fx)

	answer := result.Messages[7]
	assert.True(t, answer.ToolFailed)
	assert.Contains(t, answer.Content, "changed nothing")
	assert.Contains(t, answer.Content, "findings")
	require.Len(t, result.Delegations, 1)
	assert.True(t, result.Delegations[0].Failed)
}

// A delegate that could not be asked at all says why, and leaves nothing on
// the turn but the refused call.
func TestDelegate_ADeclinedDelegatePassesOnTheReason(t *testing.T) {
	t.Parallel()

	delegate := runtimetest.ReportBuilder()
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		delegateTurn(delegate.ID, "Build it."),
		runtimetest.TextTurn("The Report Builder is disabled."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: delegation.Run{
		Declined: "Report Builder is disabled, so it cannot take tasks.",
	}}

	result := driveWith(t, rt, delegatingRequest(delegate), fx)

	require.Len(t, result.Messages, 4)
	assert.True(t, result.Messages[2].ToolFailed)
	assert.Contains(t, result.Messages[2].Content, "is disabled")
	assert.Empty(t, result.Delegations)

	saved := result.Messages[2].DelegateReport
	require.NotNil(t, saved, "a declined hand-off keeps its account too")
	assert.Equal(t, conversation.DelegateStatusDeclined, saved.Status)
	assert.Equal(t, "Report Builder is disabled, so it cannot take tasks.", saved.Reason)
	assert.Empty(t, saved.Made)
}

// The delegate's steps are the thread's to show and never the model's to read
// again: a later turn replays only the call and its answer.
func TestOpenTurn_LeavesTheDelegatesStepsOutOfTheReplay(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.TextTurn("Done."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	delegated := conversation.Message{
		Role:           conversation.RoleUser,
		Kind:           conversation.MessageKindDelegated,
		Content:        "the task the delegate was handed",
		DelegateCallID: "call_1",
	}
	req := delegatingRequest()
	req.History = []conversation.Message{
		{Role: conversation.RoleUser, Content: "Build the dashboard."},
		{
			Role: conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{
				{ID: "call_1", Name: runtimetools.DelegateTaskName},
			},
		},
		delegated,
		{Role: conversation.RoleTool, ToolCallID: "call_1", ToolName: runtimetools.DelegateTaskName,
			Content: "the account"},
		{Role: conversation.RoleAssistant, Content: "Done."},
	}

	turn := rt.OpenTurn(t.Context(), req)

	for _, message := range turn.messages {
		assert.NotEqual(t, delegated.Content, message.Content)
	}
	assert.Len(t, turn.messages, 5, "question, call, answer, reply, and the new question")
}

// The saved account is bounded, while the delegating model reads the whole
// answer: a long reply is cut with an ellipsis on the message only.
func TestDelegate_TheSavedAccountIsBoundedAndTheModelReadsItWhole(t *testing.T) {
	t.Parallel()

	delegate := runtimetest.ReportBuilder()
	run := scriptedDelegateRun(delegate)
	run.Result.Reply = strings.Repeat("word ", 1000) + "end"
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		delegateTurn(delegate.ID, "Build it."),
		runtimetest.TextTurn("Done."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: run}

	result := driveWith(t, rt, delegatingRequest(delegate), fx)

	answer := result.Messages[7]
	assert.Contains(t, answer.Content, "word end", "the model reads the whole answer")
	require.NotNil(t, answer.DelegateReport)
	assert.True(t, strings.HasSuffix(answer.DelegateReport.Reply, "…"))
	assert.Less(t, len(answer.DelegateReport.Reply), len(run.Result.Reply))
}

func TestDelegate_SeveralLookupsInOneReplyRunTogether(t *testing.T) {
	t.Parallel()

	billing, compliance := runtimetest.ReportBuilder(), runtimetest.ReportBuilder()
	compliance.Name = "Compliance desk"
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		{
			ModelIdentifier: "test-model",
			ToolCalls: []llmports.ToolCall{
				{ID: "call_a", Name: runtimetools.DelegateTaskName, Arguments: map[string]any{
					"agentId": billing.ID.String(), "task": "Which invoices are overdue?",
				}},
				{ID: "call_b", Name: runtimetools.DelegateTaskName, Arguments: map[string]any{
					"agentId": compliance.ID.String(), "task": "Which medical cards lapse?",
				}},
			},
		},
		runtimetest.TextTurn("Both are listed."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: delegation.Run{
		Definition: &agentdefinition.Definition{ID: billing.ID, Name: billing.Name},
		Result:     &agentruntimeports.RunResult{Reply: "Found them."},
	}}

	result := driveWith(t, rt, delegatingRequest(billing, compliance), fx)

	assert.Equal(t, []int{2}, fx.batches, "both tasks are handed out at once")
	require.Len(t, result.Delegations, 2)
	assert.Equal(t, "call_a", result.Delegations[0].CallID)
	assert.Equal(t, "call_b", result.Delegations[1].CallID)
	started := 0
	for _, name := range fx.eventNames() {
		if name == assistantports.AssistantEventToolStarted {
			started++
		}
	}
	assert.Equal(t, 2, started, "each call is announced once")
}

func TestDelegate_OneLookupInAReplyRunsOnItsOwn(t *testing.T) {
	t.Parallel()

	delegate := runtimetest.ReportBuilder()
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		delegateTurn(delegate.ID, "Which invoices are overdue?"),
		runtimetest.TextTurn("Listed."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: scriptedDelegateRun(delegate)}

	driveWith(t, rt, delegatingRequest(delegate), fx)

	assert.Equal(t, []int{1}, fx.batches)
}
