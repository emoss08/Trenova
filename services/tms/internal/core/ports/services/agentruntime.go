package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type PendingAction struct {
	ToolName   string             `json:"toolName"`
	Arguments  map[string]any     `json:"arguments"`
	Rationale  string             `json:"rationale"`
	Tier       agent.AutonomyTier `json:"tier"`
	ToolCallID string             `json:"toolCallId"`
	Executed   bool               `json:"executed"`
	// ExecutionError is set when an auto-executing tool ran and failed. The
	// action is still recorded so the failure is visible next to the decision it
	// would otherwise have needed.
	ExecutionError string `json:"executionError"`
	// ExecutionResult is what an auto-executing tool made, when it reports
	// it, recorded on the proposal the action becomes.
	ExecutionResult *agent.ToolExecutionResult `json:"executionResult,omitempty"`
	// Simulated is set when the agent was in simulation and the write was
	// previewed instead of made; Simulation is the preview.
	Simulated  bool                  `json:"simulated"`
	Simulation *agent.ToolSimulation `json:"simulation,omitempty"`
	// Target is the record this action would change and its version as of the
	// proposal, when the tool names one. Nil means the tool has no single
	// target, or its version could not be read; either way the proposal is
	// still made, just without the staleness check.
	Target *ProposalTarget `json:"target,omitempty"`
	// Egress is where the call's effect reaches, as the policy classified
	// it, and HeldBy what kept it below AutoExecute.
	Egress agent.EgressClass `json:"egress,omitempty"`
	HeldBy []string          `json:"heldBy,omitempty"`
	// Tainted says the turn had read outside content when the call was
	// decided.
	Tainted bool `json:"tainted,omitempty"`
}

// ProposalTarget is a record and its version at the moment a change to it was
// proposed. The executor compares it against the record before running.
type ProposalTarget struct {
	Resource permission.Resource `json:"resource"`
	ID       pulid.ID            `json:"id"`
	Version  int64               `json:"version"`
}

type RunRequest struct {
	Definition *agentdefinition.Definition
	Actor      *RequestActor
	Context    agentdefinition.RuntimeContext
	// RunID is set for a background run so auto-executing tools can tie what
	// they do to it. A chat turn has no run until proposals are recorded.
	RunID pulid.ID
	// Unattended says nobody is reading as this runs — an event-driven or
	// scheduled agent. Tools that put a question to a person are withheld,
	// since a question nobody will answer only ends the run on it.
	Unattended bool
	// ThreadID is the conversation a chat turn belongs to, for attributing
	// what the turn cost. Empty for a background run.
	ThreadID pulid.ID
	// ToolObserver, when set, sees each tool call's outcome with the data the
	// tool returned, before it is encoded for the model. The assistant turns
	// what a person would want to see — a report's rows, a record — into
	// artifacts beside the conversation.
	ToolObserver ToolObserver
	// Publishes says the run has somewhere to keep a document it publishes
	// even though no observer rides on this request: a durable turn keeps it
	// in an activity of its own, where the observer lives.
	Publishes bool
	// History is the conversation so far, oldest first, excluding Input.
	History []conversation.Message
	Input   string
	Emit    AssistantStreamEmitter
	// PreferredProviderID is the reader's own choice for this turn, which wins
	// over the definition's. An administrator pins a default for everyone on
	// the agent; a person picking a model in the composer is overriding that
	// default for their own conversation, so the more specific choice applies.
	// Empty falls back to the definition, and then to priority order.
	PreferredProviderID pulid.ID
	// PinProvider restricts the turn to PreferredProviderID rather than
	// trying it first: set when a person chose the model themselves.
	PinProvider bool
	// Proposals is what became of the writes earlier turns of this
	// conversation proposed, so the model reads the current state of each
	// rather than the "awaiting review" it was told at the time, and does not
	// propose again what is still waiting.
	Proposals []ProposalOutcome
	// Steps, together with StepOwner, makes the run's tool calls replay-safe:
	// each call is claimed before it runs and settled after, so an attempt
	// following a failure is handed the earlier answer instead of writing a
	// second time. Nil leaves the run unguarded, which is where every run stood
	// before the ledger existed and is still right for a one-shot that is never
	// retried.
	Steps     RunStepLedger
	StepOwner RunStepOwner
	// Attempt is which try of this run is executing, recorded on each step so
	// the ledger can be read back when something went wrong. One-based.
	Attempt int
	// Delegation is set on a turn working on a task another agent handed it.
	// Such a turn runs as the same person with its own agent's tools, tiers
	// and budget, and never hands the task on.
	Delegation *Delegation
	// Taint is the outside content the turn already carries when it opens:
	// its thread's, the agent's that handed it a task. The turn adds what its
	// own subject, attachments and memories bring.
	Taint *agent.RunTaint

	UsagePurpose AIUsagePurpose
}

