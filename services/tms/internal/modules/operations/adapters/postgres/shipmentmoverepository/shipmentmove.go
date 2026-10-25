package shipmentmoverepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	moveErrors       = dberror.Mapping{Entity: "Shipment move"}
	stopErrors       = dberror.Mapping{Entity: "Shipment stop"}
	stopActualErrors = dberror.Mapping{Entity: "Stop"}
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

func New(p Params) shipmentports.ShipmentMoveRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-move-repository"),
	}
}

func (r *repository) SyncForShipment(ctx context.Context, entity *shipment.Shipment) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		return r.syncForShipment(ctx, entity)
	})
}

func (r *repository) syncForShipment(ctx context.Context, entity *shipment.Shipment) error {
	existingMoves, err := r.getExistingMoves(ctx, entity)
	if err != nil {
		return err
	}

	existingStops, err := r.getExistingStops(ctx, existingMoves)
	if err != nil {
		return err
	}

	updatedMoveIDs := make(map[pulid.ID]struct{}, len(entity.Moves))
	moveSequence := int64(0)

	for _, move := range entity.Moves {
		if move == nil {
			continue
		}

		normalizeMove(entity, move, moveSequence)
		moveSequence++

		if move.ID.IsNil() {
			move.ID = pulid.MustNew("sm_")
			if err = r.insertMove(ctx, move); err != nil {
				return err
			}
		} else if existingMove, ok := existingMoves[move.ID]; ok {
			if err = r.updateMove(ctx, move, existingMove); err != nil {
				return err
			}
			updatedMoveIDs[move.ID] = struct{}{}
		} else {
			return errortypes.NewBusinessError("Shipment contains an unknown move").
				WithParam("moveId", move.ID.String())
		}

		if err = r.syncStopsForMove(ctx, move, existingStops[move.ID]); err != nil {
			return err
		}

		if move.JurisdictionMilesDirty {
			if err = r.replaceJurisdictionMilesForMove(ctx, move); err != nil {
				return err
			}
		}
	}

	deleteIDs := make([]pulid.ID, 0, len(existingMoves))
	for moveID := range existingMoves {
		if _, ok := updatedMoveIDs[moveID]; ok {
			continue
		}
		if moveInPayload(entity, moveID) {
			continue
		}
		deleteIDs = append(deleteIDs, moveID)
	}

	if len(deleteIDs) == 0 {
		return nil
	}

	if err = r.ensureMovesAreUnassigned(ctx, deleteIDs); err != nil {
		return err
	}

	sm := shipmentdb.ShipmentMoves
	if _, err = sm.Where(sm.ID.In(deleteIDs...), sm.ShipmentID.Eq(entity.ID)).
		Delete(ctx, r.db.Keel(ctx)); err != nil {
		return fmt.Errorf("delete shipment moves: %w", err)
	}

	return nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *shipmentports.GetMoveByIDRequest,
) (*shipment.ShipmentMove, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.ShipmentMove, error) {
		sm := shipmentdb.ShipmentMoves
		q := sm.Where(sm.ID.Eq(req.MoveID))
		if req.ForUpdate {
			q = q.Lock(keel.ForUpdate)
		}
		if req.ExpandMoveDetails {
			q = q.Load(stopsWithLocation())
		}

		entity, err := q.One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, moveErrors.Map(err, req.MoveID.String())
		}

		if req.ExpandMoveDetails {
			if err = r.hydrateActiveAssignments(ctx, []*shipment.ShipmentMove{entity}); err != nil {
				return nil, err
			}
		}

		return entity, nil
	})
}

func (r *repository) GetMovesByShipmentID(
	ctx context.Context,
	req *shipmentports.GetMovesByShipmentIDRequest,
) ([]*shipment.ShipmentMove, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*shipment.ShipmentMove, error) {
		sm := shipmentdb.ShipmentMoves
		q := sm.Where(sm.ShipmentID.Eq(req.ShipmentID)).Order(sm.Sequence.Asc())
		if req.ExpandMoveDetails {
			q = q.Load(stopsWithLocation())
		}

		entities, err := q.All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, moveErrors.Map(err, req.ShipmentID.String())
		}

		if req.ExpandMoveDetails {
			if err = r.hydrateActiveAssignments(ctx, entities); err != nil {
				return nil, err
			}
		}

		return entities, nil
	})
}

