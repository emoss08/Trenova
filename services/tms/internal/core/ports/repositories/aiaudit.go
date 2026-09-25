package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// AIAuditSigner seals each new row onto its tenant's chain. The repository
// calls it in seq order under the chain head's row lock, with the hash of the
// row before, so the key never leaves the service that holds it.
type AIAuditSigner func(event *aiaudit.AIAuditEvent, prevHash string) error

// AppendAIAuditEventsRequest is one tenant's projected rows. Rows whose
// source key the tenant already holds are skipped; the rest are chained in
// the order given.
type AppendAIAuditEventsRequest struct {
	TenantInfo pagination.TenantInfo
	Events     []*aiaudit.AIAuditEvent
	Sign       AIAuditSigner
	// KeyID names the key the batch is signed with, empty when unsigned. It
	// is recorded on the chain head and the batch's seal.
	KeyID string
	Now   int64
}

type AppendAIAuditEventsResult struct {
	Inserted int
	FromSeq  int64
	ToSeq    int64
	HeadHash string
}

// ListAIAuditEventsRequest is the paged read behind GraphQL.
type ListAIAuditEventsRequest struct {
	Filter  *pagination.QueryOptions `json:"filter"`
	Cursor  pagination.CursorInfo    `json:"-"`
	Columns []string                 `json:"-"`
}

type GetAIAuditEventRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

// AIAuditEventScope bounds an export: a tenant, an occurred_at range, the
// end of the chain when the export was asked for, and the trail filter.
type AIAuditEventScope struct {
	TenantInfo  pagination.TenantInfo
	From        int64
	To          int64
	SnapshotSeq int64
	Filter      *aiaudit.ExportFilter
}

// ListAIAuditEventPageRequest reads the next page of a scope in
// (occurred_at, id) order, after the last row the caller already has.
type ListAIAuditEventPageRequest struct {
	Scope          AIAuditEventScope
	AfterOccurred  int64
	AfterID        pulid.ID
	HasAfterCursor bool
	Limit          int
}

// AIAuditEventSummary counts a scope and finds the ends of its chain range.
type AIAuditEventSummary struct {
	Count    int64 `bun:"row_count"`
	FirstSeq int64 `bun:"first_seq"`
	LastSeq  int64 `bun:"last_seq"`
}

// ListAIAuditChainRangeRequest walks one tenant's chain in seq order.
type ListAIAuditChainRangeRequest struct {
	TenantInfo pagination.TenantInfo
	AfterSeq   int64
	Limit      int
}

type RecordAIAuditVerificationRequest struct {
	TenantInfo  pagination.TenantInfo
	Status      aiaudit.VerificationStatus
	VerifiedSeq int64
	FailedSeq   *int64
	Detail      string
	At          int64
}

type ListAIAuditSealsRequest struct {
	TenantInfo pagination.TenantInfo
	FromSeq    int64
	Limit      int
}

// PruneAIAuditEventsRequest removes a tenant's chain up to and including a
// seq, in batches, under the prune setting the trigger honours.
type PruneAIAuditEventsRequest struct {
	TenantInfo pagination.TenantInfo
	ThroughSeq int64
	BatchSize  int
}

// ListCorrelatedAuditEntriesRequest finds the audit log rows written for the
// records a page of trail events touched, by record, acting principal and
// time window.
type ListCorrelatedAuditEntriesRequest struct {
	TenantInfo pagination.TenantInfo
	Matches    []AuditEntryMatch
	Limit      int
}

// AuditEntryMatch is one event's correlation key: the record, the principal
// the write was audited as (a person, or the unattended agent principal) and
// the event's time window.
type AuditEntryMatch struct {
	ResourceID  string
	PrincipalID pulid.ID
	WindowStart int64
	WindowEnd   int64
}

