/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type StopRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type stopRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewStopRepository(p StopRepositoryParams) repositories.StopRepository {
	log := p.Logger.With().
		Str("repository", "stop").
		Logger()

	return &stopRepository{
		db: p.DB,
		l:  &log,
	}
}

func (sr *stopRepository) GetByID(
	ctx context.Context,
	req repositories.GetStopByIDRequest,
) (*shipment.Stop, error) {
	dba, err := sr.db.ReadDB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "GetByID").
		Str("stopID", req.StopID.String()).
		Str("orgID", req.OrgID.String()).
		Str("buID", req.BuID.String()).
		Logger()

	stop := new(shipment.Stop)

	q := dba.NewSelect().Model(stop).
		Where("stp.id = ?", req.StopID).
		Where("stp.organization_id = ?", req.OrgID).
		Where("stp.business_unit_id = ?", req.BuID)

	if req.ExpandStopDetails {
		q.Relation("Location")
	}

	if err = q.Scan(ctx); err != nil {
		log.Error().Err(err).Msg("failed to get stop by id")
		return nil, err
	}

	return stop, nil
}

func (sr *stopRepository) BulkInsert(
	ctx context.Context,
	stops []*shipment.Stop,
) ([]*shipment.Stop, error) {
	dba, err := sr.db.WriteDB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "BulkInsert").
		Interface("stops", stops).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, err = tx.NewInsert().Model(&stops).Exec(c); err != nil {
			log.Error().Err(err).Msg("failed to bulk insert stops")
			return err
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to bulk insert stops")
		return nil, err
	}

	return stops, nil
}

func (sr *stopRepository) Update(
	ctx context.Context,
	stop *shipment.Stop,
	moveIdx, stopIdx int,
) (*shipment.Stop, error) {
	dba, err := sr.db.WriteDB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := sr.l.With().
		Str("operation", "Update").
		Str("stopID", stop.ID.String()).
		Str("orgID", stop.OrganizationID.String()).
		Str("buID", stop.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		stop.Version++

		results, rErr := tx.NewUpdate().
			Model(stop).
			Where("stp.id = ?", stop.ID).
			OmitZero().
			// Where("stp.version = ?", ov).
			Where("stp.shipment_move_id = ?", stop.ShipmentMoveID).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().Err(rErr).Msg("failed to update stop")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).Msg("failed to get rows affected")
			return roErr
		}

		log.Debug().Int("rows", int(rows)).Msg("stop rows affected")

		if rows == 0 {
			return errors.NewValidationError(
				fmt.Sprintf("move[%d].stop[%d].version", moveIdx, stopIdx),
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The Stop (%s) has either been updated or deleted since the last request.",
					stop.ID,
				),
			)
		}

		log.Debug().Int("rows", int(rows)).Msg("updated stop")

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update stop")
		return nil, err
	}

	return stop, nil
}

// HandleStopRemovals handles the removal of stops that are no longer present in the move
func (sr *stopRepository) HandleStopRemovals(
	ctx context.Context,
	tx bun.IDB,
	move *shipment.ShipmentMove,
	existingStops []*shipment.Stop,
	updatedStopIDs map[pulid.ID]struct{},
) error {
	log := sr.l.With().
		Str("operation", "HandleStopRemovals").
		Str("moveID", move.ID.String()).
		Logger()

	stopCount := len(existingStops)

	stopIDsToDelete := make([]pulid.ID, 0, stopCount)
	existingStopMap := make(map[pulid.ID]*shipment.Stop, stopCount)

	for _, stop := range existingStops {
		existingStopMap[stop.ID] = stop

		if _, ok := updatedStopIDs[stop.ID]; !ok {
			stopIDsToDelete = append(stopIDsToDelete, stop.ID)
			log.Debug().
				Str("stopID", stop.ID.String()).
				Int("sequence", stop.Sequence).
				Str("type", string(stop.Type)).
				Msg("stop marked for deletion")
		}
	}

	log.Debug().
		Interface("stopIDsToDelete", stopIDsToDelete).
		Int("deleteCount", len(stopIDsToDelete)).
		Int("existingCount", len(existingStops)).
		Int("updatedCount", len(updatedStopIDs)).
		Msg("stops to delete")

	// If there are stops to delete
	if len(stopIDsToDelete) > 0 {
		// Process the deletion with appropriate validations
		if err := sr.processStopDeletions(ctx, tx, move.ID, stopIDsToDelete); err != nil {
			return err
		}
	}

	return nil
}

// processStopDeletions handles the actual deletion process after validating requirements
func (sr *stopRepository) processStopDeletions(
	ctx context.Context,
	tx bun.IDB,
	moveID pulid.ID,
	stopIDsToDelete []pulid.ID,
) error {
	log := sr.l.With().
		Str("operation", "processStopDeletions").
		Str("moveID", moveID.String()).
		Logger()

	allStops, err := sr.getAllStopsForMove(ctx, tx, moveID)
	if err != nil {
		return err
	}

	if err = sr.validateMinimumStops(allStops); err != nil {
		return err
	}

	if err = sr.validateRemainingStopTypes(allStops, stopIDsToDelete); err != nil {
		return err
	}

	if err = sr.deleteStops(ctx, tx, stopIDsToDelete); err != nil {
		return err
	}

	if err = sr.resequenceRemainingStops(ctx, tx, moveID); err != nil {
		log.Error().Err(err).Msg("failed to resequence remaining stops")
		return err
	}

	return nil
}

