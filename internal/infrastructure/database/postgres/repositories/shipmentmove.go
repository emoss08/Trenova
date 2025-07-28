/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"
	"database/sql"
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

// ShipmentMoveRepositoryParams defines dependencies required for initializing the ShipmentMoveRepository.
// This includes database connection, stop repository, shipment control repository, and logger.
type ShipmentMoveRepositoryParams struct {
	fx.In

	DB                        db.Connection
	StopRepository            repositories.StopRepository
	ShipmentControlRepository repositories.ShipmentControlRepository
	Logger                    *logger.Logger
}

// shipmentMoveRepository implements the ShipmentMoveRepository interface
// and provides methods to manage moves, including CRUD operations, status updates,
// and bulk operations.
type shipmentMoveRepository struct {
	db   db.Connection
	stpr repositories.StopRepository
	scr  repositories.ShipmentControlRepository
	l    *zerolog.Logger
}

// NewShipmentMoveRepository initializes a new instance of shipmentMoveRepository with its dependencies.
//
// Parameters:
// - p: ShipmentMoveRepositoryParams containing database connection, stop repository, shipment control repository, and logger.
//
// Returns:
// - A new instance of shipmentMoveRepository.
func NewShipmentMoveRepository(p ShipmentMoveRepositoryParams) repositories.ShipmentMoveRepository {
	log := p.Logger.With().
		Str("repository", "shipmentmove").
		Logger()

	return &shipmentMoveRepository{
		db:   p.DB,
		stpr: p.StopRepository,
		scr:  p.ShipmentControlRepository,
		l:    &log,
	}
}

// GetByID retrieves a shipment by its unique ID, including optional expanded details
//
// Parameters:
// - ctx: The context for the operation.
// - opts: GetMoveByIDOptions containing move ID and organization ID.
//
// Returns:
//   - *shipment.ShipmentMove: The shipment move if found
//   - error: If any database operation fails.
func (sr *shipmentMoveRepository) GetByID(
	ctx context.Context,
	opts repositories.GetMoveByIDOptions,
) (*shipment.ShipmentMove, error) {
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

		// * Expand the assignment details
		q.Relation("Assignment").
			Relation("Assignment.Tractor").
			Relation("Assignment.Trailer").
			Relation("Assignment.PrimaryWorker").
			Relation("Assignment.SecondaryWorker")
	}

	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Err(err).Msg("failed to get shipment move")
			return nil, errors.NewNotFoundError("Shipment move not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get move")
		return nil, eris.Wrap(err, "get move by id")
	}

	return move, nil
}

// BulkUpdateStatus updates the status of multiple shipment moves in a single database transaction.
//
// Parameters:
// - ctx: The context for the operation.
// - req: BulkUpdateMoveStatusRequest containing move IDs and status.
//
// Returns:
//   - []*shipment.ShipmentMove: The updated shipment moves
//   - error: If any database operation fails.
func (sr *shipmentMoveRepository) BulkUpdateStatus(
	ctx context.Context,
	req repositories.BulkUpdateMoveStatusRequest,
) ([]*shipment.ShipmentMove, error) {
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
		Set("updated_at = ?", timeutils.NowUnix()).
		// Explicity set the updated_at to the current time
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
			fmt.Sprintf(
				"Version mismatch. The move (%s) has been updated since your last request.",
				moves[0].ID,
			),
		)
	}

	return moves, nil
}

