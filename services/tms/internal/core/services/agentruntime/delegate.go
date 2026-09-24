package agentruntime

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	delegateTaskName = "delegate_task"

	// maxDelegationsPerTurn bounds how many tasks one turn hands out. Each
	// is a whole turn of another agent, paid for in model calls and in the
	// reader's patience; a turn that needs more is doing too much at once.
	maxDelegationsPerTurn = 3

	// maxDelegateTaskRunes bounds a task. It is the other agent's whole
	// question, and a model pasting a conversation into it has not decided
	// what it is asking for.
	maxDelegateTaskRunes = 4000
)

// delegateTaskDescription is a constant for the same reason
// findToolsDescription is: it is addressed to a model, and the i18n extractor
// would otherwise harvest it.
const delegateTaskDescription = "Hand a task to another agent that holds tools you " +
	"do not, and get back what it did. It works on its own as the same person, with its " +
	"own tools and approvals, and cannot see this conversation, so the task must say " +
	"everything it needs and what to hand back, such as the id of what it creates. The " +
	"result names what it made, what waits on the person's approval and what it " +
	"published. Use it only when your own tools cannot do the job; one task per call."

// delegatedRefusal answers delegate_task from a turn that is itself working
// for another agent. Only the agent the person is talking to delegates, so a
// chain of agents asking agents can never form.
const delegatedRefusal = "You are working on a task another agent handed you, and you " +
	"cannot hand it on. Do what you can with your own tools, and say plainly in your " +
	"answer what is left and which tools it would need."

// delegatedAskRefusal answers ask_user from a turn working for another agent.
// Only that agent reads it, and the person answers the agent they are talking
// to, not this one.
const delegatedAskRefusal = "You are working on a task another agent handed you, and you " +
	"cannot ask the person anything. Decide as the task allows, and say in your answer " +
	"what you assumed or what you need."

// delegateJSON encodes what a delegate did, keys in order: it is built in
// workflow code, and the same account has to read the same way on replay.
var delegateJSON = sonic.Config{SortMapKeys: true}.Froze()

// delegateTaskSpec is the tool a turn hands a task to another agent with.
// agentId is limited to the agents the turn may ask, so a model cannot name
// one it was not offered.
func delegateTaskSpec(delegates []agentdefinition.RuntimeDelegate) serviceports.ToolSpec {
	ids := make([]string, 0, len(delegates))
	for _, delegate := range delegates {
		ids = append(ids, delegate.ID.String())
	}

	return serviceports.ToolSpec{
		Name:        delegateTaskName,
		Description: delegateTaskDescription,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"agentId": map[string]any{
					"type": "string",
					"enum": ids,
					"description": "The agentId of the agent to ask, from the agents you " +
						"can ask.",
				},
				"task": map[string]any{
					"type": "string",
					"description": "What the agent should do and what it should hand back, " +
						"with every name, id, date and figure it needs. \"Create a report of " +
						"this month's on-time deliveries by customer, save it privately, and " +
						"return the report's id.\"",
				},
			},
			"required":             []string{"agentId", "task"},
			"additionalProperties": false,
		},
	}
}

// DelegateCall is one task the loop has decided to hand to another agent.
type DelegateCall struct {
	Call     serviceports.ToolCall           `json:"call"`
	Delegate agentdefinition.RuntimeDelegate `json:"delegate"`
	Task     string                          `json:"task"`
	// StepScope separates the delegate's steps from this turn's in the
	// ledger they share.
	StepScope string `json:"stepScope"`
	// CallIDs are the tool call ids this conversation already holds. The
	// delegate's calls are kept in the same thread, so it is given none of
	// them.
	CallIDs []string `json:"callIds,omitempty"`
	// Taint is the outside content the delegating turn had read, which the
	// delegate's turn opens with. Nil is a turn opened before taint was kept.
	Taint *agent.RunTaint `json:"taint,omitempty"`
	// AfterExternalContent says the delegating turn has read content from
	// outside the organization. The delegate starts from where it stands, so
	// a task written under that content cannot make a write on its own.
	AfterExternalContent bool `json:"afterExternalContent,omitempty"`
}

