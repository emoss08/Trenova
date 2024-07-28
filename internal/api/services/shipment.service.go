// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package services

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/audit"
	"github.com/emoss08/trenova/pkg/constants"

	"github.com/emoss08/trenova/internal/api/common"

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type ShipmentService struct {
	common.AuditableService
	logger *config.ServerLogger
}

func NewShipmentService(s *server.Server) *ShipmentService {
	return &ShipmentService{
		AuditableService: common.AuditableService{
			DB:            s.DB,
			AuditService:  s.AuditService,
			CodeGenerator: s.CodeGenerator,
		},
		logger: s.Logger,
	}
}

// ShipmentQueryFilter Suggested additions to ShipmentQueryFilter
type ShipmentQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	CustomerID     uuid.UUID
	FromDate       *time.Time
	ToDate         *time.Time
	ShipmentTypeID uuid.UUID
	IsHazardous    bool
	Status         property.ShipmentStatus
	Limit          int
	Offset         int
}

func (s ShipmentService) filterQuery(q *bun.SelectQuery, f *ShipmentQueryFilter) *bun.SelectQuery {
	q = q.Where("sp.organization_id = ?", f.OrganizationID).
		Where("sp.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("sp.pro_number ILIKE ? OR sp.bill_of_lading ILIKE ? OR sp.tracking_number ILIKE ?",
			"%"+f.Query+"%", "%"+f.Query+"%", "%"+f.Query+"%")
	}

	// Apply additional filters
	if f.Status != "" {
		q = q.Where("sp.status = ?", f.Status)
	}
	if f.CustomerID != uuid.Nil {
		q = q.Where("sp.customer_id = ?", f.CustomerID)
	}
	if f.FromDate != nil {
		q = q.Where("sp.created_at >= ?", f.FromDate)
	}
	if f.ToDate != nil {
		q = q.Where("sp.created_at <= ?", f.ToDate)
	}
	if f.ShipmentTypeID != uuid.Nil {
		q = q.Where("sp.shipment_type_id = ?", f.ShipmentTypeID)
	}

	q = q.OrderExpr("CASE WHEN sp.pro_number = ? THEN 0 ELSE 1 END", f.Query).
		Order("sp.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

func (s ShipmentService) GetAll(ctx context.Context, filter *ShipmentQueryFilter) ([]*models.Shipment, int, error) {
	var entities []*models.Shipment

	q := s.DB.NewSelect().
		Model(&entities).
		Relation("ShipmentMoves").
		Relation("ShipmentMoves.Stops")

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get workers")
		return nil, 0, err
	}

	return entities, count, nil
}

func (s ShipmentService) Get(ctx context.Context, id uuid.UUID, orgID, buID uuid.UUID) (*models.Shipment, error) {
	entity := new(models.Shipment)
	err := s.GetByID(ctx, id, orgID, buID, entity)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to fetch shipment")
		return nil, err
	}

	return entity, nil
}

