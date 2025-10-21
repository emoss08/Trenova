package stoprepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewRepository(p Params) repositories.StopRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.stop-repository"),
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetStopByIDRequest,
) (*shipment.Stop, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("stopID", req.StopID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(shipment.Stop)
	err = db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("st.id = ?", req.StopID).
				Where("st.organization_id = ?", req.OrgID).
				Where("st.business_unit_id = ?", req.BuID)
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Stop")
	}

	return entity, nil
}

func (r *repository) BulkInsert(
	ctx context.Context,
	entities []*shipment.Stop,
) ([]*shipment.Stop, error) {
	log := r.l.With(
		zap.String("operation", "BulkInsert"),
		zap.Int("count", len(entities)),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(&entities).Exec(ctx); err != nil {
		log.Error("failed to insert stops", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *shipment.Stop,
	moveIdx, stopIdx int,
) (*shipment.Stop, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("stopID", entity.ID.String()),
		zap.Int("moveIdx", moveIdx),
		zap.Int("stopIdx", stopIdx),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := entity.Version
	entity.Version++

	results, rErr := db.NewUpdate().
		Model(entity).
		OmitZero().
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.
				Where("stp.id = ?", entity.ID).
				Where("stp.shipment_move_id = ?", entity.ShipmentMoveID).
				Where("stp.organization_id = ?", entity.OrganizationID).
				Where("stp.version = ?", ov)
		}).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update stop", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Stop", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *repository) HandleStopRemovals(
	ctx context.Context,
	tx bun.IDB,
	move *shipment.ShipmentMove,
	existingStops []*shipment.Stop,
	updatedStopIDs map[pulid.ID]struct{},
) error {
	log := r.l.With(
		zap.String("operation", "HandleStopRemovals"),
		zap.String("moveID", move.ID.String()),
	)

	stopCount := len(existingStops)

	stopIDsToDelete := make([]pulid.ID, 0, stopCount)
	existingStopMap := make(map[pulid.ID]*shipment.Stop, stopCount)

	for _, stop := range existingStops {
		existingStopMap[stop.ID] = stop

		if _, ok := updatedStopIDs[stop.ID]; !ok {
			stopIDsToDelete = append(stopIDsToDelete, stop.ID)
		}
	}

	log.Debug(
		"stops to delete",
		zap.Any("stopIDsToDelete", stopIDsToDelete),
		zap.Int("deleteCount", len(stopIDsToDelete)),
		zap.Int("existingCount", len(existingStops)),
		zap.Int("updatedCount", len(updatedStopIDs)),
	)

	if len(stopIDsToDelete) > 0 {
		if err := r.processStopDeletions(ctx, tx, move.ID, stopIDsToDelete); err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) processStopDeletions(
	ctx context.Context,
	tx bun.IDB,
	moveID pulid.ID,
	stopIDsToDelete []pulid.ID,
) error {
	allStops, err := r.getAllStopsForMove(ctx, tx, moveID)
	if err != nil {
		return err
	}

	if err = r.validateMinimumStops(allStops); err != nil {
		return err
	}

	if err = r.validateRemainingStopTypes(allStops, stopIDsToDelete); err != nil {
		return err
	}

	if err = r.deleteStops(ctx, tx, stopIDsToDelete); err != nil {
		return err
	}

	if err = r.resequenceRemainingStops(ctx, tx, moveID); err != nil {
		return err
	}

	return nil
}

func (r *repository) getAllStopsForMove(
	ctx context.Context,
	tx bun.IDB,
	moveID pulid.ID,
) ([]*shipment.Stop, error) {
	log := r.l.With(
		zap.String("operation", "getAllStopsForMove"),
		zap.String("moveID", moveID.String()),
	)

	// Pre-allocate with reasonable capacity (typical moves have 2-10 stops)
	allStops := make([]*shipment.Stop, 0, 8)
	err := tx.NewSelect().Model(&allStops).
		Where("st.shipment_move_id = ?", moveID).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get all stops for move", zap.Error(err))
		return nil, err
	}

	return allStops, nil
}

func (r *repository) validateRemainingStopTypes(
	allStops []*shipment.Stop,
	stopIDsToDelete []pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "validateRemainingStopTypes"),
		zap.Int("allStopsCount", len(allStops)),
		zap.Int("stopIDsToDelete", len(stopIDsToDelete)),
	)

	remainingPickups := 0
	remainingDeliveries := 0

	stopsToDelete := make(map[pulid.ID]struct{}, len(stopIDsToDelete))
	for _, id := range stopIDsToDelete {
		stopsToDelete[id] = struct{}{}
	}

	for _, stop := range allStops {
		_, isBeingDeleted := stopsToDelete[stop.ID]
		if !isBeingDeleted {
			switch stop.Type { //nolint:exhaustive // We only need to check for pickup and delivery
			case shipment.StopTypePickup:
				remainingPickups++
			case shipment.StopTypeDelivery:
				remainingDeliveries++
			}
		}
	}

	log.Debug("remaining stops",
		zap.Int("remainingPickups", remainingPickups),
		zap.Int("remainingDeliveries", remainingDeliveries),
	)

	if remainingPickups == 0 {
		return errortypes.NewBusinessError(
			"Cannot delete all pickup stops. At least one pickup stop is required for the move.",
		)
	}

	if remainingDeliveries == 0 {
		return errortypes.NewBusinessError(
			"Cannot delete all delivery stops. At least one delivery stop is required for the move.",
		)
	}

	return nil
}