// DelegateRun is what handing a task to another agent came to, as the
// effects that ran it report it.
type DelegateRun struct {
	// Definition is the delegate as it ran; nil when it never started.
	Definition *agentdefinition.Definition `json:"definition,omitempty"`
	// Result is everything the delegate's turn did, including when it ended
	// before it finished.
	Result *serviceports.RunResult `json:"result,omitempty"`
	// Declined is why the delegate could not be asked at all, in words the
	// delegating agent can pass on.
	Declined string `json:"declined,omitempty"`
	// Failure is why the delegate's turn ended before it finished; Stopped
	// says the person stopped it.
	Failure string `json:"failure,omitempty"`
	Stopped bool   `json:"stopped,omitempty"`
	// Documents are what it kept beside the conversation.
	Documents []serviceports.DelegateDocument `json:"documents,omitempty"`
	// ExternalContent says the delegate read content from outside the
	// organization, which now reaches this turn through its answer.
	ExternalContent bool `json:"externalContent,omitempty"`
}

// delegate hands one task to another agent and answers the call with what it
// did. The delegate's steps are folded into the turn's own record, tagged, and
// its writes are kept apart so they are recorded as its own.
func (s *Service) delegate(t *Turn, fx TurnEffects, call serviceports.ToolCall) toolOutcome {
	if t.req.Delegation != nil {
		return failedOutcome("%s", delegatedRefusal)
	}

	delegate, task, refusal := t.delegateFor(call.Arguments)
	if refusal != "" {
		return failedOutcome("Tool %q was not run: %s", delegateTaskName, refusal)
	}
	if t.delegations >= maxDelegationsPerTurn {
		return failedOutcome(
			"Tool %q was not run: this turn has already handed out %d tasks, the most one "+
				"turn may. Answer with what the agents returned, and tell the person what is "+
				"left.",
			delegateTaskName, maxDelegationsPerTurn,
		)
	}
	t.delegations++

	scope := StepKey(StepKeyParams{
		OwnerID:  t.req.StepOwner.ID,
		ToolName: delegateTaskName,
		Args:     call.Arguments,
		Ordinal:  t.counts.next(call),
	})
	if scope == "" {
		scope = "call:" + call.ID
	}

	fx.Emit(serviceports.StreamEvent{
		Event: serviceports.AssistantEventDelegateStarted,
		Data: serviceports.AssistantDelegateStartedEvent{
			DelegateCallID: call.ID,
			AgentID:        delegate.ID,
			AgentName:      delegate.Name,
			Icon:           delegate.Icon,
			Accent:         delegate.Accent,
			Task:           task,
		},
	})

	run := fx.Delegate(t, DelegateCall{
		Call:      call,
		Delegate:  delegate,
		Task:      task,
		StepScope: scope,
		// Sorted, because this is built in workflow code and a map ranges in
		// a different order every time.
		CallIDs:              slices.Sorted(maps.Keys(t.callIDs)),
		AfterExternalContent: t.external,
		Taint:                t.result.Taint.Clone(),
	})
	report := delegateReport(delegate, call.ID, run)
	if run.ExternalContent {
		t.external = true
	}

	if run.Result != nil {
		t.result.Messages = append(t.result.Messages,
			tagDelegated(run.Result.Messages, delegate.ID, call.ID)...)
		t.ReserveCallIDs(usedCallIDsOf(run.Result.Messages))
		// The delegate's reply enters this turn's context, and with it
		// whatever outside content the delegate read.
		t.inheritDelegateTaint(fx, run.Result)
	}
	if run.Definition != nil && run.Result != nil {
		t.result.Delegations = append(t.result.Delegations, serviceports.DelegatedRun{
			Definition: run.Definition,
			CallID:     call.ID,
			Actions:    run.Result.Actions,
			Model:      run.Result.Model,
			Input:      task,
			Failed:     run.Failure != "" || run.Stopped,
			Taint:      run.Result.Taint.Clone(),
		})
	}

	fx.Emit(serviceports.StreamEvent{
		Event: serviceports.AssistantEventDelegateFinished,
		Data:  report,
	})

	return delegateOutcome(report)
}

