package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
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
	// Scope narrows who reads the memory; empty is the organization. A
	// User memory is kept for OwnerUserID, a Role memory for RoleID.
	Scope       agent.MemoryScope
	OwnerUserID pulid.ID
	RoleID      pulid.ID
	// Suggest keeps the memory as a suggestion for the person to accept
	// rather than reading it into prompts at once: they asked to be asked.
	Suggest bool
	// PersonUserID is the person whose conversation the memory was picked
	// up in, recorded as who it came from; only they accept a suggestion.
	PersonUserID pulid.ID
}

type RememberPlan struct {
	Memory   *agent.Memory
	Existing *agent.Memory
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
	// ReaderUserID is the person the recall is for; what is kept for them
	// and for their roles is recalled with the organization's.
	ReaderUserID pulid.ID
	Query        string
	IDs          []pulid.ID
	Kind         agent.MemoryKind
	SubjectType  agent.MemorySubjectType
	SubjectID    pulid.ID
	ToolName     string
	Limit        int
	Attribution  AIUsageAttribution
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
	// ReaderUserID is the person the prompt is for, nil for a run nobody is
	// in. Only their own memories and their roles' are read with the rest.
	ReaderUserID pulid.ID
	ToolNames    []string
	Records      []agent.EntityRef
	Query        QueryVector
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
	Query      QueryVector
}

// MemoryRanker orders the memories a prompt may carry. Order decides which
// Facts and Corrections for tools not loaded this turn fit in the budget.
type MemoryRanker interface {
	RankMemories(ctx context.Context, req *RankMemoriesRequest) ([]*agent.Memory, error)
}

// AgentMemoryUsage is how much an organization keeps for its agents against
// what it should keep.
type AgentMemoryUsage struct {
	ActiveCount   int
	ActiveSoftCap int
	WarnAt        int
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

// SavedMemory is a memory a turn kept through the remember tool, or offered
// to keep.
type SavedMemory = conversation.SavedMemory

// MemoryRecall is a query tool's result that read memories back, naming them
// so the turn can say which memories it used.
type MemoryRecall interface {
	RecalledMemoryIDs() []pulid.ID
}

// MemoryRecordingTool is a write that keeps a memory and returns it, so the
// turn can show the person what it kept. The runtime runs it through Record
// in place of Execute when it runs in the turn; an approved proposal still
// runs it through Execute.
type MemoryRecordingTool interface {
	Record(ctx context.Context, params ToolExecuteParams) (*agent.Memory, error)
}

// DeskMemory is a memory as the person it is kept for sees it on the Desk.
type DeskMemory struct {
	Memory *agent.Memory
	// SourceTitle is the conversation the memory was saved from; empty for
	// one a person wrote themselves or one saved outside a conversation.
	SourceTitle string
	// RoleName names the role a Role memory is kept for.
	RoleName string
	// Editable says the person may change the memory: their own always, a
	// role's or the organization's only with the permission to.
	Editable bool
}

// DeskMemoryRole is one of the person's roles, which the Desk offers as a
// team to keep a memory for.
type DeskMemoryRole struct {
	ID   pulid.ID
	Name string
	// Writable says the person may keep memories for the role.
	Writable bool
}

// DeskMemoryCount is how many memories the person keeps in one scope.
type DeskMemoryCount struct {
	Scope  agent.MemoryScope
	RoleID pulid.ID
	Label  string
	Count  int
}

// DeskMemoryActor is the person on the Desk and what they may do with
// memories beyond their own.
type DeskMemoryActor struct {
	Actor *RequestActor
	// MayCreateShared and MayUpdateShared are the agent-memory create and
	// update permissions, which a role's and the organization's memories
	// need: they reach other people's conversations.
	MayCreateShared bool
	MayUpdateShared bool
}

type ListDeskMemoriesRequest struct {
	Actor  *DeskMemoryActor
	Scope  agent.MemoryScope
	RoleID pulid.ID
	Query  string
	After  string
	Limit  int
}

type DeskMemoryPage struct {
	Items []*DeskMemory
	// Next is the cursor of the following page; empty on the last.
	Next string
	// All counts every memory the search matches, and Counts each scope's.
	All    int
	Counts []DeskMemoryCount
}

type DeskMemorySettings struct {
	SavingMode               agent.MemorySavingMode
	Roles                    []DeskMemoryRole
	CanShareWithOrganization bool
}

type CreateDeskMemoryRequest struct {
	Actor   *DeskMemoryActor
	Content string
	Scope   agent.MemoryScope
	RoleID  pulid.ID
}

// ReviseDeskMemoryRequest changes what a memory says, who reads it, or both.
// Empty content keeps the words; an empty scope keeps the readers.
type ReviseDeskMemoryRequest struct {
	Actor   *DeskMemoryActor
	ID      pulid.ID
	Content string
	Scope   agent.MemoryScope
	RoleID  pulid.ID
	Version int64
}

// SetDeskMemoryStatusRequest pauses, resumes, forgets or brings back a memory:
// Paused, Active and Retired, the last undone by Active.
type SetDeskMemoryStatusRequest struct {
	Actor  *DeskMemoryActor
	ID     pulid.ID
	Status agent.MemoryStatus
}

// ConfirmDeskMemoryRequest accepts a memory an agent offered, as the person
// edited it and for whom they chose.
type ConfirmDeskMemoryRequest struct {
	Actor   *DeskMemoryActor
	ID      pulid.ID
	Content string
	Scope   agent.MemoryScope
	RoleID  pulid.ID
	Version int64
}

type DeskMemoryRef struct {
	Actor *DeskMemoryActor
	ID    pulid.ID
}

type AgentMemoryService interface {
	Remember(ctx context.Context, req *RememberRequest, actor *RequestActor) (*agent.Memory, error)
	PreviewRemember(
		ctx context.Context,
		req *RememberRequest,
		actor *RequestActor,
	) (*RememberPlan, error)
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
	// Reader is who a person reads memories as: themselves and every role
	// they hold, inherited ones included.
	Reader(ctx context.Context, tenant pagination.TenantInfo, userID pulid.ID) (agent.MemoryReader, error)
	// SavingMode is how the person wants what agents pick up kept.
	SavingMode(
		ctx context.Context,
		tenant pagination.TenantInfo,
		userID pulid.ID,
	) (agent.MemorySavingMode, error)

	ListDesk(ctx context.Context, req *ListDeskMemoriesRequest) (*DeskMemoryPage, error)
	// DeskByIDs reads the memories a conversation names, as many of them as
	// the person may see.
	DeskByIDs(ctx context.Context, actor *DeskMemoryActor, ids []pulid.ID) ([]*DeskMemory, error)
	DeskSettings(ctx context.Context, actor *DeskMemoryActor) (*DeskMemorySettings, error)
	SetSavingMode(
		ctx context.Context,
		actor *DeskMemoryActor,
		mode agent.MemorySavingMode,
	) (*DeskMemorySettings, error)
	CreateDesk(ctx context.Context, req *CreateDeskMemoryRequest) (*DeskMemory, error)
	ReviseDesk(ctx context.Context, req *ReviseDeskMemoryRequest) (*DeskMemory, error)
	SetDeskStatus(ctx context.Context, req *SetDeskMemoryStatusRequest) (*DeskMemory, error)
	ConfirmDesk(ctx context.Context, req *ConfirmDeskMemoryRequest) (*DeskMemory, error)
	DismissDesk(ctx context.Context, req DeskMemoryRef) (*DeskMemory, error)
}
