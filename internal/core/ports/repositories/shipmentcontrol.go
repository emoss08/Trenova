package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type ShipmentControlRepository interface {
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*shipment.ShipmentControl, error)
}