func stopsWithLocation() keel.Loader {
	st := shipmentdb.ShipmentMoves.Stops()
	loc := st.Location()

	return st.Order(st.Sequence.Asc()).Load(loc.Load(loc.State()))
}

func (r *repository) hydrateActiveAssignments(
	ctx context.Context,
	moves []*shipment.ShipmentMove,
) error {
	moveIDs := make([]pulid.ID, 0, len(moves))
	moveByID := make(map[pulid.ID]*shipment.ShipmentMove, len(moves))

	for _, move := range moves {
		if move == nil || move.ID.IsNil() {
			continue
		}

		moveIDs = append(moveIDs, move.ID)
		moveByID[move.ID] = move
		move.Assignment = nil
		move.CarrierAssignment = nil
	}

	if len(moveIDs) == 0 {
		return nil
	}

	a := shipmentdb.Assignments
	assignments, err := a.Where(a.ShipmentMoveID.In(moveIDs...), a.ArchivedAt.IsNull()).
		Load(a.Tractor(), a.Trailer(), a.PrimaryWorker(), a.SecondaryWorker()).
		All(ctx, r.db.Keel(ctx))
	if err != nil {
		return fmt.Errorf("load move assignments: %w", err)
	}

	for _, assignment := range assignments {
		if move := moveByID[assignment.ShipmentMoveID]; move != nil {
			move.Assignment = assignment
		}
	}

	ca := shipmentdb.CarrierAssignments
	carrierAssignments, err := ca.Where(
		ca.ShipmentMoveID.In(moveIDs...),
		ca.Status.Ne(shipment.CarrierAssignmentStatusCanceled),
	).
		Load(ca.Carrier(), ca.Accessorials()).
		All(ctx, r.db.Keel(ctx))
	if err != nil {
		return fmt.Errorf("load move carrier assignments: %w", err)
	}

	for _, carrierAssignment := range carrierAssignments {
		if move := moveByID[carrierAssignment.ShipmentMoveID]; move != nil {
			move.CarrierAssignment = carrierAssignment
		}
	}

	return nil
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req *shipmentports.UpdateMoveStatusRequest,
) (*shipment.ShipmentMove, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.ShipmentMove, error) {
		move, err := r.GetByID(ctx, &shipmentports.GetMoveByIDRequest{
			MoveID:     req.MoveID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}

		sm := shipmentdb.ShipmentMoves
		move.Status = req.Status
		if err = sm.Save(ctx, r.db.Keel(ctx), move, sm.Status); err != nil {
			return nil, fmt.Errorf(
				"update shipment move status %s: %w",
				move.ID,
				moveErrors.Map(err, move.ID.String()),
			)
		}

		return r.GetByID(ctx, &shipmentports.GetMoveByIDRequest{
			MoveID:            req.MoveID,
			TenantInfo:        req.TenantInfo,
			ExpandMoveDetails: true,
		})
	})
}

func (r *repository) UpdateStopActuals(
	ctx context.Context,
	_ pagination.TenantInfo,
	stop *shipment.Stop,
) (*shipment.Stop, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*shipment.Stop, error) {
		st := shipmentdb.Stops
		if err := st.Save(
			ctx,
			r.db.Keel(ctx),
			stop,
			st.ActualArrival,
			st.ActualDeparture,
			st.Status,
		); err != nil {
			return nil, fmt.Errorf(
				"update stop actuals %s: %w",
				stop.ID,
				stopActualErrors.Map(err, stop.ID.String()),
			)
		}

		return stop, nil
	})
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *shipmentports.BulkUpdateMoveStatusRequest,
) ([]*shipment.ShipmentMove, error) {
	if len(req.MoveIDs) == 0 {
		return []*shipment.ShipmentMove{}, nil
	}

	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*shipment.ShipmentMove, error) {
		ids := uniqueIDs(req.MoveIDs)
		sm := shipmentdb.ShipmentMoves
		n, err := sm.Where(sm.ID.In(ids...)).Update(ctx, r.db.Keel(ctx), sm.Status.To(req.Status))
		if err != nil {
			return nil, fmt.Errorf("bulk update shipment move status: %w", err)
		}
		if n != int64(len(ids)) {
			return nil, dberror.HandleNotFoundError(keel.ErrNotFound, "Shipment move")
		}

		moves, err := r.movesWithDetails(ctx, ids)
		if err != nil {
			return nil, err
		}

		result := make([]*shipment.ShipmentMove, 0, len(req.MoveIDs))
		for _, moveID := range req.MoveIDs {
			move, ok := moves[moveID]
			if !ok {
				return nil, dberror.HandleNotFoundError(keel.ErrNotFound, "Shipment move")
			}
			result = append(result, move)
		}

		return result, nil
	})
}

