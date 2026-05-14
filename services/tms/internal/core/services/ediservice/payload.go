package ediservice

import (
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

func buildTenderPayload(source *shipment.Shipment) edi.LoadTenderPayload {
	payload := edi.LoadTenderPayload{
		ShipmentID:               source.ID,
		BusinessUnitID:           source.BusinessUnitID,
		OrganizationID:           source.OrganizationID,
		ServiceTypeID:            source.ServiceTypeID,
		ShipmentTypeID:           source.ShipmentTypeID,
		CustomerID:               source.CustomerID,
		FormulaTemplateID:        source.FormulaTemplateID,
		BOL:                      source.BOL,
		Pieces:                   source.Pieces,
		Weight:                   source.Weight,
		TemperatureMin:           source.TemperatureMin,
		TemperatureMax:           source.TemperatureMax,
		FreightChargeAmount:      source.FreightChargeAmount,
		OtherChargeAmount:        source.OtherChargeAmount,
		BaseRate:                 source.BaseRate,
		TotalChargeAmount:        source.TotalChargeAmount,
		RatingUnit:               source.RatingUnit,
		Moves:                    make([]edi.LoadTenderMove, 0, len(source.Moves)),
		Commodities:              make([]edi.LoadTenderCommodity, 0, len(source.Commodities)),
		AdditionalCharges:        make([]edi.LoadTenderCharge, 0, len(source.AdditionalCharges)),
		RequiredMappingEntityIDs: map[edi.MappingEntityType][]pulid.ID{},
	}
	if source.RatingDetail != nil {
		payload.RatingDetail = jsonutils.MustToJSON(source.RatingDetail)
	}

	addRequiredID(payload.RequiredMappingEntityIDs, edi.MappingEntityTypeCustomer, source.CustomerID)
	addRequiredID(payload.RequiredMappingEntityIDs, edi.MappingEntityTypeServiceType, source.ServiceTypeID)
	addRequiredID(payload.RequiredMappingEntityIDs, edi.MappingEntityTypeFormulaTemplate, source.FormulaTemplateID)
	addRequiredID(payload.RequiredMappingEntityIDs, edi.MappingEntityTypeShipmentType, source.ShipmentTypeID)

	for _, move := range source.Moves {
		tenderMove := edi.LoadTenderMove{
			Loaded:   move.Loaded,
			Sequence: move.Sequence,
			Distance: move.Distance,
			Stops:    make([]edi.LoadTenderStop, 0, len(move.Stops)),
		}
		for _, stop := range move.Stops {
			tenderMove.Stops = append(tenderMove.Stops, edi.LoadTenderStop{
				LocationID:           stop.LocationID,
				Type:                 string(stop.Type),
				ScheduleType:         string(stop.ScheduleType),
				Sequence:             stop.Sequence,
				Pieces:               stop.Pieces,
				Weight:               stop.Weight,
				ScheduledWindowStart: stop.ScheduledWindowStart,
				ScheduledWindowEnd:   stop.ScheduledWindowEnd,
				AddressLine:          stop.AddressLine,
			})
			addRequiredID(payload.RequiredMappingEntityIDs, edi.MappingEntityTypeLocation, stop.LocationID)
		}
		payload.Moves = append(payload.Moves, tenderMove)
	}

	for _, commodity := range source.Commodities {
		payload.Commodities = append(payload.Commodities, edi.LoadTenderCommodity{
			CommodityID: commodity.CommodityID,
			Weight:      commodity.Weight,
			Pieces:      commodity.Pieces,
		})
		addRequiredID(
			payload.RequiredMappingEntityIDs,
			edi.MappingEntityTypeCommodity,
			commodity.CommodityID,
		)
	}

	for _, charge := range source.AdditionalCharges {
		payload.AdditionalCharges = append(payload.AdditionalCharges, edi.LoadTenderCharge{
			AccessorialChargeID: charge.AccessorialChargeID,
			Method:              string(charge.Method),
			Amount:              charge.Amount,
			Unit:                charge.Unit,
		})
		addRequiredID(
			payload.RequiredMappingEntityIDs,
			edi.MappingEntityTypeAccessorialCharge,
			charge.AccessorialChargeID,
		)
	}

	return payload
}

func addRequiredID(
	required map[edi.MappingEntityType][]pulid.ID,
	entityType edi.MappingEntityType,
	id pulid.ID,
) {
	if id.IsNil() {
		return
	}
	for _, existing := range required[entityType] {
		if existing == id {
			return
		}
	}
	required[entityType] = append(required[entityType], id)
}
