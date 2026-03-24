package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
)

type ShipmentMoveService interface {
	UpdateStatus(
		ctx context.Context,
		req *repositories.UpdateMoveStatusRequest,
	) (*shipment.ShipmentMove, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *repositories.BulkUpdateMoveStatusRequest,
	) ([]*shipment.ShipmentMove, error)
	SplitMove(
		ctx context.Context,
		req *repositories.SplitMoveRequest,
	) (*repositories.SplitMoveResponse, error)
}
