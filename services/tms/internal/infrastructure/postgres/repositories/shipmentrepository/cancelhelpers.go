package shipmentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/uptrace/bun"
)

func restoredShipmentStatus(req *repositories.UncancelShipmentRequest) shipment.Status {
	if req.RestoredStatus == "" || req.RestoredStatus == shipment.StatusCanceled {
		return shipment.StatusNew
	}

	return req.RestoredStatus
}

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
		Set(cols.Version.Inc(1)).
		Where(cols.ID.In(), bun.List(moveIDs)).
		Where(cols.Status.NotIn(), bun.List(shipmentstate.CancelPreservedMoveStatuses())).
		Exec(ctx); err != nil {
		return err
	}

	assignmentCols := buncolgen.AssignmentColumns
	if _, err = tx.NewUpdate().
		Model((*shipment.Assignment)(nil)).
		Set(assignmentCols.Status.Set(), shipment.AssignmentStatusCanceled).
		Set(assignmentCols.Version.Inc(1)).
		Where(assignmentCols.ShipmentMoveID.In(), bun.List(moveIDs)).
		Where(assignmentCols.ArchivedAt.IsNull()).
		Where(
			assignmentCols.Status.NotIn(),
			bun.List(shipmentstate.CancelPreservedAssignmentStatuses()),
		).
		Exec(ctx); err != nil {
		return err
	}

	stopCols := buncolgen.StopColumns
	if _, err = tx.NewUpdate().
		Model((*shipment.Stop)(nil)).
		Set(stopCols.Status.Set(), shipment.StopStatusCanceled).
		Set(stopCols.Version.Inc(1)).
		Where(stopCols.ShipmentMoveID.In(), bun.List(moveIDs)).
		Where(stopCols.Status.NotIn(), bun.List(shipmentstate.CancelPreservedStopStatuses())).
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (r *repository) uncancelShipmentComponents(
	ctx context.Context,
	tx bun.IDB,
	req *repositories.UncancelShipmentRequest,
) error {
	moveIDs, err := r.getMoveIDsForShipment(ctx, tx, req.ShipmentID)
	if err != nil {
		return err
	}

	if len(moveIDs) == 0 {
		return nil
	}

	if err = restoreMoveStatuses(ctx, tx, moveIDs, req.MoveStatuses); err != nil {
		return err
	}

	assignmentCols := buncolgen.AssignmentColumns
	if _, err = tx.NewUpdate().
		Model((*shipment.Assignment)(nil)).
		Set(assignmentCols.Status.Set(), shipment.AssignmentStatusNew).
		Set(assignmentCols.Version.Inc(1)).
		Where(assignmentCols.ShipmentMoveID.In(), bun.List(moveIDs)).
		Where(assignmentCols.ArchivedAt.IsNull()).
		Where(assignmentCols.Status.Eq(), shipment.AssignmentStatusCanceled).
		Exec(ctx); err != nil {
		return err
	}

	return restoreStopStatuses(ctx, tx, moveIDs, req.StopStatuses)
}

func restoreMoveStatuses(
	ctx context.Context,
	tx bun.IDB,
	moveIDs []pulid.ID,
	restores []repositories.MoveStatusRestore,
) error {
	cols := buncolgen.ShipmentMoveColumns
	groups := sliceutils.GroupBy(
		restores,
		func(item repositories.MoveStatusRestore) shipment.MoveStatus {
			return item.Status
		},
	)

	for status, items := range groups {
		if status == shipment.MoveStatusCanceled {
			continue
		}

		ids := make([]pulid.ID, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.MoveID)
		}

		if _, err := tx.NewUpdate().
			Model((*shipment.ShipmentMove)(nil)).
			Set(cols.Status.Set(), status).
			Set(cols.Version.Inc(1)).
			Where(cols.ID.In(), bun.List(ids)).
			Where(cols.ID.In(), bun.List(moveIDs)).
			Where(cols.Status.Eq(), shipment.MoveStatusCanceled).
			Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}

func restoreStopStatuses(
	ctx context.Context,
	tx bun.IDB,
	moveIDs []pulid.ID,
	restores []repositories.StopStatusRestore,
) error {
	cols := buncolgen.StopColumns
	groups := sliceutils.GroupBy(
		restores,
		func(item repositories.StopStatusRestore) shipment.StopStatus {
			return item.Status
		},
	)

	for status, items := range groups {
		if status == shipment.StopStatusCanceled {
			continue
		}

		ids := make([]pulid.ID, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.StopID)
		}

		if _, err := tx.NewUpdate().
			Model((*shipment.Stop)(nil)).
			Set(cols.Status.Set(), status).
			Set(cols.Version.Inc(1)).
			Where(cols.ID.In(), bun.List(ids)).
			Where(cols.ShipmentMoveID.In(), bun.List(moveIDs)).
			Where(cols.Status.Eq(), shipment.StopStatusCanceled).
			Exec(ctx); err != nil {
			return err
		}
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
