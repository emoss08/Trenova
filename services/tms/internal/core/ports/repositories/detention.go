package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type ListDetentionPoliciesRequest struct {
	Filter     *pagination.QueryOptions
	Status     string
	CustomerID *pulid.ID
	LocationID *pulid.ID
}

type ListDetentionPolicyConnectionRequest struct {
	Filter                 *pagination.QueryOptions `json:"filter"`
	Cursor                 pagination.CursorInfo    `json:"-"`
	DetentionPolicyColumns []string                 `json:"-"`
}

type GetDetentionPolicyByIDRequest struct {
	DetentionPolicyID pulid.ID
	TenantInfo        pagination.TenantInfo
	IncludeTiers      bool
}

// GetResolutionCandidatesRequest loads every policy that could conceivably
// govern a stop. Scope filtering happens in the resolver rather than in SQL so
// that non-matching candidates can still be explained back to the user.
type GetResolutionCandidatesRequest struct {
	TenantInfo pagination.TenantInfo
	CustomerID pulid.ID
	LocationID pulid.ID
}

type DetentionPolicySelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest
}

type DetentionPolicyRepository interface {
	List(
		ctx context.Context,
		req *ListDetentionPoliciesRequest,
	) (*pagination.ListResult[*detention.DetentionPolicy], error)

	ListConnection(
		ctx context.Context,
		req *ListDetentionPolicyConnectionRequest,
	) (*pagination.CursorListResult[*detention.DetentionPolicy], error)

	GetByID(
		ctx context.Context,
		req *GetDetentionPolicyByIDRequest,
	) (*detention.DetentionPolicy, error)

	GetResolutionCandidates(
		ctx context.Context,
		req *GetResolutionCandidatesRequest,
	) ([]*detention.DetentionPolicy, error)

	Create(
		ctx context.Context,
		entity *detention.DetentionPolicy,
	) (*detention.DetentionPolicy, error)

	Update(
		ctx context.Context,
		entity *detention.DetentionPolicy,
	) (*detention.DetentionPolicy, error)

	Delete(ctx context.Context, req *GetDetentionPolicyByIDRequest) error

	SelectOptions(
		ctx context.Context,
		req *DetentionPolicySelectOptionsRequest,
	) (*pagination.ListResult[*detention.DetentionPolicy], error)

	CodeExists(
		ctx context.Context,
		req *DetentionPolicyCodeExistsRequest,
	) (bool, error)

	FindOverlapping(
		ctx context.Context,
		req *FindOverlappingDetentionPoliciesRequest,
	) ([]*detention.DetentionPolicy, error)
}

type DetentionPolicyCodeExistsRequest struct {
	TenantInfo pagination.TenantInfo
	Code       string
	ExcludeID  *pulid.ID
}

// FindOverlappingDetentionPoliciesRequest finds active policies sharing the
// exact same scope whose effective windows intersect. Two such policies make
// resolution ambiguous, so creation is blocked.
type FindOverlappingDetentionPoliciesRequest struct {
	TenantInfo         pagination.TenantInfo
	ExcludeID          *pulid.ID
	CustomerID         *pulid.ID
	LocationID         *pulid.ID
	IsOrgDefault       bool
	EffectiveStartDate *int64
	EffectiveEndDate   *int64
}

type ListDetentionOccurrencesRequest struct {
	Filter     *pagination.QueryOptions
	ShipmentID *pulid.ID
	CustomerID *pulid.ID
	LocationID *pulid.ID
	Status     string
	OpenOnly   bool
}

type GetDetentionOccurrenceByIDRequest struct {
	OccurrenceID    pulid.ID
	TenantInfo      pagination.TenantInfo
	IncludeEvidence bool
	IncludeNotices  bool
}

type GetOccurrencesByShipmentRequest struct {
	ShipmentID pulid.ID
	TenantInfo pagination.TenantInfo
}

type GetOccurrenceByStopRequest struct {
	StopID     pulid.ID
	TenantInfo pagination.TenantInfo
}

// GetDayAccrualRequest sums detention already booked against each calendar day
// so a per-day cap can span stops rather than resetting at every one.
type GetDayAccrualRequest struct {
	TenantInfo    pagination.TenantInfo
	ShipmentID    pulid.ID
	ExcludeStopID pulid.ID
	DayKeys       []string
}

