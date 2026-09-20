package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
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
}

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
