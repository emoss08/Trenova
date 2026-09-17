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
	// History is the conversation so far, oldest first, excluding Input.
	History []conversation.Message
	Input   string
	Emit    AssistantStreamEmitter
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
