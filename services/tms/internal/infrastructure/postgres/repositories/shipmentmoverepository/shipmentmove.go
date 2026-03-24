package shipmentmoverepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) repositories.ShipmentMoveRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-move-repository"),
	}
}

func (r *repository) SyncForShipment(
	ctx context.Context,
	tx bun.IDB,
	entity *shipment.Shipment,
) error {
	sm := buncolgen.ShipmentMoveColumns

	existingMoves, err := r.getExistingMoves(ctx, tx, entity)
	if err != nil {
		return err
	}

	existingStops, err := r.getExistingStops(ctx, tx, entity, existingMoves)
	if err != nil {
		return err
	}

	updatedMoveIDs := make(map[pulid.ID]struct{}, len(entity.Moves))
	deleteCandidates := make([]*shipment.ShipmentMove, 0, len(existingMoves))
	moveSequence := int64(0)

	for _, move := range entity.Moves {
		if move == nil {
			continue
		}

		r.normalizeMove(entity, move, moveSequence)
		moveSequence++

		moveID := move.ID
		if moveID.IsNil() {
			moveID = pulid.MustNew("sm_")
			move.ID = moveID
			if err = r.insertMove(ctx, tx, move); err != nil {
				return err
			}
		} else if existingMove, ok := existingMoves[moveID]; ok {
			if err = r.updateMove(ctx, tx, move, existingMove); err != nil {
				return err
			}
			updatedMoveIDs[moveID] = struct{}{}
		} else {
			return errortypes.NewBusinessError("Shipment contains an unknown move").
				WithParam("moveId", move.ID.String())
		}

		if err = r.syncStopsForMove(ctx, tx, move, existingStops[moveID]); err != nil {
			return err
		}
	}

	for moveID, move := range existingMoves {
		if _, ok := updatedMoveIDs[moveID]; ok {
			continue
		}

		if _, ok := r.findMoveInPayload(entity, moveID); ok {
			continue
		}

		deleteCandidates = append(deleteCandidates, move)
	}

	if len(deleteCandidates) == 0 {
		return nil
	}

	if err = r.ensureMovesAreUnassigned(ctx, tx, deleteCandidates); err != nil {
		return err
	}

	deleteIDs := make([]pulid.ID, 0, len(deleteCandidates))
	for _, move := range deleteCandidates {
		deleteIDs = append(deleteIDs, move.ID)
	}

	if _, err = tx.NewDelete().
		Model((*shipment.ShipmentMove)(nil)).
		Where(sm.ID.In(), bun.List(deleteIDs)).
		Where(sm.ShipmentID.Eq(), entity.ID).
		Where(sm.OrganizationID.Eq(), entity.OrganizationID).
		Where(sm.BusinessUnitID.Eq(), entity.BusinessUnitID).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete shipment moves: %w", err)
	}

	return nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetMoveByIDRequest,
) (*shipment.ShipmentMove, error) {
	sm := buncolgen.ShipmentMoveColumns
	stp := buncolgen.StopColumns
	entity := new(shipment.ShipmentMove)

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(sm.ID.Eq(), req.MoveID).
		Where(sm.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(sm.BusinessUnitID.Eq(), req.TenantInfo.BuID)

	if req.ForUpdate {
		query = query.For("UPDATE")
	}

	if req.ExpandMoveDetails {
		query = query.
			RelationWithOpts(buncolgen.ShipmentMoveRelations.Stops, bun.RelationOpts{
				Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.Order(stp.Sequence.OrderAsc()).
						Relation(buncolgen.StopRelations.Location).
						Relation(buncolgen.Rel(buncolgen.StopRelations.Location, buncolgen.LocationRelations.State))
				},
			})
	}

	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Shipment move")
	}

	if req.ExpandMoveDetails {
		if err := r.hydrateActiveAssignments(
			ctx,
			req.TenantInfo,
			[]*shipment.ShipmentMove{entity},
		); err != nil {
			return nil, err
		}
	}

	return entity, nil
}

