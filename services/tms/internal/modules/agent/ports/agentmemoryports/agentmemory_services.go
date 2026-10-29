package agentmemoryports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// RememberRequest records one memory. A person records with their own id;
// an agent records through its tool with the run it was in, and the service
// works out the agent from the run.
type RememberRequest struct {
	TenantInfo  pagination.TenantInfo
	Kind        agentmemory.MemoryKind
	Content     string
	SubjectType agentmemory.MemorySubjectType
	SubjectID   pulid.ID
	ToolName    string
	ExpiresAt   *int64
	RunID       pulid.ID
	ProposalID  pulid.ID
	Taint       *agent.RunTaint
	// Scope narrows who reads the memory; empty is the organization. A
	// User memory is kept for OwnerUserID, a Role memory for RoleID.
	Scope       agentmemory.MemoryScope
	OwnerUserID pulid.ID
	RoleID      pulid.ID
	// Suggest keeps the memory as a suggestion rather than reading it into
	// prompts at once: the person asked to be asked, or the memory is for a
	// team, the organization or every agent and the actor may not manage
	// shared memory, so an administrator decides. The service sets it in the
	// second case whatever the caller asked.
	Suggest bool
	// PersonUserID is the person whose conversation the memory was picked
	// up in, recorded as who it came from; only they accept a suggestion.
	PersonUserID      pulid.ID
	Replaces          pulid.ID
	Source            agentmemory.MemorySource
	AgentDefinitionID pulid.ID
	ThreadID          pulid.ID
	ReflectionID      pulid.ID
	Evidence          *agentmemory.MemoryEvidence
}

type RememberPlan struct {
	Memory   *agentmemory.Memory
	Existing *agentmemory.Memory
	Replaced *agentmemory.Memory
	Held     bool
}

// UpdateAgentMemoryRequest rewrites what a memory says or is about. Its
// source and history stay as they were.
type UpdateAgentMemoryRequest struct {
	ID          pulid.ID
	TenantInfo  pagination.TenantInfo
	Kind        agentmemory.MemoryKind
	Content     string
	SubjectType agentmemory.MemorySubjectType
	SubjectID   pulid.ID
	ToolName    string
	ExpiresAt   *int64
	Version     int64
}

type SetAgentMemoryStatusRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Status     agentmemory.MemoryStatus
}

// ReviewAgentMemoryRequest is a person saying they read a tainted memory and
// keep it, at the version they read.
type ReviewAgentMemoryRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Version    int64
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
	Kind         agentmemory.MemoryKind
	SubjectType  agentmemory.MemorySubjectType
	SubjectID    pulid.ID
	ToolName     string
	Limit        int
	Attribution  aiusageports.AIUsageAttribution
}

// RecalledMemory is one memory a recall returned and how it was found: by
// its words, and once retrieval can compare meanings, by that too. Match is
// empty when nothing was searched for and the recall only narrowed.
type RecalledMemory struct {
	Memory *agentmemory.Memory
	Match  agentmemory.MemoryMatch
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
	Records      []agentpagecontext.EntityRef
	Query        retrievalports.QueryVector
	// Text is what the turn asked, read for the words a memory shares with
	// it when its meaning cannot be compared.
	Text string
	// Focus is the turn's own message without the one before it, read first
	// for the words it shares; Text is read too only when Focus is too short
	// to say what it is about.
	Focus string
}

// MemoryContext is what a prompt may carry, best first, and the records whose
// memories it read. The prompt keeps as many as the agent's budget holds.
type MemoryContext struct {
	Memories []*agentmemory.Memory
	Subjects []agentmemory.MemorySubject
	// Relevance is which of Memories bear on what the turn asked, by meaning
	// or by shared words.
	Relevance agentmemory.MemoryRelevance
}

// RecordMemoryUseRequest counts the memories a turn used: those that bore
// on what it asked, not every one its prompt carried.
type RecordMemoryUseRequest struct {
	TenantInfo pagination.TenantInfo
	IDs        []pulid.ID
}

