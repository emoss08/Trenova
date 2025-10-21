package shipmentmoverepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/distancecalculator"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/sourcegraph/conc/pool"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB                        *postgres.Connection
	StopRepository            repositories.StopRepository
	ShipmentControlRepository repositories.ShipmentControlRepository
	DistanceCalculatorService services.DistanceCalculatorService
	Logger                    *zap.Logger
}

type repository struct {
	db   *postgres.Connection
	stpr repositories.StopRepository
	scr  repositories.ShipmentControlRepository
	dcs  services.DistanceCalculatorService
	l    *zap.Logger
}

func NewRepository(p Params) repositories.ShipmentMoveRepository {
	return &repository{
		db:   p.DB,
		stpr: p.StopRepository,
		scr:  p.ShipmentControlRepository,
		dcs:  p.DistanceCalculatorService,
		l:    p.Logger.Named("postgres.shipment-move-repository"),
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	opts repositories.GetMoveByIDRequest,
) (*shipment.ShipmentMove, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("moveID", opts.MoveID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	move := new(shipment.ShipmentMove)

	q := db.NewSelect().Model(move).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("sm.id = ?", opts.MoveID).
				Where("sm.organization_id = ?", opts.OrgID).
				Where("sm.business_unit_id = ?", opts.BuID)
		})

	if opts.ExpandMoveDetails {
		q.RelationWithOpts("Stops", bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Relation("Location").
					Relation("Location.State")
			},
		})

		q.Relation("Assignment").
			Relation("Assignment.Tractor").
			Relation("Assignment.Trailer").
			Relation("Assignment.PrimaryWorker").
			Relation("Assignment.SecondaryWorker")
	}

	if err = q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Shipment move")
	}

	return move, nil
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req repositories.BulkUpdateMoveStatusRequest,
) ([]*shipment.ShipmentMove, error) {
	log := r.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Int("moveCount", len(req.MoveIDs)),
		zap.String("status", string(req.Status)),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if len(req.MoveIDs) == 0 {
		log.Debug("no moves to update")
		return []*shipment.ShipmentMove{}, nil
	}

	now := utils.NowUnix()
	moves := make([]*shipment.ShipmentMove, 0, len(req.MoveIDs))

	_, err = db.NewUpdate().
		Model((*shipment.ShipmentMove)(nil)).
		Set("status = ?", req.Status).
		Set("updated_at = ?", now).
		Set("version = version + 1").
		Where("sm.id IN (?)", bun.In(req.MoveIDs)).
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update move status", zap.Error(err))
		return nil, err
	}

	err = db.NewSelect().
		Model(&moves).
		Where("sm.id IN (?)", bun.In(req.MoveIDs)).
		Scan(ctx)
	if err != nil {
		log.Error("failed to fetch updated moves", zap.Error(err))
		return nil, err
	}

	if len(moves) != len(req.MoveIDs) {
		log.Error("move count mismatch after bulk update",
			zap.Int("expected", len(req.MoveIDs)),
			zap.Int("actual", len(moves)),
		)
		return nil, dberror.CreateVersionMismatchError("Move", "bulk")
	}

	return moves, nil
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req *repositories.UpdateMoveStatusRequest,
) (*shipment.ShipmentMove, error) {
	log := r.l.With(
		zap.String("operation", "UpdateStatus"),
		zap.String("moveID", req.GetMoveReq.MoveID.String()),
		zap.String("status", string(req.Status)),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	move, err := r.GetByID(ctx, req.GetMoveReq)
	if err != nil {
		log.Error("failed to get move", zap.Error(err))
		return nil, err
	}

	ov := move.Version
	move.Version++

	results, rErr := db.NewUpdate().Model(move).
		WherePK().
		Where("sm.version = ?", ov).
		Set("status = ?", req.Status).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update move version", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Move", move.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return move, nil
}

func (r *repository) GetMovesByShipmentID(
	ctx context.Context,
	opts repositories.GetMovesByShipmentIDRequest,
) ([]*shipment.ShipmentMove, error) {
	log := r.l.With(
		zap.String("operation", "GetMovesByShipmentID"),
		zap.String("shipmentID", opts.ShipmentID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	moves := make([]*shipment.ShipmentMove, 0, 4)

	q := db.NewSelect().Model(&moves).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("sm.shipment_id = ?", opts.ShipmentID).
				Where("sm.organization_id = ?", opts.OrgID).
				Where("sm.business_unit_id = ?", opts.BuID)
		})

	if err = q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Moves")
	}

	return moves, nil
}

