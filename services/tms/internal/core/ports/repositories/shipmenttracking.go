package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListTrackingShipmentsRequest struct {
	TenantInfo  pagination.TenantInfo
	ShipmentIDs []pulid.ID
}

type ShipmentTrackingRepository interface {
	ListTrackingShipments(
		ctx context.Context,
		req *ListTrackingShipmentsRequest,
	) ([]*shipment.Shipment, error)
}
