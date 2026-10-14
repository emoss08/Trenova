package ratequoteports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
)

type HydrateShipmentRequest struct {
	Shipment   *shipment.Shipment
	TenantInfo pagination.TenantInfo
	Refresh    bool
}

type HydrateShipmentsRequest struct {
	Shipments  []*shipment.Shipment
	TenantInfo pagination.TenantInfo
	Refresh    bool
}

type ShipmentRatingHydrator interface {
	HydrateShipment(ctx context.Context, req *HydrateShipmentRequest) error
	HydrateShipments(ctx context.Context, req *HydrateShipmentsRequest) error
}