func (r *repository) BulkInsert(
	ctx context.Context,
	moves []*shipment.ShipmentMove,
) ([]*shipment.ShipmentMove, error) {
	log := r.l.With(
		zap.String("operation", "BulkInsert"),
		zap.Any("moves", moves),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(&moves).Exec(ctx); err != nil {
		log.Error("failed to bulk insert moves", zap.Error(err))
		return nil, err
	}

	return moves, nil
}

func (r *repository) SplitMove(
	ctx context.Context,
	req *repositories.SplitMoveRequest,
) (*repositories.SplitMoveResponse, error) {
	log := r.l.With(
		zap.String("operation", "SplitMove"),
		zap.String("moveID", req.MoveID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	originalMove, err := r.GetByID(ctx, repositories.GetMoveByIDRequest{
		MoveID:            req.MoveID,
		OrgID:             req.OrgID,
		BuID:              req.BuID,
		ExpandMoveDetails: true,
	})
	if err != nil {
		return nil, err
	}

	var newMove *shipment.ShipmentMove
	var response *repositories.SplitMoveResponse

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if err = r.updateMoveSequences(c, tx, originalMove); err != nil {
			return err
		}

		if err = r.modifyOriginalMove(c, tx, originalMove, req); err != nil {
			return err
		}

		newMove, err = r.createSplitMove(c, tx, originalMove, req)
		if err != nil {
			return err
		}

		response, err = r.prepareSplitMoveResponse(c, tx, req, originalMove, newMove)
		return err
	})
	if err != nil {
		log.Error(
			"failed to split move",
			zap.Error(err),
			zap.Any("originalMove", originalMove),
			zap.Any("newMove", newMove),
		)
		return nil, err
	}

	return response, nil
}

func (r *repository) updateMoveSequences(
	ctx context.Context,
	tx bun.Tx,
	originalMove *shipment.ShipmentMove,
) error {
	log := r.l.With(
		zap.String("operation", "updateMoveSequences"),
		zap.String("moveID", originalMove.GetID()),
	)

	now := utils.NowUnix()
	_, err := tx.NewUpdate().
		Model((*shipment.ShipmentMove)(nil)).
		Set("sequence = sequence + 1").
		Set("version = version + 1").
		Set("updated_at = ?", now).
		Where("shipment_id = ?", originalMove.ShipmentID).
		Where("sequence > ?", originalMove.Sequence).
		Exec(ctx)
	if err != nil {
		log.Error("failed to update move sequences", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) modifyOriginalMove(
	ctx context.Context,
	tx bun.Tx,
	originalMove *shipment.ShipmentMove,
	req *repositories.SplitMoveRequest,
) error {
	log := r.l.With(
		zap.String("operation", "modifyOriginalMove"),
		zap.String("moveID", originalMove.GetID()),
	)

	_, err := tx.NewDelete().Model((*shipment.Stop)(nil)).
		Where("shipment_move_id = ? AND sequence = ?", originalMove.ID, 1).
		Exec(ctx)
	if err != nil {
		return dberror.HandleNotFoundError(err, "Original delivery stop")
	}

	splitDeliveryStop := &shipment.Stop{
		ID:               pulid.MustNew("stp_"),
		BusinessUnitID:   originalMove.BusinessUnitID,
		OrganizationID:   originalMove.OrganizationID,
		ShipmentMoveID:   originalMove.ID,
		LocationID:       req.SplitLocationID,
		Status:           shipment.StopStatusNew,
		Type:             shipment.StopTypeSplitDelivery,
		Sequence:         1,
		Pieces:           req.SplitQuantities.Pieces,
		Weight:           req.SplitQuantities.Weight,
		PlannedArrival:   req.SplitDeliveryTimes.PlannedArrival,
		PlannedDeparture: req.SplitDeliveryTimes.PlannedDeparture,
	}

	if _, err = tx.NewInsert().Model(splitDeliveryStop).Exec(ctx); err != nil {
		log.Error("failed to insert split delivery stop", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) createSplitMove(
	ctx context.Context,
	tx bun.Tx,
	originalMove *shipment.ShipmentMove,
	req *repositories.SplitMoveRequest,
) (*shipment.ShipmentMove, error) {
	log := r.l.With(
		zap.String("operation", "createSplitMove"),
		zap.String("moveID", originalMove.GetID()),
	)

	newMove := &shipment.ShipmentMove{
		ID:             pulid.MustNew("smv_"),
		BusinessUnitID: originalMove.BusinessUnitID,
		OrganizationID: originalMove.OrganizationID,
		ShipmentID:     originalMove.ShipmentID,
		Status:         shipment.MoveStatusNew,
		Loaded:         true,
		Sequence:       1, // Explicitly set to 1
		Distance:       originalMove.Distance,
	}

	if _, err := tx.NewInsert().Model(newMove).Exec(ctx); err != nil {
		log.Error("failed to insert new move", zap.Error(err))
		return nil, err
	}

	newMoveStops := r.createSplitMoveStops(newMove, originalMove, req)

	if _, err := tx.NewInsert().Model(&newMoveStops).Exec(ctx); err != nil {
		log.Error("failed to insert new move stops", zap.Error(err))
		return nil, err
	}

	return newMove, nil
}

func (r *repository) createSplitMoveStops(
	newMove, originalMove *shipment.ShipmentMove,
	req *repositories.SplitMoveRequest,
) []*shipment.Stop {
	return []*shipment.Stop{
		{
			ID:               pulid.MustNew("stp_"),
			BusinessUnitID:   originalMove.BusinessUnitID,
			OrganizationID:   originalMove.OrganizationID,
			ShipmentMoveID:   newMove.ID,
			LocationID:       req.SplitLocationID,
			Status:           shipment.StopStatusNew,
			Type:             shipment.StopTypeSplitPickup,
			Sequence:         0,
			Pieces:           req.SplitQuantities.Pieces,
			Weight:           req.SplitQuantities.Weight,
			PlannedArrival:   req.SplitPickupTimes.PlannedArrival,
			PlannedDeparture: req.SplitPickupTimes.PlannedDeparture,
		},
		{
			ID:               pulid.MustNew("stp_"),
			BusinessUnitID:   originalMove.BusinessUnitID,
			OrganizationID:   originalMove.OrganizationID,
			ShipmentMoveID:   newMove.ID,
			LocationID:       originalMove.Stops[1].LocationID,
			Status:           shipment.StopStatusNew,
			Type:             shipment.StopTypeDelivery,
			Sequence:         1,
			Pieces:           req.SplitQuantities.Pieces,
			Weight:           req.SplitQuantities.Weight,
			PlannedArrival:   originalMove.Stops[1].PlannedArrival,
			PlannedDeparture: originalMove.Stops[1].PlannedDeparture,
			AddressLine:      originalMove.Stops[1].AddressLine,
		},
	}
}

func (r *repository) prepareSplitMoveResponse(
	ctx context.Context,
	tx bun.Tx,
	req *repositories.SplitMoveRequest,
	originalMove, newMove *shipment.ShipmentMove,
) (*repositories.SplitMoveResponse, error) {
	p := pool.NewWithResults[*shipment.ShipmentMove]().
		WithErrors().
		WithContext(ctx).
		WithMaxGoroutines(2)

	p.Go(func(ctx context.Context) (*shipment.ShipmentMove, error) {
		move := new(shipment.ShipmentMove)
		err := tx.NewSelect().Model(move).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Where("sm.id = ?", originalMove.ID).
					Where("sm.organization_id = ?", req.OrgID).
					Where("sm.business_unit_id = ?", req.BuID)
			}).
			Relation("Stops", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Relation("Location").Relation("Location.State")
			}).
			Relation("Assignment").
			Relation("Assignment.Tractor").
			Relation("Assignment.Trailer").
			Relation("Assignment.PrimaryWorker").
			Relation("Assignment.SecondaryWorker").
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetch original move: %w", err)
		}
		return move, nil
	})

	p.Go(func(ctx context.Context) (*shipment.ShipmentMove, error) {
		move := new(shipment.ShipmentMove)
		err := tx.NewSelect().Model(move).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Where("sm.id = ?", newMove.ID).
					Where("sm.organization_id = ?", req.OrgID).
					Where("sm.business_unit_id = ?", req.BuID)
			}).
			Relation("Stops", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Relation("Location").Relation("Location.State")
			}).
			Relation("Assignment").
			Relation("Assignment.Tractor").
			Relation("Assignment.Trailer").
			Relation("Assignment.PrimaryWorker").
			Relation("Assignment.SecondaryWorker").
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetch new move: %w", err)
		}
		return move, nil
	})

	results, err := p.Wait()
	if err != nil {
		return nil, err
	}

	// ! Results are in order: [0]=original, [1]=new
	if len(results) != 2 {
		return nil, fmt.Errorf("expected 2 results, got %d", len(results))
	}

	return &repositories.SplitMoveResponse{
		OriginalMove: results[0],
		NewMove:      results[1],
	}, nil
}