// ReserveCallIDs marks tool call ids as taken, so a provider that reuses one
// is given a fresh id rather than a clash. A delegate's turn is handed the
// conversation's ids when it opens, and the conversation takes the delegate's
// when it ends: both keep their calls in one thread.
// CarryExternalContent starts a delegate's turn where the turn that handed it
// the task stands on outside content.
func (t *Turn) CarryExternalContent(external bool) {
	t.external = t.external || external
}

// ReadExternalContent reports whether the turn has read content from outside
// the organization.
func (t *Turn) ReadExternalContent() bool { return t.external }

func (t *Turn) ReserveCallIDs(ids []string) {
	for _, id := range ids {
		if id != "" {
			t.callIDs[id] = struct{}{}
		}
	}
}

// usedCallIDsOf is every tool call id the messages hold, in order.
func usedCallIDsOf(messages []conversation.Message) []string {
	ids := make([]string, 0, len(messages))
	for idx := range messages {
		for _, call := range messages[idx].ToolCalls {
			ids = append(ids, call.ID)
		}
	}

	return ids
}

// delegateFor reads a delegate_task call: which agent, and what to ask it. A
// refusal names what is wrong so the model can fix the call.
func (t *Turn) delegateFor(
	arguments map[string]any,
) (agentdefinition.RuntimeDelegate, string, string) {
	raw := strings.TrimSpace(stringArg(arguments, "agentId"))
	task := strings.TrimSpace(stringArg(arguments, "task"))

	switch {
	case raw == "":
		return agentdefinition.RuntimeDelegate{}, "", "agentId is required. " + t.delegateNames()
	case task == "":
		return agentdefinition.RuntimeDelegate{}, "", "task is required: say what the agent " +
			"should do and what to hand back."
	case utf8.RuneCountInString(task) > maxDelegateTaskRunes:
		return agentdefinition.RuntimeDelegate{}, "", fmt.Sprintf(
			"task is longer than %d characters. Say what to do and what to hand back, "+
				"with only the names, ids and figures it needs.", maxDelegateTaskRunes,
		)
	}

	id, err := pulid.Parse(raw)
	if err == nil {
		for _, delegate := range t.delegates {
			if delegate.ID == id {
				return delegate, task, ""
			}
		}
	}

	return agentdefinition.RuntimeDelegate{}, "", fmt.Sprintf(
		"%q is not an agent you can ask. %s", raw, t.delegateNames(),
	)
}

// delegateNames lists the agents the turn may ask, for a refusal.
func (t *Turn) delegateNames() string {
	names := make([]string, 0, len(t.delegates))
	for _, delegate := range t.delegates {
		names = append(names, fmt.Sprintf("%s (agentId %s)", delegate.Name, delegate.ID))
	}
	if len(names) == 0 {
		return "There are no agents you can ask on this turn."
	}

	return "You can ask: " + strings.Join(names, ", ") + "."
}

// tagDelegated marks a delegate's messages as its own steps on the task: kept
// with the conversation and shown nested under the call, never replayed to the
// model. The slice is copied so the delegate's own result is left as it was.
func tagDelegated(
	messages []conversation.Message,
	agentID pulid.ID,
	callID string,
) []conversation.Message {
	tagged := slices.Clone(messages)
	for idx := range tagged {
		tagged[idx].Kind = conversation.MessageKindDelegated
		tagged[idx].AgentDefinitionID = agentID
		tagged[idx].DelegateCallID = callID
	}

	return tagged
}

