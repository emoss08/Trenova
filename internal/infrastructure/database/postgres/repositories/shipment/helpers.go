/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package shipment

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
)

// EntityType represents the type of entity being processed
type EntityType string

const (
	EntityTypeShipment         EntityType = "shipments"
	EntityTypeMove             EntityType = "moves"
	EntityTypeStop             EntityType = "stops"
	EntityTypeCommodity        EntityType = "commodities"
	EntityTypeAdditionalCharge EntityType = "additional_charges"
	EntityTypeComment          EntityType = "comments"
)

// shipmentBulkData holds all entities for bulk insertion
type shipmentBulkData struct {
	shipments         []*shipment.Shipment
	moves             []*shipment.ShipmentMove
	stops             []*shipment.Stop
	commodities       []*shipment.ShipmentCommodity
	additionalCharges []*shipment.AdditionalCharge
	comments          []*shipment.ShipmentComment
}

// validateShipmentForCreation performs validation checks before creating a shipment
func (sr *shipmentRepository) validateShipmentForCreation(shp *shipment.Shipment) error {
	if shp.OrganizationID == pulid.Nil {
		return oops.In("shipment_repository").
			With("field", "organization_id").
			Errorf("organization ID is required")
	}
	if shp.BusinessUnitID == pulid.Nil {
		return oops.In("shipment_repository").
			With("field", "business_unit_id").
			Errorf("business unit ID is required")
	}
	return nil
}

// prepareShipmentForCreation prepares a shipment entity before insertion
func (sr *shipmentRepository) prepareShipmentForCreation(
	ctx context.Context,
	shp *shipment.Shipment,
	userID pulid.ID,
) error {
	// Generate pro number
	proNumber, err := sr.proNumberRepo.GetNextProNumber(ctx, &repositories.GetProNumberRequest{
		OrgID: shp.OrganizationID,
		BuID:  shp.BusinessUnitID,
	})
	if err != nil {
		return oops.In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "failed to get next pro number")
	}
	shp.ProNumber = proNumber

	// Calculate totals and status
	sr.calc.CalculateTotals(ctx, shp, userID)

	if err = sr.calc.CalculateStatus(shp); err != nil {
		return oops.In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "failed to calculate shipment status")
	}

	if err = sr.calc.CalculateTimestamps(shp); err != nil {
		return oops.In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "failed to calculate shipment timestamps")
	}

	return nil
}

// createShipmentWithRelations handles creation of shipment and all related entities in a transaction
func (sr *shipmentRepository) createShipmentWithRelations(
	ctx context.Context,
	tx bun.Tx,
	shp *shipment.Shipment,
	log *zerolog.Logger,
) error {
	if _, err := tx.NewInsert().Model(shp).Returning("*").Exec(ctx); err != nil {
		log.Error().
			Err(err).
			Interface("shipment", shp).
			Msg("failed to insert shipment")
		return err
	}

	if err := sr.shipmentCommodityRepository.HandleCommodityOperations(ctx, tx, shp, true); err != nil {
		log.Error().Err(err).Msg("failed to handle commodity operations")
		return err
	}

	if err := sr.shipmentMoveRepository.HandleMoveOperations(ctx, tx, shp, true); err != nil {
		log.Error().Err(err).Msg("failed to handle move operations")
		return err
	}

	if err := sr.additionalChargeRepository.HandleAdditionalChargeOperations(ctx, tx, shp, true); err != nil {
		log.Error().Err(err).Msg("failed to handle additional charge operations")
		return err
	}

	return nil
}

// updateShipmentWithRelations handles update of shipment and all related entities in a transaction
func (sr *shipmentRepository) updateShipmentWithRelations(
	ctx context.Context,
	tx bun.Tx,
	shp *shipment.Shipment,
	log *zerolog.Logger,
) error {
	ov := shp.Version
	shp.Version++

	results, err := tx.NewUpdate().
		Model(shp).
		WherePK().
		Where("sp.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error().
			Err(err).
			Interface("shipment", shp).
			Msg("failed to update shipment")
		return err
	}

	rows, err := results.RowsAffected()
	if err != nil {
		log.Error().
			Err(err).
			Interface("shipment", shp).
			Msg("failed to get rows affected")
		return err
	}

	if rows == 0 {
		return errors.NewValidationError(
			"version",
			errors.ErrVersionMismatch,
			fmt.Sprintf(
				"Version mismatch. The Shipment (%s) has either been updated or deleted since the last request.",
				shp.GetID(),
			),
		)
	}

	if err = sr.shipmentCommodityRepository.HandleCommodityOperations(ctx, tx, shp, false); err != nil {
		log.Error().Err(err).Msg("failed to handle commodity operations")
		return err
	}

	if err = sr.shipmentMoveRepository.HandleMoveOperations(ctx, tx, shp, false); err != nil {
		log.Error().Err(err).Msg("failed to handle move operations")
		return err
	}

	if err = sr.additionalChargeRepository.HandleAdditionalChargeOperations(ctx, tx, shp, false); err != nil {
		log.Error().Err(err).Msg("failed to handle additional charge operations")
		return err
	}

	return nil
}