func (r *repository) HandleMoveOperations(
	ctx context.Context,
	tx bun.IDB,
	shp *shipment.Shipment,
	isCreate bool,
) error {
	log := r.l.With(
		zap.String("operation", "HandleMoveOperations"),
		zap.String("shipmentID", shp.ID.String()),
		zap.Bool("isCreate", isCreate),
	)

	type parallelResult struct {
		scr       *tenant.ShipmentControl
		movesData *movesOperationData
	}

	p := pool.NewWithResults[*parallelResult]().
		WithErrors().
		WithContext(ctx).
		WithMaxGoroutines(2)

	p.Go(func(ctx context.Context) (*parallelResult, error) {
		scr, err := r.scr.GetByOrgID(ctx, shp.OrganizationID)
		if err != nil {
			log.Error("failed to get shipment control", zap.Error(err))
			return nil, fmt.Errorf("get shipment control: %w", err)
		}
		return &parallelResult{scr: scr}, nil
	})

	p.Go(func(ctx context.Context) (*parallelResult, error) {
		movesData, err := r.prepareMovesData(ctx, shp, isCreate)
		if err != nil {
			return nil, fmt.Errorf("prepare moves data: %w", err)
		}
		return &parallelResult{movesData: movesData}, nil
	})

	results, err := p.Wait()
	if err != nil {
		return err
	}

	var scr *tenant.ShipmentControl
	var movesData *movesOperationData
	for _, result := range results {
		if result.scr != nil {
			scr = result.scr
		}
		if result.movesData != nil {
			movesData = result.movesData
		}
	}

	// ! Handle database operations sequentially (they must be in transaction order)
	if err = r.processNewMoves(ctx, tx, movesData.newMoves); err != nil {
		log.Error("failed to process new moves", zap.Error(err))
		return err
	}

	if err = r.processUpdateMoves(ctx, tx, movesData.updateMoves); err != nil {
		log.Error("failed to process update moves", zap.Error(err))
		return err
	}

	if !isCreate {
		if err = r.checkAndHandleMoveDeletions(ctx, tx, shp, scr, movesData); err != nil {
			log.Error("failed to check and handle move deletions", zap.Error(err))
			return err
		}
	}

	return nil
}