func (r *RunRequest) AttributedPurpose() AIUsagePurpose {
	if r.UsagePurpose != AIUsagePurposeLive {
		return r.UsagePurpose
	}

	return UsagePurposeForRun(r.RunID)
}

// Delegation is the task another agent handed a turn.
type Delegation struct {
	// ParentAgentID and ParentAgentName are the agent that handed the task
	// over: the one the person is talking to.
	ParentAgentID   pulid.ID `json:"parentAgentId"`
	ParentAgentName string   `json:"parentAgentName"`
	// CallID is the delegate_task call the turn answers, which the thread
	// and the stream nest its steps under.
	CallID string `json:"callId"`
	// StepScope separates the turn's steps from those of the turn that
	// delegated, in the ledger both share. It is derived from what the
	// delegating model asked for, never from a provider's call id.
	StepScope string `json:"stepScope"`
}

// Scope is how the turn's events and saved steps are tagged.
func (d *Delegation) Scope(agentID pulid.ID) DelegateScope {
	if d == nil {
		return DelegateScope{}
	}

	return DelegateScope{AgentID: agentID, DelegateCallID: d.CallID}
}

// MayDelegate reports whether the turn may hand a task to another agent: a
// person is reading, in a conversation that keeps what the other agent does,
// it is not itself working for another agent, and at least one agent on its
// allowlist is one the person may use.
func (r *RunRequest) MayDelegate() bool {
	return r.Delegation == nil && !r.Unattended && r.ThreadID.IsNotNil() &&
		len(r.Context.Delegates) > 0
}

// StepScope is what the turn's step keys are namespaced by: empty for a turn
// working for the person directly.
func (r *RunRequest) StepScope() string {
	if r.Delegation == nil {
		return ""
	}

	return r.Delegation.StepScope
}

// ProposalOutcome is the current state of a proposal an earlier turn raised.
type ProposalOutcome struct {
	SourceMessageID pulid.ID
	ToolName        string
	ToolParams      map[string]any
	Rationale       string
	Status          agent.ProposalStatus
	// AutonomyTier is the tier the call was recorded at. An AutoExecute one
	// ran without a decision, so nobody approved it.
	AutonomyTier   agent.AutonomyTier
	ExecutionError string
	ExecutedAt     *int64
	// ExecutionResult is what the executed proposal made, when its tool
	// reported it.
	ExecutionResult *agent.ToolExecutionResult
	// Modifications are what the approver changed before approving, so the
	// model learns what actually ran rather than what it asked for.
	Modifications map[string]any
}

// Pending reports that the person has not decided yet.
func (o ProposalOutcome) Pending() bool {
	return o.Status == agent.ProposalStatusPending
}

// ToolObservation is one tool call as it finished.
type ToolObservation struct {
	Call ToolCall
	// Data is what a query tool returned; nil for a write, a refusal or a
	// failure.
	Data   any
	Failed bool
	// Action is the write the call proposed or made, when it was a write.
	Action *PendingAction
}

// KeepsDocuments reports whether a run may publish a document: somewhere
// beside the conversation is ready to keep it.
func (r *RunRequest) KeepsDocuments() bool {
	return r.ToolObserver != nil || r.Publishes
}

// ShownArtifact is what the person now sees for a tool call, so the model can
// be told it is there rather than repeating it.
type ShownArtifact struct {
	ID    pulid.ID
	Kind  string
	Title string
}

