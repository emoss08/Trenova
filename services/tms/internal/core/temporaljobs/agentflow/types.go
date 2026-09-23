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

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	// EventsTopic carries everything a reader of the run sees, in order.
	EventsTopic = "events"

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

	// modelHeartbeatEvery is how often a model call says it is alive, whether
	// or not tokens are arriving. Heartbeating only on output meant a model
	// thinking silently for longer than the heartbeat timeout was killed
	// mid-thought.
	modelHeartbeatEvery = 10 * time.Second

	// modelHeartbeatTimeout is how long silence means the worker is gone, so
	// the call moves to another worker quickly rather than after the full call
	// timeout.
	modelHeartbeatTimeout = 45 * time.Second

	// modelCallAttempts bounds Temporal's retries of a model call. The router
	// already falls through providers and waits out a busy one within each
	// attempt, so these attempts are for what outlasts that: a worker lost
	// mid-call, or every provider unavailable for a minute.
	modelCallAttempts = 3

	// toolTimeout bounds one tool call. Reports are the slowest tools.
	toolTimeout = 5 * time.Minute

	// findTimeout bounds a tool search, which is a catalog read plus
	// permission checks.
	findTimeout = 30 * time.Second

	// maxProviderBackoff caps how long a provider's own Retry-After is
	// honoured. A provider asking for longer is treated as unavailable rather
	// than waited on.
	maxProviderBackoff = time.Minute

	// restingBackoff is how long to wait when every provider is resting after
	// repeated failures, which is the length of the breaker's rest.
	restingBackoff = time.Minute
)

// Priority keys order a queue's work: lower runs first. They matter only
// where kinds of work share a queue, and fairness keys by organization then
// share each priority out between tenants.
const (
	PriorityInteractive = 1
	PriorityOneShot     = 2
	PriorityBackground  = 3
	PriorityEvaluation  = 5
)

// Non-retryable error types. A model call that fails with one of these would
// fail the same way however many times it was asked.
const (
	ErrTypeModelRejected        = "ModelRejected"
	ErrTypeNoProviderConfigured = "NoProviderConfigured"
	ErrTypeUnknownTool          = "UnknownTool"
	ErrTypeBadToolInput         = "BadToolInput"
)

// StreamItem is one event on a run's stream, as the reader receives it.
type StreamItem struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

// RunContext is what a run's activities need to know about the run, as data.
//
// The actor travels with the run rather than being rebuilt on a worker. A run
// acts as whoever it acts as, and a worker that reconstructed an actor from a
// tenant would be inventing an authority nobody granted.
type RunContext struct {
	Definition          *agentdefinition.Definition    `json:"definition"`
	Actor               *serviceports.RequestActor     `json:"actor"`
	Input               string                         `json:"input"`
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
}

// request is the run as the runtime reads it. The ledger and the observer are
// deliberately absent: they are services, and whichever activity needs one
// supplies its own.
func (rc *RunContext) request() *serviceports.RunRequest {
	return &serviceports.RunRequest{
		Definition:          rc.Definition,
		Actor:               rc.Actor,
		Input:               rc.Input,
		RunID:               rc.RunID,
		ThreadID:            rc.ThreadID,
		Unattended:          rc.Unattended,
		Proposals:           rc.Proposals,
		StepOwner:           rc.StepOwner,
		PreferredProviderID: rc.PreferredProviderID,
		PinProvider:         rc.PinProvider,
	}
}

type ModelCallInput struct {
	Request *serviceports.ChatCompletionRequest `json:"request"`
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
