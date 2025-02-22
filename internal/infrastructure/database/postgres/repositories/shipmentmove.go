package repositories

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ShipmentMoveRepositoryParams struct {
	fx.In

	DB             db.Connection
	StopRepository repositories.StopRepository
	Logger         *logger.Logger
}

type shipmentMoveRepository struct {
	db             db.Connection
	StopRepository repositories.StopRepository
	l              *zerolog.Logger
}

func NewShipmentMoveRepository(p ShipmentMoveRepositoryParams) repositories.ShipmentMoveRepository {
	log := p.Logger.With().
		Str("repository", "shipmentmove").
		Logger()

	return &shipmentMoveRepository{
		db:             p.DB,
		StopRepository: p.StopRepository,
		l:              &log,
	}
}

func (sr *shipmentMoveRepository) GetByID(ctx context.Context, opts repositories.GetMoveByIDOptions) (*shipment.ShipmentMove, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "GetByID").
		Str("moveID", opts.MoveID.String()).
		Str("orgID", opts.OrgID.String()).
		Str("buID", opts.BuID.String()).
		Logger()

	move := new(shipment.ShipmentMove)

	q := dba.NewSelect().Model(move).
		Where("sm.id = ?", opts.MoveID).
		Where("sm.organization_id = ?", opts.OrgID).
		Where("sm.business_unit_id = ?", opts.BuID)

	if opts.ExpandMoveDetails {
		q.Relation("Stops")
	}

	if err = q.Scan(ctx); err != nil {
		log.Error().Err(err).
			Interface("move", move).
			Msg("failed to get move by id")
		return nil, err
	}

	return move, nil
}

func (sr *shipmentMoveRepository) BulkUpdateStatus(ctx context.Context, req repositories.BulkUpdateMoveStatusRequest) ([]*shipment.ShipmentMove, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "BulkUpdateStatus").
		Interface("moveIDs", req.MoveIDs).
		Str("status", string(req.Status)).
		Logger()

	moves := make([]*shipment.ShipmentMove, len(req.MoveIDs))
	results, err := dba.NewUpdate().
		Model(&moves).
		Column("status").
		Column("updated_at").
		Set("status = ?", req.Status).
		Set("updated_at = ?", timeutils.NowUnix()). // Explicity set the updated_at to the current time
		Where("sm.id IN (?)", bun.In(req.MoveIDs)).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to bulk update move status")
		return nil, err
	}

	rows, err := results.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return nil, err
	}

	if rows != int64(len(req.MoveIDs)) {
		return nil, errors.NewValidationError(
			"move.status",
			errors.ErrVersionMismatch,
			fmt.Sprintf("Version mismatch. The move (%s) has been updated since your last request.", moves[0].ID),
		)
	}

	return moves, nil
}

func (sr *shipmentMoveRepository) UpdateStatus(ctx context.Context, opts *repositories.UpdateMoveStatusRequest) (*shipment.ShipmentMove, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "UpdateStatus").
		Str("moveID", opts.GetMoveOpts.MoveID.String()).
		Str("status", string(opts.Status)).
		Logger()

	// Get the move
	move, err := sr.GetByID(ctx, opts.GetMoveOpts)
	if err != nil {
		return nil, err
	}

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// Update the move version
		ov := move.Version
		move.Version++

		results, rErr := tx.NewUpdate().Model(move).
			WherePK().
			Where("sm.version = ?", ov).
			Set("status = ?", opts.Status).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().Err(rErr).
				Interface("move", move).
				Msg("failed to update move version")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).
				Interface("move", move).
				Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			return errors.NewValidationError(
				"move.version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The move (%s) has been updated since your last request.", move.ID),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).
			Interface("move", move).
			Msg("failed to update move status")
		return nil, err
	}

	return move, nil
}

func (sr *shipmentMoveRepository) GetMovesByShipmentID(ctx context.Context, opts repositories.GetMovesByShipmentIDOptions) ([]*shipment.ShipmentMove, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "GetMovesByShipmentID").
		Str("shipmentID", opts.ShipmentID.String()).
		Logger()

	moves := make([]*shipment.ShipmentMove, 0)

	q := dba.NewSelect().Model(&moves).
		Where("sm.shipment_id = ?", opts.ShipmentID).
		Where("sm.organization_id = ?", opts.OrgID).
		Where("sm.business_unit_id = ?", opts.BuID)

	if err = q.Scan(ctx); err != nil {
		log.Error().Err(err).Msg("failed to get moves by shipment id")
		return nil, err
	}

	return moves, nil
}

func (sr *shipmentMoveRepository) BulkInsert(ctx context.Context, moves []*shipment.ShipmentMove) ([]*shipment.ShipmentMove, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "BulkInsert").
		Interface("moves", moves).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, err = tx.NewInsert().Model(&moves).Exec(c); err != nil {
			log.Error().Err(err).Msg("failed to bulk insert moves")
			return err
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to bulk insert moves and stops")
		return nil, err
	}

	return moves, nil
}