type DetentionOccurrenceRepository interface {
	List(
		ctx context.Context,
		req *ListDetentionOccurrencesRequest,
	) (*pagination.ListResult[*detention.DetentionOccurrence], error)

	GetByID(
		ctx context.Context,
		req *GetDetentionOccurrenceByIDRequest,
	) (*detention.DetentionOccurrence, error)

	GetByStop(
		ctx context.Context,
		req *GetOccurrenceByStopRequest,
	) (*detention.DetentionOccurrence, error)

	GetByShipment(
		ctx context.Context,
		req *GetOccurrencesByShipmentRequest,
	) ([]*detention.DetentionOccurrence, error)

	ListOpen(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*detention.DetentionOccurrence, error)

	ListNoticesDue(
		ctx context.Context,
		req *ListNoticesDueRequest,
	) ([]*detention.DetentionOccurrence, error)

	Upsert(
		ctx context.Context,
		entity *detention.DetentionOccurrence,
	) (*detention.DetentionOccurrence, error)

	Update(
		ctx context.Context,
		entity *detention.DetentionOccurrence,
	) (*detention.DetentionOccurrence, error)

	ShipmentAccrued(
		ctx context.Context,
		req *GetOccurrencesByShipmentRequest,
	) (decimal.Decimal, error)
}

type ListNoticesDueRequest struct {
	TenantInfo pagination.TenantInfo
	Before     int64
}

type AppendEvidenceRequest struct {
	Entry *detention.DetentionEvidence
}

type ListEvidenceRequest struct {
	OccurrenceID pulid.ID
	TenantInfo   pagination.TenantInfo
}

type DetentionEvidenceRepository interface {
	Append(
		ctx context.Context,
		req *AppendEvidenceRequest,
	) (*detention.DetentionEvidence, error)

	List(
		ctx context.Context,
		req *ListEvidenceRequest,
	) ([]*detention.DetentionEvidence, error)

	Head(
		ctx context.Context,
		req *ListEvidenceRequest,
	) (string, int32, error)
}

type ListDetentionNoticesRequest struct {
	OccurrenceID pulid.ID
	TenantInfo   pagination.TenantInfo
}

// AttachNoticePDFDocumentRequest files a stored document against a notice.
//
// It updates the one column rather than saving the entity, because the document
// is filed after the notice was already persisted and a provider webhook may be
// recording delivery on the same row at the same time. A read-modify-write would
// make those two updates race for the same optimistic version.
type AttachNoticePDFDocumentRequest struct {
	TenantInfo pagination.TenantInfo
	NoticeID   pulid.ID
	DocumentID pulid.ID
}

type DetentionNoticeRepository interface {
	Create(
		ctx context.Context,
		entity *detention.DetentionNotice,
	) (*detention.DetentionNotice, error)

	AttachPDFDocument(
		ctx context.Context,
		req *AttachNoticePDFDocumentRequest,
	) error

	Update(
		ctx context.Context,
		entity *detention.DetentionNotice,
	) (*detention.DetentionNotice, error)

	List(
		ctx context.Context,
		req *ListDetentionNoticesRequest,
	) ([]*detention.DetentionNotice, error)

	ListQueued(
		ctx context.Context,
		req *ListNoticesDueRequest,
	) ([]*detention.DetentionNotice, error)
}

// StopFact is the settled dwell history a backtest re-prices.
type StopFact struct {
	StopID                 pulid.ID `bun:"stop_id"`
	ShipmentID             pulid.ID `bun:"shipment_id"`
	CustomerID             pulid.ID `bun:"customer_id"`
	LocationID             pulid.ID `bun:"location_id"`
	ShipmentTypeID         pulid.ID `bun:"shipment_type_id"`
	ServiceTypeID          pulid.ID `bun:"service_type_id"`
	StopType               string   `bun:"stop_type"`
	ScheduleType           string   `bun:"schedule_type"`
	ScheduledWindowStart   int64    `bun:"scheduled_window_start"`
	ScheduledWindowEnd     *int64   `bun:"scheduled_window_end"`
	ActualArrival          *int64   `bun:"actual_arrival"`
	ActualDeparture        *int64   `bun:"actual_departure"`
	CountDetentionOverride *bool    `bun:"count_detention_override"`
}

