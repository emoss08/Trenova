package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type ListShipmentOptions struct {
	Filter                  *ports.LimitOffsetQueryOptions
	IncludeMoveDetails      bool `query:"includeMoveDetails"`
	IncludeCommodityDetails bool `query:"includeCommodityDetails"`
	IncludeCustomerDetails  bool `query:"includeCustomerDetails"`
	IncludeStopDetails      bool `query:"includeStopDetails"`
}

type GetShipmentByIDOptions struct {
	ID                      pulid.ID
	OrgID                   pulid.ID
	BuID                    pulid.ID
	UserID                  pulid.ID
	IncludeMoveDetails      bool `query:"includeMoveDetails"`
	IncludeCommodityDetails bool `query:"includeCommodityDetails"`
	IncludeCustomerDetails  bool `query:"includeCustomerDetails"`
	IncludeStopDetails      bool `query:"includeStopDetails"`
}

type ShipmentRepository interface {
	List(ctx context.Context, opts *ListShipmentOptions) (*ports.ListResult[*shipment.Shipment], error)
	GetByID(ctx context.Context, opts GetShipmentByIDOptions) (*shipment.Shipment, error)
	Create(ctx context.Context, t *shipment.Shipment) (*shipment.Shipment, error)
	Update(ctx context.Context, t *shipment.Shipment) (*shipment.Shipment, error)
}