func (sr *shipmentMoveRepository) SplitMove(ctx context.Context, req *repositories.SplitMoveRequest) (*repositories.SplitMoveResponse, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "SplitMove").
		Str("moveID", req.MoveID.String()).
		Logger()

	// Get the original move with its stops
	originalMove, err := sr.GetByID(ctx, repositories.GetMoveByIDOptions{
		MoveID:            req.MoveID,
		OrgID:             req.OrgID,
		BuID:              req.BuID,
		ExpandMoveDetails: true,
	})
	if err != nil {
		return nil, err
	}

	var newMove *shipment.ShipmentMove
	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// First, get all moves for this shipment with sequence > originalMove.Sequence
		var moves []*shipment.ShipmentMove
		err = tx.NewSelect().
			Model(&moves).
			Where("shipment_id = ? AND sequence > ?", originalMove.ShipmentID, originalMove.Sequence).
			Order("sequence DESC").
			Scan(c)
		if err != nil {
			sr.l.Error().
				Err(err).
				Str("moveID", originalMove.GetID()).
				Int("sequence", originalMove.Sequence).
				Msg("failed to get moves with sequence greater than original move")
			return err
		}

		// Update sequences for existing moves, starting from the highest sequence
		for _, move := range moves {
			move.Sequence++
			if _, err = tx.NewUpdate().Model(move).
				Set("sequence = ?", move.Sequence).
				Set("version = version + 1").
				WherePK().
				Exec(c); err != nil {
				sr.l.Error().
					Err(err).
					Str("moveID", move.GetID()).
					Int("sequence", move.Sequence).
					Msg("failed to update move sequence")
				return err
			}
		}

		// Delete the original delivery stop
		_, err = tx.NewDelete().Model((*shipment.Stop)(nil)).
			Where("shipment_move_id = ? AND sequence = ?", originalMove.ID, 1).
			Exec(c)
		if err != nil {
			sr.l.Error().
				Err(err).
				Str("moveID", originalMove.GetID()).
				Msg("failed to delte the original delivery stop from the original move")
			return err
		}

		// Create split delivery stop for the original move
		splitDeliveryStop := &shipment.Stop{
			ID:               pulid.MustNew("stp_"),
			BusinessUnitID:   originalMove.BusinessUnitID,
			OrganizationID:   originalMove.OrganizationID,
			ShipmentMoveID:   originalMove.ID, // Keep it on original move
			LocationID:       req.SplitLocationID,
			Status:           shipment.StopStatusNew,
			Type:             shipment.StopTypeSplitDelivery,
			Sequence:         1,
			Pieces:           req.SplitQuantities.Pieces,
			Weight:           req.SplitQuantities.Weight,
			PlannedArrival:   req.SplitDeliveryTimes.PlannedArrival,
			PlannedDeparture: req.SplitDeliveryTimes.PlannedDeparture,
		}

		// Insert the split delivery stop
		if _, err = tx.NewInsert().Model(splitDeliveryStop).Exec(c); err != nil {
			sr.l.Error().
				Err(err).
				Str("moveID", originalMove.GetID()).
				Interface("splitDeliveryStop", splitDeliveryStop).
				Msg("failed to insert the split delivery stop")
			return err
		}

		// Create new move with sequence 1
		newMove = &shipment.ShipmentMove{
			ID:             pulid.MustNew("smv_"),
			BusinessUnitID: originalMove.BusinessUnitID,
			OrganizationID: originalMove.OrganizationID,
			ShipmentID:     originalMove.ShipmentID,
			Status:         shipment.MoveStatusNew,
			Loaded:         true,
			Sequence:       1, // Explicitly set to 1
			Distance:       originalMove.Distance,
		}

		// Insert the new move
		if _, err = tx.NewInsert().Model(newMove).Exec(c); err != nil {
			sr.l.Error().
				Err(err).
				Str("moveID", originalMove.GetID()).
				Interface("newMove", newMove).
				Msg("failed to insert the new move")
			return err
		}

		// Create stops for new move
		newMoveStops := []*shipment.Stop{
			{
				// Split Pickup
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
				// Final Delivery
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

		// Insert the stops for new move
		if _, err = tx.NewInsert().Model(&newMoveStops).Exec(c); err != nil {
			sr.l.Error().
				Err(err).
				Str("moveID", originalMove.GetID()).
				Interface("newMoveStops", newMoveStops).
				Msg("failed to insert the stops for the new move")
			return err
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to split move")
		return nil, err
	}

	// Fetch updated moves for response
	originalMove, err = sr.GetByID(ctx, repositories.GetMoveByIDOptions{
		MoveID:            originalMove.ID,
		OrgID:             req.OrgID,
		BuID:              req.BuID,
		ExpandMoveDetails: true,
	})
	if err != nil {
		return nil, err
	}

	newMove, err = sr.GetByID(ctx, repositories.GetMoveByIDOptions{
		MoveID:            newMove.ID,
		OrgID:             req.OrgID,
		BuID:              req.BuID,
		ExpandMoveDetails: true,
	})
	if err != nil {
		return nil, err
	}

	result := &repositories.SplitMoveResponse{
		OriginalMove: originalMove,
		NewMove:      newMove,
	}

	return result, nil
}

func (sr *shipmentMoveRepository) HandleMoveOperations(ctx context.Context, tx bun.IDB, shp *shipment.Shipment, isCreate bool) error {
	var err error

	// * Get the existing moves for comparison if this is an update operation
	existingMoves := make([]*shipment.ShipmentMove, 0)
	if !isCreate {
		existingMoves, err = sr.GetMovesByShipmentID(ctx, repositories.GetMovesByShipmentIDOptions{
			ShipmentID: shp.ID,
			OrgID:      shp.OrganizationID,
			BuID:       shp.BusinessUnitID,
		})
		if err != nil {
			sr.l.Error().Err(err).Msg("failed to get existing moves")
			return err
		}
	}

	// * Prepare moves for operations
	newMoves := make([]*shipment.ShipmentMove, 0)
	updateMoves := make([]*shipment.ShipmentMove, 0)
	existingMoveMap := make(map[pulid.ID]*shipment.ShipmentMove)
	updatedMoveIDs := make(map[pulid.ID]struct{})

	// * Create map of existing moves for quick lookup
	for _, move := range existingMoves {
		sr.l.Debug().Interface("move", move).Msg("existing move")
		existingMoveMap[move.ID] = move
	}

	// * Categorize moves for different operations
	for _, move := range shp.Moves {
		// * Set required fields
		move.ShipmentID = shp.ID
		move.OrganizationID = shp.OrganizationID
		move.BusinessUnitID = shp.BusinessUnitID

		if isCreate || move.ID.IsNil() {
			// * We need to set an ID new moves, because it will have stops that need to be created
			move.ID = pulid.MustNew("smv_")

			// * Append new move with an ID
			newMoves = append(newMoves, move)
		} else {
			if existing, ok := existingMoveMap[move.ID]; ok {
				// * Increment version for optimistic locking
				move.Version = existing.Version + 1
				updateMoves = append(updateMoves, move)
				updatedMoveIDs[move.ID] = struct{}{}
			}
		}
	}

	// * Handle bulk insert of new moves
	if len(newMoves) > 0 {
		if _, err := tx.NewInsert().Model(newMoves).Exec(ctx); err != nil {
			sr.l.Error().Err(err).Msg("failed to bulk insert new moves")
			return err
		}

		for _, move := range newMoves {
			sr.l.Debug().Interface("move", move).Msg("new move")
			for _, stop := range move.Stops {
				sr.l.Debug().Interface("stop", stop).Msg("stop")
			}
		}

		// TODO We also need to make sure we handle the new stops that are created with the new moves
	}

	// * Handle bulk update of new moves
	if len(updateMoves) > 0 {
		for moveIdx, move := range updateMoves {
			if err := sr.handleUpdate(ctx, tx, move, moveIdx); err != nil {
				sr.l.Error().Err(err).Msg("failed to handle bulk update of moves")
				return err
			}

			// * Handle bulk update of stops
			for stopIdx, stop := range move.Stops {
				if _, err := sr.StopRepository.Update(ctx, stop, moveIdx, stopIdx); err != nil {
					// sr.l.Error().Err(err).Msg("failed to update stop")
					return err
				}
			}
		}
	}

	// ! Moves cannot be deleted the user needs to set the status to `Canceled`

	return nil
}

func (sr *shipmentMoveRepository) handleUpdate(ctx context.Context, tx bun.IDB, move *shipment.ShipmentMove, idx int) error {
	values := tx.NewValues(move)

	// * Update the moves
	res, err := tx.NewUpdate().With("_data", values).
		Model(move).
		TableExpr("_data").
		Set("shipment_id = _data.shipment_id").
		Set("status = _data.status").
		Set("loaded = _data.loaded").
		Set("sequence = _data.sequence").
		Set("distance = _data.distance").
		Set("version = _data.version").
		Where("sm.id = _data.id").
		Where("sm.version = _data.version - 1").
		Where("sm.organization_id = _data.organization_id").
		Where("sm.business_unit_id = _data.business_unit_id").
		Exec(ctx)
	if err != nil {
		sr.l.Error().Err(err).Msg("failed to bulk update moves")
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		sr.l.Error().Err(err).Msg("failed to get rows affected for bulk update of moves")
		return err
	}

	if rowsAffected == 0 {
		return errors.NewValidationError(
			fmt.Sprintf("move[%d].version", idx),
			errors.ErrVersionMismatch,
			fmt.Sprintf("Version mismatch. The move (%s) has been updated since your last request.", move.ID),
		)
	}

	sr.l.Debug().Int("count", int(rowsAffected)).Msg("bulk updated moves")

	return nil
}
