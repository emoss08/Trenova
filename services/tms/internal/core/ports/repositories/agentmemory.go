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
	// Reader is the person the prompt is for. Memories kept for one person or
	// one role reach only that person or the role's holders.
	Reader           agent.MemoryReader
	Now              int64
	OrganizationWide bool
	Subjects         []MemorySubjectRef
	ToolNames        []string
	Limit            int
}

// SearchAgentMemoriesRequest is the recall tool's read: the words of the
// subject label, content and tool, ranked, narrowed to ids, a kind, a subject
// or a tool when given.
type SearchAgentMemoriesRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	Reader            agent.MemoryReader
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
	OwnerUserID       pulid.ID
	RoleID            pulid.ID
	Tainted           bool
	IncludeSuggested  bool
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
	// Reconsidered lets a suggestion its person turned down be accepted
	// after all: the conversation's Undo of "Don't save".
	Reconsidered bool
	// OwnerUserID and RoleID name the readers of a suggestion accepted as a
	// User or Role memory.
	OwnerUserID pulid.ID
	RoleID      pulid.ID
	ByUserID    pulid.ID
	At          int64
	Version     int64
}

// ReviseAgentMemoryRequest rewrites what a memory says and who reads it,
// under its version: the Desk's edit and its scope change.
type ReviseAgentMemoryRequest struct {
	ID          pulid.ID
	TenantInfo  pagination.TenantInfo
	Content     string
	Scope       agent.MemoryScope
	OwnerUserID pulid.ID
	RoleID      pulid.ID
	Version     int64
}

// DeskMemoryCursor is where a page of the Desk's list ends: newest first, by
// when the memory was saved and then its id.
type DeskMemoryCursor struct {
	CreatedAt int64
	ID        pulid.ID
}

// DeskMemoryFilter is what the Desk's list may be narrowed to: one scope, and
// for Role one role; text the memory says. Only memories the reader reads are
// ever listed, and only those a person keeps: active or paused ones.
type DeskMemoryFilter struct {
	TenantInfo pagination.TenantInfo
	Reader     agent.MemoryReader
	Scope      agent.MemoryScope
	RoleID     pulid.ID
	Query      string
}

type ListDeskMemoriesRequest struct {
	Filter DeskMemoryFilter
	After  *DeskMemoryCursor
	Limit  int
}

// DeskMemoryRow is a memory as the Desk lists it: with the title of the
// conversation it was saved from and the name of the role it is kept for.
type DeskMemoryRow struct {
	Memory      *agent.Memory
	SourceTitle string
	RoleName    string
}

// DeskMemoryCount is how many memories the reader keeps in one scope, and
// for Role in one role.
type DeskMemoryCount struct {
	Scope  agent.MemoryScope
	RoleID pulid.ID
	Count  int
}

type GetDeskMemoriesRequest struct {
	TenantInfo pagination.TenantInfo
	IDs        []pulid.ID
}

type GetAgentMemoryPreferenceRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
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
	Revise(ctx context.Context, req ReviseAgentMemoryRequest) (*agent.Memory, error)
	// ListDesk reads one page of the memories a person keeps, newest first.
	ListDesk(ctx context.Context, req ListDeskMemoriesRequest) ([]*DeskMemoryRow, error)
	// CountDesk counts what ListDesk would list with no scope chosen, per
	// scope and per role, in one read.
	CountDesk(ctx context.Context, filter DeskMemoryFilter) ([]DeskMemoryCount, error)
	// GetDesk reads memories by id as the Desk shows them, whatever their
	// status; the caller decides which of them the person may see.
	GetDesk(ctx context.Context, req GetDeskMemoriesRequest) ([]*DeskMemoryRow, error)
	// GetPreference returns the person's preference, or nil when they have
	// never chosen.
	GetPreference(
		ctx context.Context,
		req GetAgentMemoryPreferenceRequest,
	) (*agent.MemoryPreference, error)
	SavePreference(
		ctx context.Context,
		entity *agent.MemoryPreference,
	) (*agent.MemoryPreference, error)
}
