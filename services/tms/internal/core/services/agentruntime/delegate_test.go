package agentruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// delegateRecorder runs every effect in process except handing a task to
// another agent, which it records and answers with a scripted run.
type delegateRecorder struct {
	*localEffects

	run    DelegateRun
	calls  []DelegateCall
	events []serviceports.StreamEvent
}

func (fx *delegateRecorder) Delegate(_ *Turn, call DelegateCall) DelegateRun {
	fx.calls = append(fx.calls, call)

	return fx.run
}

func (fx *delegateRecorder) Emit(event serviceports.StreamEvent) {
	fx.events = append(fx.events, event)
}

func (fx *delegateRecorder) eventNames() []string {
	names := make([]string, 0, len(fx.events))
	for _, event := range fx.events {
		names = append(names, event.Event)
	}

	return names
}

func reportBuilder() agentdefinition.RuntimeDelegate {
	return agentdefinition.RuntimeDelegate{
		ID:          pulid.MustNew("agdef_"),
		Name:        "Report Builder",
		Description: "Builds and saves reports. It knows every dataset.",
		Icon:        "receipt",
		Accent:      "teal",
		Tools:       []string{"create_report", "run_report"},
	}
}

func delegatingRequest(delegates ...agentdefinition.RuntimeDelegate) *serviceports.RunRequest {
	return &serviceports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Context:    agentdefinition.RuntimeContext{Delegates: delegates},
		Input:      "Build a dashboard of this month's on-time delivery.",
		ThreadID:   pulid.MustNew("athr_"),
		StepOwner: serviceports.RunStepOwner{
			Kind: serviceports.RunStepOwnerAssistantTurn,
			ID:   pulid.MustNew("atrn_"),
		},
	}
}

func delegateTurn(agentID pulid.ID, task string) *serviceports.ChatCompletionResult {
	return toolTurn(delegateTaskName, map[string]any{
		"agentId": agentID.String(),
		"task":    task,
	})
}

func offered(req *serviceports.ChatCompletionRequest) []string {
	names := make([]string, 0, len(req.Tools))
	for _, spec := range req.Tools {
		names = append(names, spec.Name)
	}

	return names
}

func driveWith(
	t *testing.T,
	rt *Service,
	req *serviceports.RunRequest,
	fx *delegateRecorder,
) *serviceports.RunResult {
	t.Helper()

	// The model's streamed words go nowhere; what the loop itself emits is
	// recorded by the recorder's Emit.
	fx.localEffects = &localEffects{
		s:    rt,
		ctx:  t.Context(),
		emit: func(serviceports.StreamEvent) {},
	}
	result, err := rt.Drive(rt.OpenTurn(t.Context(), req), fx)
	require.NoError(t, err)

	return result
}