func (r *repository) SplitMove(
	ctx context.Context,
	req *shipmentports.SplitMoveRequest,
) (*shipmentports.SplitMoveResponse, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*shipmentports.SplitMoveResponse, error) {
			originalMove, err := r.GetByID(ctx, &shipmentports.GetMoveByIDRequest{
				MoveID:            req.MoveID,
				TenantInfo:        req.TenantInfo,
				ExpandMoveDetails: true,
			})
			if err != nil {
				return nil, err
			}

			split := shipment.PlanMoveSplit(originalMove, req.Spec())

			if err = r.shiftSubsequentMoveSequences(ctx, originalMove); err != nil {
				return nil, err
			}

			if err = r.updateRelayStop(ctx, split.RelayStop); err != nil {
				return nil, err
			}

			if err = r.insertSplitMove(ctx, split.NewMove); err != nil {
				return nil, err
			}

			moves, err := r.movesWithDetails(ctx, []pulid.ID{originalMove.ID, split.NewMove.ID})
			if err != nil {
				return nil, err
			}

			response := &shipmentports.SplitMoveResponse{
				OriginalMove: moves[originalMove.ID],
				NewMove:      moves[split.NewMove.ID],
			}
			if response.OriginalMove == nil || response.NewMove == nil {
				return nil, dberror.HandleNotFoundError(keel.ErrNotFound, "Shipment move")
			}

			return response, nil
		},
	)
}

func (r *repository) movesWithDetails(
	ctx context.Context,
	ids []pulid.ID,
) (map[pulid.ID]*shipment.ShipmentMove, error) {
	sm := shipmentdb.ShipmentMoves
	moves, err := sm.Where(sm.ID.In(ids...)).
		Load(stopsWithLocation()).
		Limit(len(ids)).
		List(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, fmt.Errorf("load shipment moves: %w", err)
	}

	if err = r.hydrateActiveAssignments(ctx, moves); err != nil {
		return nil, err
	}

	result := make(map[pulid.ID]*shipment.ShipmentMove, len(moves))
	for _, move := range moves {
		result[move.ID] = move
	}

	return result, nil
}

func uniqueIDs(ids []pulid.ID) []pulid.ID {
	seen := make(map[pulid.ID]struct{}, len(ids))
	out := make([]pulid.ID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}

	return out
}

