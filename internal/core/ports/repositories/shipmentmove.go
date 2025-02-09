package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetMoveByIDOptions struct {
	// ID of the move
	MoveID pulid.ID

	// ID of the organization
	OrgID pulid.ID

	// ID of the business unit
	BuID pulid.ID
}

type UpdateMoveStatusRequest struct {
	// Fetch the move
	GetMoveOpts GetMoveByIDOptions

	// Status of the move
	Status shipment.MoveStatus
}

type GetMovesByShipmentIDOptions struct {
	// ID of the shipment
	ShipmentID pulid.ID

	// ID of the organization
	OrgID pulid.ID

	// ID of the business unit
	BuID pulid.ID
}

type ShipmentMoveRepository interface {
	GetByID(ctx context.Context, opts GetMoveByIDOptions) (*shipment.ShipmentMove, error)
	UpdateStatus(ctx context.Context, opts *UpdateMoveStatusRequest) (*shipment.ShipmentMove, error)
	GetMovesByShipmentID(ctx context.Context, opts GetMovesByShipmentIDOptions) ([]*shipment.ShipmentMove, error)
}
