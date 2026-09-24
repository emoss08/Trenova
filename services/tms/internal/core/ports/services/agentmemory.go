package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// RememberRequest records one memory. A person records with their own id;
// an agent records through its tool with the run it was in, and the service
// works out the agent from the run.
type RememberRequest struct {
	TenantInfo  pagination.TenantInfo
	Kind        agent.MemoryKind
	Content     string
	SubjectType agent.MemorySubjectType
	SubjectID   pulid.ID
	ToolName    string
	ExpiresAt   *int64
	RunID       pulid.ID
	Taint       *agent.RunTaint
}

// UpdateAgentMemoryRequest rewrites what a memory says or is about. Its
// source and history stay as they were.
type UpdateAgentMemoryRequest struct {
	ID          pulid.ID
	TenantInfo  pagination.TenantInfo
	Kind        agent.MemoryKind
	Content     string
	SubjectType agent.MemorySubjectType
	SubjectID   pulid.ID
	ToolName    string
	ExpiresAt   *int64
	Version     int64
}

type SetAgentMemoryStatusRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Status     agent.MemoryStatus
}

// RecallAgentMemoriesRequest is the agent's read: what has been recorded
// about this text, this record or this tool.
type RecallAgentMemoriesRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	Query             string
	Kind              agent.MemoryKind
	SubjectType       agent.MemorySubjectType
	SubjectID         pulid.ID
	ToolName          string
	Limit             int
}

// MemoryContextRequest is the prompt builder's read: everything a run of
// this agent should start knowing.
type MemoryContextRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	ToolNames         []string
	Subjects          []repositories.MemorySubjectRef
	Limit             int
}

type ApproveAgentMemorySuggestionRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Kind       agent.MemoryKind
	Content    string
	Scope      agent.MemoryScope
	Version    int64
}

type DismissAgentMemorySuggestionRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Version    int64
}

type AgentMemoryService interface {
	Remember(ctx context.Context, req *RememberRequest, actor *RequestActor) (*agent.Memory, error)
	Update(
		ctx context.Context,
		req *UpdateAgentMemoryRequest,
		actor *RequestActor,
	) (*agent.Memory, error)
	SetStatus(
		ctx context.Context,
		req SetAgentMemoryStatusRequest,
		actor *RequestActor,
	) (*agent.Memory, error)
	GetByID(ctx context.Context, req repositories.GetAgentMemoryByIDRequest) (*agent.Memory, error)
	ListConnection(
		ctx context.Context,
		req *repositories.ListAgentMemoryConnectionRequest,
	) (*pagination.CursorListResult[*agent.Memory], error)
	Recall(ctx context.Context, req RecallAgentMemoriesRequest) ([]*agent.Memory, error)
	// ForContext returns what a prompt should carry and counts each memory
	// as used.
	ForContext(ctx context.Context, req MemoryContextRequest) ([]*agent.Memory, error)
	// RecordCorrection turns a decision that changed or refused a proposal
	// into a correction, when the decision says enough to learn from.
	RecordCorrection(
		ctx context.Context,
		proposal *agent.AgentProposal,
		decision *agent.AgentDecision,
	) (*agent.Memory, error)
	ApproveSuggestion(
		ctx context.Context,
		req *ApproveAgentMemorySuggestionRequest,
		actor *RequestActor,
	) (*agent.Memory, error)
	DismissSuggestion(
		ctx context.Context,
		req DismissAgentMemorySuggestionRequest,
		actor *RequestActor,
	) (*agent.Memory, error)
}
