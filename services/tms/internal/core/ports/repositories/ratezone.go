package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ratezone"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListRateZoneRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type ListRateZoneConnectionRequest struct {
	Filter          *pagination.QueryOptions `json:"filter"`
	Cursor          pagination.CursorInfo    `json:"-"`
	RateZoneColumns []string                 `json:"-"`
}

type GetRateZoneByIDRequest struct {
	RateZoneID     pulid.ID              `json:"rateZoneId"`
	TenantInfo     pagination.TenantInfo `json:"-"`
	IncludeMembers bool                  `json:"includeMembers"`
}

// ResolveZoneMembershipRequest asks which zones contain a place.
//
// The keys are the shipment's own match keys, produced by rategeo, so a zone
// member and the place it covers are compared as strings the database can index
// rather than by re-deriving geography at read time.
type ResolveZoneMembershipRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	MatchKeys  []string              `json:"matchKeys"`
}

type RateZoneRepository interface {
	List(
		ctx context.Context,
		req *ListRateZoneRequest,
	) (*pagination.ListResult[*ratezone.RateZone], error)
	ListConnection(
		ctx context.Context,
		req *ListRateZoneConnectionRequest,
	) (*pagination.CursorListResult[*ratezone.RateZone], error)
	SelectOptions(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*ratezone.RateZone], error)
	GetByID(ctx context.Context, req *GetRateZoneByIDRequest) (*ratezone.RateZone, error)
	Create(ctx context.Context, entity *ratezone.RateZone) (*ratezone.RateZone, error)
	Update(ctx context.Context, entity *ratezone.RateZone) (*ratezone.RateZone, error)
	Delete(ctx context.Context, req *GetRateZoneByIDRequest) error

	// ResolveMembership returns the zones whose members cover any of the given
	// keys. It runs twice per rating, once per end of the lane.
	ResolveMembership(ctx context.Context, req *ResolveZoneMembershipRequest) ([]pulid.ID, error)
}