func (r *repository) GetMovesByShipmentID(
	ctx context.Context,
	req *repositories.GetMovesByShipmentIDRequest,
) ([]*shipment.ShipmentMove, error) {
	sm := buncolgen.ShipmentMoveColumns
	stp := buncolgen.StopColumns
	entities := make([]*shipment.ShipmentMove, 0)

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(sm.ShipmentID.Eq(), req.ShipmentID).
		Where(sm.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(sm.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Order(sm.Sequence.OrderAsc())

	if req.ExpandMoveDetails {
		query = query.
			RelationWithOpts(buncolgen.ShipmentMoveRelations.Stops, bun.RelationOpts{
				Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.Order(stp.Sequence.OrderAsc()).
						Relation(buncolgen.StopRelations.Location).
						Relation(buncolgen.Rel(buncolgen.StopRelations.Location, buncolgen.LocationRelations.State))
				},
			})
	}

	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Shipment moves")
	}

	if req.ExpandMoveDetails {
		if err := r.hydrateActiveAssignments(ctx, req.TenantInfo, entities); err != nil {
			return nil, err
		}
	}

	return entities, nil
}

func (r *repository) hydrateActiveAssignments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moves []*shipment.ShipmentMove,
) error {
	a := buncolgen.AssignmentColumns
	moveIDs := make([]pulid.ID, 0, len(moves))
	moveByID := make(map[pulid.ID]*shipment.ShipmentMove, len(moves))

	for _, move := range moves {
		if move == nil || move.ID.IsNil() {
			continue
		}

		moveIDs = append(moveIDs, move.ID)
		moveByID[move.ID] = move
		move.Assignment = nil
	}

	if len(moveIDs) == 0 {
		return nil
	}

	assignments := make([]*shipment.Assignment, 0, len(moveIDs))
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&assignments).
		Where(a.ShipmentMoveID.In(), bun.List(moveIDs)).
		Where(a.OrganizationID.Eq(), tenantInfo.OrgID).
		Where(a.BusinessUnitID.Eq(), tenantInfo.BuID).
		Where(a.ArchivedAt.IsNull()).
		Relation(buncolgen.AssignmentRelations.Tractor).
		Relation(buncolgen.AssignmentRelations.Trailer).
		Relation(buncolgen.AssignmentRelations.PrimaryWorker).
		Relation(buncolgen.AssignmentRelations.SecondaryWorker).
		Scan(ctx); err != nil {
		if dberror.IsNotFoundError(err) {
			return nil
		}

		return fmt.Errorf("load move assignments: %w", err)
	}

	for _, assignment := range assignments {
		if assignment == nil {
			continue
		}

		if move := moveByID[assignment.ShipmentMoveID]; move != nil {
			move.Assignment = assignment
		}
	}

	return nil
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req *repositories.UpdateMoveStatusRequest,
) (*shipment.ShipmentMove, error) {
	sm := buncolgen.ShipmentMoveColumns

	move, err := r.GetByID(ctx, &repositories.GetMoveByIDRequest{
		MoveID:            req.MoveID,
		TenantInfo:        req.TenantInfo,
		ExpandMoveDetails: false,
	})
	if err != nil {
		return nil, err
	}

	ov := move.Version
	move.Status = req.Status
	move.Version++

	results, err := r.db.DBForContext(ctx).NewUpdate().
		Model(move).
		Column(sm.Status.String(), sm.Version.String(), sm.UpdatedAt.String()).
		Where(sm.ID.Eq(), move.ID).
		Where(sm.OrganizationID.Eq(), move.OrganizationID).
		Where(sm.BusinessUnitID.Eq(), move.BusinessUnitID).
		Where(sm.Version.Eq(), ov).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update shipment move status %s: %w", move.ID, err)
	}

	if err = dberror.CheckRowsAffected(results, "Shipment move", move.ID.String()); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, &repositories.GetMoveByIDRequest{
		MoveID:            req.MoveID,
		TenantInfo:        req.TenantInfo,
		ExpandMoveDetails: true,
	})
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateMoveStatusRequest,
) ([]*shipment.ShipmentMove, error) {
	sm := buncolgen.ShipmentMoveColumns

	if len(req.MoveIDs) == 0 {
		return []*shipment.ShipmentMove{}, nil
	}

	entities := make([]*shipment.ShipmentMove, 0, len(req.MoveIDs))
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		for _, moveID := range req.MoveIDs {
			move, err := r.GetByID(c, &repositories.GetMoveByIDRequest{
				MoveID:            moveID,
				TenantInfo:        req.TenantInfo,
				ExpandMoveDetails: false,
			})
			if err != nil {
				return err
			}

			ov := move.Version
			move.Status = req.Status
			move.Version++

			results, err := tx.NewUpdate().
				Model(move).
				Column(sm.Status.String(), sm.Version.String(), sm.UpdatedAt.String()).
				Where(sm.ID.Eq(), move.ID).
				Where(sm.OrganizationID.Eq(), move.OrganizationID).
				Where(sm.BusinessUnitID.Eq(), move.BusinessUnitID).
				Where(sm.Version.Eq(), ov).
				Exec(c)
			if err != nil {
				return fmt.Errorf("bulk update shipment move status %s: %w", move.ID, err)
			}

			if err = dberror.CheckRowsAffected(results, "Shipment move", move.ID.String()); err != nil {
				return err
			}

			entities = append(entities, move)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	result := make([]*shipment.ShipmentMove, 0, len(req.MoveIDs))
	for _, moveID := range req.MoveIDs {
		move, err := r.GetByID(ctx, &repositories.GetMoveByIDRequest{
			MoveID:            moveID,
			TenantInfo:        req.TenantInfo,
			ExpandMoveDetails: true,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, move)
	}

	return result, nil
}

func (r *repository) SplitMove(
	ctx context.Context,
	req *repositories.SplitMoveRequest,
) (*repositories.SplitMoveResponse, error) {
	var response *repositories.SplitMoveResponse
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		originalMove, err := r.GetByID(c, &repositories.GetMoveByIDRequest{
			MoveID:            req.MoveID,
			TenantInfo:        req.TenantInfo,
			ExpandMoveDetails: true,
		})
		if err != nil {
			return err
		}

		if err = r.shiftSubsequentMoveSequences(c, tx, originalMove); err != nil {
			return err
		}

		if err = r.updateOriginalMoveForSplit(c, tx, originalMove); err != nil {
			return err
		}

		newMove, err := r.insertSplitMove(c, tx, originalMove, req)
		if err != nil {
			return err
		}

		response, err = r.loadSplitMoveResponse(c, tx, req.TenantInfo, originalMove.ID, newMove.ID)
		return err
	})
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (r *repository) getExistingMoves(
	ctx context.Context,
	tx bun.IDB,
	entity *shipment.Shipment,
) (map[pulid.ID]*shipment.ShipmentMove, error) {
	sm := buncolgen.ShipmentMoveColumns
	moves := make([]*shipment.ShipmentMove, 0, len(entity.Moves))
	if err := tx.NewSelect().
		Model(&moves).
		Where(sm.ShipmentID.Eq(), entity.ID).
		Where(sm.OrganizationID.Eq(), entity.OrganizationID).
		Where(sm.BusinessUnitID.Eq(), entity.BusinessUnitID).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("select shipment moves: %w", err)
	}

	result := make(map[pulid.ID]*shipment.ShipmentMove, len(moves))
	for _, move := range moves {
		result[move.ID] = move
	}

	return result, nil
}

