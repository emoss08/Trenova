package detentionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
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

type GetDetentionPoliciesByIDsRequest struct {
	TenantInfo         pagination.TenantInfo
	DetentionPolicyIDs []pulid.ID
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
	) (*pagination.ListResult[*detentionpolicy.DetentionPolicy], error)

	ListConnection(
		ctx context.Context,
		req *ListDetentionPolicyConnectionRequest,
	) (*pagination.CursorListResult[*detentionpolicy.DetentionPolicy], error)

	GetByID(
		ctx context.Context,
		req *GetDetentionPolicyByIDRequest,
	) (*detentionpolicy.DetentionPolicy, error)

	GetByIDs(
		ctx context.Context,
		req *GetDetentionPoliciesByIDsRequest,
	) ([]*detentionpolicy.DetentionPolicy, error)

	GetResolutionCandidates(
		ctx context.Context,
		req *GetResolutionCandidatesRequest,
	) ([]*detentionpolicy.DetentionPolicy, error)

	Create(
		ctx context.Context,
		entity *detentionpolicy.DetentionPolicy,
	) (*detentionpolicy.DetentionPolicy, error)

	Update(
		ctx context.Context,
		entity *detentionpolicy.DetentionPolicy,
	) (*detentionpolicy.DetentionPolicy, error)

	Delete(ctx context.Context, req *GetDetentionPolicyByIDRequest) error

	SelectOptions(
		ctx context.Context,
		req *DetentionPolicySelectOptionsRequest,
	) (*pagination.ListResult[*detentionpolicy.DetentionPolicy], error)

	CodeExists(
		ctx context.Context,
		req *DetentionPolicyCodeExistsRequest,
	) (bool, error)

	FindOverlapping(
		ctx context.Context,
		req *FindOverlappingDetentionPoliciesRequest,
	) ([]*detentionpolicy.DetentionPolicy, error)
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
	// IncludeEvidence loads each occurrence's evidence chain, which is what
	// billing reads to set the billed wait against the truck's own clock.
	IncludeEvidence bool
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

type LinkOccurrenceChargesRequest struct {
	TenantInfo        pagination.TenantInfo
	ShipmentID        pulid.ID
	ChargeOccurrences map[pulid.ID][]pulid.ID
}

// ListDetentionBillingHoldsRequest asks for the charges holding the shipments
// off an invoice.
type ListDetentionBillingHoldsRequest struct {
	TenantInfo  pagination.TenantInfo
	ShipmentIDs []pulid.ID
}

// ListOccurrencesByChargesRequest asks for the billable occurrences behind
// shipment charges.
type ListOccurrencesByChargesRequest struct {
	TenantInfo  pagination.TenantInfo
	ShipmentIDs []pulid.ID
	ChargeIDs   []pulid.ID
	Statuses    []detention.OccurrenceStatus
}

type DetentionOccurrenceRepository interface {
	// ListBillingHoldsByShipments returns every occurrence on the shipments
	// that holds billing, ordered by shipment and then by clock.
	ListBillingHoldsByShipments(
		ctx context.Context,
		req *ListDetentionBillingHoldsRequest,
	) ([]*detention.DetentionOccurrence, error)

	// ListBillingHolds returns every occurrence in the tenant that holds
	// billing, oldest clock first.
	ListBillingHolds(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*detention.DetentionOccurrence, error)

	// ListByAdditionalCharges returns the occurrences in the requested
	// statuses that the shipment charges bill.
	ListByAdditionalCharges(
		ctx context.Context,
		req *ListOccurrencesByChargesRequest,
	) ([]*detention.DetentionOccurrence, error)

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

	LinkCharges(
		ctx context.Context,
		req *LinkOccurrenceChargesRequest,
	) error
}

type ListNoticesDueRequest struct {
	TenantInfo pagination.TenantInfo
	Before     int64
}

type AppendEvidenceRequest struct {
	Entry *detentionevidence.DetentionEvidence
}

type ListEvidenceRequest struct {
	OccurrenceID pulid.ID
	TenantInfo   pagination.TenantInfo
}

type DetentionEvidenceRepository interface {
	Append(
		ctx context.Context,
		req *AppendEvidenceRequest,
	) (*detentionevidence.DetentionEvidence, error)

	List(
		ctx context.Context,
		req *ListEvidenceRequest,
	) ([]*detentionevidence.DetentionEvidence, error)

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
		entity *detentionnotice.DetentionNotice,
	) (*detentionnotice.DetentionNotice, error)

	AttachPDFDocument(
		ctx context.Context,
		req *AttachNoticePDFDocumentRequest,
	) error

	Update(
		ctx context.Context,
		entity *detentionnotice.DetentionNotice,
	) (*detentionnotice.DetentionNotice, error)

	List(
		ctx context.Context,
		req *ListDetentionNoticesRequest,
	) ([]*detentionnotice.DetentionNotice, error)

	ListQueued(
		ctx context.Context,
		req *ListNoticesDueRequest,
	) ([]*detentionnotice.DetentionNotice, error)
}

// StopFact is the settled dwell history a backtest re-prices.
type StopFact struct {
	StopID                 pulid.ID
	ShipmentID             pulid.ID
	CustomerID             pulid.ID
	LocationID             pulid.ID
	ShipmentTypeID         pulid.ID
	ServiceTypeID          pulid.ID
	StopType               string
	ScheduleType           string
	ScheduledWindowStart   int64
	ScheduledWindowEnd     *int64
	ActualArrival          *int64
	ActualDeparture        *int64
	CountDetentionOverride *bool
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
	LocationID pulid.ID
}

type FacilityDetentionStat struct {
	LocationID         pulid.ID            `json:"locationId"`
	LocationName       string              `json:"locationName"`
	StopCount          int                 `json:"stopCount"`
	BreachCount        int                 `json:"breachCount"`
	AvgDwellMinutes    int                 `json:"avgDwellMinutes"`
	MedianDwellMinutes float64             `json:"medianDwellMinutes"`
	P90DwellMinutes    float64             `json:"p90DwellMinutes"`
	BilledAmount       decimal.Decimal     `json:"billedAmount"`
	DriverPayAmount    decimal.NullDecimal `json:"driverPayAmount"`
	NetMargin          decimal.NullDecimal `json:"netMargin"`
	WaivedAmount       decimal.Decimal     `json:"waivedAmount"`
	DisputeCount       int                 `json:"disputeCount"`
	SuppressedCount    int                 `json:"suppressedCount"`
}

type CustomerDetentionStat struct {
	CustomerID      pulid.ID            `json:"customerId"`
	CustomerName    string              `json:"customerName"`
	StopCount       int                 `json:"stopCount"`
	BreachCount     int                 `json:"breachCount"`
	BilledAmount    decimal.Decimal     `json:"billedAmount"`
	DriverPayAmount decimal.NullDecimal `json:"driverPayAmount"`
	NetMargin       decimal.NullDecimal `json:"netMargin"`
	WaivedAmount    decimal.Decimal     `json:"waivedAmount"`
	DisputeCount    int                 `json:"disputeCount"`
	SuppressedCount int                 `json:"suppressedCount"`
}

type WaiverLeakageStat struct {
	Reason        string          `json:"reason"`
	WaiverCount   int             `json:"waiverCount"`
	ApproverCount int             `json:"approverCount"`
	WaivedAmount  decimal.Decimal `json:"waivedAmount"`
}

// StopBaseline is what a stop actually billed, keyed for backtest comparison.
type StopBaseline struct {
	StopID         pulid.ID
	BillableAmount decimal.Decimal
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
