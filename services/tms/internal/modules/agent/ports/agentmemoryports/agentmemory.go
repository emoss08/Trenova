package agentmemoryports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// MemorySubjectRef names one record a memory can be about.
type MemorySubjectRef struct {
	Type agentmemory.MemorySubjectType
	ID   pulid.ID
}

type GetAgentMemoryByIDRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListAgentMemoriesByIDsRequest struct {
	TenantInfo pagination.TenantInfo
	IDs        []pulid.ID
}

type ListAgentMemoryReplacementsRequest struct {
	TenantInfo  pagination.TenantInfo
	ReplacedIDs []pulid.ID
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
	Reader           agentmemory.MemoryReader
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
	Reader            agentmemory.MemoryReader
	Now               int64
	Query             string
	// Terms are the query's meaningful words. When no memory matches every
	// word, a memory sharing two of them is found instead.
	Terms    []string
	IDs      []pulid.ID
	Kind     agentmemory.MemoryKind
	Subject  *MemorySubjectRef
	ToolName string
	Limit    int
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
	Scope             agentmemory.MemoryScope
	AgentDefinitionID pulid.ID
	OwnerUserID       pulid.ID
	RoleID            pulid.ID
	Tainted           bool
	IncludeSuggested  bool
}

type SetAgentMemoryStatusQuery struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Status     agentmemory.MemoryStatus
	ByUserID   pulid.ID
	At         int64
}

// ReviewAgentMemoryRequest records that a person read a tainted memory and
// kept it, at the version they read: a memory changed since is not cleared.
type ReviewAgentMemoryQuery struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	ByUserID   pulid.ID
	At         int64
	Version    int64
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
	Status     agentmemory.MemoryStatus
	Kind       agentmemory.MemoryKind
	Content    string
	Scope      agentmemory.MemoryScope
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
	Scope       agentmemory.MemoryScope
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
	Reader     agentmemory.MemoryReader
	Scope      agentmemory.MemoryScope
	RoleID     pulid.ID
	Query      string
}

type ListDeskMemoriesQuery struct {
	Filter DeskMemoryFilter
	After  *DeskMemoryCursor
	Limit  int
}

// DeskMemoryRow is a memory as the Desk lists it: with the title of the
// conversation it was saved from and the name of the role it is kept for.
type DeskMemoryRow struct {
	Memory      *agentmemory.Memory
	SourceTitle string
	RoleName    string
}

// DeskMemoryCount is how many memories the reader keeps in one scope, and
// for Role in one role.
type DeskMemoryCountRow struct {
	Scope  agentmemory.MemoryScope
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
	Create(ctx context.Context, entity *agentmemory.Memory) (*agentmemory.Memory, error)
	Update(ctx context.Context, entity *agentmemory.Memory) (*agentmemory.Memory, error)
	GetByID(ctx context.Context, req GetAgentMemoryByIDRequest) (*agentmemory.Memory, error)
	ListByIDs(ctx context.Context, req ListAgentMemoriesByIDsRequest) ([]*agentmemory.Memory, error)
	ListReplacements(
		ctx context.Context,
		req ListAgentMemoryReplacementsRequest,
	) ([]*agentmemory.Memory, error)
	ListConnection(
		ctx context.Context,
		req *ListAgentMemoryConnectionRequest,
	) (*pagination.CursorListResult[*agentmemory.Memory], error)
	ListActive(
		ctx context.Context,
		req ListActiveAgentMemoriesRequest,
	) ([]*agentmemory.Memory, error)
	Search(ctx context.Context, req SearchAgentMemoriesRequest) ([]*agentmemory.Memory, error)
	FindActive(ctx context.Context, req FindActiveAgentMemoryRequest) (*agentmemory.Memory, error)
	CountActive(ctx context.Context, req CountActiveAgentMemoriesRequest) (int, error)
	SetStatus(ctx context.Context, req SetAgentMemoryStatusQuery) (*agentmemory.Memory, error)
	// Review clears an Active or Paused memory's taint for the person who
	// read it; a memory that does not taint, or that changed, is refused.
	Review(ctx context.Context, req ReviewAgentMemoryQuery) (*agentmemory.Memory, error)
	MarkUsed(ctx context.Context, req MarkAgentMemoriesUsedRequest) error
	ListSuggestionContext(
		ctx context.Context,
		req ListAgentMemorySuggestionContextRequest,
	) ([]*agentmemory.Memory, error)
	ResolveSuggestion(
		ctx context.Context,
		req ResolveAgentMemorySuggestionRequest,
	) (*agentmemory.Memory, error)
	Revise(ctx context.Context, req ReviseAgentMemoryRequest) (*agentmemory.Memory, error)
	// ListDesk reads one page of the memories a person keeps, newest first.
	ListDesk(ctx context.Context, req ListDeskMemoriesQuery) ([]*DeskMemoryRow, error)
	// CountDesk counts what ListDesk would list with no scope chosen, per
	// scope and per role, in one read.
	CountDesk(ctx context.Context, filter DeskMemoryFilter) ([]DeskMemoryCountRow, error)
	// GetDesk reads memories by id as the Desk shows them, whatever their
	// status; the caller decides which of them the person may see.
	GetDesk(ctx context.Context, req GetDeskMemoriesRequest) ([]*DeskMemoryRow, error)
	// GetPreference returns the person's preference, or nil when they have
	// never chosen.
	GetPreference(
		ctx context.Context,
		req GetAgentMemoryPreferenceRequest,
	) (*agentmemory.MemoryPreference, error)
	SavePreference(
		ctx context.Context,
		entity *agentmemory.MemoryPreference,
	) (*agentmemory.MemoryPreference, error)
}