// RankMemoriesRequest orders the memories a prompt may carry among those of
// equal standing, most useful first.
type RankMemoriesRequest struct {
	TenantInfo pagination.TenantInfo
	Now        int64
	Memories   []*agentmemory.Memory
	Query      retrievalports.QueryVector
}

// MemoryRanker orders the memories a prompt may carry. Order decides which
// Facts and Corrections for tools not loaded this turn fit in the budget.
type MemoryRanker interface {
	RankMemories(ctx context.Context, req *RankMemoriesRequest) (RankedMemories, error)
}

// RankedMemories is the ranker's order and, when it could compare meanings,
// the memories at or above the similarity floor for the turn's query.
type RankedMemories struct {
	Memories []*agentmemory.Memory
	Similar  []pulid.ID
	Semantic bool
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
	Kind       agentmemory.MemoryKind
	Content    string
	Scope      agentmemory.MemoryScope
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
	Record(
		ctx context.Context,
		params agenttoolports.ToolExecuteParams,
	) (*agentmemory.Memory, error)
}

// DeskMemory is a memory as the person it is kept for sees it on the Desk.
type DeskMemory struct {
	Memory *agentmemory.Memory
	// SourceTitle is the conversation the memory was saved from; empty for
	// one a person wrote themselves or one saved outside a conversation.
	SourceTitle string
	// RoleName names the role a Role memory is kept for.
	RoleName string
	// Editable says the person may change the memory: their own always, a
	// role's or the organization's only with the permission to.
	Editable   bool
	Replaces   *DeskMemoryLink
	ReplacedBy *DeskMemoryLink
}

type DeskMemoryLink struct {
	ID      pulid.ID
	Content string
	Status  agentmemory.MemoryStatus
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
	Scope  agentmemory.MemoryScope
	RoleID pulid.ID
	Label  string
	Count  int
}

// DeskMemoryActor is the person on the Desk and what they may do with
// memories beyond their own.
type DeskMemoryActor struct {
	Actor *permissionports.RequestActor
	// MayManageShared is the agent-memory manage permission, which keeping,
	// changing, pausing or forgetting a role's, the organization's or every
	// agent's memories needs: they reach other people's conversations.
	// Without it a person offers such a memory and an administrator keeps it.
	MayManageShared bool
}

type ListDeskMemoriesRequest struct {
	Actor  *DeskMemoryActor
	Scope  agentmemory.MemoryScope
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
	SavingMode agentmemory.MemorySavingMode
	Roles      []DeskMemoryRole
	// CanShareWithOrganization says the person may put a memory in front of
	// the whole organization, kept at once or offered for an administrator.
	CanShareWithOrganization bool
	// ManagesShared says the person keeps shared memories themselves; without
	// it what they keep for a team or the organization is offered.
	ManagesShared bool
}

type CreateDeskMemoryRequest struct {
	Actor   *DeskMemoryActor
	Content string
	Scope   agentmemory.MemoryScope
	RoleID  pulid.ID
}

// ReviseDeskMemoryRequest changes what a memory says, who reads it, or both.
// Empty content keeps the words; an empty scope keeps the readers.
type ReviseDeskMemoryRequest struct {
	Actor   *DeskMemoryActor
	ID      pulid.ID
	Content string
	Scope   agentmemory.MemoryScope
	RoleID  pulid.ID
	Version int64
}

// SetDeskMemoryStatusRequest pauses, resumes, forgets or brings back a memory:
// Paused, Active and Retired, the last undone by Active.
type SetDeskMemoryStatusRequest struct {
	Actor  *DeskMemoryActor
	ID     pulid.ID
	Status agentmemory.MemoryStatus
}

// ConfirmDeskMemoryRequest accepts a memory an agent offered, as the person
// edited it and for whom they chose.
type ConfirmDeskMemoryRequest struct {
	Actor   *DeskMemoryActor
	ID      pulid.ID
	Content string
	Scope   agentmemory.MemoryScope
	RoleID  pulid.ID
	Version int64
}

type DeskMemoryRef struct {
	Actor *DeskMemoryActor
	ID    pulid.ID
}