type movesOperationData struct {
	newMoves        []*shipment.ShipmentMove
	updateMoves     []*shipment.ShipmentMove
	existingMoveMap map[pulid.ID]*shipment.ShipmentMove
	updatedMoveIDs  map[pulid.ID]struct{}
	moveToDelete    []*shipment.ShipmentMove
	existingMoves   []*shipment.ShipmentMove
}

func (r *repository) prepareMovesData(
	ctx context.Context,
	shp *shipment.Shipment,
	isCreate bool,
) (*movesOperationData, error) {
	log := r.l.With(
		zap.String("operation", "prepareMovesData"),
		zap.String("shipmentID", shp.ID.String()),
		zap.Bool("isCreate", isCreate),
	)

	moveCount := len(shp.Moves)
	data := &movesOperationData{
		newMoves:        make([]*shipment.ShipmentMove, 0, moveCount),
		updateMoves:     make([]*shipment.ShipmentMove, 0, moveCount),
		existingMoveMap: make(map[pulid.ID]*shipment.ShipmentMove, moveCount),
		updatedMoveIDs:  make(map[pulid.ID]struct{}, moveCount),
		moveToDelete:    make([]*shipment.ShipmentMove, 0, moveCount),
		existingMoves:   make([]*shipment.ShipmentMove, 0, moveCount),
	}

	if !isCreate {
		var err error
		data.existingMoves, err = r.GetMovesByShipmentID(
			ctx,
			repositories.GetMovesByShipmentIDRequest{
				ShipmentID: shp.ID,
				OrgID:      shp.OrganizationID,
				BuID:       shp.BusinessUnitID,
			},
		)
		if err != nil {
			log.Error("failed to get existing moves", zap.Error(err))
			return nil, err
		}

		for _, move := range data.existingMoves {
			data.existingMoveMap[move.ID] = move
		}
	}

	r.categorizeMoves(shp, data, isCreate)
	return data, nil
}

