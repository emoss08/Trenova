package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetShipmentHoldByShipmentIDRequest struct {
	ShipmentID pulid.ID
	OrgID      pulid.ID
	BuID       pulid.ID
	UserID     pulid.ID
}

type GetShipmentHoldByIDRequest struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type ShipmentHoldRepository interface {
	GetByShipmentID(
		ctx context.Context,
		req *GetShipmentHoldByShipmentIDRequest,
	) (*ports.ListResult[*shipment.ShipmentHold], error)
	GetByID(
		ctx context.Context,
		req *GetShipmentHoldByIDRequest,
	) (*shipment.ShipmentHold, error)
	Create(ctx context.Context, hold *shipment.ShipmentHold) (*shipment.ShipmentHold, error)
	Update(ctx context.Context, hold *shipment.ShipmentHold) (*shipment.ShipmentHold, error)
}
