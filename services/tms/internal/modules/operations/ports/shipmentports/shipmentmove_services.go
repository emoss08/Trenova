package shipmentports

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
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

type MoveSplitPlan struct {
	Move  *shipment.ShipmentMove
	Split *shipment.MoveSplit
}

type ShipmentMoveService interface {
	UpdateStatus(
		ctx context.Context,
		req *UpdateMoveStatusRequest,
	) (*shipment.ShipmentMove, error)
	RecordStopActual(
		ctx context.Context,
		req *RecordStopActualRequest,
	) (*shipment.ShipmentMove, error)
	PreviewStopActual(
		ctx context.Context,
		req *RecordStopActualRequest,
	) (*StopActualPlan, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateMoveStatusRequest,
	) ([]*shipment.ShipmentMove, error)
	PreviewUpdateStatus(
		ctx context.Context,
		req *BulkUpdateMoveStatusRequest,
	) (*MoveStatusPlan, error)
	SplitMove(
		ctx context.Context,
		req *SplitMoveRequest,
	) (*SplitMoveResponse, error)
	PreviewSplitMove(
		ctx context.Context,
		req *SplitMoveRequest,
	) (*MoveSplitPlan, error)
}
