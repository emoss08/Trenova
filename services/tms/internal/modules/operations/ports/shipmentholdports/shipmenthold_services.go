package shipmentholdports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold"
	"github.com/emoss08/trenova/pkg/pagination"
)

type ShipmentHoldService interface {
	ListByShipmentID(
		ctx context.Context,
		req *ListShipmentHoldsRequest,
	) (*pagination.ListResult[*shipmenthold.ShipmentHold], error)
	GetByID(
		ctx context.Context,
		req *GetShipmentHoldByIDRequest,
	) (*shipmenthold.ShipmentHold, error)
	Create(
		ctx context.Context,
		req *CreateShipmentHoldRequest,
		actor *permissionports.RequestActor,
	) (*shipmenthold.ShipmentHold, error)
	Update(
		ctx context.Context,
		req *UpdateShipmentHoldRequest,
		actor *permissionports.RequestActor,
	) (*shipmenthold.ShipmentHold, error)
	Release(
		ctx context.Context,
		req *ReleaseShipmentHoldRequest,
		actor *permissionports.RequestActor,
	) (*shipmenthold.ShipmentHold, error)
	PreviewCreate(
		ctx context.Context,
		req *CreateShipmentHoldRequest,
		actor *permissionports.RequestActor,
	) (*shipmenthold.ShipmentHold, error)
	PreviewRelease(
		ctx context.Context,
		req *ReleaseShipmentHoldRequest,
		actor *permissionports.RequestActor,
	) (*shipmenthold.ShipmentHold, error)
	PreviewUpdate(
		ctx context.Context,
		req *UpdateShipmentHoldRequest,
		actor *permissionports.RequestActor,
	) (*shipmenthold.ShipmentHold, error)
}