// UpdateStatus updates the status of a shipment move
//
// Parameters:
// - ctx: The context for the operation.
// - opts: UpdateMoveStatusRequest containing move ID and status.
//
// Returns:
//   - *shipment.ShipmentMove: The updated shipment move
//   - error: If any database operation fails.
func (sr *shipmentMoveRepository) UpdateStatus(
	ctx context.Context,
	opts *repositories.UpdateMoveStatusRequest,
) (*shipment.ShipmentMove, error) {
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
		log.Error().Err(err).Msg("failed to get move")
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
				fmt.Sprintf(
					"Version mismatch. The move (%s) has been updated since your last request.",
					move.ID,
				),
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

// GetMovesByShipmentID retrieves all shipment moves for a given shipment ID.
//
// Parameters:
// - ctx: The context for the operation.
// - opts: GetMovesByShipmentIDOptions containing shipment ID, organization ID, and business unit ID.
//
// Returns:
//   - []*shipment.ShipmentMove: The shipment moves
//   - error: If any database operation fails.
func (sr *shipmentMoveRepository) GetMovesByShipmentID(
	ctx context.Context,
	opts repositories.GetMovesByShipmentIDOptions,
) ([]*shipment.ShipmentMove, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "GetMovesByShipmentID").
		Str("shipmentID", opts.ShipmentID.String()).
		Logger()

	moves := make([]*shipment.ShipmentMove, 0)

	// * Craft the query using a where group to ensure all conditions are met
	q := dba.NewSelect().Model(&moves).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("sm.shipment_id = ?", opts.ShipmentID).
				Where("sm.organization_id = ?", opts.OrgID).
				Where("sm.business_unit_id = ?", opts.BuID)
		})

	// * Execute the query
	if err = q.Scan(ctx); err != nil {
		// * If the query is [sql.ErrNoRows], return a not found error
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Err(err).Msg("failed to get moves by shipment id")
			return nil, errors.NewNotFoundError("Moves not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get moves by shipment id")
		return nil, eris.Wrap(err, "get moves by shipment id")
	}

	return moves, nil
}

// BulkInsert inserts multiple shipment moves in a single database transaction.
//
// Parameters:
// - ctx: The context for the operation.
// - moves: The shipment moves to insert.
//
// Returns:
//   - []*shipment.ShipmentMove: The inserted shipment moves
//   - error: If any database operation fails.
func (sr *shipmentMoveRepository) BulkInsert(
	ctx context.Context,
	moves []*shipment.ShipmentMove,
) ([]*shipment.ShipmentMove, error) {
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
		return nil, eris.Wrap(err, "bulk insert moves")
	}

	return moves, nil
}

// SplitMove splits a shipment move into two new moves.
//
// Parameters:
// - ctx: The context for the operation.
// - req: SplitMoveRequest containing move ID, organization ID, business unit ID, split location ID, split quantities, and split delivery times.
//
// Returns:
//   - *SplitMoveResponse: The response containing the original and new moves
//   - error: If any database operation fails.
func (sr *shipmentMoveRepository) SplitMove(
	ctx context.Context,
	req *repositories.SplitMoveRequest,
) (*repositories.SplitMoveResponse, error) {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "SplitMove").
		Str("moveID", req.MoveID.String()).
		Logger()

	//  * Get the original move with its stops
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
		// Update sequences for subsequent moves
		if err = sr.updateMoveSequences(c, tx, originalMove); err != nil {
			return err
		}

		// Handle original move modifications
		if err = sr.modifyOriginalMove(c, tx, originalMove, req); err != nil {
			return err
		}

		// Create new split move
		newMove, err = sr.createSplitMove(c, tx, originalMove, req)
		return err
	})
	if err != nil {
		log.Error().Err(err).Interface("originalMove", originalMove).
			Interface("newMove", newMove).
			Msg("failed to split move")
		return nil, eris.Wrap(err, "split move")
	}

	// Fetch updated moves for response
	return sr.prepareSplitMoveResponse(ctx, req, originalMove, newMove)
}