// delegateReport is the account of a task handed to another agent: how it
// ended, what it answered, and every write it made or proposed. The reader is
// shown it as delegate_finished and the delegating model reads it as the
// call's result, so both are told the same thing.
func delegateReport(
	delegate agentdefinition.RuntimeDelegate,
	callID string,
	run DelegateRun,
) serviceports.AssistantDelegateFinishedEvent {
	report := serviceports.AssistantDelegateFinishedEvent{
		DelegateCallID: callID,
		AgentID:        delegate.ID,
		AgentName:      delegate.Name,
		Icon:           delegate.Icon,
		Accent:         delegate.Accent,
		Made:           []serviceports.DelegateWrite{},
		Awaiting:       []serviceports.DelegateWrite{},
		Published:      slices.Clone(run.Documents),
	}
	if report.Published == nil {
		report.Published = []serviceports.DelegateDocument{}
	}

	result := run.Result
	if result != nil {
		report.ToolCallsUsed = result.ToolCallsUsed
		for idx := range result.Actions {
			action := &result.Actions[idx]
			write := serviceports.DelegateWrite{
				ToolName:  action.ToolName,
				CallID:    action.ToolCallID,
				Tier:      action.Tier,
				Summary:   argumentSummary(action.Arguments),
				Result:    action.ExecutionResult,
				Error:     action.ExecutionError,
				Simulated: action.Simulated,
			}
			if action.Executed || action.Simulated || action.ExecutionError != "" {
				report.Made = append(report.Made, write)
				continue
			}
			report.Awaiting = append(report.Awaiting, write)
		}
	}

	switch {
	case run.Declined != "":
		report.Status = serviceports.DelegateStatusDeclined
		report.Reason = run.Declined
	case run.Stopped:
		report.Status = serviceports.DelegateStatusStopped
		report.Reason = "The person stopped the reply before " + delegate.Name + " finished."
	case run.Failure != "":
		report.Status = serviceports.DelegateStatusFailed
		report.Reason = run.Failure
	case result == nil:
		report.Status = serviceports.DelegateStatusFailed
		report.Reason = delegate.Name + " did not report back."
	case result.OutputRefused:
		report.Status = serviceports.DelegateStatusRefused
		report.Reason = result.Reply
	case result.Exhausted:
		report.Status = serviceports.DelegateStatusExhausted
		report.Reply = result.Reply
		report.Reason = delegate.Name + " used every tool call its settings allow before it " +
			"finished."
	default:
		report.Status = serviceports.DelegateStatusCompleted
		report.Reply = result.Reply
	}

	return report
}

// delegateOutcome is what the delegating model reads about the task. A task
// that ended partway is a failed call whose writes are still named: they
// happened, and what else the delegate may have begun is unconfirmed rather
// than undone, so nothing here says a change did not happen.
func delegateOutcome(report serviceports.AssistantDelegateFinishedEvent) toolOutcome {
	encoded, err := delegateJSON.MarshalToString(report)
	if err != nil {
		encoded = fmt.Sprintf(`{"agent":%q,"status":%q}`, report.AgentName, report.Status)
	}

	outcome := toolOutcome{
		content: FenceToolResult(delegateTaskName, encoded) + "\n\n" + delegateNote(report),
		summary: summaryLine(report.AgentName),
		// The saved account is bounded so the message row stays small; the
		// model reads the whole account above.
		delegateReport: report.Bounded(),
	}

	switch report.Status {
	case serviceports.DelegateStatusDeclined:
		declined := failedOutcome("Tool %q was not run: %s", delegateTaskName, report.Reason)
		declined.delegateReport = outcome.delegateReport

		return declined
	case serviceports.DelegateStatusFailed, serviceports.DelegateStatusStopped:
		outcome.failed = true
	case serviceports.DelegateStatusCompleted,
		serviceports.DelegateStatusExhausted,
		serviceports.DelegateStatusRefused:
	}

	return outcome
}