func (r *repository) categorizeMoves(
	shp *shipment.Shipment,
	data *movesOperationData,
	isCreate bool,
) {
	for _, move := range shp.Moves {
		move.ShipmentID = shp.ID
		move.OrganizationID = shp.OrganizationID
		move.BusinessUnitID = shp.BusinessUnitID

		if isCreate || move.ID.IsNil() {
			move.ID = pulid.MustNew("smv_")
			data.newMoves = append(data.newMoves, move)
		} else {
			if existing, ok := data.existingMoveMap[move.ID]; ok {
				move.Version = existing.Version + 1
				data.updateMoves = append(data.updateMoves, move)
				data.updatedMoveIDs[move.ID] = struct{}{}
			}
		}
	}
}

func (r *repository) processNewMoves(
	ctx context.Context,
	tx bun.IDB,
	newMoves []*shipment.ShipmentMove,
) error {
	log := r.l.With(
		zap.String("operation", "processNewMoves"),
		zap.Any("newMoves", newMoves),
	)

	if len(newMoves) == 0 {
		log.Debug("no new moves to process")
		return nil
	}

	if _, err := tx.NewInsert().Model(&newMoves).Exec(ctx); err != nil {
		log.Error("failed to bulk insert new moves", zap.Error(err))
		return err
	}

	for _, move := range newMoves {
		log.Debug("new move", zap.Any("move", move))
		if err := r.insertStopsForMove(ctx, tx, move); err != nil {
			return err
		}

		if err := r.calculateAndUpdateMoveDistance(ctx, tx, move); err != nil {
			log.Error("failed to calculate move distance", zap.Error(err))
			return err
		}
	}

	return nil
}