func (r *repository) getExistingStops(
	ctx context.Context,
	tx bun.IDB,
	entity *shipment.Shipment,
	existingMoves map[pulid.ID]*shipment.ShipmentMove,
) (map[pulid.ID]map[pulid.ID]*shipment.Stop, error) {
	stp := buncolgen.StopColumns
	result := make(map[pulid.ID]map[pulid.ID]*shipment.Stop, len(existingMoves))
	if len(existingMoves) == 0 {
		return result, nil
	}

	moveIDs := make([]pulid.ID, 0, len(existingMoves))
	for moveID := range existingMoves {
		moveIDs = append(moveIDs, moveID)
	}

	stops := make([]*shipment.Stop, 0)
	if err := tx.NewSelect().
		Model(&stops).
		Where(stp.ShipmentMoveID.In(), bun.List(moveIDs)).
		Where(stp.OrganizationID.Eq(), entity.OrganizationID).
		Where(stp.BusinessUnitID.Eq(), entity.BusinessUnitID).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("select shipment stops: %w", err)
	}

	for _, stop := range stops {
		if _, ok := result[stop.ShipmentMoveID]; !ok {
			result[stop.ShipmentMoveID] = make(map[pulid.ID]*shipment.Stop)
		}
		result[stop.ShipmentMoveID][stop.ID] = stop
	}

	return result, nil
}

