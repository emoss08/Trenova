package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GetShipmentControlRequest struct {
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type ShipmentControlRepository interface {
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*tenant.ShipmentControl, error)
	Update(ctx context.Context, sc *tenant.ShipmentControl) (*tenant.ShipmentControl, error)
}
type ShipmentControlCacheRepository interface {
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*tenant.ShipmentControl, error)
	Set(ctx context.Context, sc *tenant.ShipmentControl) error
	Invalidate(ctx context.Context, orgID pulid.ID) error
}