// scriptedDelegateRun is what the Report Builder did when asked: it saved a
// private report on its own and proposed sharing it, which waits on the person.
func scriptedDelegateRun(delegate agentdefinition.RuntimeDelegate) DelegateRun {
	definition := &agentdefinition.Definition{ID: delegate.ID, Name: delegate.Name}

	return DelegateRun{
		Definition: definition,
		Result: &serviceports.RunResult{
			Reply: "I saved the report \"On-time this month\".",
			Model: "delegate-model",
			Messages: []conversation.Message{
				{Role: conversation.RoleUser, Content: "Create the report."},
				{
					Role: conversation.RoleAssistant,
					ToolCalls: []conversation.ToolCallRecord{
						{ID: "call_d1", Name: "create_report"},
						{ID: "call_d2", Name: "share_report"},
					},
				},
				{Role: conversation.RoleTool, ToolCallID: "call_d1", ToolName: "create_report"},
				{Role: conversation.RoleTool, ToolCallID: "call_d2", ToolName: "share_report"},
				{Role: conversation.RoleAssistant, Content: "I saved the report."},
			},
			Actions: []serviceports.PendingAction{
				{
					ToolName:   "create_report",
					ToolCallID: "call_d1",
					Tier:       agent.TierAutoExecute,
					Executed:   true,
					Arguments:  map[string]any{"name": "On-time this month"},
					ExecutionResult: &agent.ToolExecutionResult{
						Action: "created",
						Kind:   "report",
						Name:   "On-time this month",
						IDs:    map[string]string{"definitionId": "rd_123"},
						Record: &agent.RecordRef{EntityType: "report", ID: "rd_123"},
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
		Documents: []serviceports.DelegateDocument{
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

	delegate := reportBuilder()
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		delegateTurn(delegate.ID, "Create a report of this month's on-time deliveries and "+
			"return its id."),
		textTurn("The report is saved and the tile is ready."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: scriptedDelegateRun(delegate)}

	result := driveWith(t, rt, delegatingRequest(delegate), fx)

	first := completion.Requests[0]
	assert.Contains(t, offered(first), delegateTaskName)
	assert.Contains(t, first.System, "## Agents you can ask")
	assert.Contains(t, first.System, "Report Builder (agentId "+delegate.ID.String()+")")
	assert.Contains(t, first.System, "create_report, run_report")

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
	assert.Equal(t, agent.ToolEffectDelegate, answer.ToolEffect)
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
	assert.Equal(t, &agent.RecordRef{EntityType: "report", ID: "rd_123"},
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
		serviceports.AssistantEventMessage,
		serviceports.AssistantEventToolStarted,
		serviceports.AssistantEventDelegateStarted,
		serviceports.AssistantEventDelegateFinished,
		serviceports.AssistantEventToolFinished,
	}, fx.eventNames())
	finished, ok := fx.events[3].Data.(serviceports.AssistantDelegateFinishedEvent)
	require.True(t, ok)
	assert.Equal(t, serviceports.DelegateStatusCompleted, finished.Status)
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

	delegate := reportBuilder()
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		delegateTurn(delegate.ID, "Do it for me."),
		textTurn("I could not finish the tile."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	req := delegatingRequest(delegate)
	req.Delegation = &serviceports.Delegation{
		ParentAgentName: "Homepage Widget Builder",
		CallID:          "call_parent",
		StepScope:       "scope",
	}
	fx := &delegateRecorder{run: scriptedDelegateRun(delegate)}

	result := driveWith(t, rt, req, fx)

	first := completion.Requests[0]
	assert.NotContains(t, offered(first), delegateTaskName)
	assert.NotContains(t, offered(first), askUserName)
	assert.NotContains(t, first.System, "## Agents you can ask")
	assert.Empty(t, fx.calls, "nobody was asked")
	require.Len(t, result.Messages, 4)
	assert.True(t, result.Messages[2].ToolFailed)
	assert.Equal(t, delegatedRefusal, result.Messages[2].Content)
	assert.Empty(t, result.Delegations)
}

// A run nobody is watching, and a turn outside a conversation, never delegate:
// there is nowhere to keep what the other agent does.
func TestDelegate_IsOfferedOnlyInAConversationSomebodyIsReading(t *testing.T) {
	t.Parallel()

	delegate := reportBuilder()
	for name, change := range map[string]func(*serviceports.RunRequest){
		"unattended":     func(req *serviceports.RunRequest) { req.Unattended = true },
		"no thread":      func(req *serviceports.RunRequest) { req.ThreadID = pulid.Nil },
		"no delegates":   func(req *serviceports.RunRequest) { req.Context.Delegates = nil },
		"a sub-agent":    func(req *serviceports.RunRequest) { req.Delegation = &serviceports.Delegation{} },
		"all conditions": func(*serviceports.RunRequest) {},
	} {
		completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
			textTurn("Done."),
		}}
		rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
		req := delegatingRequest(delegate)
		change(req)

		state := rt.OpenTurn(t.Context(), req).State()
		holds := strings.Contains(strings.Join(state.Held, ","), delegateTaskName)
		assert.Equal(t, name == "all conditions", holds, name)
	}
}

// An agent the model names that is not on the list is refused with the names
// it may use, and nobody is asked.
func TestDelegate_RefusesAnAgentItWasNotOffered(t *testing.T) {
	t.Parallel()

	delegate := reportBuilder()
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		delegateTurn(pulid.MustNew("agdef_"), "Build it."),
		textTurn("I could not ask that agent."),
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

	delegate := reportBuilder()
	turns := make([]*serviceports.ChatCompletionResult, 0, maxDelegationsPerTurn+2)
	for idx := 0; idx <= maxDelegationsPerTurn; idx++ {
		turns = append(turns, delegateTurn(delegate.ID, "Task "+string(rune('A'+idx))))
	}
	turns = append(turns, textTurn("That is everything."))
	completion := &scriptedCompletion{Turns: turns}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: scriptedDelegateRun(delegate)}

	result := driveWith(t, rt, delegatingRequest(delegate), fx)

	assert.Len(t, fx.calls, maxDelegationsPerTurn)
	last := result.Messages[len(result.Messages)-2]
	assert.True(t, last.ToolFailed)
	assert.Contains(t, last.Content, "already handed out")
	assert.Len(t, result.Delegations, maxDelegationsPerTurn)
}

// A delegate that ended partway is a failed call, but what it made is still
// named, and nothing says the rest did not happen: it is unconfirmed.
func TestDelegate_AFailedDelegateKeepsWhatItMadeAndDeniesNothing(t *testing.T) {
	t.Parallel()

	delegate := reportBuilder()
	run := scriptedDelegateRun(delegate)
	run.Failure = "The model provider did not answer Report Builder in time."
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		delegateTurn(delegate.ID, "Build it."),
		textTurn("The report was saved, but the rest is unconfirmed."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: run}

	result := driveWith(t, rt, delegatingRequest(delegate), fx)

	answer := result.Messages[7]
	assert.True(t, answer.ToolFailed)
	assert.Contains(t, answer.Content, "rd_123")
	assert.Contains(t, answer.Content, "unconfirmed")
	assert.NotContains(t, answer.Content, "did not happen")
	require.Len(t, result.Delegations, 1)
	assert.True(t, result.Delegations[0].Failed)
}

// A delegate that could not be asked at all says why, and leaves nothing on
// the turn but the refused call.
func TestDelegate_ADeclinedDelegatePassesOnTheReason(t *testing.T) {
	t.Parallel()

	delegate := reportBuilder()
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		delegateTurn(delegate.ID, "Build it."),
		textTurn("The Report Builder is disabled."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	fx := &delegateRecorder{run: DelegateRun{
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

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		textTurn("Done."),
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
			Role:      conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{ID: "call_1", Name: delegateTaskName}},
		},
		delegated,
		{Role: conversation.RoleTool, ToolCallID: "call_1", ToolName: delegateTaskName,
			Content: "the account"},
		{Role: conversation.RoleAssistant, Content: "Done."},
	}

	turn := rt.OpenTurn(t.Context(), req)

	for _, message := range turn.messages {
		assert.NotEqual(t, delegated.Content, message.Content)
	}
	assert.Len(t, turn.messages, 5, "question, call, answer, reply, and the new question")
}

// A delegate's step keys are its own, and the turn's own keys are exactly what
// they were before delegation existed, so an in-flight turn's ledger still
// answers for its steps.
func TestStepKey_ScopeSeparatesADelegatesStepsAndLeavesTheOwnersAlone(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("atrn_")
	params := StepKeyParams{
		OwnerID:  owner,
		ToolName: "create_report",
		Args:     map[string]any{"name": "On-time"},
	}
	own := StepKey(params)

	base, ok := callKey(serviceports.ToolCall{Name: params.ToolName, Arguments: params.Args})
	require.True(t, ok)
	legacy := sha256.Sum256([]byte(owner.String() + "\x00" + base + "\x00" + "0"))
	assert.Equal(t, hex.EncodeToString(legacy[:]), own)

	params.Scope = "delegate-a"
	scoped := StepKey(params)
	params.Scope = "delegate-b"
	other := StepKey(params)
	assert.NotEqual(t, own, scoped)
	assert.NotEqual(t, scoped, other)
}

// find_tools that finds nothing the turn can call points at an agent it may
// ask that holds what matched, rather than at an administrator.
func TestDelegatedSearchNote_NamesTheAgentThatHoldsIt(t *testing.T) {
	t.Parallel()

	delegate := reportBuilder()
	note := delegatedSearchNote(
		[]agentdefinition.RuntimeDelegate{delegate},
		[]string{"create_report", "list_invoices"},
	)

	assert.Contains(t, note, "Report Builder (agentId "+delegate.ID.String()+"): create_report")
	assert.NotContains(t, note, "list_invoices")
	assert.Contains(t, note, delegateTaskName)
	assert.Empty(t, delegatedSearchNote(nil, []string{"create_report"}))
}

// The saved account is bounded, while the delegating model reads the whole
// answer: a long reply is cut with an ellipsis on the message only.
func TestDelegate_TheSavedAccountIsBoundedAndTheModelReadsItWhole(t *testing.T) {
	t.Parallel()

	delegate := reportBuilder()
	run := scriptedDelegateRun(delegate)
	run.Result.Reply = strings.Repeat("word ", 1000) + "end"
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		delegateTurn(delegate.ID, "Build it."),
		textTurn("Done."),
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