// getAllStopsForMove fetches all stops for the given move
func (sr *stopRepository) getAllStopsForMove(
	ctx context.Context,
	tx bun.IDB,
	moveID pulid.ID,
) ([]*shipment.Stop, error) {
	log := sr.l.With().
		Str("operation", "getAllStopsForMove").
		Str("moveID", moveID.String()).
		Logger()

	allStops := make([]*shipment.Stop, 0)
	err := tx.NewSelect().
		Model(&allStops).
		Where("shipment_move_id = ?", moveID).
		Scan(ctx)
	if err != nil {
		log.Error().Err(err).Str("moveID", moveID.String()).
			Msg("failed to get all stops for move")
		return nil, err
	}

	return allStops, nil
}

// validateMinimumStops ensures there are enough stops in total
func (sr *stopRepository) validateMinimumStops(allStops []*shipment.Stop) error {
	log := sr.l.With().Str("operation", "validateMinimumStops").Logger()

	if len(allStops) < 2 {
		log.Error().Msg("move has less than 2 stops, cannot proceed with deletion")
		return errors.NewBusinessError(
			"A move must have at least a pickup and delivery stop",
		)
	}

	return nil
}

// validateRemainingStopTypes ensures at least one pickup and delivery stop will remain
func (sr *stopRepository) validateRemainingStopTypes(
	allStops []*shipment.Stop,
	stopIDsToDelete []pulid.ID,
) error {
	log := sr.l.With().Str("operation", "validateRemainingStopTypes").Logger()

	remainingPickups := 0
	remainingDeliveries := 0

	stopsToDelete := make(map[pulid.ID]struct{})
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

	log.Debug().
		Int("remainingPickups", remainingPickups).
		Int("remainingDeliveries", remainingDeliveries).
		Msg("stops that will remain after deletion")

	if remainingPickups == 0 {
		return errors.NewBusinessError(
			"Cannot delete all pickup stops. At least one pickup stop is required for the move.",
		)
	}

	if remainingDeliveries == 0 {
		return errors.NewBusinessError(
			"Cannot delete all delivery stops. At least one delivery stop is required for the move.",
		)
	}

	return nil
}

func (sr *stopRepository) deleteStops(
	ctx context.Context,
	tx bun.IDB,
	stopIDsToDelete []pulid.ID,
) error {
	log := sr.l.With().
		Str("operation", "deleteStops").
		Interface("stopIDsToDelete", stopIDsToDelete).
		Logger()

	result, err := tx.NewDelete().
		Model((*shipment.Stop)(nil)).
		Where("id IN (?)", bun.In(stopIDsToDelete)).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Interface("stopIDs", stopIDsToDelete).
			Msg("failed to delete stops")
		return err
	}

	// Check that the expected number of stops were deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected for stop deletion")
		return err
	}

	log.Info().Int64("deletedStopCount", rowsAffected).
		Interface("stopIDs", stopIDsToDelete).
		Msg("successfully deleted stops")

	return nil
}

// resequenceRemainingStops reorders the sequence numbers of all stops for a move to ensure
// they are sequential with no gaps
func (sr *stopRepository) resequenceRemainingStops(
	ctx context.Context,
	tx bun.IDB,
	moveID pulid.ID,
) error {
	log := sr.l.With().
		Str("operation", "resequenceRemainingStops").
		Str("moveID", moveID.String()).
		Logger()

	stops := make([]*shipment.Stop, 0)
	err := tx.NewSelect().
		Model(&stops).
		Where("shipment_move_id = ?", moveID).
		Order("sequence ASC").
		Scan(ctx)
	if err != nil {
		log.Error().Err(err).Str("moveID", moveID.String()).
			Msg("failed to get remaining stops for resequencing")
		return err
	}

	if len(stops) <= 1 {
		return nil
	}

	needsResequencing := false
	for i, stop := range stops {
		if stop.Sequence != i {
			needsResequencing = true
			break
		}
	}

	if !needsResequencing {
		log.Debug().Msg("stops already properly sequenced, skipping resequencing")
		return nil
	}

	for i, stop := range stops {
		if stop.Sequence == i {
			continue
		}

		_, err = tx.NewUpdate().
			Model(stop).
			Set("sequence = ?", i).
			Set("version = version + 1").
			Where("id = ?", stop.ID).
			Exec(ctx)
		if err != nil {
			log.Error().Err(err).
				Str("stopID", stop.ID.String()).
				Int("oldSequence", stop.Sequence).
				Int("newSequence", i).
				Msg("failed to update stop sequence during resequencing")
			return err
		}

		log.Debug().
			Str("stopID", stop.ID.String()).
			Int("oldSequence", stop.Sequence).
			Int("newSequence", i).
			Msg("resequenced stop")
	}

	log.Info().Int("stopCount", len(stops)).Msg("successfully resequenced stops")
	return nil
}
