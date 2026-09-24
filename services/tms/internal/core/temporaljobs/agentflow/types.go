// Package agentflow runs an agent's turn as Temporal workflow code.
//
// The loop lives in the workflow, and each model call and each tool call is an
// activity of its own. The loop is agentruntime's, driven through the same
// TurnEffects seam the in-process runtime uses, so there is one loop and it is
// only the effects that differ. Doing it this way buys what one opaque activity
// per run never could:
//
//   - A failure retries the one call that failed. It no longer re-enters the
//     whole turn and re-asks the model everything it already answered.
//   - Every model call and every tool call is visible by name in the Temporal
//     UI, with its input, result, retries and timing.
//   - A worker that dies mid-turn is replaced and the turn carries on from the
//     last completed step, because the workflow replays to where it was.
//
// A run's reader follows it on a Workflow Stream hosted by the run's workflow:
// the model activity publishes the reply as it streams, and the loop publishes
// every other event. Because the stream lives in the workflow, a reader can
// never attach before it exists.
package agentflow

import (
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	// EventsTopic carries everything a reader of the run sees, in order.
	EventsTopic = temporaltype.StreamEventsTopic

	// streamBatchInterval is how long the model activity holds streamed text
	// before publishing it. Each publish is a signal in the run's history, so
	// sending every token would bloat the history for nothing. A tenth of a
	// second is below what reads as lag and is the value Temporal's own AI
	// integrations use.
	streamBatchInterval = 100 * time.Millisecond

	// modelCallTimeout bounds one model call. A reasoning model can think for
	// minutes before it writes anything, so this is generous. What catches a
	// dead call early is the heartbeat, not this.
	modelCallTimeout = 10 * time.Minute

	// modelCallAttempts bounds Temporal's retries of a model call. The router
	// already falls through providers and waits out a busy one within each
	// attempt, so these attempts are for what outlasts that: a worker lost
	// mid-call, or every provider unavailable for a minute.
	modelCallAttempts = 3

	// toolTimeout bounds one tool call. Reports are the slowest tools.
	toolTimeout = 5 * time.Minute

	// heavyToolWait bounds how long a heavy tool may wait for a worker on
	// the heavy queue before the model is told it could not be run. A
	// deployment that polls no heavy queue would otherwise hang the turn.
	heavyToolWait = 15 * time.Minute

	// findTimeout bounds a tool search, which is a catalog read plus
	// permission checks.
	findTimeout = 30 * time.Second

	// openDelegateTimeout bounds opening another agent's turn on a task:
	// reading it, checking the person may use it and its budget, and
	// building its prompt and tools.
	openDelegateTimeout = time.Minute
)

// heavyTools run on the heavy queue rather than on the queue of the run that
// called them. Each builds or compares a report or an optimisation, which takes
// a worker for minutes and would otherwise hold a slot a person's reply needs.
var heavyTools = map[string]struct{}{
	"run_report":          {},
	"compare_report_runs": {},
	"plan_dispatch":       {},
}

// Priority keys order a queue's work: lower runs first. They matter only
// where kinds of work share a queue, and fairness keys by organization then
// share each priority out between tenants.
const (
	PriorityInteractive = 1
	PriorityOneShot     = 2
	PriorityBackground  = 3
	PriorityEvaluation  = 5
)

// Non-retryable error types of a tool call. A call that fails with one of these
// would fail the same way however many times it was asked.
const (
	ErrTypeUnknownTool  = "UnknownTool"
	ErrTypeBadToolInput = "BadToolInput"
	// ErrTypeDelegateDeclined is another agent that may not be handed the
	// task: its message says why, for the agent that asked.
	ErrTypeDelegateDeclined = "DelegateDeclined"
)

// StreamItem is one event on a run's stream, as the reader receives it.
type StreamItem = temporaltype.StreamItem

// RunContext is what a run's activities need to know about the run, as data.
//
// The actor travels with the run rather than being rebuilt on a worker. A run
// acts as whoever it acts as, and a worker that reconstructed an actor from a
// tenant would be inventing an authority nobody granted.
type RunContext struct {
	Definition *agentdefinition.Definition `json:"definition"`
	Actor      *serviceports.RequestActor  `json:"actor"`
	Input      string                      `json:"input"`
	// Timezone is the organization's, which is what a tool reads a bare date
	// in.
	Timezone            string                         `json:"timezone,omitempty"`
	RunID               pulid.ID                       `json:"runId,omitzero"`
	ThreadID            pulid.ID                       `json:"threadId,omitzero"`
	Unattended          bool                           `json:"unattended"`
	Proposals           []serviceports.ProposalOutcome `json:"proposals,omitempty"`
	StepOwner           serviceports.RunStepOwner      `json:"stepOwner"`
	PreferredProviderID pulid.ID                       `json:"preferredProviderId,omitzero"`
	PinProvider         bool                           `json:"pinProvider"`
	// PriorityKey orders this run's work against other kinds of work on the
	// same queue. The organization is its fairness key.
	PriorityKey int `json:"priorityKey"`
	// Delegation is set on the turn of an agent working on a task the run's
	// own agent handed it.
	Delegation *serviceports.Delegation `json:"delegation,omitempty"`
	// Records are the records the turn is about, so an agent it hands a task
	// to reads their memories too. A run recorded before they were kept has
	// none, and its delegates read the memories of no record.
	Records []agent.EntityRef `json:"records,omitempty"`
}

