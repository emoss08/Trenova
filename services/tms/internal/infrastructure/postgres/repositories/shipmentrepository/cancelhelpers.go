package shipmentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
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

	if _, err = tx.NewUpdate().
		Model((*shipment.ShipmentMove)(nil)).
		Set("status = ?", shipment.MoveStatusCanceled).
		Where("sm.id IN (?)", bun.List(moveIDs)).
		Exec(ctx); err != nil {
		return err
	}

	if _, err = tx.NewUpdate().
		Model((*shipment.Assignment)(nil)).
		Set("status = ?", shipment.AssignmentStatusCanceled).
		Where("a.shipment_move_id IN (?)", bun.List(moveIDs)).
		Where("a.archived_at IS NULL").
		Exec(ctx); err != nil {
		return err
	}

	if _, err = tx.NewUpdate().
		Model((*shipment.Stop)(nil)).
		Set("status = ?", shipment.StopStatusCanceled).
		Where("stp.shipment_move_id IN (?)", bun.List(moveIDs)).
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

	if _, err = tx.NewUpdate().
		Model((*shipment.ShipmentMove)(nil)).
		Set("status = ?", shipment.MoveStatusNew).
		Where("sm.id IN (?)", bun.List(moveIDs)).
		Exec(ctx); err != nil {
		return err
	}

	if _, err = tx.NewUpdate().
		Model((*shipment.Assignment)(nil)).
		Set("status = ?", shipment.AssignmentStatusNew).
		Where("a.shipment_move_id IN (?)", bun.List(moveIDs)).
		Where("a.archived_at IS NULL").
		Exec(ctx); err != nil {
		return err
	}

	if _, err = tx.NewUpdate().
		Model((*shipment.Stop)(nil)).
		Set("status = ?", shipment.StopStatusNew).
		Where("stp.shipment_move_id IN (?)", bun.List(moveIDs)).
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
	if err := tx.NewSelect().
		Model((*shipment.ShipmentMove)(nil)).
		Column("id").
		Where("sm.shipment_id = ?", shipmentID).
		Scan(ctx, &moveIDs); err != nil {
		return nil, err
	}

	return moveIDs, nil
}
