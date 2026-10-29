package delegation

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/argcheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

// HandingTurn is what handing tasks to other agents reads and changes of
// the turn that hands them.
type HandingTurn interface {
	Turn
	Request() *agentruntimeports.RunRequest
	Holds(name string) bool
	// Delegations is how many tasks the turn has handed out, and
	// CountDelegation counts one more.
	Delegations() int
	CountDelegation()
	// NextOrdinal numbers this exact call within the run, so the step key of
	// a legitimately repeated call differs from the first one's.
	NextOrdinal(call *llmports.ToolCall) int
	// CallIDs are the tool call ids the conversation holds, sorted, and
	// ReserveCallIDs marks more as taken.
	CallIDs() []string
	ReserveCallIDs(ids []string)
	ReadExternalContent() bool
	CarryExternalContent(external bool)
	// MoneySpent is what the turn's automatic money writes, its delegates'
	// and the delegating turn's, have moved and not yet recorded.
	MoneySpent() agenttooltrust.MoneyUse
	// InheritDelegateTaint folds into the turn whatever outside content a
	// delegate read, announcing each new mark.
	InheritDelegateTaint(emit func(kernelports.StreamEvent), result *agentruntimeports.RunResult)
}

// Effects is what handing tasks does outside the turn.
type Effects interface {
	Emit(event kernelports.StreamEvent)
	// Delegate runs each agent's turn on its task to its end, side by side
	// where it can, reporting what each did in the order given.
	Delegate(calls []Call) []Run
	// ToolStarted announces a call the loop is about to answer.
	ToolStarted(call *llmports.ToolCall)
}

// Delegate hands one task to another agent and answers the call with what
// it did. The delegate's steps are folded into the turn's own record, tagged,
// and its writes are kept apart so they are recorded as its own.
func Delegate(t HandingTurn, fx Effects, call *llmports.ToolCall) toolcall.Result {
	task, refusal := Prepare(t, call)
	if task == nil {
		return refusal
	}
	outcome, _ := Hand(t, fx, task)

	return outcome
}

// Prepare checks a delegate_task call and counts the task it hands out; a
// nil task comes with the refusal the call is answered with.
func Prepare(t HandingTurn, call *llmports.ToolCall) (*HandedTask, toolcall.Result) {
	if t.Request().Delegation != nil {
		return nil, toolcall.Failed("%s", DelegatedRefusal)
	}

	delegate, task, refusal := For(t, call.Arguments)
	if refusal != "" {
		return nil, toolcall.Failed(
			"Tool %q was not run: %s",
			runtimetools.DelegateTaskName,
			refusal,
		)
	}
	spec := runtimetools.DelegateTaskSpec(t.Delegates())
	if err := toolschema.Validate(spec.Parameters, call.Arguments); err != nil {
		return nil, argcheck.Refusal(runtimetools.DelegateTaskName, spec.Parameters, err)
	}
	handed, refusal := HandedOver(t, call.Arguments)
	if refusal != "" {
		return nil, toolcall.Failed(
			"Tool %q was not run: %s",
			runtimetools.DelegateTaskName,
			refusal,
		)
	}
	if t.Delegations() >= MaxPerTurn {
		return nil, toolcall.Failed(
			"Tool %q was not run: this turn has already handed out %d tasks, the most one "+
				"turn may. Answer with what the agents returned, and tell the person what is "+
				"left.",
			runtimetools.DelegateTaskName, MaxPerTurn,
		)
	}
	t.CountDelegation()

	return &HandedTask{
		Call:     *call,
		Delegate: delegate,
		Task:     task,
		Handed:   handed,
	}, toolcall.Result{}
}

// FanOut hands every delegate_task call of one reply out at once, when the
// reply makes two or more the budget allows, and answers each by its call
// id. Nil when the reply's tasks are handed one at a time.
func FanOut(
	t HandingTurn,
	fx Effects,
	calls []llmports.ToolCall,
	used, budget int,
) map[string]toolcall.Result {
	if !t.Holds(runtimetools.DelegateTaskName) || t.Request().Delegation != nil {
		return nil
	}
	eligible := make([]llmports.ToolCall, 0, len(calls))
	for idx, call := range calls {
		if call.Name == runtimetools.DelegateTaskName && call.ArgumentsError == "" &&
			used+idx < budget {
			eligible = append(eligible, call)
		}
	}
	if len(eligible) < 2 {
		return nil
	}

	outcomes := make(map[string]toolcall.Result, len(eligible))
	tasks := make([]*HandedTask, 0, len(eligible))
	for idx := range eligible {
		call := &eligible[idx]
		task, refusal := Prepare(t, call)
		if task == nil {
			fx.ToolStarted(call)
			outcomes[call.ID] = refusal
			continue
		}
		task.FannedOut = true
		tasks = append(tasks, task)
	}
	settled := handAll(t, fx, tasks)
	for idx := range settled {
		outcomes[tasks[idx].Call.ID] = settled[idx]
	}

	return outcomes
}

