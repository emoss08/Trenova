package dedicatedlane

import (
	"context"
	"database/sql"
	"time"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// AssignmentServiceParams defines dependencies for the dedicated lane assignment service
type AssignmentServiceParams struct {
	fx.In

	DB             db.Connection
	AssignmentRepo repositories.AssignmentRepository
	ShipmentRepo   repositories.ShipmentRepository
	Logger         *logger.Logger
}

// AssignmentService handles dedicated lane assignment operations
type AssignmentService struct {
	db             db.Connection
	assignmentRepo repositories.AssignmentRepository
	shipmentRepo   repositories.ShipmentRepository
	l              *zerolog.Logger
}

// NewAssignmentService creates a new dedicated lane assignment service
func NewAssignmentService(p AssignmentServiceParams) *AssignmentService {
	log := p.Logger.With().
		Str("service", "dedicated_lane_assignment").
		Logger()

	return &AssignmentService{
		db:             p.DB,
		assignmentRepo: p.AssignmentRepo,
		shipmentRepo:   p.ShipmentRepo,
		l:              &log,
	}
}

// HandleDedicatedLaneOperations checks if a shipment matches a dedicated lane and auto-assigns if configured
func (s *AssignmentService) HandleDedicatedLaneOperations(
	ctx context.Context,
	shp *shipment.Shipment,
) error {
	dba, err := s.db.DB(ctx)
	if err != nil {
		return oops.In("dedicated_lane_assignment_service").
			With("op", "handle_dedicated_lane_operations").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

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

	// Find matching dedicated lane
	dl, err := s.findMatchingDedicatedLane(ctx, dba, shp, originLocationID, destinationLocationID)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Info().Msg("no dedicated lane found for shipment")
			return nil
		}
		log.Error().Err(err).Msg("failed to query dedicated lane")
		return err
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

// findMatchingDedicatedLane queries for a dedicated lane matching the shipment criteria
func (s *AssignmentService) findMatchingDedicatedLane(
	ctx context.Context,
	dba *bun.DB,
	shp *shipment.Shipment,
	originLocationID, destinationLocationID pulid.ID,
) (*dedicatedlane.DedicatedLane, error) {
	dl := new(dedicatedlane.DedicatedLane)

	query := dba.NewSelect().Model(dl).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			q := sq.
				Where("dl.status = ?", domain.StatusActive).
				Where("dl.organization_id = ?", shp.OrganizationID).
				Where("dl.business_unit_id = ?", shp.BusinessUnitID).
				Where("dl.customer_id = ?", shp.CustomerID).
				Where("dl.origin_location_id = ?", originLocationID).
				Where("dl.destination_location_id = ?", destinationLocationID)

			// ServiceTypeID and ShipmentTypeID are required fields
			q = q.Where("dl.service_type_id = ?", shp.ServiceTypeID).
				Where("dl.shipment_type_id = ?", shp.ShipmentTypeID)

			// Handle optional trailer type - match if both are specified and equal, or both are null
			if shp.TrailerTypeID != nil {
				q = q.Where("dl.trailer_type_id = ?", *shp.TrailerTypeID)
			} else {
				q = q.Where("dl.trailer_type_id IS NULL")
			}

			// Handle optional tractor type - match if both are specified and equal, or both are null
			if shp.TractorTypeID != nil {
				q = q.Where("dl.tractor_type_id = ?", *shp.TractorTypeID)
			} else {
				q = q.Where("dl.tractor_type_id IS NULL")
			}

			return q
		}).
		Order("dl.created_at DESC").
		Limit(1)

	err := query.Scan(ctx)

	return dl, err
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

	// Bulk assign shipment moves to dedicated lane
	_, err := s.assignmentRepo.BulkAssign(ctx, &repositories.AssignmentRequest{
		ShipmentID:        shp.ID,
		PrimaryWorkerID:   dl.PrimaryWorkerID,
		SecondaryWorkerID: dl.SecondaryWorkerID,
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
