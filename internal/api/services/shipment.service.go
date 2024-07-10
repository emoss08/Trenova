package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/gen"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type ShipmentService struct {
	db      *bun.DB
	logger  *zerolog.Logger
	codeGen *gen.CodeGenerator
}

func NewShipmentService(s *server.Server) *ShipmentService {
	return &ShipmentService{
		db:      s.DB,
		logger:  s.Logger,
		codeGen: s.CodeGenerator,
	}
}

type StopInput struct {
	LocationID       uuid.UUID           `json:"locationId"`
	Type             property.StopType   `json:"type"`
	PlannedArrival   time.Time           `json:"plannedArrival"`
	PlannedDeparture time.Time           `json:"plannedDeparture"`
	Weight           decimal.NullDecimal `json:"weight"`
	Pieces           decimal.NullDecimal `json:"pieces"`
}

type CreateShipmentInput struct {
	BusinessUnitID              uuid.UUID                     `json:"businessUnitId"`
	OrganizationID              uuid.UUID                     `json:"organizationId"`
	CustomerID                  uuid.UUID                     `json:"customerId"`
	OriginLocationID            uuid.UUID                     `json:"originLocationId"`
	OriginPlannedArrival        time.Time                     `json:"originPlannedArrival"`
	OriginPlannedDeparture      time.Time                     `json:"originPlannedDeparture"`
	DestinationLocationID       uuid.UUID                     `json:"destinationLocationId"`
	DestinationPlannedArrival   time.Time                     `json:"destinationPlannedArrival"`
	DestinationPlannedDeparture time.Time                     `json:"destinationPlannedDeparture"`
	ShipmentTypeID              uuid.UUID                     `json:"shipmentTypeId"`
	RevenueCodeID               *uuid.UUID                    `json:"revenueCodeId"`
	ServiceTypeID               *uuid.UUID                    `json:"serviceTypeId"`
	RatingMethod                property.ShipmentRatingMethod `json:"ratingMethod"`
	RatingUnit                  int                           `json:"ratingUnit"`
	OtherChargeAmount           decimal.Decimal               `json:"otherChargeAmount"`
	FreightChargeamount         decimal.Decimal               `json:"freightChargeAmount"`
	TotalChargeAmount           decimal.Decimal               `json:"totalChargeAmount"`
	Pieces                      decimal.NullDecimal           `json:"pieces"`
	Weight                      decimal.NullDecimal           `json:"weight"`
	TractorID                   uuid.UUID                     `json:"tractorId"`
	TrailerID                   uuid.UUID                     `json:"trailerId"`
	PrimaryWorkerID             uuid.UUID                     `json:"primaryWorkerId"`
	SecondaryWorkerID           *uuid.UUID                    `json:"secondaryWorkerId"`
	TrailerTypeID               *uuid.UUID                    `json:"trailerTypeId"`
	TractorTypeID               *uuid.UUID                    `json:"tractorTypeId"`
	TemperatureMin              int                           `json:"temperatureMin"`
	TemperatureMax              int                           `json:"temperatureMax"`
	BillOfLading                string                        `json:"billOfLading"`
	SpecialInstructions         string                        `json:"specialInstructions"`
	TrackingNumber              string                        `json:"trackingNumber"`
	Priority                    int                           `json:"priority"`
	TotalDistance               decimal.NullDecimal           `json:"totalDistance"`
	Stops                       []StopInput                   `json:"stops"`
}

// Suggested additions to ShipmentQueryFilter
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

func (s *ShipmentService) filterQuery(q *bun.SelectQuery, f *ShipmentQueryFilter) *bun.SelectQuery {
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

func (s *ShipmentService) GetAll(ctx context.Context, filter *ShipmentQueryFilter) ([]*models.Shipment, int, error) {
	var entities []*models.Shipment

	q := s.db.NewSelect().
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

func (s *ShipmentService) Get(ctx context.Context, id uuid.UUID, orgID, buID uuid.UUID) (*models.Shipment, error) {
	entity := new(models.Shipment)
	err := s.db.NewSelect().
		Model(entity).
		Where("sp.organization_id = ?", orgID).
		Where("sp.business_unit_id = ?", buID).
		Where("sp.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

type AssignTractorInput struct {
	TractorID  uuid.UUID `json:"tractorId"`
	ShipmentID uuid.UUID `json:"shipmentId"`
}

func (s *ShipmentService) AssignTractorToShipment(ctx context.Context, input *AssignTractorInput, orgID, buID uuid.UUID) error {
	if input.ShipmentID == uuid.Nil || input.TractorID == uuid.Nil {
		return validator.DBValidationError{Message: "shipmentId and tractorId are required"}
	}

	shipment, err := s.Get(ctx, input.ShipmentID, orgID, buID)
	if err != nil {
		return err
	}

	if err = shipment.AssignTractorToMovement(ctx, s.db, input.TractorID); err != nil {
		var dbErr validator.DBValidationError
		if errors.As(err, &dbErr) {
			return dbErr
		}

		return err
	}
	return nil
}

func (s *ShipmentService) Create(ctx context.Context, input *CreateShipmentInput) (*models.Shipment, error) {
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

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Insert the shipment.
		if err := shipment.InsertShipment(ctx, s.db); err != nil {
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

		// Create stops
		stops, err := s.createStops(ctx, tx, shipment, move, input.Stops)
		if err != nil {
			return err
		}

		move.Stops = stops

		// Validate stop sequence
		if err = move.ValidateStopSequence(); err != nil {
			return err
		}

		// Insert the move
		if _, err = tx.NewInsert().Model(move).Exec(ctx); err != nil {
			return err
		}

		// Insert all stops
		if _, err = tx.NewInsert().Model(&stops).Exec(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return shipment, nil
}

func (s *ShipmentService) createStops(ctx context.Context, tx bun.Tx, shipment *models.Shipment, move *models.ShipmentMove, stopInputs []StopInput) ([]*models.Stop, error) {
	stops := make([]*models.Stop, len(stopInputs))
	for i, input := range stopInputs {
		location, err := s.getLocation(ctx, tx, input.LocationID)
		if err != nil {
			return nil, err
		}

		stop := &models.Stop{
			ShipmentMoveID:   move.ID,
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

		// Validate the stop
		if err = stop.Validate(); err != nil {
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

func (s *ShipmentService) getLocation(ctx context.Context, tx bun.Tx, locationID uuid.UUID) (*models.Location, error) {
	location := new(models.Location)
	err := tx.NewSelect().Model(location).Where("id = ?", locationID).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch location: %w", err)
	}
	return location, nil
}

func (s *ShipmentService) consolidateAddress(entity *models.Location) string {
	return fmt.Sprintf("%s, %s, %s, %s, %s", entity.AddressLine1, entity.AddressLine2, entity.City, entity.State, entity.PostalCode)
}