// Hand runs another agent on a task and answers the call with what it did,
// also handing back the account the call was answered with.
func Hand(
	t HandingTurn,
	fx Effects,
	h *HandedTask,
) (outcome toolcall.Result, report assistantstreamports.AssistantDelegateFinishedEvent) {
	call := start(t, fx, h)
	runs := fx.Delegate([]Call{call})

	return settle(t, fx, h, &runs[0])
}

func handAll(t HandingTurn, fx Effects, tasks []*HandedTask) []toolcall.Result {
	if len(tasks) == 0 {
		return nil
	}
	calls := make([]Call, 0, len(tasks))
	for _, task := range tasks {
		calls = append(calls, start(t, fx, task))
	}
	runs := fx.Delegate(calls)
	outcomes := make([]toolcall.Result, 0, len(tasks))
	for idx, task := range tasks {
		outcome, _ := settle(t, fx, task, &runs[idx])
		outcomes = append(outcomes, outcome)
	}

	return outcomes
}

func start(t HandingTurn, fx Effects, h *HandedTask) Call {
	call, delegate, task := h.Call, h.Delegate, h.Task
	if h.FannedOut {
		fx.ToolStarted(&call)
	}
	scope := toolcall.StepKey(toolcall.StepKeyParams{
		OwnerID:  t.Request().StepOwner.ID,
		ToolName: runtimetools.DelegateTaskName,
		Args:     call.Arguments,
		Ordinal:  t.NextOrdinal(&call),
	})
	if scope == "" {
		scope = "call:" + call.ID
	}

	fx.Emit(kernelports.StreamEvent{
		Event: assistantstreamports.AssistantEventDelegateStarted,
		Data: assistantstreamports.AssistantDelegateStartedEvent{
			DelegateCallID: call.ID,
			AgentID:        delegate.ID,
			AgentName:      delegate.Name,
			Icon:           delegate.Icon,
			Accent:         delegate.Accent,
			Task:           task,
		},
	})

	return Call{
		Call:      call,
		Delegate:  delegate,
		Task:      task,
		StepScope: scope,
		// Sorted, because this is built in workflow code and a map ranges in
		// a different order every time.
		CallIDs:              t.CallIDs(),
		AfterExternalContent: t.ReadExternalContent(),
		Taint:                t.Result().Taint.HandedOn(),
		Context:              h.Handed,
		Directed:             h.Directed,
		MoneySpent:           t.MoneySpent(),
	}
}

func settle(
	t HandingTurn,
	fx Effects,
	h *HandedTask,
	run *Run,
) (outcome toolcall.Result, report assistantstreamports.AssistantDelegateFinishedEvent) {
	call, delegate := h.Call, h.Delegate
	report = Report(&delegate, call.ID, run)
	if run.ExternalContent {
		t.CarryExternalContent(true)
	}

	result := t.Result()
	if run.Result != nil {
		result.Messages = append(result.Messages,
			TagDelegated(run.Result.Messages, delegate.ID, call.ID)...)
		t.ReserveCallIDs(UsedCallIDsOf(run.Result.Messages))
		// The delegate's reply enters this turn's context, and with it
		// whatever outside content the delegate read.
		t.InheritDelegateTaint(fx.Emit, run.Result)
	}
	if run.Definition != nil && run.Result != nil {
		result.Delegations = append(result.Delegations, agentruntimeports.DelegatedRun{
			Definition: run.Definition,
			CallID:     call.ID,
			Actions:    run.Result.Actions,
			Model:      run.Result.Model,
			Input:      h.Task,
			Failed:     run.Failure != "" || run.Stopped,
			Taint:      run.Result.Taint.Clone(),
			Usage:      run.Result.Usage.Clone(),
		})
	}

	fx.Emit(kernelports.StreamEvent{
		Event: assistantstreamports.AssistantEventDelegateFinished,
		Data:  report,
	})

	outcome = Outcome(&report, !h.Directed)

	return outcome, report
}
