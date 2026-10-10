package shipmentports

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type ShipmentHoldService interface {
	ListByShipmentID(
		ctx context.Context,
		req *ListShipmentHoldsRequest,
	) (*pagination.ListResult[*shipment.ShipmentHold], error)
	GetByID(
		ctx context.Context,
		req *GetShipmentHoldByIDRequest,
	) (*shipment.ShipmentHold, error)
	Create(
		ctx context.Context,
		req *CreateShipmentHoldRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentHold, error)
	Update(
		ctx context.Context,
		req *UpdateShipmentHoldRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentHold, error)
	Release(
		ctx context.Context,
		req *ReleaseShipmentHoldRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentHold, error)
	PreviewCreate(
		ctx context.Context,
		req *CreateShipmentHoldRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentHold, error)
	PreviewRelease(
		ctx context.Context,
		req *ReleaseShipmentHoldRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentHold, error)
	PreviewUpdate(
		ctx context.Context,
		req *UpdateShipmentHoldRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentHold, error)
}
