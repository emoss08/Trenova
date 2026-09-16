package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// AllowedDetectorKeys restricts a read to the detectors a reader may see.
//
// It is pushed into the query rather than applied to the results, which is what
// makes a paginated page correct: filtering after the fact gives uneven pages
// and a total that counts rows the reader never receives.
//
// An empty slice means nothing is visible, not everything. A reader holding none
// of the relevant permissions must get an empty list, and the one shape that
// could plausibly be read as "no restriction" is the one that would silently
// hand them every finding in the organization.
type AllowedDetectorKeys []string

// ListActiveInsightsRequest reads what the home screen shows: the findings that
// were present at the last refresh and that nobody has dismissed.
type ListActiveInsightsRequest struct {
	TenantInfo          pagination.TenantInfo `json:"-"`
	AllowedDetectorKeys AllowedDetectorKeys   `json:"-"`
	// Categories narrows to particular kinds of finding. Empty means all.
	Categories []insight.Category `json:"categories"`
	// Limit bounds what a widget asks for. A home screen shows a handful.
	Limit int `json:"limit"`
}

// ListInsightsRequest browses the whole history rather than the home screen's
// slice of it: any status, any severity, a page at a time.
type ListInsightsRequest struct {
	TenantInfo          pagination.TenantInfo `json:"-"`
	AllowedDetectorKeys AllowedDetectorKeys   `json:"-"`
	Categories          []insight.Category    `json:"categories"`
	Severities          []insight.Severity    `json:"severities"`
	// Statuses narrows the lifecycle. Empty means active only, because that is
	// what someone opening the page is asking about; seeing what was dismissed or
	// has since resolved is a deliberate act.
	Statuses []insight.Status `json:"statuses"`
	Limit    int              `json:"limit"`
	Offset   int              `json:"offset"`
}

// RestoreInsightRequest undoes a dismissal.
//
// Dismissing is one click next to the card, so it is one misclick away, and
// without this the mistake stands for a month. Restoring only ever returns a
// dismissed finding to active; it cannot revive one the system resolved,
// because that condition is no longer true.
type RestoreInsightRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type GetInsightByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