type AgentMemoryService interface {
	Remember(
		ctx context.Context,
		req *RememberRequest,
		actor *permissionports.RequestActor,
	) (*agentmemory.Memory, error)
	PreviewRemember(
		ctx context.Context,
		req *RememberRequest,
		actor *permissionports.RequestActor,
	) (*RememberPlan, error)
	Update(
		ctx context.Context,
		req *UpdateAgentMemoryRequest,
		actor *permissionports.RequestActor,
	) (*agentmemory.Memory, error)
	// PreviewSetStatus is the memory SetStatus would change, after the same
	// checks, so a preview can say when the change would be refused.
	PreviewSetStatus(
		ctx context.Context,
		req *SetAgentMemoryStatusRequest,
		actor *permissionports.RequestActor,
	) (*agentmemory.Memory, error)
	SetStatus(
		ctx context.Context,
		req SetAgentMemoryStatusRequest,
		actor *permissionports.RequestActor,
	) (*agentmemory.Memory, error)
	// Review clears a tainted memory's taint for the person who read it:
	// from then on it is the organization's own, followed as written, and
	// a turn that reads it is not tainted by it.
	Review(
		ctx context.Context,
		req ReviewAgentMemoryRequest,
		actor *permissionports.RequestActor,
	) (*agentmemory.Memory, error)
	GetByID(ctx context.Context, req GetAgentMemoryByIDRequest) (*agentmemory.Memory, error)
	ListConnection(
		ctx context.Context,
		req *ListAgentMemoryConnectionRequest,
	) (*pagination.CursorListResult[*agentmemory.Memory], error)
	Recall(ctx context.Context, req RecallAgentMemoriesRequest) ([]RecalledMemory, error)
	// MayManageShared reports whether the person holds agent-memory manage:
	// the right to keep, change and approve what reaches other people's
	// conversations. Nobody but a person holds it, and a check that cannot
	// be made reads as not held.
	MayManageShared(ctx context.Context, tenant pagination.TenantInfo, userID pulid.ID) bool
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
		decision *agentdecision.AgentDecision,
	) (*agentmemory.Memory, error)
	ApproveSuggestion(
		ctx context.Context,
		req *ApproveAgentMemorySuggestionRequest,
		actor *permissionports.RequestActor,
	) (*agentmemory.Memory, error)
	DismissSuggestion(
		ctx context.Context,
		req DismissAgentMemorySuggestionRequest,
		actor *permissionports.RequestActor,
	) (*agentmemory.Memory, error)
	// Reader is who a person reads memories as: themselves and every role
	// they hold, inherited ones included.
	Reader(
		ctx context.Context,
		tenant pagination.TenantInfo,
		userID pulid.ID,
	) (agentmemory.MemoryReader, error)
	// SavingMode is how the person wants what agents pick up kept.
	SavingMode(
		ctx context.Context,
		tenant pagination.TenantInfo,
		userID pulid.ID,
	) (agentmemory.MemorySavingMode, error)

	ListDesk(ctx context.Context, req *ListDeskMemoriesRequest) (*DeskMemoryPage, error)
	// DeskByIDs reads the memories a conversation names, as many of them as
	// the person may see.
	DeskByIDs(ctx context.Context, actor *DeskMemoryActor, ids []pulid.ID) ([]*DeskMemory, error)
	DeskSettings(ctx context.Context, actor *DeskMemoryActor) (*DeskMemorySettings, error)
	SetSavingMode(
		ctx context.Context,
		actor *DeskMemoryActor,
		mode agentmemory.MemorySavingMode,
	) (*DeskMemorySettings, error)
	CreateDesk(ctx context.Context, req *CreateDeskMemoryRequest) (*DeskMemory, error)
	ReviseDesk(ctx context.Context, req *ReviseDeskMemoryRequest) (*DeskMemory, error)
	SetDeskStatus(ctx context.Context, req *SetDeskMemoryStatusRequest) (*DeskMemory, error)
	ConfirmDesk(ctx context.Context, req *ConfirmDeskMemoryRequest) (*DeskMemory, error)
	DismissDesk(ctx context.Context, req DeskMemoryRef) (*DeskMemory, error)
}