// prepareBulkShipmentData prepares all entities for bulk duplication
func (sr *shipmentRepository) prepareBulkShipmentData(
	ctx context.Context,
	originalShipment *shipment.Shipment,
	req *repositories.DuplicateShipmentRequest,
) (*shipmentBulkData, error) {
	data := &shipmentBulkData{
		shipments:         make([]*shipment.Shipment, 0, req.Count),
		moves:             make([]*shipment.ShipmentMove, 0),
		stops:             make([]*shipment.Stop, 0),
		commodities:       make([]*shipment.ShipmentCommodity, 0),
		additionalCharges: make([]*shipment.AdditionalCharge, 0),
		comments:          make([]*shipment.ShipmentComment, 0),
	}

	for i := range req.Count {
		newShipment, err := sr.duplicateShipmentFields(ctx, originalShipment)
		if err != nil {
			return nil, oops.In("shipment_repository").
				With("iteration", i).
				Time(time.Now()).
				Wrapf(err, "failed to duplicate shipment fields")
		}

		data.shipments = append(data.shipments, newShipment)

		moves, stops := sr.prepareMovesAndStops(
			originalShipment,
			newShipment,
			req.OverrideDates,
		)
		data.moves = append(data.moves, moves...)
		data.stops = append(data.stops, stops...)

		if req.IncludeCommodities {
			commodities := sr.prepareCommodities(originalShipment, newShipment)
			data.commodities = append(data.commodities, commodities...)
		}

		if req.IncludeAdditionalCharges {
			additionalCharges := sr.prepareAdditionalCharges(originalShipment, newShipment)
			data.additionalCharges = append(data.additionalCharges, additionalCharges...)
		}

		if req.IncludeComments {
			comments := sr.prepareShipmentComments(originalShipment, newShipment)
			data.comments = append(data.comments, comments...)
		}
	}

	return data, nil
}

// bulkInsertShipmentData performs bulk insertion of all shipment-related entities
func (sr *shipmentRepository) bulkInsertShipmentData(
	ctx context.Context,
	tx bun.Tx,
	data *shipmentBulkData,
	log *zerolog.Logger,
) error {
	if err := sr.insertEntities(ctx, tx, log, EntityTypeShipment, &data.shipments); err != nil {
		return err
	}

	if len(data.moves) > 0 {
		if err := sr.insertEntities(ctx, tx, log, EntityTypeMove, &data.moves); err != nil {
			return err
		}
	}

	if len(data.stops) > 0 {
		if err := sr.insertEntities(ctx, tx, log, EntityTypeStop, &data.stops); err != nil {
			return err
		}
	}

	if len(data.commodities) > 0 {
		if err := sr.insertEntities(ctx, tx, log, EntityTypeCommodity, &data.commodities); err != nil {
			return err
		}
	}

	if len(data.additionalCharges) > 0 {
		if err := sr.insertEntities(ctx, tx, log, EntityTypeAdditionalCharge, &data.additionalCharges); err != nil {
			return err
		}
	}

	if len(data.comments) > 0 {
		if err := sr.insertEntities(ctx, tx, log, EntityTypeComment, &data.comments); err != nil {
			return err
		}
	}

	return nil
}