// updateMoveSequences updates the sequence numbers of moves after the original move
func (sr *shipmentMoveRepository) updateMoveSequences(
	ctx context.Context,
	tx bun.Tx,
	originalMove *shipment.ShipmentMove,
) error {
	log := sr.l.With().
		Str("operation", "updateMoveSequences").
		Str("moveID", originalMove.GetID()).
		Logger()

	// Get all moves for this shipment with sequence > originalMove.Sequence
	var moves []*shipment.ShipmentMove
	err := tx.NewSelect().
		Model(&moves).
		Where("shipment_id = ? AND sequence > ?", originalMove.ShipmentID, originalMove.Sequence).
		Order("sequence DESC").
		Scan(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Err(err).Msg("failed to get moves with sequence greater than original move")
			return errors.NewNotFoundError("Moves not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get moves with sequence greater than original move")
		return eris.Wrap(err, "get moves with sequence greater than original move")
	}

	// Update sequences for existing moves, starting from the highest sequence
	for _, move := range moves {
		move.Sequence++
		if _, err = tx.NewUpdate().Model(move).
			Set("sequence = ?", move.Sequence).
			Set("version = version + 1").
			WherePK().
			Exec(ctx); err != nil {
			log.Error().Err(err).
				Str("moveID", move.GetID()).
				Int("sequence", move.Sequence).
				Msg("failed to update move sequence")
			return err
		}
	}

	return nil
}

// modifyOriginalMove removes the original delivery stop and adds a split delivery stop
func (sr *shipmentMoveRepository) modifyOriginalMove(
	ctx context.Context,
	tx bun.Tx,
	originalMove *shipment.ShipmentMove,
	req *repositories.SplitMoveRequest,
) error {
	log := sr.l.With().
		Str("operation", "modifyOriginalMove").
		Str("moveID", originalMove.GetID()).
		Logger()

	// Delete the original delivery stop
	_, err := tx.NewDelete().Model((*shipment.Stop)(nil)).
		Where("shipment_move_id = ? AND sequence = ?", originalMove.ID, 1).
		Exec(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().
				Err(err).
				Msg("failed to delete the original delivery stop from the original move")
			return errors.NewNotFoundError(
				"Original delivery stop not found within your organization",
			)
		}

		log.Error().
			Err(err).
			Msg("failed to delete the original delivery stop from the original move")
		return eris.Wrap(err, "delete original delivery stop")
	}

	// Create and insert split delivery stop for the original move
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
		log.Error().Err(err).
			Str("moveID", originalMove.GetID()).
			Interface("splitDeliveryStop", splitDeliveryStop).
			Msg("failed to insert the split delivery stop")
		return eris.Wrap(err, "insert split delivery stop")
	}

	return nil
}

// createSplitMove creates a new move with split pickup and final delivery stops
func (sr *shipmentMoveRepository) createSplitMove(
	ctx context.Context,
	tx bun.Tx,
	originalMove *shipment.ShipmentMove,
	req *repositories.SplitMoveRequest,
) (*shipment.ShipmentMove, error) {
	log := sr.l.With().
		Str("operation", "createSplitMove").
		Str("moveID", originalMove.GetID()).
		Logger()

	// Create new move with sequence 1
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

	// Insert the new move
	if _, err := tx.NewInsert().Model(newMove).Exec(ctx); err != nil {
		log.Error().Err(err).
			Str("moveID", originalMove.GetID()).
			Interface("newMove", newMove).
			Msg("failed to insert the new move")
		return nil, eris.Wrap(err, "insert new move")
	}

	// Create stops for new move
	newMoveStops := sr.createSplitMoveStops(newMove, originalMove, req)

	// Insert the stops for new move
	if _, err := tx.NewInsert().Model(&newMoveStops).Exec(ctx); err != nil {
		log.Error().Err(err).
			Str("moveID", originalMove.GetID()).
			Interface("newMoveStops", newMoveStops).
			Msg("failed to insert the stops for the new move")
		return nil, eris.Wrap(err, "insert new move stops")
	}

	return newMove, nil
}