func (r *repository) normalizeMove(
	entity *shipment.Shipment,
	move *shipment.ShipmentMove,
	sequence int64,
) {
	move.ShipmentID = entity.ID
	move.OrganizationID = entity.OrganizationID
	move.BusinessUnitID = entity.BusinessUnitID
	move.Sequence = sequence
}

func (r *repository) insertMove(
	ctx context.Context,
	tx bun.IDB,
	move *shipment.ShipmentMove,
) error {
	sm := buncolgen.ShipmentMoveColumns
	if _, err := tx.NewInsert().
		Model(move).
		Column(
			sm.ID.String(),
			sm.BusinessUnitID.String(),
			sm.OrganizationID.String(),
			sm.ShipmentID.String(),
			sm.Status.String(),
			sm.Loaded.String(),
			sm.Sequence.String(),
			sm.Distance.String(),
			sm.Version.String(),
			sm.CreatedAt.String(),
			sm.UpdatedAt.String(),
		).
		Exec(ctx); err != nil {
		return fmt.Errorf("insert shipment move %s: %w", move.ID, err)
	}

	return nil
}

func (r *repository) updateMove(
	ctx context.Context,
	tx bun.IDB,
	move *shipment.ShipmentMove,
	existingMove *shipment.ShipmentMove,
) error {
	sm := buncolgen.ShipmentMoveColumns
	move.Version = existingMove.Version + 1

	results, err := tx.NewUpdate().
		Model(move).
		Column(
			sm.Status.String(),
			sm.Loaded.String(),
			sm.Sequence.String(),
			sm.Distance.String(),
			sm.Version.String(),
			sm.UpdatedAt.String(),
		).
		Where(sm.ID.Eq(), existingMove.ID).
		Where(sm.OrganizationID.Eq(), existingMove.OrganizationID).
		Where(sm.BusinessUnitID.Eq(), existingMove.BusinessUnitID).
		Where(sm.Version.Eq(), existingMove.Version).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("update shipment move %s: %w", move.ID, err)
	}

	if err = dberror.CheckRowsAffected(results, "Shipment move", move.ID.String()); err != nil {
		return err
	}

	return nil
}