func (r *repository) deleteStops(
	ctx context.Context,
	tx bun.IDB,
	stopIDsToDelete []pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "deleteStops"),
		zap.Int("stopIDsToDelete", len(stopIDsToDelete)),
	)

	result, err := tx.NewDelete().
		Model((*shipment.Stop)(nil)).
		Where("id IN (?)", bun.In(stopIDsToDelete)).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete stops", zap.Error(err), zap.Any("stopIDs", stopIDsToDelete))
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error("failed to get rows affected for stop deletion", zap.Error(err))
		return err
	}

	log.Info(
		"successfully deleted stops",
		zap.Int64("deletedStopCount", rowsAffected),
		zap.Any("stopIDs", stopIDsToDelete),
	)

	return nil
}

func (r *repository) validateMinimumStops(allStops []*shipment.Stop) error {
	if len(allStops) < 2 {
		return errortypes.NewBusinessError("A move must have at least 2 stops")
	}

	return nil
}

func (r *repository) resequenceRemainingStops(
	ctx context.Context,
	tx bun.IDB,
	moveID pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "resequenceRemainingStops"),
		zap.String("moveID", moveID.String()),
	)

	now := utils.NowUnix()

	cteQuery := tx.NewSelect().
		Model((*shipment.Stop)(nil)).
		ColumnExpr("id").
		ColumnExpr("ROW_NUMBER() OVER (ORDER BY sequence ASC) - 1 as new_seq").
		ColumnExpr("sequence as old_seq").
		Where("shipment_move_id = ?", moveID)

	res, err := tx.NewUpdate().
		With("reseq", cteQuery).
		Model((*shipment.Stop)(nil)).
		TableExpr("reseq").
		Set("sequence = reseq.new_seq").
		Set("version = version + 1").
		Set("updated_at = ?", now).
		Where("st.id = reseq.id").
		Where("st.sequence != reseq.new_seq").
		Exec(ctx)
	if err != nil {
		log.Error(
			"failed to resequence remaining stops",
			zap.Error(err),
			zap.String("moveID", moveID.String()),
		)
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		log.Error("failed to get rows affected", zap.Error(err))
		return err
	}

	if rows > 0 {
		log.Info(
			"successfully resequenced stops",
			zap.Int64("updatedCount", rows),
		)
	} else {
		log.Debug("stops already properly sequenced, no updates needed")
	}

	return nil
}