// AIAuditRepository is the trail and its chain. There is no update of an
// event and no single-row delete: rows are appended under a chain lock and
// removed only as a pruned prefix of a tenant's chain.
type AIAuditRepository interface {
	Append(
		ctx context.Context,
		req *AppendAIAuditEventsRequest,
	) (*AppendAIAuditEventsResult, error)
	ExistingSourceKeys(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		keys []string,
	) (map[string]struct{}, error)
	ListConnection(
		ctx context.Context,
		req *ListAIAuditEventsRequest,
	) (*pagination.CursorListResult[*aiaudit.AIAuditEvent], error)
	GetByID(ctx context.Context, req GetAIAuditEventRequest) (*aiaudit.AIAuditEvent, error)
	ListByIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) ([]*aiaudit.AIAuditEvent, error)
	Summarize(ctx context.Context, scope *AIAuditEventScope) (*AIAuditEventSummary, error)
	ListPage(
		ctx context.Context,
		req *ListAIAuditEventPageRequest,
	) ([]*aiaudit.AIAuditEvent, error)
	ListChainRange(
		ctx context.Context,
		req ListAIAuditChainRangeRequest,
	) ([]*aiaudit.AIAuditEvent, error)
	FirstSeq(ctx context.Context, tenantInfo pagination.TenantInfo) (int64, bool, error)
	GetChainHead(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*aiaudit.AIAuditChainHead, error)
	ListChainHeads(ctx context.Context) ([]*aiaudit.AIAuditChainHead, error)
	RecordVerification(
		ctx context.Context,
		req *RecordAIAuditVerificationRequest,
	) (*aiaudit.AIAuditChainHead, error)
	GetSealEndingAt(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		toSeq int64,
	) (*aiaudit.AIAuditSeal, error)
	ListSeals(ctx context.Context, req ListAIAuditSealsRequest) ([]*aiaudit.AIAuditSeal, error)
	LastSealBefore(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		sealedBefore int64,
	) (*aiaudit.AIAuditSeal, error)
	Prune(ctx context.Context, req PruneAIAuditEventsRequest) (int, error)
	GetWatermarks(ctx context.Context) (map[aiaudit.Source]*aiaudit.AIAuditProjectorState, error)
	SaveWatermark(ctx context.Context, state *aiaudit.AIAuditProjectorState) error
	ListCorrelatedAuditEntries(
		ctx context.Context,
		req *ListCorrelatedAuditEntriesRequest,
	) ([]*audit.Entry, error)
}

// AIAuditSourcePage reads one source across every tenant in (ts, id) order,
// after a cursor and no later than a bound.
type AIAuditSourcePage struct {
	AfterTS int64
	AfterID string
	UntilTS int64
	Limit   int
}

// AIAuditTurnWindow finds the turns of some conversations that were running
// in a period, for usage rows written before they named their turn.
type AIAuditTurnWindow struct {
	TenantInfo pagination.TenantInfo
	ThreadIDs  []pulid.ID
	From       int64
	To         int64
}

// OwnerCall names a tool call by the run or turn that made it.
type OwnerCall struct {
	OwnerID pulid.ID
	CallID  string
}

// AIAuditSourceRepository is what the projector reads: each source table in
// watermark order across tenants, and the rows a batch needs to explain
// itself, by id within a tenant.
type AIAuditSourceRepository interface {
	ListRuns(ctx context.Context, page AIAuditSourcePage) ([]*agent.AgentRun, error)
	ListTurns(ctx context.Context, page AIAuditSourcePage) ([]*conversation.AssistantTurn, error)
	ListUsage(ctx context.Context, page AIAuditSourcePage) ([]*aiusage.AIUsageRecord, error)
	ListSteps(ctx context.Context, page AIAuditSourcePage) ([]*agent.AgentRunStep, error)
	ListEvents(ctx context.Context, page AIAuditSourcePage) ([]*agent.AgentRunEvent, error)
	ListProposals(ctx context.Context, page AIAuditSourcePage) ([]*agent.AgentProposal, error)
	ListDecisions(ctx context.Context, page AIAuditSourcePage) ([]*agent.AgentDecision, error)

	RunsByIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) ([]*agent.AgentRun, error)
	TurnsByIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) ([]*conversation.AssistantTurn, error)
	TurnsByRunIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		runIDs []pulid.ID,
	) ([]*conversation.AssistantTurn, error)
	TurnsInWindow(ctx context.Context, req AIAuditTurnWindow) ([]*conversation.AssistantTurn, error)
	ThreadsByIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) ([]*conversation.Thread, error)
	EvaluationsByIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) ([]*agent.Evaluation, error)
	ProposalsByIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) ([]*agent.AgentProposal, error)
	StartedToolSteps(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ownerIDs []pulid.ID,
	) ([]*agent.AgentRunStep, error)
	ToolStepCalls(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		calls []OwnerCall,
	) (map[OwnerCall]struct{}, error)
	UserNames(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) (map[pulid.ID]string, error)
	AgentNames(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) (map[pulid.ID]string, error)
}

// AIAuditExportRepository stores export requests and their files.
type AIAuditExportRepository interface {
	Create(ctx context.Context, export *aiaudit.AIAuditExport) (*aiaudit.AIAuditExport, error)
	Update(ctx context.Context, export *aiaudit.AIAuditExport) (*aiaudit.AIAuditExport, error)
	GetByID(ctx context.Context, req GetAIAuditExportRequest) (*aiaudit.AIAuditExport, error)
	ListConnection(
		ctx context.Context,
		req *ListAIAuditExportsRequest,
	) (*pagination.CursorListResult[*aiaudit.AIAuditExport], error)
	ListExpired(ctx context.Context, before int64, limit int) ([]*aiaudit.AIAuditExport, error)
	ListStale(ctx context.Context, updatedBefore int64, limit int) ([]*aiaudit.AIAuditExport, error)
}

type GetAIAuditExportRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type ListAIAuditExportsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"-"`
}
