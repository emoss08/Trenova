package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type DedicatedLaneFilterOptions struct {
	ExpandDetails bool `json:"expandDetails" form:"expandDetails"`
}

type ListDedicatedLaneRequest struct {
	Filter        *pagination.QueryOptions   `json:"filter"        form:"filter"`
	FilterOptions DedicatedLaneFilterOptions `json:"filterOptions" form:"filterOptions"`
}

type GetDedicatedLaneByIDRequest struct {
	ID            pulid.ID                   `json:"id"            form:"id"`
	OrgID         pulid.ID                   `json:"orgId"         form:"orgId"`
	BuID          pulid.ID                   `json:"buId"          form:"buId"`
	UserID        pulid.ID                   `json:"userId"        form:"userId"`
	FilterOptions DedicatedLaneFilterOptions `json:"filterOptions" form:"filterOptions"`
}

type FindDedicatedLaneByShipmentRequest struct {
	OrganizationID        pulid.ID  `json:"organizationId"        form:"organizationId"`
	BusinessUnitID        pulid.ID  `json:"businessUnitId"        form:"businessUnitId"`
	CustomerID            pulid.ID  `json:"customerId"            form:"customerId"`
	OriginLocationID      pulid.ID  `json:"originLocationId"      form:"originLocationId"`
	DestinationLocationID pulid.ID  `json:"destinationLocationId" form:"destinationLocationId"`
	TrailerTypeID         *pulid.ID `json:"trailerTypeId"         form:"trailerTypeId"`
	TractorTypeID         *pulid.ID `json:"tractorTypeId"         form:"tractorTypeId"`
	ServiceTypeID         *pulid.ID `json:"serviceTypeId"         form:"serviceTypeId"`
	ShipmentTypeID        *pulid.ID `json:"shipmentTypeId"        form:"shipmentTypeId"`
}

type DedicatedLaneRepository interface {
	List(
		ctx context.Context,
		req *ListDedicatedLaneRequest,
	) (*pagination.ListResult[*dedicatedlane.DedicatedLane], error)
	GetByID(
		ctx context.Context,
		req *GetDedicatedLaneByIDRequest,
	) (*dedicatedlane.DedicatedLane, error)
	FindByShipment(
		ctx context.Context,
		req *FindDedicatedLaneByShipmentRequest,
	) (*dedicatedlane.DedicatedLane, error)
	Create(
		ctx context.Context,
		dl *dedicatedlane.DedicatedLane,
	) (*dedicatedlane.DedicatedLane, error)
	Update(
		ctx context.Context,
		dl *dedicatedlane.DedicatedLane,
	) (*dedicatedlane.DedicatedLane, error)
}