// DismissInsightRequest records that a person judged a finding not worth acting
// on. The reason is optional and is kept because "why did we ignore this" is the
// question asked three months later.
type DismissInsightRequest struct {
	ID         pulid.ID              `json:"id"`
	UserID     pulid.ID              `json:"-"`
	Reason     string                `json:"reason"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

// ReplaceDetectorFindingsRequest is one detector's whole result for one tenant.
//
// It is deliberately a single call rather than per-insight writes. A refresh has
// to supersede what the detector found last time, insert what it found now, and
// resolve anything that has stopped being true, and those only make sense
// together: doing them separately leaves a home screen showing a finding that
// was superseded but not yet replaced.
type ReplaceDetectorFindingsRequest struct {
	TenantInfo  pagination.TenantInfo
	DetectorKey string
	Insights    []*insight.Insight
	// SuppressedBefore is the cutoff for honouring a dismissal: a finding
	// dismissed more recently than this is not recreated.
	SuppressedBefore int64
}

// ReplaceDetectorFindingsResult reports what a refresh actually changed, so a
// run can be logged as work done rather than merely attempted.
type ReplaceDetectorFindingsResult struct {
	Created    int
	Superseded int
	Resolved   int
	Suppressed int
}

type InsightRepository interface {
	ListActive(
		ctx context.Context,
		req ListActiveInsightsRequest,
	) ([]*insight.Insight, error)
	List(
		ctx context.Context,
		req ListInsightsRequest,
	) (*pagination.ListResult[*insight.Insight], error)
	GetByID(ctx context.Context, req GetInsightByIDRequest) (*insight.Insight, error)
	Dismiss(ctx context.Context, req DismissInsightRequest) (*insight.Insight, error)
	Restore(ctx context.Context, req RestoreInsightRequest) (*insight.Insight, error)
	ReplaceDetectorFindings(
		ctx context.Context,
		req ReplaceDetectorFindingsRequest,
	) (ReplaceDetectorFindingsResult, error)
}

// InsightMetricsRepository holds the aggregate queries the detectors read.
//
// It is separate from InsightRepository because the two have nothing to do with
// each other: one stores findings, the other measures the operation. Keeping
// them apart is also what lets a detector be tested against a fake without
// standing up the storage side.
//
// Every method takes and returns plain rows. Thresholds, severity and wording
// are the detector's business, not the query's.
type InsightMetricsRepository interface {
	// CustomerOnTimeComparison counts delivery stops made on time in the window
	// and in the equally-long window before it, per customer.
	CustomerOnTimeComparison(
		ctx context.Context,
		req InsightWindowRequest,
	) ([]CustomerOnTimeRow, error)
	// UnbilledDeliveredShipments summarises shipments delivered before the cutoff
	// that have not been billed, per customer.
	UnbilledDeliveredShipments(
		ctx context.Context,
		req UnbilledShipmentsRequest,
	) ([]UnbilledShipmentRow, error)
	// UnbilledDetention summarises detention that accrued in the window and never
	// became a billed charge, per location.
	UnbilledDetention(
		ctx context.Context,
		req InsightWindowRequest,
	) ([]UnbilledDetentionRow, error)
	// CustomerEmptyMiles compares each customer's empty-mile share against the
	// organization's own average over the same window.
	CustomerEmptyMiles(
		ctx context.Context,
		req InsightWindowRequest,
	) ([]CustomerEmptyMilesRow, error)
	// ExpiringCredentials counts credentials falling due, grouped by type.
	ExpiringCredentials(
		ctx context.Context,
		req ExpiringCredentialsRequest,
	) ([]ExpiringCredentialRow, error)
}

// InsightWindowRequest is the common shape: one tenant, one period.
type InsightWindowRequest struct {
	TenantInfo  pagination.TenantInfo
	WindowStart int64
	WindowEnd   int64
}

type UnbilledShipmentsRequest struct {
	TenantInfo pagination.TenantInfo
	// DeliveredBefore is the age cutoff. A shipment delivered after this is not
	// yet late to bill.
	DeliveredBefore int64
}

type ExpiringCredentialsRequest struct {
	TenantInfo pagination.TenantInfo
	// From and Through bound the expiry window being looked at. From is normally
	// now, so an already-expired credential is reported too.
	From    int64
	Through int64
}

type CustomerOnTimeRow struct {
	CustomerID    pulid.ID `bun:"customer_id"`
	CustomerName  string   `bun:"customer_name"`
	CurrentTotal  int64    `bun:"current_total"`
	CurrentOnTime int64    `bun:"current_on_time"`
	PriorTotal    int64    `bun:"prior_total"`
	PriorOnTime   int64    `bun:"prior_on_time"`
}

type UnbilledShipmentRow struct {
	CustomerID     pulid.ID `bun:"customer_id"`
	CustomerName   string   `bun:"customer_name"`
	ShipmentCount  int64    `bun:"shipment_count"`
	TotalAmount    string   `bun:"total_amount"`
	OldestDelivery int64    `bun:"oldest_delivery"`
}

type UnbilledDetentionRow struct {
	LocationID        pulid.ID `bun:"location_id"`
	LocationName      string   `bun:"location_name"`
	OccurrenceCount   int64    `bun:"occurrence_count"`
	UnbilledAmount    string   `bun:"unbilled_amount"`
	TotalDwellMinutes int64    `bun:"total_dwell_minutes"`
}

type CustomerEmptyMilesRow struct {
	CustomerID    pulid.ID `bun:"customer_id"`
	CustomerName  string   `bun:"customer_name"`
	EmptyMiles    string   `bun:"empty_miles"`
	TotalMiles    string   `bun:"total_miles"`
	MoveCount     int64    `bun:"move_count"`
	OrgEmptyMiles string   `bun:"org_empty_miles"`
	OrgTotalMiles string   `bun:"org_total_miles"`
}

type ExpiringCredentialRow struct {
	CredentialTypeID   pulid.ID `bun:"credential_type_id"`
	CredentialTypeName string   `bun:"credential_type_name"`
	IsRequired         bool     `bun:"is_required"`
	WorkerCount        int64    `bun:"worker_count"`
	// EarliestExpiry is the soonest a credential in this group runs out, which is
	// what decides how urgent the group is.
	EarliestExpiry int64 `bun:"earliest_expiry"`
	// AlreadyExpired counts those past their date, which are a different problem
	// from those merely approaching one.
	AlreadyExpired int64 `bun:"already_expired"`
}