func (r *repository) insertStopsForMove(
	ctx context.Context,
	tx bun.IDB,
	move *shipment.ShipmentMove,
) error {
	log := r.l.With(
		zap.String("operation", "insertStopsForMove"),
		zap.String("moveID", move.ID.String()),
		zap.Int("stopCount", len(move.Stops)),
	)

	if len(move.Stops) == 0 {
		log.Debug("no stops to insert")
		return nil
	}

	for _, stop := range move.Stops {
		stop.ShipmentMoveID = move.ID
		stop.OrganizationID = move.OrganizationID
		stop.BusinessUnitID = move.BusinessUnitID
	}

	if _, err := tx.NewInsert().Model(&move.Stops).Exec(ctx); err != nil {
		log.Error("failed to bulk insert stops", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) processUpdateMoves(
	ctx context.Context,
	tx bun.IDB,
	updateMoves []*shipment.ShipmentMove,
) error {
	log := r.l.With(
		zap.String("operation", "processUpdateMoves"),
		zap.Int("moveCount", len(updateMoves)),
	)

	if len(updateMoves) == 0 {
		log.Debug("no update moves to process")
		return nil
	}

	if err := r.handleBulkUpdate(ctx, tx, updateMoves); err != nil {
		log.Error("failed to handle bulk update of moves", zap.Error(err))
		return err
	}

	for moveIdx, move := range updateMoves {
		if err := r.processStopsForExistingMove(ctx, tx, move, moveIdx); err != nil {
			return err
		}

		if err := r.calculateAndUpdateMoveDistance(ctx, tx, move); err != nil {
			log.Error("failed to calculate and update move distance", zap.Error(err))
			return err
		}
	}

	return nil
}

func (r *repository) processStopsForExistingMove(
	ctx context.Context,
	tx bun.IDB,
	move *shipment.ShipmentMove,
	moveIdx int,
) error {
	log := r.l.With(
		zap.String("operation", "processStopsForExistingMove"),
		zap.String("moveID", move.ID.String()),
	)

	existingStops := make([]*shipment.Stop, 0, 8)
	err := tx.NewSelect().Model(&existingStops).
		Where("shipment_move_id = ?", move.ID).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get existing stops for move", zap.Error(err))
		return err
	}

	stopCount := len(move.Stops)
	updatedStopIDs := make(map[pulid.ID]struct{}, stopCount)
	newStops := make([]*shipment.Stop, 0, stopCount)

	for stopIdx, stop := range move.Stops {
		stop.ShipmentMoveID = move.ID
		stop.OrganizationID = move.OrganizationID
		stop.BusinessUnitID = move.BusinessUnitID

		if stop.ID.IsNil() {
			stop.ID = pulid.MustNew("stp_")
			newStops = append(newStops, stop)
		} else {
			if _, err = r.stpr.Update(ctx, stop, moveIdx, stopIdx); err != nil {
				log.Error("failed to update stop",
					zap.Error(err),
					zap.Int("moveIdx", moveIdx),
					zap.Int("stopIdx", stopIdx),
				)
				return err
			}
			updatedStopIDs[stop.ID] = struct{}{}
		}
	}

	if len(newStops) > 0 {
		if _, err = tx.NewInsert().Model(&newStops).Exec(ctx); err != nil {
			log.Error("failed to bulk insert new stops", zap.Error(err))
			return err
		}
	}

	if len(existingStops) > 0 {
		if err = r.stpr.HandleStopRemovals(ctx, tx, move, existingStops, updatedStopIDs); err != nil {
			log.Error("failed to handle stop removals", zap.Error(err), zap.Int("moveIdx", moveIdx))
			return err
		}
	}

	if err = r.calculateAndUpdateMoveDistance(ctx, tx, move); err != nil {
		log.Error("failed to calculate move distance after stop changes", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) checkAndHandleMoveDeletions(
	ctx context.Context,
	tx bun.IDB,
	shp *shipment.Shipment,
	scr *tenant.ShipmentControl,
	data *movesOperationData,
) error {
	log := r.l.With(
		zap.String("operation", "checkAndHandleMoveDeletions"),
		zap.String("shipmentID", shp.ID.String()),
	)

	deletionRequired := false
	for moveID := range data.existingMoveMap {
		if _, ok := data.updatedMoveIDs[moveID]; !ok {
			deletionRequired = true

			if !scr.AllowMoveRemovals {
				log.Debug(
					"Organization does not allow move removals, returning error...",
					zap.String("organizationID", shp.OrganizationID.String()),
					zap.Any("data", data),
				)
				return errortypes.NewBusinessError(
					"Your organization does not allow move removals",
				)
			}
			break
		}
	}

	if deletionRequired {
		if err := r.handleMoveDeletions(ctx, tx, &repositories.HandleMoveDeletionsRequest{
			ExistingMoveMap: data.existingMoveMap,
			UpdatedMoveIDs:  data.updatedMoveIDs,
			MoveToDelete:    data.moveToDelete,
		}); err != nil {
			log.Error("failed to handle move deletions", zap.Error(err))
			return err
		}
	}

	return nil
}

func (r *repository) handleBulkUpdate(
	ctx context.Context,
	tx bun.IDB,
	moves []*shipment.ShipmentMove,
) error {
	log := r.l.With(
		zap.String("operation", "handleBulkUpdate"),
		zap.Int("moveCount", len(moves)),
	)

	if len(moves) == 0 {
		return nil
	}

	movesInterface := make([]any, len(moves))
	for i, move := range moves {
		movesInterface[i] = move
	}

	values := tx.NewValues(&movesInterface)

	res, err := tx.NewUpdate().
		With("_data", values).
		Model((*shipment.ShipmentMove)(nil)).
		TableExpr("_data").
		Set("shipment_id = _data.shipment_id").
		Set("status = _data.status").
		Set("loaded = _data.loaded").
		Set("sequence = _data.sequence").
		Set("distance = _data.distance").
		Set("version = _data.version").
		Set("updated_at = ?", utils.NowUnix()).
		Where("sm.id = _data.id").
		Where("sm.version = _data.version - 1").
		Where("sm.organization_id = _data.organization_id").
		Where("sm.business_unit_id = _data.business_unit_id").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update moves", zap.Error(err))
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		log.Error("failed to get rows affected", zap.Error(err))
		return err
	}

	if rows != int64(len(moves)) {
		log.Error("move count mismatch in bulk update",
			zap.Int64("expected", int64(len(moves))),
			zap.Int64("actual", rows),
		)
		return dberror.CreateVersionMismatchError("Move", "bulk_update")
	}

	return nil
}

func (r *repository) handleMoveDeletions(
	ctx context.Context,
	tx bun.IDB,
	req *repositories.HandleMoveDeletionsRequest,
) error {
	log := r.l.With(
		zap.String("operation", "handleMoveDeletions"),
	)

	moveIDsToDelete := make([]pulid.ID, 0, len(req.ExistingMoveMap))

	for moveID, move := range req.ExistingMoveMap {
		if _, ok := req.UpdatedMoveIDs[moveID]; !ok {
			moveIDsToDelete = append(moveIDsToDelete, moveID)
			req.MoveToDelete = append(req.MoveToDelete, move)
		}
	}

	if len(moveIDsToDelete) > 0 {
		shipmentID := req.ExistingMoveMap[moveIDsToDelete[0]].ShipmentID

		if err := r.deleteMovesAndAssociatedData(ctx, tx, moveIDsToDelete); err != nil {
			log.Error("failed to delete moves and associated data", zap.Error(err))
			return err
		}

		if err := r.resequenceRemainingMoves(ctx, tx, shipmentID); err != nil {
			log.Error(
				"failed to resequence remaining moves",
				zap.Error(err),
				zap.Any("moveIDs", moveIDsToDelete),
			)
			return err
		}
	}

	return nil
}

func (r *repository) deleteMovesAndAssociatedData(
	ctx context.Context,
	tx bun.IDB,
	moveIDsToDelete []pulid.ID,
) error {
	if err := r.deleteAssociatedStops(ctx, tx, moveIDsToDelete); err != nil {
		return err
	}

	if err := r.deleteAssociatedAssignments(ctx, tx, moveIDsToDelete); err != nil {
		return err
	}

	return r.deleteMoves(ctx, tx, moveIDsToDelete)
}

func (r *repository) deleteAssociatedStops(
	ctx context.Context,
	tx bun.IDB,
	moveIDs []pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "deleteAssociatedStops"),
		zap.Any("moveIDs", moveIDs),
	)

	_, err := tx.NewDelete().
		Model((*shipment.Stop)(nil)).
		Where("shipment_move_id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete associated stops", zap.Error(err), zap.Any("moveIDs", moveIDs))

		return dberror.HandleNotFoundError(err, "Stop")
	}

	return nil
}

