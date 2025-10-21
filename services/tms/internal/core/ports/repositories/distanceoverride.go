package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/distanceoverride"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type ListDistanceOverrideRequest struct {
	Filter        *pagination.QueryOptions `json:"filter"        form:"filter"`
	ExpandDetails bool                     `json:"expandDetails" form:"expandDetails"`
}

type GetDistanceOverrideRequest struct {
	ID     pulid.ID `json:"id"     form:"id"`
	OrgID  pulid.ID `json:"orgId"  form:"orgId"`
	BuID   pulid.ID `json:"buId"   form:"buId"`
	UserID pulid.ID `json:"userId" form:"userId"`
}

type DeleteDistanceOverrideRequest struct {
	ID     pulid.ID `json:"id"     form:"id"`
	OrgID  pulid.ID `json:"orgId"  form:"orgId"`
	BuID   pulid.ID `json:"buId"   form:"buId"`
	UserID pulid.ID `json:"userId" form:"userId"`
}

type GetByLocationIDsRequest struct {
	OriginLocationID      pulid.ID `json:"originLocationId"      form:"originLocationId"`
	DestinationLocationID pulid.ID `json:"destinationLocationId" form:"destinationLocationId"`
	OrgID                 pulid.ID `json:"orgId"                 form:"orgId"`
	BuID                  pulid.ID `json:"buId"                  form:"buId"`
}

type DistanceOverrideRepository interface {
	List(
		ctx context.Context,
		req *ListDistanceOverrideRequest,
	) (*pagination.ListResult[*distanceoverride.Override], error)
	GetByID(
		ctx context.Context,
		req *GetDistanceOverrideRequest,
	) (*distanceoverride.Override, error)
	GetByLocationIDs(
		ctx context.Context,
		req *GetByLocationIDsRequest,
	) (*distanceoverride.Override, error)
	Create(
		ctx context.Context,
		entity *distanceoverride.Override,
	) (*distanceoverride.Override, error)
	Update(
		ctx context.Context,
		entity *distanceoverride.Override,
	) (*distanceoverride.Override, error)
	Delete(
		ctx context.Context,
		req *DeleteDistanceOverrideRequest,
	) error
}
