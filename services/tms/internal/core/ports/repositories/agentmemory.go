package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// MemorySubjectRef names one record a memory can be about.
type MemorySubjectRef struct {
	Type agent.MemorySubjectType
	ID   pulid.ID
}

type GetAgentMemoryByIDRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListAgentMemoryConnectionRequest struct {
	Filter  *pagination.QueryOptions `json:"filter"`
	Cursor  pagination.CursorInfo    `json:"-"`
	Columns []string                 `json:"-"`
}

// ListActiveAgentMemoriesRequest reads the memories a prompt may carry: the
// organization-wide ones, plus any about the subjects and tools named. Rows
// about a subject come first, then organization-wide instructions, then rows
// about a tool, then the rest; instructions before corrections before facts
// within each, newest first, and the limit cuts from the end of that order.
type ListActiveAgentMemoriesRequest struct {
	TenantInfo pagination.TenantInfo
	// AgentDefinitionID is the agent the prompt is for. Memories kept for one
	// agent reach only that agent; without an agent none of them are read.
	AgentDefinitionID pulid.ID
	Now               int64
	OrganizationWide  bool
	Subjects          []MemorySubjectRef
	ToolNames         []string
	Limit             int
}

// SearchAgentMemoriesRequest is the recall tool's read: the words of the
// subject label, content and tool, ranked, narrowed to ids, a kind, a subject
// or a tool when given.
type SearchAgentMemoriesRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	Now               int64
	Query             string
	IDs               []pulid.ID
	Kind              agent.MemoryKind
	Subject           *MemorySubjectRef
	ToolName          string
	Limit             int
}

type CountActiveAgentMemoriesRequest struct {
	TenantInfo pagination.TenantInfo
	Now        int64
}

// FindActiveAgentMemoryRequest looks for a memory that already says this,
// so recording the same thing twice keeps one row. Only a row the new one
// would have been read as counts: unexpired, kept for the same readers, and
// never a tainted row standing in for a clean write.
type FindActiveAgentMemoryRequest struct {
	TenantInfo        pagination.TenantInfo
	Now               int64
	Content           string
	Subject           *MemorySubjectRef
	ToolName          string
	Scope             agent.MemoryScope
	AgentDefinitionID pulid.ID
	Tainted           bool
}

type SetAgentMemoryStatusRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Status     agent.MemoryStatus
	ByUserID   pulid.ID
	At         int64
}

type MarkAgentMemoriesUsedRequest struct {
	TenantInfo pagination.TenantInfo
	IDs        []pulid.ID
	At         int64
}

type ListAgentMemorySuggestionContextRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	Now               int64
	DismissedSince    int64
	Limit             int
}

type ResolveAgentMemorySuggestionRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Status     agent.MemoryStatus
	Kind       agent.MemoryKind
	Content    string
	Scope      agent.MemoryScope
	ByUserID   pulid.ID
	At         int64
	Version    int64
}

type AgentMemoryRepository interface {
	Create(ctx context.Context, entity *agent.Memory) (*agent.Memory, error)
	Update(ctx context.Context, entity *agent.Memory) (*agent.Memory, error)
	GetByID(ctx context.Context, req GetAgentMemoryByIDRequest) (*agent.Memory, error)
	ListConnection(
		ctx context.Context,
		req *ListAgentMemoryConnectionRequest,
	) (*pagination.CursorListResult[*agent.Memory], error)
	ListActive(ctx context.Context, req ListActiveAgentMemoriesRequest) ([]*agent.Memory, error)
	Search(ctx context.Context, req SearchAgentMemoriesRequest) ([]*agent.Memory, error)
	FindActive(ctx context.Context, req FindActiveAgentMemoryRequest) (*agent.Memory, error)
	CountActive(ctx context.Context, req CountActiveAgentMemoriesRequest) (int, error)
	SetStatus(ctx context.Context, req SetAgentMemoryStatusRequest) (*agent.Memory, error)
	MarkUsed(ctx context.Context, req MarkAgentMemoriesUsedRequest) error
	ListSuggestionContext(
		ctx context.Context,
		req ListAgentMemorySuggestionContextRequest,
	) ([]*agent.Memory, error)
	ResolveSuggestion(
		ctx context.Context,
		req ResolveAgentMemorySuggestionRequest,
	) (*agent.Memory, error)
}
