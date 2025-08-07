/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/shared/pulid"
)

type DedicatedLaneFilterOptions struct {
	ExpandDetails bool `query:"expandDetails"`
}

type ListDedicatedLaneRequest struct {
	Filter        *ports.LimitOffsetQueryOptions
	FilterOptions DedicatedLaneFilterOptions `query:"filterOptions"`
}

type GetDedicatedLaneByIDRequest struct {
	ID            pulid.ID
	OrgID         pulid.ID
	BuID          pulid.ID
	UserID        pulid.ID
	FilterOptions DedicatedLaneFilterOptions `query:"filterOptions"`
}

type FindDedicatedLaneByShipmentRequest struct {
	OrganizationID        pulid.ID
	BusinessUnitID        pulid.ID
	CustomerID            pulid.ID
	OriginLocationID      pulid.ID
	DestinationLocationID pulid.ID
	TrailerTypeID         *pulid.ID
	TractorTypeID         *pulid.ID
	ServiceTypeID         *pulid.ID
	ShipmentTypeID        *pulid.ID
}

type DedicatedLaneRepository interface {
	List(
		ctx context.Context,
		req *ListDedicatedLaneRequest,
	) (*ports.ListResult[*dedicatedlane.DedicatedLane], error)
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
