package watchtowerports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// WatchtowerItemInput is what a source says about one record worth a
// person's attention. The projector turns it into an item and keeps it in
// step with later calls for the same kind and id.
type WatchtowerItemInput struct {
	TenantInfo  pagination.TenantInfo
	SourceKind  watchtower.SourceKind
	SourceID    string
	Severity    watchtower.Severity
	Title       string
	Summary     string
	SubjectType agent.SubjectType
	SubjectID   pulid.ID
	EventKind   agenttrigger.EventKind
	// RaisesEvent has the projector publish EventKind to the agents that
	// subscribe to it when the item is first raised, and again when it
	// reopens. It is for a source with no moment of its own to publish from:
	// a sweep that finds a condition rather than a service that causes it.
	RaisesEvent bool
	Path        string
	OccurredAt  int64
	OwnerID     pulid.ID
	Deadline    *watchtower.Deadline
	Record      *watchtower.RecordRef
	About       pulid.ID
	Facts       []watchtower.Fact
	Impact      []string
	Trail       []watchtower.TrailEvent
	// Panel is what is happening, in the shape made for it: a stop and its
	// window, a clock, a paper that is missing. Left nil, the item is read
	// from its facts.
	Panel *watchtower.Panel
}

// WatchtowerProjector is what a source calls as its records open and close.
// Every method is best effort: a lost projection costs a reconcile, never
// the write that raised it.
type WatchtowerProjector interface {
	Upsert(ctx context.Context, input WatchtowerItemInput)
	Resolve(
		ctx context.Context,
		tenant pagination.TenantInfo,
		kind watchtower.SourceKind,
		sourceID string,
	)
}

type ReconcileSubjectRequest struct {
	TenantInfo pagination.TenantInfo
	SourceKind watchtower.SourceKind
	SubjectID  pulid.ID
	Open       []WatchtowerItemInput
}

// WatchtowerSubjectReconciler settles one record's items of a kind at once:
// what is open is raised, and what that record no longer reports is
// resolved. It is best effort, like the projector.
type WatchtowerSubjectReconciler interface {
	ReconcileSubject(ctx context.Context, req *ReconcileSubjectRequest)
}

// WatchtowerSource is a source that can say what is open right now, so the
// watchtower can be filled in when it is switched on and corrected each
// night against the record.
type WatchtowerSource interface {
	Kind() watchtower.SourceKind
	Snapshot(ctx context.Context, tenant pagination.TenantInfo) ([]WatchtowerItemInput, error)
}

type ListWatchtowerItemsRequest struct {
	TenantInfo     pagination.TenantInfo
	Kinds          []watchtower.SourceKind
	Severities     []watchtower.Severity
	UnresolvedOnly bool
	Since          int64
	After          string
	First          int
	Statuses       []watchtower.Status
	AssignedToMe   bool
	AssigneeIDs    []pulid.ID
	Unassigned     bool
	OverdueOnly    bool
	SubjectID      pulid.ID
	AboutIDs       []pulid.ID
	OrderByDue     bool
	AgentWorking   bool
}

type WatchtowerPage struct {
	Items       []*watchtower.Item
	HasNextPage bool
	EndCursor   string
	Cursors     []string
	// SeenAt is the reader's cursor, so a client can draw the unseen line.
	SeenAt int64
}

type WatchtowerCounts struct {
	Unresolved     int
	Critical       int
	Active         int
	ActiveCritical int
	UnseenCritical int
	Unseen         int
	Overdue        int
	Unassigned     int
	Mine           int
	MineOverdue    int
	WithAgents     int
	ByKind         map[watchtower.SourceKind]int
	SeenAt         int64
}

type HandOffWatchtowerItemRequest struct {
	TenantInfo pagination.TenantInfo
	ItemID     pulid.ID
	// AgentDefinitionID starts that agent on the item's subject. Empty
	// publishes the item's event to whoever subscribes to it.
	AgentDefinitionID pulid.ID
}

// HandOffWatchtowerItemResult says what the hand-off did. With a run, an
// agent is working on it; with subscribers, the event went to them; with
// neither, the candidates are who could take it, for the person to pick.
type HandOffWatchtowerItemResult struct {
	// Item is the item as it stands after the hand-off, so a client can
	// redraw the row without fetching the feed again.
	Item        *watchtower.Item
	Run         *agent.AgentRun
	Subscribers []*agentdefinition.Definition
	Candidates  []*agentdefinition.Definition
	Templates   []agenttemplate.Template
}

type WatchtowerSweepResult struct {
	Upserted int
	Resolved int
	// Failed names the sources that could not be read.
	Failed []string
}

type ExceptionTarget struct {
	TenantInfo      pagination.TenantInfo
	ItemID          pulid.ID
	ExpectedVersion int64
	DryRun          bool
}

type AssignExceptionRequest struct {
	ExceptionTarget
	AssigneeID pulid.ID
	Note       string
}

type SnoozeExceptionRequest struct {
	ExceptionTarget
	Until int64
	Note  string
}

type SetExceptionDueRequest struct {
	ExceptionTarget
	DueAt *int64
	Note  string
}

type CommentOnExceptionRequest struct {
	ExceptionTarget
	Body string
}

type ResolveExceptionRequest struct {
	ExceptionTarget
	Code watchtower.ResolutionCode
	Note string
}

type ReopenExceptionRequest struct {
	ExceptionTarget
	Note string
}

type UnsnoozeExceptionRequest struct {
	ExceptionTarget
}

type ExceptionHistoryRequest struct {
	TenantInfo pagination.TenantInfo
	ItemID     pulid.ID
	Limit      int
}

