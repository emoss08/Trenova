package shipmentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/uptrace/bun"
)

func (r *repository) cancelShipmentComponents(
	ctx context.Context,
	tx bun.IDB,
	req *repositories.CancelShipmentRequest,
) error {
	moveIDs, err := r.getMoveIDsForShipment(ctx, tx, req.ShipmentID)
	if err != nil {
		return err
	}

	if len(moveIDs) == 0 {
		return nil
	}

	return r.bulkCancelShipmentComponents(ctx, tx, moveIDs)
}

func (r *repository) getMoveIDsForShipment(
	ctx context.Context,
	tx bun.IDB,
	shipmentID pulid.ID,
) (moveIDs []*pulid.ID, err error) {
	moves := make([]*shipment.ShipmentMove, 0)

	err = tx.NewSelect().Model(&moves).Where("sm.shipment_id = ?", shipmentID).Scan(ctx)
	if err != nil {
		return nil, err
	}

	moveIDs = make([]*pulid.ID, len(moves))
	for i, move := range moves {
		moveIDs[i] = &move.ID
	}

	return moveIDs, nil
}

func (r *repository) bulkCancelShipmentComponents(
	ctx context.Context,
	tx bun.IDB,
	moveIDs []*pulid.ID,
) error {
	if _, err := tx.NewUpdate().
		Model((*shipment.ShipmentMove)(nil)).
		Set("status = ?", shipment.MoveStatusCanceled).
		Where("sm.id IN (?)", bun.In(moveIDs)).
		Exec(ctx); err != nil {
		return err
	}

	if _, err := tx.NewUpdate().
		Model((*shipment.Assignment)(nil)).
		Set("status = ?", shipment.AssignmentStatusCanceled).
		Where("a.shipment_move_id IN (?)", bun.In(moveIDs)).
		Exec(ctx); err != nil {
		return err
	}

	if _, err := tx.NewUpdate().
		Model((*shipment.Stop)(nil)).
		Set("status = ?", shipment.StopStatusCanceled).
		Where("stp.shipment_move_id IN (?)", bun.In(moveIDs)).
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (r *repository) bulkUnCancelShipmentComponents(
	ctx context.Context,
	tx bun.Tx,
	moveIDs []pulid.ID,
	updateAppointments bool,
) error {
	now := utils.NowUnix()
	if _, err := tx.NewUpdate().
		Model((*shipment.ShipmentMove)(nil)).
		OmitZero().
		Set("status = ?", shipment.MoveStatusNew).
		Where("sm.id IN (?)", bun.In(moveIDs)).
		Exec(ctx); err != nil {
		return err
	}

	if _, err := tx.NewUpdate().
		Model((*shipment.Assignment)(nil)).
		OmitZero().
		Set("status = ?", shipment.AssignmentStatusNew).
		Where("a.shipment_move_id IN (?)", bun.In(moveIDs)).
		Exec(ctx); err != nil {
		return err
	}

	stpQuery := tx.NewUpdate().
		Model((*shipment.Stop)(nil)).
		Set("status = ?", shipment.StopStatusNew).
		OmitZero().
		Where("stp.shipment_move_id IN (?)", bun.In(moveIDs))

	if updateAppointments {
		stpQuery.Set("planned_arrival = ?", now)
		stpQuery.Set("planned_departure = ?", now+utils.DaysToSeconds(1))
	}

	if _, err := stpQuery.Exec(ctx); err != nil {
		return err
	}

	return nil
}