// createSplitMoveStops creates the stops for the new split move
func (sr *shipmentMoveRepository) createSplitMoveStops(
	newMove, originalMove *shipment.ShipmentMove,
	req *repositories.SplitMoveRequest,
) []*shipment.Stop {
	return []*shipment.Stop{
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
}

// prepareSplitMoveResponse fetches the updated moves and prepares the response
func (sr *shipmentMoveRepository) prepareSplitMoveResponse(
	ctx context.Context,
	req *repositories.SplitMoveRequest,
	originalMove, newMove *shipment.ShipmentMove,
) (*repositories.SplitMoveResponse, error) {
	// Fetch updated original move
	updatedOriginalMove, err := sr.GetByID(ctx, repositories.GetMoveByIDOptions{
		MoveID:            originalMove.ID,
		OrgID:             req.OrgID,
		BuID:              req.BuID,
		ExpandMoveDetails: true,
	})
	if err != nil {
		return nil, err
	}

	// Fetch updated new move
	updatedNewMove, err := sr.GetByID(ctx, repositories.GetMoveByIDOptions{
		MoveID:            newMove.ID,
		OrgID:             req.OrgID,
		BuID:              req.BuID,
		ExpandMoveDetails: true,
	})
	if err != nil {
		return nil, err
	}

	return &repositories.SplitMoveResponse{
		OriginalMove: updatedOriginalMove,
		NewMove:      updatedNewMove,
	}, nil
}

// HandleMoveOperations handles the operations for a shipment move.
//
// Parameters:
// - ctx: The context for the operation.
// - tx: The database transaction.
// - shp: The shipment to operate on.
// - isCreate: Whether the operation is a create or update.
//
// Returns:
//   - error: If any database operation fails.
func (sr *shipmentMoveRepository) HandleMoveOperations(
	ctx context.Context,
	tx bun.IDB,
	shp *shipment.Shipment,
	isCreate bool,
) error {
	log := sr.l.With().
		Str("operation", "HandleMoveOperations").
		Str("shipmentID", shp.ID.String()).
		Logger()

	// Check organization settings
	scr, err := sr.scr.GetByOrgID(ctx, shp.OrganizationID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get shipment control")
		return err
	}

	// Prepare data for operations
	movesData, err := sr.prepareMovesData(ctx, shp, isCreate)
	if err != nil {
		return err
	}

	// Handle operations in sequence
	if err = sr.processNewMoves(ctx, tx, movesData.newMoves); err != nil {
		return err
	}

	if err = sr.processUpdateMoves(ctx, tx, movesData.updateMoves); err != nil {
		return err
	}

	if !isCreate {
		if err = sr.checkAndHandleMoveDeletions(ctx, tx, shp, scr, movesData); err != nil {
			return err
		}
	}

	log.Debug().Int("new_moves", len(movesData.newMoves)).
		Int("updated_moves", len(movesData.updateMoves)).
		Msg("move operations completed")

	return nil
}

// movesOperationData contains the data needed for move operations
type movesOperationData struct {
	newMoves        []*shipment.ShipmentMove
	updateMoves     []*shipment.ShipmentMove
	existingMoveMap map[pulid.ID]*shipment.ShipmentMove
	updatedMoveIDs  map[pulid.ID]struct{}
	moveToDelete    []*shipment.ShipmentMove
	existingMoves   []*shipment.ShipmentMove
}

// prepareMovesData prepares the data needed for move operations
func (sr *shipmentMoveRepository) prepareMovesData(
	ctx context.Context,
	shp *shipment.Shipment,
	isCreate bool,
) (*movesOperationData, error) {
	log := sr.l.With().
		Str("operation", "prepareMovesData").
		Str("shipmentID", shp.ID.String()).
		Logger()

	data := &movesOperationData{
		newMoves:        make([]*shipment.ShipmentMove, 0),
		updateMoves:     make([]*shipment.ShipmentMove, 0),
		existingMoveMap: make(map[pulid.ID]*shipment.ShipmentMove),
		updatedMoveIDs:  make(map[pulid.ID]struct{}),
		moveToDelete:    make([]*shipment.ShipmentMove, 0),
		existingMoves:   make([]*shipment.ShipmentMove, 0),
	}

	// Get existing moves if this is an update operation
	if !isCreate {
		var err error
		data.existingMoves, err = sr.GetMovesByShipmentID(
			ctx,
			repositories.GetMovesByShipmentIDOptions{
				ShipmentID: shp.ID,
				OrgID:      shp.OrganizationID,
				BuID:       shp.BusinessUnitID,
			},
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to get existing moves")
			return nil, err
		}

		// Create map of existing moves for quick lookup
		for _, move := range data.existingMoves {
			log.Debug().Interface("move", move).Msg("existing move")
			data.existingMoveMap[move.ID] = move
		}
	}

	sr.categorizeMoves(shp, data, isCreate)
	return data, nil
}

// categorizeMoves categorizes moves for different operations
func (sr *shipmentMoveRepository) categorizeMoves(
	shp *shipment.Shipment,
	data *movesOperationData,
	isCreate bool,
) {
	for _, move := range shp.Moves {
		// Set required fields
		move.ShipmentID = shp.ID
		move.OrganizationID = shp.OrganizationID
		move.BusinessUnitID = shp.BusinessUnitID

		if isCreate || move.ID.IsNil() {
			// Set ID for new moves
			move.ID = pulid.MustNew("smv_")
			data.newMoves = append(data.newMoves, move)
		} else {
			if existing, ok := data.existingMoveMap[move.ID]; ok {
				// Increment version for optimistic locking
				move.Version = existing.Version + 1
				data.updateMoves = append(data.updateMoves, move)
				data.updatedMoveIDs[move.ID] = struct{}{}
			}
		}
	}
}

// processNewMoves handles the insertion of new moves and their stops
func (sr *shipmentMoveRepository) processNewMoves(
	ctx context.Context,
	tx bun.IDB,
	newMoves []*shipment.ShipmentMove,
) error {
	if len(newMoves) == 0 {
		return nil
	}

	log := sr.l.With().Str("operation", "processNewMoves").Logger()

	// Insert new moves
	if _, err := tx.NewInsert().Model(&newMoves).Exec(ctx); err != nil {
		log.Error().Err(err).Msg("failed to bulk insert new moves")
		return err
	}

	// Insert stops for each new move
	for _, move := range newMoves {
		log.Debug().Interface("move", move).Msg("new move")
		if err := sr.insertStopsForMove(ctx, tx, move); err != nil {
			return err
		}
	}

	return nil
}

// insertStopsForMove inserts stops for a new move
func (sr *shipmentMoveRepository) insertStopsForMove(
	ctx context.Context,
	tx bun.IDB,
	move *shipment.ShipmentMove,
) error {
	log := sr.l.With().
		Str("operation", "insertStopsForMove").
		Str("moveID", move.ID.String()).
		Logger()

	for _, stop := range move.Stops {
		// Set required fields
		stop.ShipmentMoveID = move.ID
		stop.OrganizationID = move.OrganizationID
		stop.BusinessUnitID = move.BusinessUnitID

		// Insert the stop
		if _, err := tx.NewInsert().Model(stop).Exec(ctx); err != nil {
			log.Error().Err(err).Msg("failed to insert stop")
			return err
		}
	}
	return nil
}

// processUpdateMoves handles updates to existing moves and their stops
func (sr *shipmentMoveRepository) processUpdateMoves(
	ctx context.Context,
	tx bun.IDB,
	updateMoves []*shipment.ShipmentMove,
) error {
	if len(updateMoves) == 0 {
		return nil
	}

	log := sr.l.With().Str("operation", "processUpdateMoves").Logger()

	for moveIdx, move := range updateMoves {
		if err := sr.handleUpdate(ctx, tx, move, moveIdx); err != nil {
			log.Error().Err(err).Msg("failed to handle bulk update of moves")
			return err
		}

		if err := sr.processStopsForExistingMove(ctx, tx, move, moveIdx); err != nil {
			return err
		}
	}

	return nil
}

// processStopsForExistingMove processes stops for an existing move
func (sr *shipmentMoveRepository) processStopsForExistingMove(
	ctx context.Context,
	tx bun.IDB,
	move *shipment.ShipmentMove,
	moveIdx int,
) error {
	log := sr.l.With().
		Str("operation", "processStopsForExistingMove").
		Str("moveID", move.ID.String()).
		Logger()

	// Get existing stops for this move
	existingStops := make([]*shipment.Stop, 0)
	err := tx.NewSelect().Model(&existingStops).
		Where("shipment_move_id = ?", move.ID).
		Scan(ctx)
	if err != nil {
		log.Error().Err(err).Str("moveID", move.ID.String()).
			Msg("failed to get existing stops for move")
		return err
	}

	// Track which stops are being updated
	updatedStopIDs := make(map[pulid.ID]struct{})

	// Process each stop
	for stopIdx, stop := range move.Stops {
		// Set required fields
		stop.ShipmentMoveID = move.ID
		stop.OrganizationID = move.OrganizationID
		stop.BusinessUnitID = move.BusinessUnitID

		if stop.ID.IsNil() {
			// New stop
			stop.ID = pulid.MustNew("stp_")
			if _, err = tx.NewInsert().Model(stop).Exec(ctx); err != nil {
				log.Error().Err(err).
					Int("moveIdx", moveIdx).
					Int("stopIdx", stopIdx).
					Interface("stop", stop).
					Msg("failed to insert new stop")
				return err
			}
		} else {
			// Existing stop
			if _, err = sr.stpr.Update(ctx, stop, moveIdx, stopIdx); err != nil {
				log.Error().Err(err).
					Int("moveIdx", moveIdx).
					Int("stopIdx", stopIdx).
					Interface("stop", stop).
					Msg("failed to update stop")
				return err
			}
			updatedStopIDs[stop.ID] = struct{}{}
		}
	}

	// Handle stop removals if needed
	if len(existingStops) > 0 {
		if err = sr.stpr.HandleStopRemovals(ctx, tx, move, existingStops, updatedStopIDs); err != nil {
			log.Error().Err(err).
				Int("moveIdx", moveIdx).
				Msg("failed to handle stop removals")
			return err
		}
	}

	return nil
}

// checkAndHandleMoveDeletions checks if moves can be deleted and handles the deletion
func (sr *shipmentMoveRepository) checkAndHandleMoveDeletions(
	ctx context.Context,
	tx bun.IDB,
	shp *shipment.Shipment,
	scr *shipment.ShipmentControl,
	data *movesOperationData,
) error {
	log := sr.l.With().
		Str("operation", "checkAndHandleMoveDeletions").
		Logger()

	// Check if there are moves to delete and if organization allows it
	deletionRequired := false
	for moveID := range data.existingMoveMap {
		if _, ok := data.updatedMoveIDs[moveID]; !ok {
			deletionRequired = true

			// Check if the organization allows move removals
			if !scr.AllowMoveRemovals {
				log.Debug().
					Msgf("Organization %s does not allow move removals, returning error...", shp.OrganizationID)
				return errors.NewBusinessError(
					"Your organization does not allow move removals",
				)
			}
			break
		}
	}

	// If no deletion needed or already checked permission above
	if deletionRequired {
		if err := sr.handleMoveDeletions(ctx, tx, &repositories.HandleMoveDeletionsRequest{
			ExistingMoveMap: data.existingMoveMap,
			UpdatedMoveIDs:  data.updatedMoveIDs,
			MoveToDelete:    data.moveToDelete,
		}); err != nil {
			log.Error().Err(err).Msg("failed to handle move deletions")
			return err
		}
	}

	return nil
}

// handleUpdate handles the update of a shipment move.
//
// Parameters:
// - ctx: The context for the operation.
// - tx: The database transaction.
// - move: The move to update.
// - idx: The index of the move in the update list.
func (sr *shipmentMoveRepository) handleUpdate(
	ctx context.Context,
	tx bun.IDB,
	move *shipment.ShipmentMove,
	idx int,
) error {
	log := sr.l.With().
		Str("operation", "handleUpdate").
		Int("idx", idx).
		Interface("move", move).
		Logger()

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
		log.Error().Err(err).Msg("failed to bulk update moves")
		return err
	}

	// * Get the rows affected
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected for bulk update of moves")
		return err
	}

	if rowsAffected == 0 {
		return errors.NewValidationError(
			fmt.Sprintf("move[%d].version", idx),
			errors.ErrVersionMismatch,
			fmt.Sprintf(
				"Version mismatch. The move (%s) has been updated since your last request.",
				move.ID,
			),
		)
	}

	log.Debug().Int("count", int(rowsAffected)).Msg("bulk updated moves")

	return nil
}

