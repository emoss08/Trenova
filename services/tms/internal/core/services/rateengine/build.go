package rateengine

import (
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rategeo"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// ErrNoParty is returned when a rating has nobody to price against — a shipment
// without a customer on the sell side, or without a carrier on the buy side.
var ErrNoParty = errors.New("rating has no party to price against")

// buildContext freezes everything a rating will read.
//
// Once this returns, nothing else about the shipment is consulted. That is the
// property the whole engine rests on: a quote records this context, and
// replaying it later gives the same answer even though the stops have been
// edited, the mileage engine upgraded and the commodity reclassified since.
func (s *Service) buildContext(
	req *services.RateShipmentRequest,
) (*RateContext, error) {
	entity := req.Shipment
	if entity == nil {
		return nil, ErrNoParty
	}

	partyID, err := partyFor(req)
	if err != nil {
		return nil, err
	}

	asOf := req.AsOf
	if asOf == 0 {
		asOf = ratingDate(entity, s.now)
	}

	rateCtx := &RateContext{
		TenantInfo:      req.TenantInfo,
		PartyType:       req.PartyType,
		PartyID:         partyID,
		AsOf:            asOf,
		Origin:          placeOf(entity.ShipperStop()),
		Destination:     placeOf(entity.DeliveryStop()),
		Distance:        totalDistance(entity),
		Weight:          totalWeight(entity),
		Pieces:          totalPieces(entity),
		Stops:           countStops(entity),
		LinearFeet:      totalLinearFeet(entity),
		CubicFeet:       cubicFeet(entity),
		ServiceTypeID:   entity.ServiceTypeID,
		ShipmentTypeID:  entity.ShipmentTypeID,
		TractorTypeID:   entity.TractorTypeID,
		TrailerTypeID:   entity.TrailerTypeID,
		CommodityIDs:    commodityIDs(entity),
		FreightClasses:  freightClasses(entity),
		HasHazmat:       hasHazmat(entity),
		BillingCurrency: billingCurrency(entity),
		SellTotal:       req.SellTotal,
		Entity:          entity,
	}

	rateCtx.RequiresTempControl = requiresTempControl(entity)
	rateCtx.PartyName = partyName(entity, req.PartyType)

	applyCoordinates(rateCtx, entity)

	return rateCtx, nil
}

func partyFor(req *services.RateShipmentRequest) (pulid.ID, error) {
	if req.PartyType == rateagreement.PartyTypeCarrier {
		if req.PartyID.IsNil() {
			return pulid.Nil, ErrNoParty
		}

		return req.PartyID, nil
	}

	// A shipment with no customer yet is not an engine failure. Every shipment
	// needs one and the validator says so far more clearly than this could, so
	// rating treats it as a lane no contract covers and lets the save fail on
	// the field that is actually wrong.
	return req.Shipment.CustomerID, nil
}

func partyName(entity *shipment.Shipment, partyType rateagreement.PartyType) string {
	if partyType == rateagreement.PartyTypeCustomer && entity.Customer != nil {
		return entity.Customer.Name
	}

	return ""
}

// ratingDate is when the contract terms are read at.
//
// The actual ship date leads because that is the day the service was performed,
// and a contract amended between then and invoicing must not change what was
// already earned. It matches how the existing commercial calculator picks its
// rating date, so a shipment does not silently rate against two different days.
func ratingDate(entity *shipment.Shipment, now func() int64) int64 {
	if entity.ActualShipDate != nil && *entity.ActualShipDate > 0 {
		return *entity.ActualShipDate
	}

	if entity.CreatedAt > 0 {
		return entity.CreatedAt
	}

	return now()
}

// placeOf reads one end of the lane from a stop.
//
// The stop's location must be preloaded. Without it the place carries only the
// location id, which still matches a location-scoped rule but nothing broader —
// so the rating quietly falls through to a wider rate. The caller preloads it;
// this is noted because the failure is silent rather than loud.
func placeOf(stop *shipment.Stop) rategeo.Place {
	if stop == nil {
		return rategeo.Place{}
	}

	place := rategeo.Place{LocationID: stop.LocationID}

	if stop.Location == nil {
		return place
	}

	place.PostalCode = stop.Location.PostalCode
	place.City = stop.Location.City
	place.StateID = stop.Location.StateID

	if stop.Location.State != nil {
		place.CountryISO3 = stop.Location.State.CountryIso3
	}

	return place
}

func applyCoordinates(rateCtx *RateContext, entity *shipment.Shipment) {
	if origin := entity.ShipperStop(); origin != nil && origin.Location != nil {
		rateCtx.OriginLatitude = origin.Location.Latitude
		rateCtx.OriginLongitude = origin.Location.Longitude
	}

	if destination := entity.DeliveryStop(); destination != nil &&
		destination.Location != nil {
		rateCtx.DestinationLatitude = destination.Location.Latitude
		rateCtx.DestinationLongitude = destination.Location.Longitude
	}
}

func totalDistance(entity *shipment.Shipment) decimal.Decimal {
	total := decimal.Zero

	for _, move := range entity.Moves {
		if move == nil || move.Distance == nil {
			continue
		}
		total = total.Add(decimal.NewFromFloat(*move.Distance))
	}

	return total
}

// totalWeight prefers the commodities' own weights and falls back to the
// shipment header, which is what a load entered without a commodity breakdown
// carries.
func totalWeight(entity *shipment.Shipment) decimal.Decimal {
	total := decimal.Zero

	for _, item := range entity.Commodities {
		if item == nil {
			continue
		}
		total = total.Add(decimal.NewFromInt(item.Weight))
	}

	if total.IsPositive() {
		return total
	}

	if entity.Weight != nil {
		return decimal.NewFromInt(*entity.Weight)
	}

	return decimal.Zero
}

func totalPieces(entity *shipment.Shipment) int64 {
	var total int64

	for _, item := range entity.Commodities {
		if item == nil {
			continue
		}
		total += item.Pieces
	}

	if total > 0 {
		return total
	}

	if entity.Pieces != nil {
		return *entity.Pieces
	}

	return 0
}

func countStops(entity *shipment.Shipment) int16 {
	var total int16

	for _, move := range entity.Moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop == nil || stop.Status == shipment.StopStatusCanceled {
				continue
			}
			total++
		}
	}

	return total
}

