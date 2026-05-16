package ediservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/location"
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
		ServiceTypeLabel:         serviceTypeLabel(source),
		ShipmentTypeID:           source.ShipmentTypeID,
		ShipmentTypeLabel:        shipmentTypeLabel(source),
		CustomerID:               source.CustomerID,
		CustomerLabel:            customerLabel(source),
		FormulaTemplateID:        source.FormulaTemplateID,
		FormulaTemplateLabel:     formulaTemplateLabel(source),
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
			locationLabel, locationAddress := stopLocationDetails(stop)
			tenderMove.Stops = append(tenderMove.Stops, edi.LoadTenderStop{
				LocationID:           stop.LocationID,
				LocationLabel:        locationLabel,
				Type:                 string(stop.Type),
				ScheduleType:         string(stop.ScheduleType),
				Sequence:             stop.Sequence,
				Pieces:               stop.Pieces,
				Weight:               stop.Weight,
				ScheduledWindowStart: stop.ScheduledWindowStart,
				ScheduledWindowEnd:   stop.ScheduledWindowEnd,
				AddressLine:          firstNonEmpty(stop.AddressLine, locationAddress),
			})
			if stop.Location != nil {
				tenderMove.Stops[len(tenderMove.Stops)-1].LocationName = stop.Location.Name
				tenderMove.Stops[len(tenderMove.Stops)-1].LocationCode = stop.Location.Code
				tenderMove.Stops[len(tenderMove.Stops)-1].LocationAddressLine1 = stop.Location.AddressLine1
				tenderMove.Stops[len(tenderMove.Stops)-1].LocationAddressLine2 = stop.Location.AddressLine2
				tenderMove.Stops[len(tenderMove.Stops)-1].LocationCity = stop.Location.City
				tenderMove.Stops[len(tenderMove.Stops)-1].LocationPostalCode = stop.Location.PostalCode
				if stop.Location.State != nil {
					tenderMove.Stops[len(tenderMove.Stops)-1].LocationStateCode = stop.Location.State.Abbreviation
				}
			}
			addRequiredID(payload.RequiredMappingEntityIDs, edi.MappingEntityTypeLocation, stop.LocationID)
		}
		payload.Moves = append(payload.Moves, tenderMove)
	}

	for _, commodity := range source.Commodities {
		payload.Commodities = append(payload.Commodities, edi.LoadTenderCommodity{
			CommodityID:          commodity.CommodityID,
			CommodityLabel:       commodityLabel(commodity),
			CommodityName:        commodityName(commodity),
			CommodityDescription: commodityDescription(commodity),
			Weight:               commodity.Weight,
			Pieces:               commodity.Pieces,
		})
		addRequiredID(
			payload.RequiredMappingEntityIDs,
			edi.MappingEntityTypeCommodity,
			commodity.CommodityID,
		)
	}

	for _, charge := range source.AdditionalCharges {
		payload.AdditionalCharges = append(payload.AdditionalCharges, edi.LoadTenderCharge{
			AccessorialChargeID:    charge.AccessorialChargeID,
			AccessorialLabel:       accessorialLabel(charge),
			AccessorialCode:        accessorialCode(charge),
			AccessorialDescription: accessorialDescription(charge),
			Method:                 string(charge.Method),
			Amount:                 charge.Amount,
			Unit:                   charge.Unit,
		})
		addRequiredID(
			payload.RequiredMappingEntityIDs,
			edi.MappingEntityTypeAccessorialCharge,
			charge.AccessorialChargeID,
		)
	}

	return payload
}