// PublishedDocument is a write-up the model asked to keep beside the
// conversation. ArtifactID, when set, names the document it revises.
type PublishedDocument struct {
	Title      string
	Body       string
	ArtifactID pulid.ID
}

// ToolObserver is told about each tool call as it finishes, and answers with
// the artifact it showed the person for it, if any. An error is a document
// that could not be kept, which the model is told.
type ToolObserver func(observation ToolObservation) (*ShownArtifact, error)

type RunResult struct {
	Reply    string
	Messages []conversation.Message
	Actions  []PendingAction
	// OutputRefused is set when the answer, rather than the question, was
	// declined by the output guard. Reply then carries the refusal.
	OutputRefused bool
	OutputRule    string
	Model         string
	ProviderID    pulid.ID
	ToolCallsUsed int
	Exhausted     bool
	// Truncated reports that the provider stopped partway through the reply.
	// The turn still counts as finished and Reply holds what arrived.
	Truncated bool
	// Delegations are the tasks the turn handed to other agents, with what
	// each one's writes came to. Their steps are among Messages, tagged; their
	// writes are recorded as their own agent's, not this one's.
	Delegations []DelegatedRun `json:",omitempty"`
	// Taint is the outside content the turn read. Nil is a turn opened
	// before taint was kept, which counts as tainted wherever it matters.
	Taint       *agent.RunTaint    `json:",omitempty"`
	Fingerprint *agent.Fingerprint `json:",omitempty"`
}

func (r *RunResult) ServedFingerprint() *agent.Fingerprint {
	if r == nil {
		return nil
	}

	return r.Fingerprint.Served(r.Model, r.ProviderID)
}

// DelegatedRun is one task a turn handed to another agent, as it is saved:
// the agent it ran as, and every write it proposed or made.
type DelegatedRun struct {
	// Definition is the delegate as it ran. Its proposals are recorded
	// against it, so a decision on one is a decision on that agent.
	Definition *agentdefinition.Definition `json:"definition"`
	CallID     string                      `json:"callId"`
	Actions    []PendingAction             `json:"actions,omitempty"`
	Model      string                      `json:"model,omitempty"`
	Input      string                      `json:"input"`
	// Failed says the delegate's turn ended before it finished.
	Failed bool `json:"failed,omitempty"`
	// Taint is the outside content the delegate's turn read, which its
	// proposals carry.
	Taint *agent.RunTaint `json:"taint,omitempty"`
}

type AgentRuntime interface {
	Run(ctx context.Context, req *RunRequest) (*RunResult, error)
	// ToolSummaries describes the tools a definition may use, in the shape the
	// prompt lists them. A configured tool that is no longer registered is
	// omitted rather than described.
	ToolSummaries(definition *agentdefinition.Definition) []agentdefinition.ToolSummary
	// PermittedTools keeps the named tools the actor may use, the check a
	// turn's own tool set is narrowed by.
	PermittedTools(ctx context.Context, actor *RequestActor, names []string) []string
}

// AgentSubjectDescriber renders the record a run or a conversation is about
// as the model should first see it: a label and the notes that matter, read
// from the services that own the record. Nil, nil means there is nothing to
// describe, as for an organization-wide run.
type AgentSubjectDescriber interface {
	Describe(
		ctx context.Context,
		tenant pagination.TenantInfo,
		subjectType agent.SubjectType,
		subjectID pulid.ID,
	) (*agentdefinition.RuntimeSubject, error)
}

type RuntimeContextBuilder interface {
	Build(ctx context.Context, req *RuntimeContextRequest) (agentdefinition.RuntimeContext, error)
}

type RuntimeContextRequest struct {
	Definition *agentdefinition.Definition
	Actor      *RequestActor
	Trigger    agent.RunTrigger
	Subject    *agentdefinition.RuntimeSubject
	Page       *agentdefinition.PageContext
	// Attachments and Mentions are what the person handed over with the
	// message that started this turn.
	Attachments []agentdefinition.RuntimeAttachment
	Mentions    []agentdefinition.RuntimeMention
	// DelegatedBy names the agent that handed this turn its task, when it is
	// working for another agent. Such a turn is offered no delegates.
	DelegatedBy string
}