func (r *repository) syncStopsForMove(
	ctx context.Context,
	tx bun.IDB,
	move *shipment.ShipmentMove,
	existingStops map[pulid.ID]*shipment.Stop,
) error {
	stp := buncolgen.StopColumns

	if existingStops == nil {
		existingStops = make(map[pulid.ID]*shipment.Stop)
	}

	updatedStopIDs := make(map[pulid.ID]struct{}, len(move.Stops))
	stopSequence := int64(0)

	for _, stop := range move.Stops {
		if stop == nil {
			continue
		}

		r.normalizeStop(move, stop, stopSequence)
		stopSequence++

		stopID := stop.ID
		if stopID.IsNil() {
			stopID = pulid.MustNew("stp_")
			stop.ID = stopID
			if err := r.insertStop(ctx, tx, stop); err != nil {
				return err
			}
			updatedStopIDs[stopID] = struct{}{}
		} else if existingStop, ok := existingStops[stopID]; ok {
			if err := r.updateStop(ctx, tx, stop, existingStop); err != nil {
				return err
			}
			updatedStopIDs[stopID] = struct{}{}
		} else {
			return errortypes.NewBusinessError("Shipment move contains an unknown stop").
				WithParam("moveId", move.ID.String()).
				WithParam("stopId", stop.ID.String())
		}
	}

	deleteIDs := make([]pulid.ID, 0, len(existingStops))
	for stopID := range existingStops {
		if _, ok := updatedStopIDs[stopID]; ok {
			continue
		}

		if _, ok := r.findStopInPayload(move, stopID); ok {
			continue
		}

		deleteIDs = append(deleteIDs, stopID)
	}

	if len(deleteIDs) == 0 {
		return nil
	}

	if _, err := tx.NewDelete().
		Model((*shipment.Stop)(nil)).
		Where(stp.ID.In(), bun.List(deleteIDs)).
		Where(stp.ShipmentMoveID.Eq(), move.ID).
		Where(stp.OrganizationID.Eq(), move.OrganizationID).
		Where(stp.BusinessUnitID.Eq(), move.BusinessUnitID).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete shipment stops for move %s: %w", move.ID, err)
	}

	return nil
}

func (r *repository) normalizeStop(
	move *shipment.ShipmentMove,
	stop *shipment.Stop,
	sequence int64,
) {
	stop.ShipmentMoveID = move.ID
	stop.OrganizationID = move.OrganizationID
	stop.BusinessUnitID = move.BusinessUnitID
	stop.Sequence = sequence
}

func (r *repository) insertStop(
	ctx context.Context,
	tx bun.IDB,
	stop *shipment.Stop,
) error {
	if _, err := tx.NewInsert().Model(stop).Exec(ctx); err != nil {
		return fmt.Errorf("insert shipment stop %s: %w", stop.ID, err)
	}

	return nil
}

func (r *repository) updateStop(
	ctx context.Context,
	tx bun.IDB,
	stop *shipment.Stop,
	existingStop *shipment.Stop,
) error {
	stp := buncolgen.StopColumns
	stop.Version = existingStop.Version + 1

	results, err := tx.NewUpdate().
		Model(stop).
		Column(
			stp.LocationID.String(),
			stp.Status.String(),
			stp.Type.String(),
			stp.ScheduleType.String(),
			stp.Sequence.String(),
			stp.Pieces.String(),
			stp.Weight.String(),
			stp.ScheduledWindowStart.String(),
			stp.ScheduledWindowEnd.String(),
			stp.ActualArrival.String(),
			stp.ActualDeparture.String(),
			stp.CountLateOverride.String(),
			stp.CountDetentionOverride.String(),
			stp.AddressLine.String(),
			stp.Version.String(),
			stp.UpdatedAt.String(),
		).
		Where(stp.ID.Eq(), existingStop.ID).
		Where(stp.ShipmentMoveID.Eq(), existingStop.ShipmentMoveID).
		Where(stp.OrganizationID.Eq(), existingStop.OrganizationID).
		Where(stp.BusinessUnitID.Eq(), existingStop.BusinessUnitID).
		Where(stp.Version.Eq(), existingStop.Version).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("update shipment stop %s: %w", stop.ID, err)
	}

	if err = dberror.CheckRowsAffected(results, "Shipment stop", stop.ID.String()); err != nil {
		return err
	}

	return nil
}

