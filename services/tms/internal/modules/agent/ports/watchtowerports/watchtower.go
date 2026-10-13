package watchtowerports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ResolveWatchtowerItemRequest closes the item standing in for one source
// record. Resolving what is already resolved, or absent, is not an error.
type ResolveWatchtowerItemRequest struct {
	TenantInfo pagination.TenantInfo
	SourceKind watchtower.SourceKind
	SourceID   string
	ResolvedAt int64
}

// ResolveMissingWatchtowerItemsRequest closes every open item of a kind
// whose source is not among the ones still open, for a reconcile that reads
// the source's own state.
type ResolveMissingWatchtowerItemsRequest struct {
	TenantInfo    pagination.TenantInfo
	SourceKind    watchtower.SourceKind
	OpenSourceIDs []string
	ResolvedAt    int64
}

// ListWatchtowerItemsRequest pages the feed newest first by occurrence. The
// cursor is the last row's occurred_at and id, so a feed that grows while
// it is read does not skip or repeat.
type ListWatchtowerItemsQuery struct {
	TenantInfo pagination.TenantInfo
	// Kinds narrows the feed; empty means every kind the reader may see,
	// which the service resolves before asking.
	Kinds      []watchtower.SourceKind
	Severities []watchtower.Severity
	// UnresolvedOnly hides what the source has since closed.
	UnresolvedOnly bool
	// Since keeps only items that occurred after it.
	Since int64
	// BeforeOccurredAt and BeforeID are the keyset cursor; zero reads from
	// the top.
	BeforeOccurredAt int64
	BeforeID         pulid.ID
	Limit            int

	Statuses    []watchtower.Status
	AssigneeIDs []pulid.ID
	Unassigned  bool
	OverdueAt   int64
	SubjectID   pulid.ID
	OrderByDue  bool
	BeforeDueAt int64
}

// CountWatchtowerItemsRequest asks how much is open, and how much of it is
// critical and unseen by a reader whose cursor is SeenAt.
type CountWatchtowerItemsRequest struct {
	TenantInfo pagination.TenantInfo
	Kinds      []watchtower.SourceKind
	SeenAt     int64
	AssigneeID pulid.ID
	Now        int64
}

type WatchtowerKindCount struct {
	SourceKind watchtower.SourceKind `bun:"source_kind"`
	Count      int                   `bun:"count"`
}

type WatchtowerCountsRow struct {
	Unresolved     int
	Critical       int
	UnseenCritical int
	Unseen         int
	Overdue        int
	Unassigned     int
	Mine           int
	MineOverdue    int
	ByKind         []WatchtowerKindCount
}

type UpdateWatchtowerWorkflowRequest struct {
	Item            *watchtower.Item
	ExpectedVersion int64
	Events          []*watchtower.Event
}

type ListWatchtowerEventsRequest struct {
	TenantInfo pagination.TenantInfo
	ItemID     pulid.ID
	Limit      int
}

type ListAttentionWatchtowerItemsRequest struct {
	TenantInfo         pagination.TenantInfo
	Now                int64
	Limit              int
	BelowEscalationCap int16
}

type GetWatchtowerItemRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type GetWatchtowerCursorRequest struct {
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

// DeleteResolvedWatchtowerItemsRequest removes items resolved before a
// cut-off, a batch at a time, across every tenant.
type DeleteResolvedWatchtowerItemsRequest struct {
	Before int64
	Limit  int
}

type WatchtowerRepository interface {
	// Upsert replaces the item for its source, keeping the id of the row that
	// was already there. It reports whether the row is new.
	Upsert(ctx context.Context, item *watchtower.Item) (*watchtower.Item, bool, error)
	Resolve(ctx context.Context, req ResolveWatchtowerItemRequest) (*watchtower.Item, error)
	UpdateWorkflow(
		ctx context.Context,
		req *UpdateWatchtowerWorkflowRequest,
	) (*watchtower.Item, error)
	InsertEvents(ctx context.Context, events []*watchtower.Event) error
	ListEvents(ctx context.Context, req ListWatchtowerEventsRequest) ([]*watchtower.Event, error)
	ListOverdue(
		ctx context.Context,
		req ListAttentionWatchtowerItemsRequest,
	) ([]*watchtower.Item, error)
	ListSnoozeExpired(
		ctx context.Context,
		req ListAttentionWatchtowerItemsRequest,
	) ([]*watchtower.Item, error)
	ListTenantsNeedingAttention(ctx context.Context, now int64) ([]pagination.TenantInfo, error)
	ResolveMissing(ctx context.Context, req *ResolveMissingWatchtowerItemsRequest) (int, error)
	GetByID(ctx context.Context, req GetWatchtowerItemRequest) (*watchtower.Item, error)
	List(ctx context.Context, req ListWatchtowerItemsQuery) ([]*watchtower.Item, error)
	Counts(ctx context.Context, req CountWatchtowerItemsRequest) (*WatchtowerCountsRow, error)
	GetCursor(ctx context.Context, req GetWatchtowerCursorRequest) (*watchtower.Cursor, error)
	SetCursor(ctx context.Context, cursor *watchtower.Cursor) (*watchtower.Cursor, error)
	DeleteResolvedBefore(ctx context.Context, req DeleteResolvedWatchtowerItemsRequest) (int, error)
}
