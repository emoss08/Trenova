/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package dedicatedlane

import (
	"context"
	"database/sql"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

// AssignmentServiceParams defines dependencies for the dedicated lane assignment service
type AssignmentServiceParams struct {
	fx.In

	DB                db.Connection
	AssignmentRepo    repositories.AssignmentRepository
	ShipmentRepo      repositories.ShipmentRepository
	TractorRepo       repositories.TractorRepository
	DedicatedLaneRepo repositories.DedicatedLaneRepository
	Logger            *logger.Logger
}

// AssignmentService handles dedicated lane assignment operations
type AssignmentService struct {
	db                db.Connection
	assignmentRepo    repositories.AssignmentRepository
	shipmentRepo      repositories.ShipmentRepository
	tractorRepo       repositories.TractorRepository
	dedicatedLaneRepo repositories.DedicatedLaneRepository
	l                 *zerolog.Logger
}

// NewAssignmentService creates a new dedicated lane assignment service
//
//nolint:gocritic // This is dependency injection
func NewAssignmentService(p AssignmentServiceParams) *AssignmentService {
	log := p.Logger.With().
		Str("service", "dedicated_lane_assignment").
		Logger()

	return &AssignmentService{
		db:                p.DB,
		assignmentRepo:    p.AssignmentRepo,
		shipmentRepo:      p.ShipmentRepo,
		tractorRepo:       p.TractorRepo,
		dedicatedLaneRepo: p.DedicatedLaneRepo,
		l:                 &log,
	}
}

// HandleDedicatedLaneOperations checks if a shipment matches a dedicated lane and auto-assigns if configured
func (s *AssignmentService) HandleDedicatedLaneOperations(
	ctx context.Context,
	shp *shipment.Shipment,
) error {
	log := s.l.With().
		Str("op", "handle_dedicated_lane_operations").
		Str("shipmentID", shp.ID.String()).
		Logger()

	// Extract origin and destination from shipment moves
	originLocationID, destinationLocationID := s.extractLocations(shp)
	if originLocationID == "" || destinationLocationID == "" {
		log.Info().Msg("insufficient location data for dedicated lane matching")
		return nil
	}

	// Find matching dedicated lane using repository
	dl, err := s.dedicatedLaneRepo.FindByShipment(
		ctx,
		&repositories.FindDedicatedLaneByShipmentRequest{
			OrganizationID:        shp.OrganizationID,
			BusinessUnitID:        shp.BusinessUnitID,
			CustomerID:            shp.CustomerID,
			ServiceTypeID:         &shp.ServiceTypeID,
			ShipmentTypeID:        &shp.ShipmentTypeID,
			OriginLocationID:      originLocationID,
			DestinationLocationID: destinationLocationID,
			TrailerTypeID:         shp.TrailerTypeID,
			TractorTypeID:         shp.TractorTypeID,
		},
	)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Info().Msg("no dedicated lane found for shipment")
			return nil
		}
		log.Error().Err(err).Msg("failed to query dedicated lane")
		return err
	}

	// Check if dedicated lane was found
	if dl == nil {
		log.Info().Msg("no dedicated lane found for shipment")
		return nil
	}

	// Auto-assign if configured
	if dl.AutoAssign {
		return s.performAutoAssignment(ctx, shp, dl, &log)
	}

	log.Info().Msg("dedicated lane found but auto-assign disabled")
	return nil
}

// extractLocations gets origin and destination location IDs from shipment moves
func (s *AssignmentService) extractLocations(
	shp *shipment.Shipment,
) (originLocationID, destinationLocationID pulid.ID) {
	if len(shp.Moves) == 0 {
		return originLocationID, destinationLocationID
	}

	// Get origin from first move's first stop
	if len(shp.Moves[0].Stops) > 0 {
		originLocationID = shp.Moves[0].Stops[0].LocationID
	}

	// Get destination from last move's last stop
	if len(shp.Moves) > 1 && len(shp.Moves[len(shp.Moves)-1].Stops) > 0 {
		lastMove := shp.Moves[len(shp.Moves)-1]
		destinationLocationID = lastMove.Stops[len(lastMove.Stops)-1].LocationID
	} else if len(shp.Moves[0].Stops) > 1 {
		// Single move with multiple stops
		destinationLocationID = shp.Moves[0].Stops[len(shp.Moves[0].Stops)-1].LocationID
	}

	return originLocationID, destinationLocationID
}

// performAutoAssignment executes the auto-assignment process
func (s *AssignmentService) performAutoAssignment(
	ctx context.Context,
	shp *shipment.Shipment,
	dl *dedicatedlane.DedicatedLane,
	log *zerolog.Logger,
) error {
	log.Info().
		Str("dedicatedLaneID", dl.ID.String()).
		Msg("auto-assigning shipment to dedicated lane")

	// ! Do nothing if the primary worker is not set
	if dl.PrimaryWorkerID == nil {
		return nil
	}

	// * Fetch the tractor by it's primary worker id
	tractor, err := s.tractorRepo.GetByPrimaryWorkerID(
		ctx,
		repositories.GetTractorByPrimaryWorkerIDRequest{
			WorkerID: dl.PrimaryWorkerID,
			OrgID:    shp.OrganizationID,
			BuID:     shp.BusinessUnitID,
		},
	)
	if err != nil {
		return oops.In("dedicated_lane_assignment_service").
			With("op", "perform_auto_assignment").
			Time(time.Now()).
			Wrapf(err, "get tractor by primary worker id")
	}

	// Bulk assign shipment moves to dedicated lane
	_, err = s.assignmentRepo.BulkAssign(ctx, &repositories.AssignmentRequest{
		ShipmentID:        shp.ID,
		PrimaryWorkerID:   pulid.ConvertFromPtr(dl.PrimaryWorkerID),
		SecondaryWorkerID: dl.SecondaryWorkerID,
		TractorID:         tractor.ID,
		OrgID:             shp.OrganizationID,
		BuID:              shp.BusinessUnitID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to bulk assign shipment moves to dedicated lane")
		return oops.In("dedicated_lane_assignment_service").
			With("op", "perform_auto_assignment").
			Time(time.Now()).
			Wrapf(err, "bulk assign shipment moves")
	}

	// Update shipment status to assigned
	_, err = s.shipmentRepo.UpdateStatus(ctx, &repositories.UpdateShipmentStatusRequest{
		GetOpts: &repositories.GetShipmentByIDOptions{
			ID:    shp.ID,
			OrgID: shp.OrganizationID,
			BuID:  shp.BusinessUnitID,
			ShipmentOptions: repositories.ShipmentOptions{
				ExpandShipmentDetails: true,
			},
		},
		Status: shipment.StatusAssigned,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update shipment status")
		return oops.In("dedicated_lane_assignment_service").
			With("op", "perform_auto_assignment").
			Time(time.Now()).
			Wrapf(err, "update shipment status")
	}

	log.Info().Msg("successfully auto-assigned shipment to dedicated lane")
	return nil
}