func (r *repository) ensureMovesAreUnassigned(
	ctx context.Context,
	tx bun.IDB,
	moves []*shipment.ShipmentMove,
) error {
	a := buncolgen.AssignmentColumns
	moveIDs := make([]pulid.ID, 0, len(moves))
	for _, move := range moves {
		moveIDs = append(moveIDs, move.ID)
	}

	count, err := tx.NewSelect().
		Table("assignments").
		Where(a.ShipmentMoveID.In(), bun.List(moveIDs)).
		Where(a.OrganizationID.Eq(), moves[0].OrganizationID).
		Where(a.BusinessUnitID.Eq(), moves[0].BusinessUnitID).
		Where(a.ArchivedAt.IsNull()).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("check move assignments: %w", err)
	}

	if count == 0 {
		return nil
	}

	return errortypes.NewBusinessError("Cannot remove moves that already have assignments")
}

func (r *repository) findMoveInPayload(
	entity *shipment.Shipment,
	moveID pulid.ID,
) (*shipment.ShipmentMove, bool) {
	for _, move := range entity.Moves {
		if move == nil {
			continue
		}

		if move.ID == moveID {
			return move, true
		}
	}

	return nil, false
}

func (r *repository) findStopInPayload(
	move *shipment.ShipmentMove,
	stopID pulid.ID,
) (*shipment.Stop, bool) {
	for _, stop := range move.Stops {
		if stop == nil {
			continue
		}

		if stop.ID == stopID {
			return stop, true
		}
	}

	return nil, false
}

func (r *repository) shiftSubsequentMoveSequences(
	ctx context.Context,
	tx bun.IDB,
	originalMove *shipment.ShipmentMove,
) error {
	sm := buncolgen.ShipmentMoveColumns
	now := timeutils.NowUnix()

	_, err := tx.NewUpdate().
		Model((*shipment.ShipmentMove)(nil)).
		Set(sm.Sequence.Inc(1)).
		Set(sm.Version.Inc(1)).
		Set(sm.UpdatedAt.Set(), now).
		Where(sm.ShipmentID.Eq(), originalMove.ShipmentID).
		Where(sm.OrganizationID.Eq(), originalMove.OrganizationID).
		Where(sm.BusinessUnitID.Eq(), originalMove.BusinessUnitID).
		Where(sm.Sequence.Gt(), originalMove.Sequence).
		Exec(ctx)
	return err
}

func (r *repository) updateOriginalMoveForSplit(
	ctx context.Context,
	tx bun.IDB,
	originalMove *shipment.ShipmentMove,
) error {
	stp := buncolgen.StopColumns
	deliveryStop := originalMove.Stops[1]
	deliveryStop.Type = shipment.StopTypeSplitDelivery
	deliveryStop.Status = shipment.StopStatusNew
	deliveryStop.Version++

	results, err := tx.NewUpdate().
		Model(deliveryStop).
		Column(stp.Status.String(), stp.Type.String(), stp.Version.String(), stp.UpdatedAt.String()).
		Where(stp.ID.Eq(), deliveryStop.ID).
		Where(stp.ShipmentMoveID.Eq(), originalMove.ID).
		Where(stp.OrganizationID.Eq(), originalMove.OrganizationID).
		Where(stp.BusinessUnitID.Eq(), originalMove.BusinessUnitID).
		Where(stp.Version.Eq(), deliveryStop.Version-1).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("update original split-delivery stop %s: %w", deliveryStop.ID, err)
	}

	return dberror.CheckRowsAffected(results, "Shipment stop", deliveryStop.ID.String())
}