func (r *repository) deleteAssociatedAssignments(
	ctx context.Context,
	tx bun.IDB,
	moveIDs []pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "deleteAssociatedAssignments"),
		zap.Any("moveIDs", moveIDs),
	)

	_, err := tx.NewDelete().
		Model((*shipment.Assignment)(nil)).
		Where("shipment_move_id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		log.Error(
			"failed to delete associated assignments",
			zap.Error(err),
			zap.Any("moveIDs", moveIDs),
		)
		return dberror.HandleNotFoundError(err, "Assignment")
	}

	return nil
}

func (r *repository) deleteMoves(
	ctx context.Context,
	tx bun.IDB,
	moveIDs []pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "deleteMoves"),
		zap.Any("moveIDs", moveIDs),
	)

	result, err := tx.NewDelete().
		Model((*shipment.ShipmentMove)(nil)).
		Where("id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete moves", zap.Error(err), zap.Any("moveIDs", moveIDs))
		return err
	}

	roErr := dberror.CheckRowsAffected(result, "Move", moveIDs[0].String())
	if roErr != nil {
		return roErr
	}

	return nil
}

func (r *repository) resequenceRemainingMoves(
	ctx context.Context,
	tx bun.IDB,
	shipmentID pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "resequenceRemainingMoves"),
		zap.String("shipmentID", shipmentID.String()),
	)

	now := utils.NowUnix()

	cteQuery := tx.NewSelect().
		Model((*shipment.ShipmentMove)(nil)).
		ColumnExpr("id").
		ColumnExpr("ROW_NUMBER() OVER (ORDER BY sequence ASC) - 1 as new_seq").
		ColumnExpr("sequence as old_seq").
		Where("shipment_id = ?", shipmentID)

	res, err := tx.NewUpdate().
		With("reseq", cteQuery).
		Model((*shipment.ShipmentMove)(nil)).
		TableExpr("reseq").
		Set("sequence = reseq.new_seq").
		Set("version = version + 1").
		Set("updated_at = ?", now).
		Where("sm.id = reseq.id").
		Where("sm.sequence != reseq.new_seq").
		Exec(ctx)
	if err != nil {
		log.Error(
			"failed to resequence remaining moves",
			zap.Error(err),
			zap.String("shipmentID", shipmentID.String()),
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
			"successfully resequenced moves",
			zap.Int64("updatedCount", rows),
		)
	} else {
		log.Debug("moves already properly sequenced, no updates needed")
	}

	return nil
}