func (r *repository) getExistingMoves(
	ctx context.Context,
	entity *shipment.Shipment,
) (map[pulid.ID]*shipment.ShipmentMove, error) {
	sm := shipmentdb.ShipmentMoves
	moves, err := sm.Where(sm.ShipmentID.Eq(entity.ID)).All(ctx, r.db.Keel(ctx))
	if err != nil {
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
	existingMoves map[pulid.ID]*shipment.ShipmentMove,
) (map[pulid.ID]map[pulid.ID]*shipment.Stop, error) {
	result := make(map[pulid.ID]map[pulid.ID]*shipment.Stop, len(existingMoves))
	if len(existingMoves) == 0 {
		return result, nil
	}

	moveIDs := make([]pulid.ID, 0, len(existingMoves))
	for moveID := range existingMoves {
		moveIDs = append(moveIDs, moveID)
	}

	st := shipmentdb.Stops
	stops, err := st.Where(st.ShipmentMoveID.In(moveIDs...)).All(ctx, r.db.Keel(ctx))
	if err != nil {
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

func normalizeMove(entity *shipment.Shipment, move *shipment.ShipmentMove, sequence int64) {
	move.ShipmentID = entity.ID
	move.OrganizationID = entity.OrganizationID
	move.BusinessUnitID = entity.BusinessUnitID
	move.Sequence = sequence
}

func (r *repository) insertMove(ctx context.Context, move *shipment.ShipmentMove) error {
	row := &shipment.ShipmentMove{
		ID:             move.ID,
		BusinessUnitID: move.BusinessUnitID,
		OrganizationID: move.OrganizationID,
		ShipmentID:     move.ShipmentID,
		Status:         move.Status,
		CoverageType:   move.CoverageType,
		Loaded:         move.Loaded,
		Sequence:       move.Sequence,
		Distance:       move.Distance,
		Version:        move.Version,
	}
	if err := shipmentdb.ShipmentMoves.Create(ctx, r.db.Keel(ctx), row); err != nil {
		return fmt.Errorf(
			"insert shipment move %s: %w",
			move.ID,
			moveErrors.Map(err, move.ID.String()),
		)
	}

	move.CoverageType = row.CoverageType
	move.CreatedAt = row.CreatedAt
	move.UpdatedAt = row.UpdatedAt

	return nil
}

func (r *repository) updateMove(
	ctx context.Context,
	move *shipment.ShipmentMove,
	existingMove *shipment.ShipmentMove,
) error {
	if move.CoverageType == "" {
		move.CoverageType = existingMove.CoverageType
	}
	if move.CoverageType == "" {
		move.CoverageType = shipment.MoveCoverageTypeUnassigned
	}
	move.Version = existingMove.Version

	sm := shipmentdb.ShipmentMoves
	if err := sm.Save(
		ctx,
		r.db.Keel(ctx),
		move,
		sm.Status,
		sm.CoverageType,
		sm.Loaded,
		sm.Sequence,
		sm.Distance,
	); err != nil {
		return fmt.Errorf(
			"update shipment move %s: %w",
			move.ID,
			moveErrors.Map(err, move.ID.String()),
		)
	}

	return nil
}

func (r *repository) syncStopsForMove(
	ctx context.Context,
	move *shipment.ShipmentMove,
	existingStops map[pulid.ID]*shipment.Stop,
) error {
	updatedStopIDs := make(map[pulid.ID]struct{}, len(move.Stops))
	inserts := make([]*shipment.Stop, 0, len(move.Stops))
	stopSequence := int64(0)

	for _, stop := range move.Stops {
		if stop == nil {
			continue
		}

		normalizeStop(move, stop, stopSequence)
		stopSequence++

		if stop.ID.IsNil() {
			stop.ID = pulid.MustNew("stp_")
			inserts = append(inserts, stop)
			updatedStopIDs[stop.ID] = struct{}{}
		} else if existingStop, ok := existingStops[stop.ID]; ok {
			if err := r.updateStop(ctx, stop, existingStop); err != nil {
				return err
			}
			updatedStopIDs[stop.ID] = struct{}{}
		} else {
			return errortypes.NewBusinessError("Shipment move contains an unknown stop").
				WithParam("moveId", move.ID.String()).
				WithParam("stopId", stop.ID.String())
		}
	}

	if len(inserts) > 0 {
		if err := shipmentdb.Stops.CreateMany(ctx, r.db.Keel(ctx), inserts); err != nil {
			return fmt.Errorf(
				"insert shipment stops for move %s: %w",
				move.ID,
				stopErrors.Map(err, inserts[0].ID.String()),
			)
		}
	}

	deleteIDs := make([]pulid.ID, 0, len(existingStops))
	for stopID := range existingStops {
		if _, ok := updatedStopIDs[stopID]; ok {
			continue
		}
		if stopInPayload(move, stopID) {
			continue
		}
		deleteIDs = append(deleteIDs, stopID)
	}

	if len(deleteIDs) == 0 {
		return nil
	}

	st := shipmentdb.Stops
	if _, err := st.Where(st.ID.In(deleteIDs...), st.ShipmentMoveID.Eq(move.ID)).
		Delete(ctx, r.db.Keel(ctx)); err != nil {
		return fmt.Errorf("delete shipment stops for move %s: %w", move.ID, err)
	}

	return nil
}

func (r *repository) replaceJurisdictionMilesForMove(
	ctx context.Context,
	move *shipment.ShipmentMove,
) error {
	jm := shipmentdb.ShipmentMoveJurisdictionMiles
	if _, err := jm.Where(jm.ShipmentMoveID.Eq(move.ID)).Delete(ctx, r.db.Keel(ctx)); err != nil {
		return fmt.Errorf("delete jurisdiction miles for move %s: %w", move.ID, err)
	}

	rows := make([]*shipment.ShipmentMoveJurisdictionMile, 0, len(move.JurisdictionMiles))
	for _, row := range move.JurisdictionMiles {
		if row == nil {
			continue
		}
		row.OrganizationID = move.OrganizationID
		row.BusinessUnitID = move.BusinessUnitID
		row.ShipmentMoveID = move.ID
		row.ShipmentID = move.ShipmentID
		rows = append(rows, row)
	}
	if len(rows) > 0 {
		if err := jm.CreateMany(ctx, r.db.Keel(ctx), rows); err != nil {
			return fmt.Errorf("insert jurisdiction miles for move %s: %w", move.ID, err)
		}
	}

	move.JurisdictionMiles = rows
	move.JurisdictionMilesDirty = false
	return nil
}

func normalizeStop(move *shipment.ShipmentMove, stop *shipment.Stop, sequence int64) {
	stop.ShipmentMoveID = move.ID
	stop.OrganizationID = move.OrganizationID
	stop.BusinessUnitID = move.BusinessUnitID
	stop.Sequence = sequence
}

func (r *repository) updateStop(
	ctx context.Context,
	stop *shipment.Stop,
	existingStop *shipment.Stop,
) error {
	if stop.ScheduleType == "" {
		stop.ScheduleType = shipment.StopScheduleTypeOpen
	}
	stop.Version = existingStop.Version

	st := shipmentdb.Stops
	if err := st.Save(
		ctx,
		r.db.Keel(ctx),
		stop,
		st.LocationID,
		st.Status,
		st.Type,
		st.ScheduleType,
		st.Sequence,
		st.Pieces,
		st.Weight,
		st.ScheduledWindowStart,
		st.ScheduledWindowEnd,
		st.ActualArrival,
		st.ActualDeparture,
		st.CountLateOverride,
		st.CountDetentionOverride,
		st.AddressLine,
	); err != nil {
		return fmt.Errorf(
			"update shipment stop %s: %w",
			stop.ID,
			stopErrors.Map(err, stop.ID.String()),
		)
	}

	return nil
}

func (r *repository) ensureMovesAreUnassigned(ctx context.Context, moveIDs []pulid.ID) error {
	a := shipmentdb.Assignments
	assigned, err := a.Where(a.ShipmentMoveID.In(moveIDs...), a.ArchivedAt.IsNull()).
		Exists(ctx, r.db.Keel(ctx))
	if err != nil {
		return fmt.Errorf("check move assignments: %w", err)
	}

	if !assigned {
		return nil
	}

	return errortypes.NewBusinessError("Cannot remove moves that already have assignments")
}

func moveInPayload(entity *shipment.Shipment, moveID pulid.ID) bool {
	for _, move := range entity.Moves {
		if move != nil && move.ID == moveID {
			return true
		}
	}

	return false
}

func stopInPayload(move *shipment.ShipmentMove, stopID pulid.ID) bool {
	for _, stop := range move.Stops {
		if stop != nil && stop.ID == stopID {
			return true
		}
	}

	return false
}

func (r *repository) shiftSubsequentMoveSequences(
	ctx context.Context,
	originalMove *shipment.ShipmentMove,
) error {
	sm := shipmentdb.ShipmentMoves
	_, err := sm.Where(
		sm.ShipmentID.Eq(originalMove.ShipmentID),
		sm.Sequence.Gt(originalMove.Sequence),
	).Update(ctx, r.db.Keel(ctx), sm.Sequence.Add(1))
	if err != nil {
		return fmt.Errorf("shift subsequent move sequences: %w", err)
	}

	return nil
}

func (r *repository) updateRelayStop(ctx context.Context, relay *shipment.Stop) error {
	st := shipmentdb.Stops
	deliveryStop := *relay
	if err := st.Save(ctx, r.db.Keel(ctx), &deliveryStop, st.Status, st.Type); err != nil {
		return fmt.Errorf(
			"update original split-delivery stop %s: %w",
			deliveryStop.ID,
			stopErrors.Map(err, deliveryStop.ID.String()),
		)
	}

	return nil
}

func (r *repository) insertSplitMove(ctx context.Context, newMove *shipment.ShipmentMove) error {
	stops := newMove.Stops
	newMove.Stops = nil
	if err := shipmentdb.ShipmentMoves.Create(ctx, r.db.Keel(ctx), newMove); err != nil {
		return fmt.Errorf(
			"insert split move %s: %w",
			newMove.ID,
			moveErrors.Map(err, newMove.ID.String()),
		)
	}

	if err := shipmentdb.Stops.CreateMany(ctx, r.db.Keel(ctx), stops); err != nil {
		return fmt.Errorf(
			"insert split move stops for move %s: %w",
			newMove.ID,
			stopErrors.Map(err, newMove.ID.String()),
		)
	}
	newMove.Stops = stops

	return nil
}
