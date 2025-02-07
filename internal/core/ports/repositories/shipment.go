package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type ShipmentOptions struct {
	ExpandShipmentDetails bool `query:"expandShipmentDetails"`
}

type ListShipmentOptions struct {
	Filter          *ports.LimitOffsetQueryOptions
	ShipmentOptions ShipmentOptions
}

type GetShipmentByIDOptions struct {
	ID              pulid.ID
	OrgID           pulid.ID
	BuID            pulid.ID
	UserID          pulid.ID
	ShipmentOptions ShipmentOptions
}

type CancelShipmentRequest struct {
	ShipmentID   pulid.ID `json:"shipmentId"`
	OrgID        pulid.ID `json:"orgId"`
	BuID         pulid.ID `json:"buId"`
	CanceledByID pulid.ID `json:"canceledById"`
	CanceledAt   int64    `json:"canceledAt"`
	CancelReason string   `json:"cancelReason"`
}

type ShipmentRepository interface {
	List(ctx context.Context, opts *ListShipmentOptions) (*ports.ListResult[*shipment.Shipment], error)
	GetByID(ctx context.Context, opts GetShipmentByIDOptions) (*shipment.Shipment, error)
	Create(ctx context.Context, t *shipment.Shipment) (*shipment.Shipment, error)
	Update(ctx context.Context, t *shipment.Shipment) (*shipment.Shipment, error)
	Cancel(ctx context.Context, req *CancelShipmentRequest) error
}
