package shipmentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

func (r *repository) cancelShipmentComponents(
	ctx context.Context,
	tx bun.IDB,
	shipmentID pulid.ID,
) error {
	moveIDs, err := r.getMoveIDsForShipment(ctx, tx, shipmentID)
	if err != nil {
		return err
	}

	if len(moveIDs) == 0 {
		return nil
	}

	cols := buncolgen.ShipmentMoveColumns
	if _, err = tx.NewUpdate().
		Model((*shipment.ShipmentMove)(nil)).
		Set(cols.Status.Set(), shipment.MoveStatusCanceled).
		Where(cols.ID.In(), bun.List(moveIDs)).
		Exec(ctx); err != nil {
		return err
	}

	assignmentCols := buncolgen.AssignmentColumns
	if _, err = tx.NewUpdate().
		Model((*shipment.Assignment)(nil)).
		Set(assignmentCols.Status.Set(), shipment.AssignmentStatusCanceled).
		Where(assignmentCols.ShipmentMoveID.In(), bun.List(moveIDs)).
		Where(assignmentCols.ArchivedAt.IsNotNull()).
		Exec(ctx); err != nil {
		return err
	}

	stopCols := buncolgen.StopColumns
	if _, err = tx.NewUpdate().
		Model((*shipment.Stop)(nil)).
		Set(stopCols.Status.Set(), shipment.StopStatusCanceled).
		Where(stopCols.ShipmentMoveID.In(), bun.List(moveIDs)).
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (r *repository) uncancelShipmentComponents(
	ctx context.Context,
	tx bun.IDB,
	shipmentID pulid.ID,
) error {
	moveIDs, err := r.getMoveIDsForShipment(ctx, tx, shipmentID)
	if err != nil {
		return err
	}

	if len(moveIDs) == 0 {
		return nil
	}

	moveCols := buncolgen.ShipmentMoveColumns
	if _, err = tx.NewUpdate().
		Model((*shipment.ShipmentMove)(nil)).
		Set(moveCols.Status.Set(), shipment.MoveStatusNew).
		Where(moveCols.ID.In(), bun.List(moveIDs)).
		Exec(ctx); err != nil {
		return err
	}

	assignmentCols := buncolgen.AssignmentColumns
	if _, err = tx.NewUpdate().
		Model((*shipment.Assignment)(nil)).
		Set(assignmentCols.Status.Set(), shipment.AssignmentStatusNew).
		Where(assignmentCols.ShipmentMoveID.In(), bun.List(moveIDs)).
		Where(assignmentCols.ArchivedAt.IsNull()).
		Exec(ctx); err != nil {
		return err
	}

	stopCols := buncolgen.StopColumns
	if _, err = tx.NewUpdate().
		Model((*shipment.Stop)(nil)).
		Set(stopCols.Status.Set(), shipment.StopStatusNew).
		Where(stopCols.ShipmentMoveID.In(), bun.List(moveIDs)).
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (r *repository) getMoveIDsForShipment(
	ctx context.Context,
	tx bun.IDB,
	shipmentID pulid.ID,
) ([]pulid.ID, error) {
	moveIDs := make([]pulid.ID, 0)
	cols := buncolgen.ShipmentMoveColumns

	if err := tx.NewSelect().
		Model((*shipment.ShipmentMove)(nil)).
		Column(cols.ID.Bare()).
		Where(cols.ShipmentID.Eq(), shipmentID).
		Scan(ctx, &moveIDs); err != nil {
		return nil, err
	}

	return moveIDs, nil
}