func sourceLabelIndex(payload edi.LoadTenderPayload) map[edi.MappingEntityType]map[pulid.ID]string {
	labels := map[edi.MappingEntityType]map[pulid.ID]string{}
	setLabel(labels, edi.MappingEntityTypeCustomer, payload.CustomerID, payload.CustomerLabel)
	setLabel(labels, edi.MappingEntityTypeServiceType, payload.ServiceTypeID, payload.ServiceTypeLabel)
	setLabel(labels, edi.MappingEntityTypeShipmentType, payload.ShipmentTypeID, payload.ShipmentTypeLabel)
	setLabel(
		labels,
		edi.MappingEntityTypeFormulaTemplate,
		payload.FormulaTemplateID,
		payload.FormulaTemplateLabel,
	)

	for _, move := range payload.Moves {
		for _, stop := range move.Stops {
			setLabel(labels, edi.MappingEntityTypeLocation, stop.LocationID, stopLabel(stop))
		}
	}
	for _, commodity := range payload.Commodities {
		setLabel(
			labels,
			edi.MappingEntityTypeCommodity,
			commodity.CommodityID,
			commodity.CommodityLabel,
		)
	}
	for _, charge := range payload.AdditionalCharges {
		setLabel(
			labels,
			edi.MappingEntityTypeAccessorialCharge,
			charge.AccessorialChargeID,
			charge.AccessorialLabel,
		)
	}

	return labels
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

func setLabel(
	labels map[edi.MappingEntityType]map[pulid.ID]string,
	entityType edi.MappingEntityType,
	id pulid.ID,
	label string,
) {
	if id.IsNil() || strings.TrimSpace(label) == "" {
		return
	}
	if _, ok := labels[entityType]; !ok {
		labels[entityType] = map[pulid.ID]string{}
	}
	labels[entityType][id] = label
}

func customerLabel(source *shipment.Shipment) string {
	if source.Customer == nil {
		return ""
	}
	return joinCodeName(source.Customer.Code, source.Customer.Name)
}

func serviceTypeLabel(source *shipment.Shipment) string {
	if source.ServiceType == nil {
		return ""
	}
	return firstNonEmpty(source.ServiceType.Code, source.ServiceType.Description)
}

func shipmentTypeLabel(source *shipment.Shipment) string {
	if source.ShipmentType == nil {
		return ""
	}
	return firstNonEmpty(source.ShipmentType.Code, source.ShipmentType.Description)
}

func formulaTemplateLabel(source *shipment.Shipment) string {
	if source.FormulaTemplate == nil {
		return ""
	}
	return source.FormulaTemplate.Name
}

func stopLocationDetails(stop *shipment.Stop) (string, string) {
	if stop.Location == nil {
		return "", ""
	}
	address := locationAddress(stop.Location)
	label := joinCodeName(stop.Location.Code, stop.Location.Name)
	return firstNonEmpty(label, address), address
}

func stopLabel(stop edi.LoadTenderStop) string {
	return firstNonEmpty(
		stop.LocationLabel,
		joinCodeName(stop.LocationCode, stop.LocationName),
		stop.AddressLine,
		stop.LocationID.String(),
	)
}

func commodityLabel(commodity *shipment.ShipmentCommodity) string {
	if commodity.Commodity == nil {
		return ""
	}
	return firstNonEmpty(commodity.Commodity.Name, commodity.Commodity.Description)
}

func commodityName(commodity *shipment.ShipmentCommodity) string {
	if commodity.Commodity == nil {
		return ""
	}
	return commodity.Commodity.Name
}

func commodityDescription(commodity *shipment.ShipmentCommodity) string {
	if commodity.Commodity == nil {
		return ""
	}
	return commodity.Commodity.Description
}

func accessorialLabel(charge *shipment.AdditionalCharge) string {
	if charge.AccessorialCharge == nil {
		return ""
	}
	return firstNonEmpty(
		joinCodeName(charge.AccessorialCharge.Code, charge.AccessorialCharge.Description),
		charge.AccessorialCharge.Code,
		charge.AccessorialCharge.Description,
	)
}

func accessorialCode(charge *shipment.AdditionalCharge) string {
	if charge.AccessorialCharge == nil {
		return ""
	}
	return charge.AccessorialCharge.Code
}

func accessorialDescription(charge *shipment.AdditionalCharge) string {
	if charge.AccessorialCharge == nil {
		return ""
	}
	return charge.AccessorialCharge.Description
}

func locationAddress(loc *location.Location) string {
	if loc == nil {
		return ""
	}
	state := ""
	if loc.State != nil {
		state = loc.State.Abbreviation
	}
	return strings.Join(nonEmptyStrings(
		loc.AddressLine1,
		loc.AddressLine2,
		strings.Join(nonEmptyStrings(loc.City, state, loc.PostalCode), ", "),
	), ", ")
}

func joinCodeName(code, name string) string {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	if code != "" && name != "" {
		return code + " - " + name
	}
	return firstNonEmpty(code, name)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func nonEmptyStrings(values ...string) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			items = append(items, value)
		}
	}
	return items
}
