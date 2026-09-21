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
	// Simulated is set when the agent was in simulation and the write was
	// previewed instead of made; Simulation is the preview.
	Simulated  bool                  `json:"simulated"`
	Simulation *agent.ToolSimulation `json:"simulation,omitempty"`
	// Target is the record this action would change and its version as of the
	// proposal, when the tool names one. Nil means the tool has no single
	// target, or its version could not be read; either way the proposal is
	// still made, just without the staleness check.
	Target *ProposalTarget `json:"target,omitempty"`
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
}

// ProposalOutcome is the current state of a proposal an earlier turn raised.
type ProposalOutcome struct {
	SourceMessageID pulid.ID
	ToolName        string
	ToolParams      map[string]any
	Rationale       string
	Status          agent.ProposalStatus
	ExecutionError  string
	ExecutedAt      *int64
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

// ToolObserver is told about each tool call as it finishes.
type ToolObserver func(observation ToolObservation)

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
}

type AgentRuntime interface {
	Run(ctx context.Context, req *RunRequest) (*RunResult, error)
	// ToolSummaries describes the tools a definition may use, in the shape the
	// prompt lists them. A configured tool that is no longer registered is
	// omitted rather than described.
	ToolSummaries(definition *agentdefinition.Definition) []agentdefinition.ToolSummary
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
}