func totalLinearFeet(entity *shipment.Shipment) decimal.Decimal {
	total := decimal.Zero

	for _, item := range entity.Commodities {
		if item == nil || item.Commodity == nil ||
			item.Commodity.LinearFeetPerUnit == nil {
			continue
		}

		perUnit := decimal.NewFromFloat(*item.Commodity.LinearFeetPerUnit)
		total = total.Add(perUnit.Mul(decimal.NewFromInt(item.Pieces)))
	}

	return total
}

// cubicFeet is what density classification divides weight by.
//
// The shipment's own load envelope is preferred because it describes the space
// the freight actually occupies on the trailer. Falling back to summing the
// commodities' dimensions is less accurate — it ignores how the freight stacks
// — but it is what a rater would do with the numbers available.
func cubicFeet(entity *shipment.Shipment) decimal.Decimal {
	if envelope := envelopeCubicFeet(entity); envelope.IsPositive() {
		return envelope
	}

	total := decimal.Zero

	for _, item := range entity.Commodities {
		if item == nil {
			continue
		}

		volume := commodityCubicFeet(item)
		if volume.IsPositive() {
			total = total.Add(volume.Mul(decimal.NewFromInt(item.Pieces)))
		}
	}

	return total
}

func envelopeCubicFeet(entity *shipment.Shipment) decimal.Decimal {
	if entity.EnvelopeLengthFeet == nil || entity.EnvelopeWidthFeet == nil ||
		entity.EnvelopeHeightFeet == nil {
		return decimal.Zero
	}

	return decimal.NewFromFloat(*entity.EnvelopeLengthFeet).
		Mul(decimal.NewFromFloat(*entity.EnvelopeWidthFeet)).
		Mul(decimal.NewFromFloat(*entity.EnvelopeHeightFeet))
}

func commodityCubicFeet(item *shipment.ShipmentCommodity) decimal.Decimal {
	length, width, height := itemDimensions(item)
	if length == nil || width == nil || height == nil {
		return decimal.Zero
	}

	return decimal.NewFromFloat(*length).
		Mul(decimal.NewFromFloat(*width)).
		Mul(decimal.NewFromFloat(*height))
}

// itemDimensions prefers the dimensions recorded on this shipment's line over
// the commodity's catalogue defaults, since the line describes what actually
// shipped.
func itemDimensions(item *shipment.ShipmentCommodity) (length, width, height *float64) {
	length, width, height = item.LengthFeet, item.WidthFeet, item.HeightFeet
	if length != nil && width != nil && height != nil {
		return length, width, height
	}

	if item.Commodity == nil {
		return nil, nil, nil
	}

	return item.Commodity.LengthPerUnit,
		item.Commodity.WidthPerUnit,
		item.Commodity.HeightPerUnit
}

func commodityIDs(entity *shipment.Shipment) []pulid.ID {
	ids := make([]pulid.ID, 0, len(entity.Commodities))

	for _, item := range entity.Commodities {
		if item == nil || item.CommodityID.IsNil() {
			continue
		}
		ids = append(ids, item.CommodityID)
	}

	return ids
}

func freightClasses(entity *shipment.Shipment) []commodity.FreightClass {
	classes := make([]commodity.FreightClass, 0, len(entity.Commodities))
	seen := make(map[commodity.FreightClass]struct{}, len(entity.Commodities))

	for _, item := range entity.Commodities {
		if item == nil || item.Commodity == nil || item.Commodity.FreightClass == "" {
			continue
		}

		class := item.Commodity.FreightClass
		if _, dup := seen[class]; dup {
			continue
		}
		seen[class] = struct{}{}
		classes = append(classes, class)
	}

	return classes
}

func hasHazmat(entity *shipment.Shipment) bool {
	for _, item := range entity.Commodities {
		if item == nil || item.Commodity == nil {
			continue
		}
		if !item.Commodity.HazardousMaterialID.IsNil() {
			return true
		}
	}

	return false
}

func requiresTempControl(entity *shipment.Shipment) bool {
	return entity.TemperatureMin != nil || entity.TemperatureMax != nil
}

// billingCurrency is what the organization bills this customer in, which a
// contract written in another currency is converted to at the end.
func billingCurrency(entity *shipment.Shipment) string {
	if entity.Customer != nil && entity.Customer.BillingProfile != nil &&
		entity.Customer.BillingProfile.BillingCurrency != "" {
		return entity.Customer.BillingProfile.BillingCurrency
	}

	return money.DefaultCurrencyCode
}

// LocationOf exposes the place a stop describes, for callers that need to build
// a context without a shipment — a quote for a lane nobody has booked yet.
func LocationOf(loc *location.Location) rategeo.Place {
	if loc == nil {
		return rategeo.Place{}
	}

	place := rategeo.Place{
		LocationID: loc.ID,
		PostalCode: loc.PostalCode,
		City:       loc.City,
		StateID:    loc.StateID,
	}

	if loc.State != nil {
		place.CountryISO3 = loc.State.CountryIso3
	}

	return place
}