// delegateNote tells the delegating model how to read the account above it.
func delegateNote(report serviceports.AssistantDelegateFinishedEvent) string {
	name := report.AgentName
	switch report.Status {
	case serviceports.DelegateStatusFailed, serviceports.DelegateStatusStopped:
		return "[" + name + " stopped before it finished. Everything under made happened. " +
			"Anything else it may have begun is unconfirmed: tell the person plainly, and do " +
			"not claim it happened or that it did not.]"
	case serviceports.DelegateStatusRefused:
		return "[" + name + "'s answer was withheld. Everything under made happened and " +
			"everything under awaiting is waiting on the person; tell them so, without the " +
			"withheld answer.]"
	default:
		return "[" + name + " did this as the same person. Tell the person what it did, " +
			"naming what it made. Everything under awaiting is a proposal waiting for their " +
			"approval on its card in this conversation and has not run. Use the ids under " +
			"made for your next step; never invent one.]"
	}
}

// UsableAgentsFor is what the person may use of the organization's agents in
// a conversation: whether they may talk to agents at all, and which agents
// restricted to roles their roles grant. Nobody but a person talks to an
// agent, and a check that cannot be made offers nothing.
func UsableAgentsFor(
	ctx context.Context,
	permissions serviceports.PermissionEngine,
	actor *serviceports.RequestActor,
	logger *zap.Logger,
) *serviceports.UsableAgents {
	if permissions == nil || actor == nil ||
		actor.PrincipalType != serviceports.PrincipalTypeUser {
		return &serviceports.UsableAgents{}
	}

	usable, err := permissions.AgentsUsable(ctx, actor, permission.OpCreate)
	if err != nil {
		if logger != nil {
			logger.Warn("could not check which agents the person may use; "+
				"withholding them", zap.Error(err))
		}

		return &serviceports.UsableAgents{}
	}
	if usable == nil {
		return &serviceports.UsableAgents{}
	}

	return usable
}

// PermittedTools keeps the named tools the actor may use, the same check the
// turn's own tool set is narrowed by.
func (s *Service) PermittedTools(
	ctx context.Context,
	actor *serviceports.RequestActor,
	names []string,
) []string {
	return s.permittedTools(ctx, actor, names)
}

// delegatedSearchNote answers a search that found nothing the turn can call,
// when an agent it may ask holds what matched. Pointing the model at the agent
// is the answer the person wants; "an administrator can add it" is not.
func delegatedSearchNote(
	delegates []agentdefinition.RuntimeDelegate,
	unheld []string,
) string {
	holding := delegatesHolding(delegates, unheld)
	if len(holding) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Nothing you can call matched. These agents you can ask hold tools that do:\n")
	for _, delegate := range holding {
		matched := make([]string, 0, len(unheld))
		for _, name := range unheld {
			if slices.Contains(delegate.Tools, name) {
				matched = append(matched, name)
			}
		}
		fmt.Fprintf(&b, "- %s (agentId %s): %s\n",
			delegate.Name, delegate.ID, strings.Join(matched, ", "))
	}
	b.WriteString("\nHand the task to one of them with delegate_task, saying what it should " +
		"do and what to hand back.")

	return b.String()
}

// delegatesHolding names the agents the turn may ask that hold a tool, for a
// search that found nothing the turn itself can call.
func delegatesHolding(
	delegates []agentdefinition.RuntimeDelegate,
	names []string,
) []agentdefinition.RuntimeDelegate {
	holding := make([]agentdefinition.RuntimeDelegate, 0, len(delegates))
	for _, delegate := range delegates {
		if slices.ContainsFunc(names, func(name string) bool {
			return slices.Contains(delegate.Tools, name)
		}) {
			holding = append(holding, delegate)
		}
	}

	return holding
}