// handleMoveDeletions handles the deletion of moves that are no longer present.
//
// Parameters:
// - ctx: The context for the operation.
// - tx: The database transaction.
// - req: The request containing the existing and updated moves.
func (sr *shipmentMoveRepository) handleMoveDeletions(
	ctx context.Context,
	tx bun.IDB,
	req *repositories.HandleMoveDeletionsRequest,
) error {
	log := sr.l.With().
		Str("operation", "handleMoveDeletions").
		Logger()

	// * Create a slice to hold the IDs of moves to delete
	moveIDsToDelete := make([]pulid.ID, 0)

	// * For each existing move, check if it is still present in the updated move list
	for moveID, move := range req.ExistingMoveMap {
		if _, ok := req.UpdatedMoveIDs[moveID]; !ok {
			moveIDsToDelete = append(moveIDsToDelete, moveID)
			req.MoveToDelete = append(req.MoveToDelete, move)
		}
	}

	log.Debug().
		Interface("moveIDsToDelete", moveIDsToDelete).
		Msg("moves to delete")

	// * If there are moves to delete
	if len(moveIDsToDelete) > 0 {
		// Get the shipment ID from the first move (all moves being deleted are from the same shipment)
		shipmentID := req.ExistingMoveMap[moveIDsToDelete[0]].ShipmentID

		// Delete associated data and the moves themselves
		if err := sr.deleteMovesAndAssociatedData(ctx, tx, moveIDsToDelete); err != nil {
			return err
		}

		// Resequence the remaining moves
		if err := sr.resequenceRemainingMoves(ctx, tx, shipmentID); err != nil {
			log.Error().Err(err).
				Interface("moveIDs", moveIDsToDelete).
				Msg("failed to resequence remaining moves")
			return err
		}
	}

	return nil
}

