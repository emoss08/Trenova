package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
)

type ShipmentHoldService interface {
	ListByShipmentID(
		ctx context.Context,
		req *repositories.ListShipmentHoldsRequest,
	) (*pagination.ListResult[*shipment.ShipmentHold], error)
	GetByID(
		ctx context.Context,
		req *repositories.GetShipmentHoldByIDRequest,
	) (*shipment.ShipmentHold, error)
	Create(
		ctx context.Context,
		req *repositories.CreateShipmentHoldRequest,
		actor *RequestActor,
	) (*shipment.ShipmentHold, error)
	Update(
		ctx context.Context,
		req *repositories.UpdateShipmentHoldRequest,
		actor *RequestActor,
	) (*shipment.ShipmentHold, error)
	Release(
		ctx context.Context,
		req *repositories.ReleaseShipmentHoldRequest,
		actor *RequestActor,
	) (*shipment.ShipmentHold, error)
}
