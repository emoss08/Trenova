package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
)

type StopActualPlan struct {
	Before *shipment.ShipmentMove
	After  *shipment.ShipmentMove
}

// MoveStatusPlan is what setting moves' status would leave: each move before
// and after, in the order asked, and each shipment they belong to with its
// state re-derived. Nothing in it has been saved.
type MoveStatusPlan struct {
	MovesBefore     []*shipment.ShipmentMove
	MovesAfter      []*shipment.ShipmentMove
	ShipmentsBefore []*shipment.Shipment
	ShipmentsAfter  []*shipment.Shipment
}

type ShipmentMoveService interface {
	UpdateStatus(
		ctx context.Context,
		req *repositories.UpdateMoveStatusRequest,
	) (*shipment.ShipmentMove, error)
	RecordStopActual(
		ctx context.Context,
		req *repositories.RecordStopActualRequest,
	) (*shipment.ShipmentMove, error)
	PreviewStopActual(
		ctx context.Context,
		req *repositories.RecordStopActualRequest,
	) (*StopActualPlan, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *repositories.BulkUpdateMoveStatusRequest,
	) ([]*shipment.ShipmentMove, error)
	PreviewUpdateStatus(
		ctx context.Context,
		req *repositories.BulkUpdateMoveStatusRequest,
	) (*MoveStatusPlan, error)
	SplitMove(
		ctx context.Context,
		req *repositories.SplitMoveRequest,
	) (*repositories.SplitMoveResponse, error)
}