func (r *repository) insertSplitMove(
	ctx context.Context,
	tx bun.IDB,
	originalMove *shipment.ShipmentMove,
	req *repositories.SplitMoveRequest,
) (*shipment.ShipmentMove, error) {
	newMove := &shipment.ShipmentMove{
		ID:             pulid.MustNew("sm_"),
		BusinessUnitID: originalMove.BusinessUnitID,
		OrganizationID: originalMove.OrganizationID,
		ShipmentID:     originalMove.ShipmentID,
		Status:         shipment.MoveStatusNew,
		Loaded:         true,
		Sequence:       originalMove.Sequence + 1,
		Distance:       originalMove.Distance,
	}

	if _, err := tx.NewInsert().Model(newMove).Exec(ctx); err != nil {
		return nil, fmt.Errorf("insert split move %s: %w", newMove.ID, err)
	}

	bridgeLocationID := originalMove.Stops[1].LocationID
	newStops := []*shipment.Stop{
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       originalMove.BusinessUnitID,
			OrganizationID:       originalMove.OrganizationID,
			ShipmentMoveID:       newMove.ID,
			LocationID:           bridgeLocationID,
			Status:               shipment.StopStatusNew,
			Type:                 shipment.StopTypeSplitPickup,
			Sequence:             0,
			Pieces:               req.Pieces,
			Weight:               req.Weight,
			ScheduledWindowStart: req.SplitPickupTimes.ScheduledWindowStart,
			ScheduledWindowEnd:   req.SplitPickupTimes.ScheduledWindowEnd,
		},
		{
			ID:                   pulid.MustNew("stp_"),
			BusinessUnitID:       originalMove.BusinessUnitID,
			OrganizationID:       originalMove.OrganizationID,
			ShipmentMoveID:       newMove.ID,
			LocationID:           req.NewDeliveryLocationID,
			Status:               shipment.StopStatusNew,
			Type:                 shipment.StopTypeDelivery,
			Sequence:             1,
			Pieces:               req.Pieces,
			Weight:               req.Weight,
			ScheduledWindowStart: req.NewDeliveryTimes.ScheduledWindowStart,
			ScheduledWindowEnd:   req.NewDeliveryTimes.ScheduledWindowEnd,
		},
	}

	if _, err := tx.NewInsert().Model(&newStops).Exec(ctx); err != nil {
		return nil, fmt.Errorf("insert split move stops for move %s: %w", newMove.ID, err)
	}

	return newMove, nil
}

func (r *repository) loadSplitMoveResponse(
	ctx context.Context,
	tx bun.IDB,
	tenantInfo pagination.TenantInfo,
	originalMoveID, newMoveID pulid.ID,
) (*repositories.SplitMoveResponse, error) {
	originalMove, err := r.loadMoveWithDetails(ctx, tx, tenantInfo, originalMoveID)
	if err != nil {
		return nil, err
	}

	newMove, err := r.loadMoveWithDetails(ctx, tx, tenantInfo, newMoveID)
	if err != nil {
		return nil, err
	}

	return &repositories.SplitMoveResponse{
		OriginalMove: originalMove,
		NewMove:      newMove,
	}, nil
}

func (r *repository) loadMoveWithDetails(
	ctx context.Context,
	tx bun.IDB,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (*shipment.ShipmentMove, error) {
	sm := buncolgen.ShipmentMoveColumns
	stp := buncolgen.StopColumns
	entity := new(shipment.ShipmentMove)
	err := tx.NewSelect().
		Model(entity).
		Where(sm.ID.Eq(), moveID).
		Where(sm.OrganizationID.Eq(), tenantInfo.OrgID).
		Where(sm.BusinessUnitID.Eq(), tenantInfo.BuID).
		RelationWithOpts(buncolgen.ShipmentMoveRelations.Stops, bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order(stp.Sequence.OrderAsc()).
					Relation(buncolgen.StopRelations.Location).
					Relation(buncolgen.Rel(buncolgen.StopRelations.Location, buncolgen.LocationRelations.State))
			},
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Shipment move")
	}

	if err = r.hydrateActiveAssignments(ctx, tenantInfo, []*shipment.ShipmentMove{entity}); err != nil {
		return nil, err
	}

	return entity, nil
}