type ListStopFactsRequest struct {
	TenantInfo pagination.TenantInfo
	From       int64
	To         int64
	CustomerID *pulid.ID
	LocationID *pulid.ID
	Limit      int
}

type DetentionStatsRequest struct {
	TenantInfo pagination.TenantInfo
	From       int64
	To         int64
	Limit      int
}

type FacilityDetentionStat struct {
	LocationID         pulid.ID        `bun:"location_id"          json:"locationId"`
	LocationName       string          `bun:"location_name"        json:"locationName"`
	StopCount          int             `bun:"stop_count"           json:"stopCount"`
	BreachCount        int             `bun:"breach_count"         json:"breachCount"`
	AvgDwellMinutes    int             `bun:"avg_dwell_minutes"    json:"avgDwellMinutes"`
	MedianDwellMinutes float64         `bun:"median_dwell_minutes" json:"medianDwellMinutes"`
	P90DwellMinutes    float64         `bun:"p90_dwell_minutes"    json:"p90DwellMinutes"`
	BilledAmount       decimal.Decimal `bun:"billed_amount"        json:"billedAmount"`
	DriverPayAmount    decimal.Decimal `bun:"driver_pay_amount"    json:"driverPayAmount"`
	NetMargin          decimal.Decimal `bun:"net_margin"           json:"netMargin"`
	WaivedAmount       decimal.Decimal `bun:"waived_amount"        json:"waivedAmount"`
	DisputeCount       int             `bun:"dispute_count"        json:"disputeCount"`
	SuppressedCount    int             `bun:"suppressed_count"     json:"suppressedCount"`
}

type CustomerDetentionStat struct {
	CustomerID      pulid.ID        `bun:"customer_id"       json:"customerId"`
	CustomerName    string          `bun:"customer_name"     json:"customerName"`
	StopCount       int             `bun:"stop_count"        json:"stopCount"`
	BreachCount     int             `bun:"breach_count"      json:"breachCount"`
	BilledAmount    decimal.Decimal `bun:"billed_amount"     json:"billedAmount"`
	DriverPayAmount decimal.Decimal `bun:"driver_pay_amount" json:"driverPayAmount"`
	NetMargin       decimal.Decimal `bun:"net_margin"        json:"netMargin"`
	WaivedAmount    decimal.Decimal `bun:"waived_amount"     json:"waivedAmount"`
	DisputeCount    int             `bun:"dispute_count"     json:"disputeCount"`
	SuppressedCount int             `bun:"suppressed_count"  json:"suppressedCount"`
}

type WaiverLeakageStat struct {
	Reason        string          `bun:"reason"         json:"reason"`
	WaiverCount   int             `bun:"waiver_count"   json:"waiverCount"`
	ApproverCount int             `bun:"approver_count" json:"approverCount"`
	WaivedAmount  decimal.Decimal `bun:"waived_amount"  json:"waivedAmount"`
}

// StopBaseline is what a stop actually billed, keyed for backtest comparison.
type StopBaseline struct {
	StopID         pulid.ID        `bun:"stop_id"`
	BillableAmount decimal.Decimal `bun:"billable_amount"`
}

type DetentionAnalyticsRepository interface {
	ListStopFacts(ctx context.Context, req *ListStopFactsRequest) ([]*StopFact, error)

	BaselineByStop(
		ctx context.Context,
		req *DetentionStatsRequest,
	) ([]*StopBaseline, error)

	FacilityStats(
		ctx context.Context,
		req *DetentionStatsRequest,
	) ([]*FacilityDetentionStat, error)

	CustomerStats(
		ctx context.Context,
		req *DetentionStatsRequest,
	) ([]*CustomerDetentionStat, error)

	WaiverStats(ctx context.Context, req *DetentionStatsRequest) ([]*WaiverLeakageStat, error)

	EntityNames(
		ctx context.Context,
		req *DetentionEntityNamesRequest,
	) (*DetentionEntityNames, error)
}

type DetentionEntityNamesRequest struct {
	TenantInfo  pagination.TenantInfo
	LocationIDs []pulid.ID
	CustomerIDs []pulid.ID
}

type DetentionEntityNames struct {
	Locations map[pulid.ID]string
	Customers map[pulid.ID]string
}
