package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type ListShipmentTypeRequest struct {
	Filter *ports.LimitOffsetQueryOptions `query:"filter"`
	Status string                         `query:"status"`
}

type GetShipmentTypeByIDOptions struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type ShipmentTypeRepository interface {
	List(
		ctx context.Context,
		req *ListShipmentTypeRequest,
	) (*ports.ListResult[*shipmenttype.ShipmentType], error)
	GetByID(
		ctx context.Context,
		opts GetShipmentTypeByIDOptions,
	) (*shipmenttype.ShipmentType, error)
	Create(ctx context.Context, st *shipmenttype.ShipmentType) (*shipmenttype.ShipmentType, error)
	Update(ctx context.Context, st *shipmenttype.ShipmentType) (*shipmenttype.ShipmentType, error)
}
