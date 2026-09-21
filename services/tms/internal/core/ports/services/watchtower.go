package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
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
	EventKind   agent.EventKind
	Path        string
	OccurredAt  int64
}

// WatchtowerProjector is what a source calls as its records open and close.
// Every method is best effort: a lost projection costs a reconcile, never
// the write that raised it.
type WatchtowerProjector interface {
	Upsert(ctx context.Context, input WatchtowerItemInput)
	Resolve(ctx context.Context, tenant pagination.TenantInfo, kind watchtower.SourceKind, sourceID string)
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
}

type WatchtowerPage struct {
	Items       []*watchtower.Item
	HasNextPage bool
	EndCursor   string
	// SeenAt is the reader's cursor, so a client can draw the unseen line.
	SeenAt int64
}

type WatchtowerCounts struct {
	Unresolved     int
	Critical       int
	UnseenCritical int
	Unseen         int
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
	Templates   []agentdefinition.Template
}

type WatchtowerSweepResult struct {
	Upserted int
	Resolved int
	// Failed names the sources that could not be read.
	Failed []string
}

type WatchtowerService interface {
	List(ctx context.Context, req ListWatchtowerItemsRequest, actor *RequestActor) (*WatchtowerPage, error)
	Counts(ctx context.Context, tenant pagination.TenantInfo, actor *RequestActor) (*WatchtowerCounts, error)
	MarkSeen(ctx context.Context, tenant pagination.TenantInfo, actor *RequestActor, seenAt int64) (*WatchtowerCounts, error)
	Dismiss(ctx context.Context, tenant pagination.TenantInfo, id pulid.ID, actor *RequestActor) (*watchtower.Item, error)
	HandOff(ctx context.Context, req HandOffWatchtowerItemRequest, actor *RequestActor) (*HandOffWatchtowerItemResult, error)
	// Backfill fills the feed from every source's snapshot; Reconcile
	// resolves what the sources no longer report as open and adds what they
	// do. Both run per tenant.
	Backfill(ctx context.Context, tenant pagination.TenantInfo) (*WatchtowerSweepResult, error)
	Reconcile(ctx context.Context, tenant pagination.TenantInfo) (*WatchtowerSweepResult, error)
}