type ExceptionService interface {
	GetException(
		ctx context.Context,
		req GetWatchtowerItemRequest,
		actor *permissionports.RequestActor,
	) (*watchtower.Item, error)
	AssignException(
		ctx context.Context,
		req *AssignExceptionRequest,
		actor *permissionports.RequestActor,
	) (*watchtower.Item, error)
	AcknowledgeException(
		ctx context.Context,
		req *ExceptionTarget,
		actor *permissionports.RequestActor,
	) (*watchtower.Item, error)
	StartException(
		ctx context.Context,
		req *ExceptionTarget,
		actor *permissionports.RequestActor,
	) (*watchtower.Item, error)
	SnoozeException(
		ctx context.Context,
		req *SnoozeExceptionRequest,
		actor *permissionports.RequestActor,
	) (*watchtower.Item, error)
	SetExceptionDue(
		ctx context.Context,
		req *SetExceptionDueRequest,
		actor *permissionports.RequestActor,
	) (*watchtower.Item, error)
	CommentOnException(
		ctx context.Context,
		req *CommentOnExceptionRequest,
		actor *permissionports.RequestActor,
	) (*watchtower.Event, error)
	ResolveException(
		ctx context.Context,
		req *ResolveExceptionRequest,
		actor *permissionports.RequestActor,
	) (*watchtower.Item, error)
	ReopenException(
		ctx context.Context,
		req *ReopenExceptionRequest,
		actor *permissionports.RequestActor,
	) (*watchtower.Item, error)
	UnsnoozeException(
		ctx context.Context,
		req *UnsnoozeExceptionRequest,
		actor *permissionports.RequestActor,
	) (*watchtower.Item, error)
	ExceptionHistory(
		ctx context.Context,
		req *ExceptionHistoryRequest,
		actor *permissionports.RequestActor,
	) ([]*watchtower.Event, error)
}

type WatchtowerService interface {
	List(
		ctx context.Context,
		req ListWatchtowerItemsRequest,
		actor *permissionports.RequestActor,
	) (*WatchtowerPage, error)
	Counts(
		ctx context.Context,
		tenant pagination.TenantInfo,
		actor *permissionports.RequestActor,
	) (*WatchtowerCounts, error)
	MarkSeen(
		ctx context.Context,
		tenant pagination.TenantInfo,
		actor *permissionports.RequestActor,
		seenAt int64,
	) (*WatchtowerCounts, error)
	Dismiss(
		ctx context.Context,
		tenant pagination.TenantInfo,
		id pulid.ID,
		actor *permissionports.RequestActor,
	) (*watchtower.Item, error)
	HandOff(
		ctx context.Context,
		req HandOffWatchtowerItemRequest,
		actor *permissionports.RequestActor,
	) (*HandOffWatchtowerItemResult, error)
	HandOffAgents(
		ctx context.Context,
		tenant pagination.TenantInfo,
		actor *permissionports.RequestActor,
	) ([]*agentdefinition.Definition, error)
	MarkItemSeen(
		ctx context.Context,
		req GetWatchtowerItemRequest,
		actor *permissionports.RequestActor,
	) (*watchtower.Item, error)
	Cleared(
		ctx context.Context,
		tenant pagination.TenantInfo,
		actor *permissionports.RequestActor,
		since int64,
	) (int, error)
	// Backfill fills the feed from every source's snapshot; Reconcile
	// resolves what the sources no longer report as open and adds what they
	// do. Both run per tenant.
	Backfill(ctx context.Context, tenant pagination.TenantInfo) (*WatchtowerSweepResult, error)
	Reconcile(ctx context.Context, tenant pagination.TenantInfo) (*WatchtowerSweepResult, error)
	ReconcileKind(
		ctx context.Context,
		tenant pagination.TenantInfo,
		kind watchtower.SourceKind,
	) (*WatchtowerSweepResult, error)
}

type ExceptionSweepResult struct {
	Woke      int
	Escalated int
	Failed    int
}

type ExceptionSweeper interface {
	SweepAttention(
		ctx context.Context,
		tenant pagination.TenantInfo,
		now int64,
	) (*ExceptionSweepResult, error)
}

// SettleRecordRequest closes what is open about one record because the work
// that answers it has been done.
type SettleRecordRequest struct {
	TenantInfo pagination.TenantInfo
	// RecordID is the record the conditions are about.
	RecordID pulid.ID
	// Note says what settled them, in the trail of each item it closes.
	Note string
}

// RecordSettler closes the conditions open about a record when a plan about
// it has run to its end. Decisions waiting on a person are left alone: they
// are other work, not the condition the plan answered. A source that still
// finds the problem raises it again.
type RecordSettler interface {
	SettleRecord(
		ctx context.Context,
		req *SettleRecordRequest,
		actor *permissionports.RequestActor,
	) int
}

// SettleRunRequest closes what was handed to an agent's run that looked and
// found nothing to change.
type SettleRunRequest struct {
	TenantInfo pagination.TenantInfo
	// RunID is the run the items were handed to.
	RunID pulid.ID
	// Note is what the agent found, kept in the trail of each item it closes.
	Note string
}

// RunSettler closes the items an agent's run was working once the run ends
// having proposed nothing: the agent looked, there was nothing to change, and
// leaving the item in progress would say work is under way that is not. The
// item is closed as a person's Done closes it, so it stays closed while the
// source keeps reporting the same problem and comes back when the problem
// clears and returns.
type RunSettler interface {
	SettleRun(
		ctx context.Context,
		req *SettleRunRequest,
		actor *permissionports.RequestActor,
	) int
}