// deleteMovesAndAssociatedData deletes moves and their associated data (stops and assignments)
func (sr *shipmentMoveRepository) deleteMovesAndAssociatedData(
	ctx context.Context,
	tx bun.IDB,
	moveIDsToDelete []pulid.ID,
) error {
	// Delete associated stops
	if err := sr.deleteAssociatedStops(ctx, tx, moveIDsToDelete); err != nil {
		return err
	}

	// Delete associated assignments
	if err := sr.deleteAssociatedAssignments(ctx, tx, moveIDsToDelete); err != nil {
		return err
	}

	// Delete the moves themselves
	return sr.deleteMoves(ctx, tx, moveIDsToDelete)
}

// deleteAssociatedStops deletes all stops associated with the specified moves
func (sr *shipmentMoveRepository) deleteAssociatedStops(
	ctx context.Context,
	tx bun.IDB,
	moveIDs []pulid.ID,
) error {
	log := sr.l.With().
		Str("operation", "deleteAssociatedStops").
		Logger()

	_, err := tx.NewDelete().
		Model((*shipment.Stop)(nil)).
		Where("shipment_move_id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Err(err).
				Interface("moveIDs", moveIDs).
				Msg("failed to delete associated stops")
			return errors.NewNotFoundError("Associated stops not found within your organization")
		}

		log.Error().Err(err).
			Interface("moveIDs", moveIDs).
			Msg("failed to delete associated stops")
		return err
	}
	return nil
}

