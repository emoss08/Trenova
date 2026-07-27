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
	TenantInfo       pagination.TenantInfo
	ShipmentID       pulid.ID
	ExcludeStopID    pulid.ID
	DayKeys          []string
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

type DetentionNoticeRepository interface {
	Create(
		ctx context.Context,
		entity *detention.DetentionNotice,
	) (*detention.DetentionNotice, error)

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
