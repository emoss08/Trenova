package ratezoneports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/rating/domain/ratezone"
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

type GetRateZonesByIDsRequest struct {
	TenantInfo  pagination.TenantInfo `json:"-"`
	RateZoneIDs []pulid.ID            `json:"rateZoneIds"`
}

type ResolveLaneZonesRequest struct {
	TenantInfo      pagination.TenantInfo `json:"-"`
	OriginKeys      []string              `json:"originKeys"`
	DestinationKeys []string              `json:"destinationKeys"`
}

type ZoneRef struct {
	ID   pulid.ID `json:"id"`
	Code string   `json:"code"`
}

type LaneZones struct {
	Origin      []ZoneRef `json:"origin"`
	Destination []ZoneRef `json:"destination"`
}

func ZoneIDs(refs []ZoneRef) []pulid.ID {
	ids := make([]pulid.ID, 0, len(refs))
	for _, ref := range refs {
		ids = append(ids, ref.ID)
	}
	return ids
}

func ZoneCodes(refs []ZoneRef) []string {
	codes := make([]string, 0, len(refs))
	for _, ref := range refs {
		codes = append(codes, ref.Code)
	}
	return codes
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
	GetByIDs(ctx context.Context, req *GetRateZonesByIDsRequest) ([]*ratezone.RateZone, error)
	Create(ctx context.Context, entity *ratezone.RateZone) (*ratezone.RateZone, error)
	Update(ctx context.Context, entity *ratezone.RateZone) (*ratezone.RateZone, error)
	Delete(ctx context.Context, req *GetRateZoneByIDRequest) error

	ResolveLaneZones(ctx context.Context, req *ResolveLaneZonesRequest) (*LaneZones, error)
	GetCodesByIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) (map[pulid.ID]string, error)
}