// Scope is how the run's events are tagged for its reader: empty for the
// run's own agent, the agent and the delegate call for one working on a task
// it was handed.
func (rc *RunContext) Scope() serviceports.DelegateScope {
	if rc.Delegation == nil || rc.Definition == nil {
		return serviceports.DelegateScope{}
	}

	return rc.Delegation.Scope(rc.Definition.ID)
}

// NewRunContext is a run request as the data a run's activities carry. The
// services on the request stay behind: whichever activity needs one supplies
// its own.
func NewRunContext(req *serviceports.RunRequest, priorityKey int) RunContext {
	return RunContext{
		Definition:          req.Definition,
		Actor:               req.Actor,
		Input:               req.Input,
		Timezone:            req.Context.Timezone,
		RunID:               req.RunID,
		ThreadID:            req.ThreadID,
		Unattended:          req.Unattended,
		Proposals:           req.Proposals,
		StepOwner:           req.StepOwner,
		PreferredProviderID: req.PreferredProviderID,
		PinProvider:         req.PinProvider,
		PriorityKey:         priorityKey,
		Delegation:          req.Delegation,
		Records:             req.Records,
	}
}

// request is the run as the runtime reads it. The ledger and the observer are
// deliberately absent: they are services, and whichever activity needs one
// supplies its own.
func (rc *RunContext) request() *serviceports.RunRequest {
	return &serviceports.RunRequest{
		Definition:          rc.Definition,
		Actor:               rc.Actor,
		Context:             agentdefinition.RuntimeContext{Timezone: rc.Timezone},
		Input:               rc.Input,
		RunID:               rc.RunID,
		ThreadID:            rc.ThreadID,
		Unattended:          rc.Unattended,
		Proposals:           rc.Proposals,
		StepOwner:           rc.StepOwner,
		PreferredProviderID: rc.PreferredProviderID,
		PinProvider:         rc.PinProvider,
		Delegation:          rc.Delegation,
		Records:             rc.Records,
	}
}

type ModelCallInput struct {
	Request *serviceports.ChatCompletionRequest `json:"request"`
	// Stream says somebody is reading the run live, so the reply is
	// published as it arrives. A run nobody watches is not streamed: every
	// publish is a signal in the run's history.
	Stream bool `json:"stream"`
	// Scope tags what is streamed when the reply is another agent's, on a
	// task the run's own agent handed it.
	Scope serviceports.DelegateScope `json:"scope,omitzero"`
}

// OpenDelegateInput asks for another agent's turn on a task the run's agent
// handed it.
type OpenDelegateInput struct {
	Run  RunContext                `json:"run"`
	Call agentruntime.DelegateCall `json:"call"`
}

// DelegateOpening is the other agent's turn, ready to drive: its run as data
// and its turn as the activity built it.
type DelegateOpening struct {
	Run  RunContext             `json:"run"`
	Turn agentruntime.TurnState `json:"turn"`
}

type FindToolsInput struct {
	Run       RunContext                `json:"run"`
	Tools     agentruntime.ToolSetState `json:"tools"`
	Arguments map[string]any            `json:"arguments"`
}

type FindToolsResult struct {
	Content string   `json:"content"`
	Loaded  []string `json:"loaded,omitempty"`
}

// ToolInput is one tool call. The activity running it is named for the tool,
// so each tool reads as itself in the Temporal UI.
type ToolInput struct {
	Run  RunContext                `json:"run"`
	Call agentruntime.DispatchCall `json:"call"`
}

// PublishInput is a document the model asked to publish, as its call.
type PublishInput struct {
	Run  RunContext            `json:"run"`
	Call serviceports.ToolCall `json:"call"`
}

type ToolResult struct {
	Outcome agentruntime.ToolOutcome `json:"outcome"`
	// Artifacts are what the call produced for a person to see beside the
	// run. They are made where the call ran, since only there does the
	// tool's raw result exist.
	Artifacts []*assistantartifact.Artifact `json:"artifacts,omitempty"`
}

// Outcome is what one driven turn came to.
type Outcome struct {
	Result *serviceports.RunResult `json:"result"`
	// Artifacts are everything the turn's tool calls produced.
	Artifacts []*assistantartifact.Artifact `json:"artifacts,omitempty"`
	// Events are what the turn said, for its durable account. The reply's
	// streamed text is not among them; the transcript already keeps it whole.
	Events []StreamItem `json:"events,omitempty"`
}