// deleteAssociatedAssignments deletes all assignments associated with the specified moves
func (sr *shipmentMoveRepository) deleteAssociatedAssignments(
	ctx context.Context,
	tx bun.IDB,
	moveIDs []pulid.ID,
) error {
	log := sr.l.With().
		Str("operation", "deleteAssociatedAssignments").
		Logger()

	_, err := tx.NewDelete().
		Model((*shipment.Assignment)(nil)).
		Where("shipment_move_id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Err(err).
				Interface("moveIDs", moveIDs).
				Msg("failed to delete associated assignments")
			return errors.NewNotFoundError(
				"Associated assignments not found within your organization",
			)
		}

		log.Error().
			Err(err).
			Interface("moveIDs", moveIDs).
			Msg("failed to delete associated assignments")
		return err
	}
	return nil
}

// deleteMoves deletes the specified moves and logs the result
func (sr *shipmentMoveRepository) deleteMoves(
	ctx context.Context,
	tx bun.IDB,
	moveIDs []pulid.ID,
) error {
	log := sr.l.With().
		Str("operation", "deleteMoves").
		Logger()

	result, err := tx.NewDelete().
		Model((*shipment.ShipmentMove)(nil)).
		Where("id IN (?)", bun.In(moveIDs)).
		Exec(ctx)
	if err != nil {
		log.Error().
			Err(err).
			Interface("moveIDs", moveIDs).
			Msg("failed to delete moves")
		return err
	}

	// Check that the expected number of moves were deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected for move deletion")
		return err
	}

	log.Info().
		Int64("deletedMoveCount", rowsAffected).
		Interface("moveIDs", moveIDs).
		Msg("successfully deleted moves")

	return nil
}

