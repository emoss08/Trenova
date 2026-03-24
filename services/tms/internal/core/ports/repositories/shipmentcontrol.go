package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
)

type GetShipmentControlRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ShipmentControlRepository interface {
	Get(
		ctx context.Context,
		req GetShipmentControlRequest,
	) (*tenant.ShipmentControl, error)
	Update(
		ctx context.Context,
		sc *tenant.ShipmentControl,
	) (*tenant.ShipmentControl, error)
}
