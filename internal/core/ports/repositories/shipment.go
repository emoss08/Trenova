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
	// The ID of the shipment
	ID pulid.ID

	// The ID of the organization
	OrgID pulid.ID

	// The ID of the business unit
	BuID pulid.ID

	// The ID of the user (Optional)
	UserID pulid.ID

	// Shipment options (Optional)
	ShipmentOptions ShipmentOptions
}

type UpdateShipmentStatusRequest struct {
	// Fetch the shipment
	GetOpts GetShipmentByIDOptions

	// The status of the shipment
	Status shipment.Status
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
	UpdateStatus(ctx context.Context, opts *UpdateShipmentStatusRequest) (*shipment.Shipment, error)
	Cancel(ctx context.Context, req *CancelShipmentRequest) (*shipment.Shipment, error)
}