// resequenceRemainingMoves reorders the sequence numbers of all moves for a shipment to ensure
// they are sequential (0, 1, 2, ...) with no gaps
func (sr *shipmentMoveRepository) resequenceRemainingMoves(
	ctx context.Context,
	tx bun.IDB,
	shipmentID pulid.ID,
) error {
	log := sr.l.With().
		Str("operation", "resequenceRemainingMoves").
		Str("shipmentID", shipmentID.String()).
		Logger()

	// * Get all remaining moves for this shipment, ordered by their current sequence
	var moves []*shipment.ShipmentMove
	err := tx.NewSelect().
		Model(&moves).
		Where("shipment_id = ?", shipmentID).
		Order("sequence ASC").
		Scan(ctx)
	if err != nil {
		log.Error().Err(err).
			Str("shipmentID", shipmentID.String()).
			Msg("failed to get remaining moves for resequencing")
		return err
	}

	// * Nothing to resequence if there are no moves or just one move
	if len(moves) <= 1 {
		return nil
	}

	// * Check if sequences are already contiguous and start from 0
	needsResequencing := false
	for i, move := range moves {
		if move.Sequence != i {
			needsResequencing = true
			break
		}
	}

	// * Skip resequencing if already in order
	if !needsResequencing {
		log.Debug().Msg("moves already properly sequenced, skipping resequencing")
		return nil
	}

	// * Update each move with its new sequence number
	for i, move := range moves {
		if move.Sequence == i {
			continue // Skip if already has the correct sequence
		}

		_, err = tx.NewUpdate().
			Model(move).
			Set("sequence = ?", i).
			Set("version = version + 1"). // Increment version for optimistic locking
			Where("id = ?", move.ID).
			Exec(ctx)
		if err != nil {
			log.Error().Err(err).
				Str("moveID", move.ID.String()).
				Int("oldSequence", move.Sequence).
				Int("newSequence", i).
				Msg("failed to update move sequence during resequencing")
			return err
		}

		log.Debug().
			Str("moveID", move.ID.String()).
			Int("oldSequence", move.Sequence).
			Int("newSequence", i).
			Msg("resequenced move")
	}

	log.Info().Int("moveCount", len(moves)).Msg("successfully resequenced moves")
	return nil
}