func (r *repository) calculateAndUpdateMoveDistance(
	ctx context.Context,
	tx bun.IDB,
	move *shipment.ShipmentMove,
) error {
	log := r.l.With(
		zap.String("operation", "calculateAndUpdateMoveDistance"),
		zap.String("moveID", move.ID.String()),
	)

	if len(move.Stops) < 2 {
		log.Debug(
			"insufficient stops for distance calculation",
			zap.Int("stopCount", len(move.Stops)),
		)
		return nil
	}

	result, err := r.dcs.CalculateDistance(ctx, tx, &services.DistanceCalculationRequest{
		MoveID:         move.ID,
		OrganizationID: move.OrganizationID,
		BusinessUnitID: move.BusinessUnitID,
	})
	if err != nil {
		if errors.Is(err, distancecalculator.ErrInsufficientStops) {
			log.Debug("service reported insufficient stops")
			return nil
		}
		log.Error("failed to calculate distance", zap.Error(err))
		return fmt.Errorf("calculate distance: %w", err)
	}

	move.Distance = &result.Distance

	if err = r.updateMoveDistance(ctx, tx, move); err != nil {
		log.Error("failed to persist move distance", zap.Error(err))
		return fmt.Errorf("update move distance: %w", err)
	}

	log.Debug("move distance updated successfully",
		zap.Float64("distance", result.Distance),
		zap.String("source", string(result.Source)),
	)

	return nil
}

func (r *repository) updateMoveDistance(
	ctx context.Context,
	tx bun.IDB,
	move *shipment.ShipmentMove,
) error {
	log := r.l.With(
		zap.String("operation", "updateMoveDistance"),
		zap.String("moveID", move.ID.String()),
	)

	_, err := tx.NewUpdate().
		Model(move).
		Column("distance").
		WherePK().
		Exec(ctx)
	if err != nil {
		log.Error("failed to update move distance", zap.Error(err))
		return err
	}

	return nil
}
