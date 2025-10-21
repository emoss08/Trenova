package shipmentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type EntityType string

const (
	EntityTypeShipment         EntityType = "shipments"
	EntityTypeMove             EntityType = "moves"
	EntityTypeStop             EntityType = "stops"
	EntityTypeCommodity        EntityType = "commodities"
	EntityTypeAdditionalCharge EntityType = "additional_charges"
	EntityTypeComment          EntityType = "comments"
)

type shipmentBulkData struct {
	shipments         []*shipment.Shipment
	moves             []*shipment.ShipmentMove
	stops             []*shipment.Stop
	commodities       []*shipment.ShipmentCommodity
	additionalCharges []*shipment.AdditionalCharge
	comments          []*shipment.ShipmentComment
}

func (r *repository) bulkInsertShipmentData(
	ctx context.Context,
	tx bun.IDB,
	data *shipmentBulkData,
	log *zap.Logger,
) error {
	if err := r.insertEntities(ctx, tx, log, EntityTypeShipment, &data.shipments); err != nil {
		return err
	}

	if len(data.moves) > 0 {
		if err := r.insertEntities(ctx, tx, log, EntityTypeMove, &data.moves); err != nil {
			return err
		}
	}

	if len(data.stops) > 0 {
		if err := r.insertEntities(ctx, tx, log, EntityTypeStop, &data.stops); err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) insertEntities(
	ctx context.Context,
	tx bun.IDB,
	log *zap.Logger,
	entityType EntityType,
	entities any,
) error {
	if _, err := tx.NewInsert().Model(entities).Exec(ctx); err != nil {
		log.Error(
			"failed to insert entities",
			zap.String("entityType", string(entityType)),
			zap.Error(err),
		)
		return err
	}

	return nil
}

func (r *repository) prepareBulkShipmentData(
	ctx context.Context,
	entity *shipment.Shipment,
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

	proNumbers, err := r.generator.GenerateShipmentProNumberBatch(
		ctx,
		entity.OrganizationID,
		entity.BusinessUnitID,
		req.Count,
	)
	if err != nil {
		return nil, fmt.Errorf("generate shipment pro number batch: %w", err)
	}

	for i := range req.Count {
		newShipment := r.duplicateShipmentFields(entity, proNumbers[i])
		data.shipments = append(data.shipments, newShipment)

		moves, stops := r.prepareMovesAndStops(entity, newShipment, req.OverrideDates)
		data.moves = append(data.moves, moves...)
		data.stops = append(data.stops, stops...)

		if req.IncludeAdditionalCharges {
			charges := r.prepareCharges(entity, newShipment)
			data.additionalCharges = append(data.additionalCharges, charges...)
		}
	}

	return data, nil
}

func (r *repository) duplicateShipmentFields(
	original *shipment.Shipment,
	proNumber string,
) *shipment.Shipment {
	shp := &shipment.Shipment{
		ID:                  pulid.MustNew("shp_"),
		BusinessUnitID:      original.BusinessUnitID,
		OrganizationID:      original.OrganizationID,
		ServiceTypeID:       original.ServiceTypeID,
		ShipmentTypeID:      original.ShipmentTypeID,
		CustomerID:          original.CustomerID,
		TractorTypeID:       original.TractorTypeID,
		TrailerTypeID:       original.TrailerTypeID,
		Status:              shipment.StatusNew,
		ProNumber:           proNumber,
		RatingUnit:          original.RatingUnit,
		OtherChargeAmount:   original.OtherChargeAmount,
		RatingMethod:        original.RatingMethod,
		FreightChargeAmount: original.FreightChargeAmount,
		TotalChargeAmount:   original.TotalChargeAmount,
		Pieces:              original.Pieces,
		Weight:              original.Weight,
		TemperatureMin:      original.TemperatureMin,
		TemperatureMax:      original.TemperatureMax,
		BOL:                 "GENERATED-COPY",
	}

	return shp
}

func (r *repository) prepareMovesAndStops(
	original *shipment.Shipment, newShipment *shipment.Shipment, overrideDates bool,
) ([]*shipment.ShipmentMove, []*shipment.Stop) {
	moves := make([]*shipment.ShipmentMove, 0, len(original.Moves))
	stops := make([]*shipment.Stop, 0)

	for _, originalMove := range original.Moves {
		newMove := &shipment.ShipmentMove{
			ID:             pulid.MustNew("smv_"),
			BusinessUnitID: original.BusinessUnitID,
			OrganizationID: original.OrganizationID,
			ShipmentID:     newShipment.ID,
			Status:         shipment.MoveStatusNew,
			Loaded:         originalMove.Loaded,
			Sequence:       originalMove.Sequence,
			Distance:       originalMove.Distance,
		}
		moves = append(moves, newMove)

		moveStops := r.prepareStops(originalMove, newMove, overrideDates)
		stops = append(stops, moveStops...)
	}

	return moves, stops
}

func (r *repository) prepareStops(
	originalMove *shipment.ShipmentMove, newMove *shipment.ShipmentMove, overrideDates bool,
) []*shipment.Stop {
	stops := make([]*shipment.Stop, 0, len(originalMove.Stops))

	for _, stop := range originalMove.Stops {
		newStop := r.prepareStopForDuplication(stop, newMove.ID, overrideDates)
		stops = append(stops, newStop)
	}

	return stops
}

func (r *repository) prepareStopForDuplication(
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

	if overrideDates {
		now := utils.NowUnix()
		oneDay := utils.DaysToSeconds(1)
		newStop.PlannedArrival = now
		newStop.PlannedDeparture = now + oneDay
	} else {
		newStop.PlannedDeparture = original.PlannedDeparture
	}

	return newStop
}

func (r *repository) prepareCharges(
	original, newShipment *shipment.Shipment,
) []*shipment.AdditionalCharge {
	entities := make([]*shipment.AdditionalCharge, 0, len(original.AdditionalCharges))

	for _, ac := range original.AdditionalCharges {
		newCharge := &shipment.AdditionalCharge{
			ID:                  pulid.MustNew("ac_"),
			BusinessUnitID:      original.BusinessUnitID,
			OrganizationID:      original.OrganizationID,
			ShipmentID:          newShipment.ID,
			AccessorialChargeID: ac.AccessorialChargeID,
			Unit:                ac.Unit,
			Method:              ac.Method,
			Amount:              ac.Amount,
		}

		entities = append(entities, newCharge)
	}

	return entities
}
