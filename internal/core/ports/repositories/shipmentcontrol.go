package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetShipmentControlRequest struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type ShipmentControlRepository interface {
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*shipment.ShipmentControl, error)
	GetByID(ctx context.Context, opts *GetShipmentControlRequest) (*shipment.ShipmentControl, error)
	Update(ctx context.Context, sc *shipment.ShipmentControl) (*shipment.ShipmentControl, error)
}
