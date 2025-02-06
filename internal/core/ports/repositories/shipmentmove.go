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

type UpdateStatusOptions struct {
	// Fetch the move
	GetMoveOpts GetMoveByIDOptions

	// Status of the move
	Status shipment.MoveStatus
}

type ShipmentMoveRepository interface {
	GetByID(ctx context.Context, opts GetMoveByIDOptions) (*shipment.ShipmentMove, error)
	UpdateStatus(ctx context.Context, opts UpdateStatusOptions) (*shipment.ShipmentMove, error)
}