// executeInTransaction wraps database operations in a transaction with consistent error handling
func (sr *shipmentRepository) executeInTransaction(
	ctx context.Context,
	fn func(context.Context, bun.Tx) error,
) error {
	dba, err := sr.db.DB(ctx)
	if err != nil {
		return oops.
			In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	return dba.RunInTx(ctx, nil, fn)
}

// prepareStopForDuplication creates a new stop based on the original with optional date overrides
func (sr *shipmentRepository) prepareStopForDuplication(
	original *shipment.Stop,
	newMoveID pulid.ID,
	overrideDates bool,
) *shipment.Stop {
	newStop := &shipment.Stop{
		ID:             pulid.MustNew("stp_"),
		BusinessUnitID: original.BusinessUnitID,
		OrganizationID: original.OrganizationID,
		ShipmentMoveID: newMoveID,
		LocationID:     original.LocationID,
		Status:         shipment.StopStatusNew,
		Type:           original.Type,
		Sequence:       original.Sequence,
		Pieces:         original.Pieces,
		Weight:         original.Weight,
		PlannedArrival: original.PlannedArrival,
		AddressLine:    original.AddressLine,
	}

	// Override dates if requested
	if overrideDates {
		now := timeutils.NowUnix()
		oneDay := timeutils.DaysToSeconds(1)
		newStop.PlannedArrival = now
		newStop.PlannedDeparture = now + oneDay
	} else {
		newStop.PlannedDeparture = original.PlannedDeparture
	}

	return newStop
}

// bulkCancelShipmentComponents cancels multiple shipment components in bulk
func (sr *shipmentRepository) bulkCancelShipmentComponents(
	ctx context.Context,
	tx bun.Tx,
	moveIDs []pulid.ID,
) error {
	// Cancel moves
	if _, err := tx.NewUpdate().
		Model((*shipment.ShipmentMove)(nil)).
		Set("status = ?", shipment.MoveStatusCanceled).
		Where("sm.id IN (?)", bun.In(moveIDs)).
		Exec(ctx); err != nil {
		return oops.In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "failed to cancel moves")
	}

	// Cancel assignments
	if _, err := tx.NewUpdate().
		Model((*shipment.Assignment)(nil)).
		Set("status = ?", shipment.AssignmentStatusCanceled).
		Where("a.shipment_move_id IN (?)", bun.In(moveIDs)).
		Exec(ctx); err != nil {
		return oops.In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "failed to cancel assignments")
	}

	// Cancel stops
	if _, err := tx.NewUpdate().
		Model((*shipment.Stop)(nil)).
		Set("status = ?", shipment.StopStatusCanceled).
		Where("stp.shipment_move_id IN (?)", bun.In(moveIDs)).
		Exec(ctx); err != nil {
		return oops.In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "failed to cancel stops")
	}

	return nil
}

// bulkUnCancelShipmentComponents un-cancels multiple shipment components in bulk
func (sr *shipmentRepository) bulkUnCancelShipmentComponents(
	ctx context.Context,
	tx bun.Tx,
	moveIDs []pulid.ID,
	updateAppointments bool,
) error {
	if _, err := tx.NewUpdate().
		Model((*shipment.ShipmentMove)(nil)).
		OmitZero().
		Set("status = ?", shipment.MoveStatusNew).
		Where("sm.id IN (?)", bun.In(moveIDs)).
		Exec(ctx); err != nil {
		return oops.In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "failed to un-cancel moves")
	}

	if _, err := tx.NewUpdate().
		Model((*shipment.Assignment)(nil)).
		OmitZero().
		Set("status = ?", shipment.AssignmentStatusNew).
		Where("a.shipment_move_id IN (?)", bun.In(moveIDs)).
		Exec(ctx); err != nil {
		return oops.In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "failed to un-cancel assignments")
	}

	stpQuery := tx.NewUpdate().
		Model((*shipment.Stop)(nil)).
		Set("status = ?", shipment.StopStatusNew).
		OmitZero().
		Where("stp.shipment_move_id IN (?)", bun.In(moveIDs))

	if updateAppointments {
		stpQuery.Set("planned_arrival = ?", timeutils.NowUnix())
		stpQuery.Set("planned_departure = ?", timeutils.NowUnix()+timeutils.DaysToSeconds(1))
	}

	if _, err := stpQuery.Exec(ctx); err != nil {
		return oops.In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "failed to un-cancel stops")
	}

	return nil
}

// getMoveIDsForShipment retrieves all move IDs for a given shipment
func (sr *shipmentRepository) getMoveIDsForShipment(
	ctx context.Context,
	tx bun.Tx,
	shipmentID pulid.ID,
) ([]pulid.ID, error) {
	moves := make([]*shipment.ShipmentMove, 0)
	err := tx.NewSelect().
		Model(&moves).
		Where("sm.shipment_id = ?", shipmentID).
		Scan(ctx)
	if err != nil {
		return nil, oops.In("shipment_repository").
			Time(time.Now()).
			Wrapf(err, "failed to fetch shipment moves")
	}

	if len(moves) == 0 {
		return []pulid.ID{}, nil
	}

	moveIDs := make([]pulid.ID, len(moves))
	for i, move := range moves {
		moveIDs[i] = move.ID
	}

	return moveIDs, nil
}
