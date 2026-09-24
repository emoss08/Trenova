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
	ProposalID  pulid.ID
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
	IDs               []pulid.ID
	Kind              agent.MemoryKind
	SubjectType       agent.MemorySubjectType
	SubjectID         pulid.ID
	ToolName          string
	Limit             int
}

// RecalledMemory is one memory a recall returned and how it was found: by
// its words, and once retrieval can compare meanings, by that too. Match is
// empty when nothing was searched for and the recall only narrowed.
type RecalledMemory struct {
	Memory *agent.Memory
	Match  agent.MemoryMatch
}

// MemoryContextRequest is the prompt builder's read: everything a run of
// this agent may start knowing. Records are what the turn is about (its
// subject, the page, the records the person named, and those of the turn
// that handed it its task), as page entity types or subject types with ids.
type MemoryContextRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	ToolNames         []string
	Records           []agent.EntityRef
}

// MemoryContext is what a prompt may carry, best first, and the records whose
// memories it read. The prompt keeps as many as the agent's budget holds.
type MemoryContext struct {
	Memories []*agent.Memory
	Subjects []agent.MemorySubject
}

// RecordMemoryUseRequest counts the memories a prompt carried as used.
type RecordMemoryUseRequest struct {
	TenantInfo pagination.TenantInfo
	IDs        []pulid.ID
}

// RankMemoriesRequest orders the memories a prompt may carry among those of
// equal standing, most useful first.
type RankMemoriesRequest struct {
	TenantInfo pagination.TenantInfo
	Now        int64
	Memories   []*agent.Memory
}

// MemoryRanker orders the memories a prompt may carry. Order decides which
// Facts and Corrections for tools not loaded this turn fit in the budget.
type MemoryRanker interface {
	RankMemories(ctx context.Context, req RankMemoriesRequest) ([]*agent.Memory, error)
}

// AgentMemoryUsage is how much an organization keeps for its agents against
// what it should keep.
type AgentMemoryUsage struct {
	ActiveCount     int
	ActiveSoftCap   int
	WarnAt          int
	ContentMaxChars int
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
	Recall(ctx context.Context, req RecallAgentMemoriesRequest) ([]RecalledMemory, error)
	// ForContext returns the memories a prompt may carry, best first, with
	// the records they were read for. Nothing is counted as used until a
	// prompt carries it; RecordUse does that.
	ForContext(ctx context.Context, req MemoryContextRequest) (*MemoryContext, error)
	RecordUse(ctx context.Context, req RecordMemoryUseRequest) error
	Usage(ctx context.Context, tenant pagination.TenantInfo) (*AgentMemoryUsage, error)
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
