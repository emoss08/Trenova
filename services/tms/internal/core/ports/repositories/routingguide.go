package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type RoutingGuideFilterOptions struct {
	IncludeEntries   bool `form:"includeEntries"`
	IncludeLocations bool `form:"includeLocations"`
}

type ListRoutingGuideRequest struct {
	Filter                    *pagination.QueryOptions `json:"filter" form:"filter"`
	RoutingGuideFilterOptions ` form:"routingGuideFilterOptions"`
}

type GetRoutingGuideByIDRequest struct {
	ID                        pulid.ID              `json:"id"         form:"id"`
	TenantInfo                pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
	RoutingGuideFilterOptions ` form:"routingGuideFilterOptions"`
}

type DeleteRoutingGuideRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ID         pulid.ID              `json:"id"`
}

type MatchLaneRequest struct {
	TenantInfo            pagination.TenantInfo `json:"-"`
	OriginLocationID      pulid.ID              `json:"originLocationId"`
	DestinationLocationID pulid.ID              `json:"destinationLocationId"`
	OriginCity            string                `json:"originCity"`
	OriginState           string                `json:"originState"`
	DestinationCity       string                `json:"destinationCity"`
	DestinationState      string                `json:"destinationState"`
}

type RoutingGuideRepository interface {
	List(
		ctx context.Context,
		req *ListRoutingGuideRequest,
	) (*pagination.ListResult[*tender.RoutingGuide], error)
	GetByID(
		ctx context.Context,
		req GetRoutingGuideByIDRequest,
	) (*tender.RoutingGuide, error)
	Create(
		ctx context.Context,
		entity *tender.RoutingGuide,
	) (*tender.RoutingGuide, error)
	Update(
		ctx context.Context,
		entity *tender.RoutingGuide,
	) (*tender.RoutingGuide, error)
	Delete(
		ctx context.Context,
		req DeleteRoutingGuideRequest,
	) error
	MatchLane(
		ctx context.Context,
		req *MatchLaneRequest,
	) (*tender.RoutingGuide, error)
}