func (s ShipmentService) AssignTractorToShipment(ctx context.Context, input *types.AssignTractorInput, orgID, buID uuid.UUID) ([]models.TractorAssignment, error) {
	if input.TractorID == uuid.Nil || len(input.Assignments) == 0 {
		return nil, validator.DBValidationError{Message: "tractorId and at least one assignment are required"}
	}

	var assignedAssignments []models.TractorAssignment

	err := s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Fetch all existing assignments for this tractor
		var existingAssignments []models.TractorAssignment
		err := tx.NewSelect().
			Model(&existingAssignments).
			Where("tractor_id = ? AND organization_id = ? AND business_unit_id = ?",
				input.TractorID, orgID, buID).
			Scan(ctx)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to fetch existing assignments")
			return err
		}

		// Create a map of existing assignments for easy lookup
		existingMap := make(map[string]*models.TractorAssignment)
		for i := range existingAssignments {
			key := fmt.Sprintf("%s-%s", existingAssignments[i].ShipmentID, existingAssignments[i].ShipmentMoveID)
			existingMap[key] = &existingAssignments[i]
		}

		// Process new assignments
		for i, assignment := range input.Assignments {
			key := fmt.Sprintf("%s-%s", assignment.ShipmentID, assignment.ShipmentMoveID)
			if existing, ok := existingMap[key]; ok {
				// Update existing assignment
				existing.Sequence = i + 1
				existing.Status = "Active"
				if _, err = tx.NewUpdate().Model(existing).WherePK().Exec(ctx); err != nil {
					s.logger.Error().Err(err).Msg("failed to update assignment sequence")
					return err
				}

				assignedAssignments = append(assignedAssignments, *existing)
				delete(existingMap, key)
			} else {
				// Insert new assignment
				newAssignment := &models.TractorAssignment{
					TractorID:      input.TractorID,
					ShipmentID:     assignment.ShipmentID,
					ShipmentMoveID: assignment.ShipmentMoveID,
					Sequence:       i + 1,
					AssignedByID:   assignment.AssignedByID,
					AssignedAt:     time.Now(),
					Status:         "Active",
					OrganizationID: orgID,
					BusinessUnitID: buID,
				}

				if _, err = tx.NewInsert().Model(newAssignment).Exec(ctx); err != nil {
					s.logger.Error().Err(err).Msg("failed to insert new assignment")
					return err
				}

				assignedAssignments = append(assignedAssignments, *newAssignment)
			}
		}

		// Deactivate assignments that are no longer in the list
		for _, assignment := range existingMap {
			if assignment.Status == "Active" {
				assignment.Status = "Inactive"
				_, err = tx.NewUpdate().
					Model(assignment).
					Set("status = ?", "Inactive").
					WherePK().
					Exec(ctx)
				if err != nil {
					s.logger.Error().Err(err).Msg("failed to deactivate old assignment")
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return assignedAssignments, nil
}

func (s ShipmentService) Create(ctx context.Context, input *types.CreateShipmentInput, userID uuid.UUID) (*models.Shipment, error) {
	shipment := &models.Shipment{
		EntryMethod:           "Manual",
		Status:                property.ShipmentStatusNew,
		BusinessUnitID:        input.BusinessUnitID,
		OrganizationID:        input.OrganizationID,
		CustomerID:            input.CustomerID,
		OriginLocationID:      input.OriginLocationID,
		DestinationLocationID: input.DestinationLocationID,
		ShipmentTypeID:        input.ShipmentTypeID,
		RevenueCodeID:         input.RevenueCodeID,
		ServiceTypeID:         input.ServiceTypeID,
		RatingMethod:          input.RatingMethod,
		RatingUnit:            input.RatingUnit,
		OtherChargeAmount:     input.OtherChargeAmount,
		FreightChargeAmount:   input.FreightChargeamount,
		TotalChargeAmount:     input.TotalChargeAmount,
		Pieces:                input.Pieces,
		Weight:                input.Weight,
		TrailerTypeID:         input.TrailerTypeID,
		TractorTypeID:         input.TractorTypeID,
		TemperatureMin:        input.TemperatureMin,
		TemperatureMax:        input.TemperatureMax,
		BillOfLading:          input.BillOfLading,
		SpecialInstructions:   input.SpecialInstructions,
		TrackingNumber:        input.TrackingNumber,
		Priority:              input.Priority,
		TotalDistance:         input.TotalDistance,
	}

	err := s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Insert the shipment.
		auditUser, err := s.CreateWithAudit(ctx, shipment, userID)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to create Shipment")
			return err
		}

		// Create the initial shipment move.
		move := &models.ShipmentMove{
			BusinessUnitID:    input.BusinessUnitID,
			OrganizationID:    input.OrganizationID,
			ShipmentID:        shipment.ID,
			Status:            property.ShipmentMoveStatusNew,
			IsLoaded:          true, // All first movements are loaded by default.
			SequenceNumber:    1,
			TractorID:         input.TractorID,
			TrailerID:         input.TrailerID,
			PrimaryWorkerID:   input.PrimaryWorkerID,
			SecondaryWorkerID: input.SecondaryWorkerID,
		}

		// Create stops (but don't insert yet)
		stops, err := s.createStops(ctx, tx, shipment, move, input.Stops)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to create stops")
			return err
		}

		move.Stops = stops

		// Validate stop sequence
		if err = move.ValidateStopSequence(); err != nil {
			s.logger.Error().Err(err).Msg("failed to validate stop sequence")
			return err
		}

		// Insert the move
		if _, err = move.Insert(ctx, tx, s.AuditService, auditUser); err != nil {
			s.logger.Error().Err(err).Msg("failed to insert shipment move")
			return err
		}

		// Now insert and audit stops
		for _, stop := range stops {
			stop.ShipmentMoveID = move.ID
			if err = stop.InsertPrepared(ctx, tx); err != nil {
				s.logger.Error().Err(err).Msg("failed to insert stop")
				return err
			}

			s.AuditService.LogAction(
				constants.TableStop,
				stop.ID.String(),
				property.AuditLogActionCreate,
				auditUser,
				stop.OrganizationID,
				stop.BusinessUnitID,
				audit.WithDiff(nil, stop),
			)
		}

		return nil
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to create shipment")
		return nil, err
	}

	return shipment, nil
}

func (s ShipmentService) createStops(ctx context.Context, tx bun.Tx, shipment *models.Shipment, move *models.ShipmentMove, stopInputs []types.StopInput) ([]*models.Stop, error) {
	stops := make([]*models.Stop, len(stopInputs))
	for i, input := range stopInputs {
		location, err := s.getLocation(ctx, tx, input.LocationID)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to fetch location")
			return nil, err
		}

		stop := &models.Stop{
			ShipmentMoveID:   move.ID, // This will be set later
			LocationID:       input.LocationID,
			Type:             input.Type,
			SequenceNumber:   i + 1,
			Status:           property.ShipmentMoveStatusNew,
			PlannedArrival:   input.PlannedArrival,
			PlannedDeparture: input.PlannedDeparture,
			Weight:           input.Weight,
			Pieces:           input.Pieces,
			BusinessUnitID:   shipment.BusinessUnitID,
			OrganizationID:   shipment.OrganizationID,
			AddressLine:      s.consolidateAddress(location),
		}

		// Prepare the stop
		if err := stop.PrepareForInsert(); err != nil {
			s.logger.Error().Err(err).Msg("failed to prepare stop")
			return nil, err
		}

		stops[i] = stop
	}

	// Set origin and destination on shipment
	if len(stops) > 0 {
		shipment.OriginLocationID = stops[0].LocationID
		shipment.DestinationLocationID = stops[len(stops)-1].LocationID
	}

	return stops, nil
}

func (s ShipmentService) getLocation(ctx context.Context, tx bun.Tx, locationID uuid.UUID) (*models.Location, error) {
	location := new(models.Location)
	if err := tx.NewSelect().Model(location).Where("id = ?", locationID).Scan(ctx); err != nil {
		s.logger.Error().Err(err).Msg("failed to fetch location")
		return nil, err
	}
	return location, nil
}

func (s ShipmentService) consolidateAddress(entity *models.Location) string {
	return fmt.Sprintf("%s, %s, %s, %s, %s", entity.AddressLine1, entity.AddressLine2, entity.City, entity.State, entity.PostalCode)
}
