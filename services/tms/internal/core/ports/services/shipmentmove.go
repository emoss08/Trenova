package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type StopActualPlanner func(
	ctx context.Context,
	move *shipment.ShipmentMove,
) ([]shipment.StopActualChange, error)

type ReconcileStopActualsRequest struct {
	TenantInfo pagination.TenantInfo
	MoveID     pulid.ID
	Plan       StopActualPlanner
}

type ReconcileStopActualsResult struct {
	Move    *shipment.ShipmentMove
	Changes []shipment.StopActualChange
}

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
	ReconcileStopActuals(
		ctx context.Context,
		req *ReconcileStopActualsRequest,
	) (*ReconcileStopActualsResult, error)
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
	PreviewSplitMove(
		ctx context.Context,
		req *repositories.SplitMoveRequest,
	) (*MoveSplitPlan, error)
}
