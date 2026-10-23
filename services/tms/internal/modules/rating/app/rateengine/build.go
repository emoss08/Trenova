package rateengine

import (
	"context"
	"errors"
	"math"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/modeprofileports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
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
	ctx context.Context,
	req *ratequoteports.RateShipmentRequest,
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
		asOf = entity.RatingAsOf(s.now())
	}

	rateCtx := &RateContext{
		TenantInfo:      req.TenantInfo,
		PartyType:       req.PartyType,
		PartyID:         partyID,
		AsOf:            asOf,
		Origin:          placeOf(entity.ShipperStop()),
		Destination:     placeOf(entity.DeliveryStop()),
		Distance:        entity.RatingDistance(),
		Weight:          entity.RatingWeight(),
		Pieces:          entity.RatingPieces(),
		Stops:           ratedStops(entity),
		LinearFeet:      entity.RatingLinearFeet(),
		CubicFeet:       entity.RatingCubicFeet(),
		ServiceTypeID:   entity.ServiceTypeID,
		ShipmentTypeID:  entity.ShipmentTypeID,
		TractorTypeID:   entity.TractorTypeID,
		TrailerTypeID:   entity.TrailerTypeID,
		CommodityIDs:    commodityIDs(entity),
		FreightClasses:  entity.FreightClasses(),
		HasHazmat:       entity.HasHazardousMaterial(),
		BillingCurrency: tariff.BillingCurrency(entity),
		SellTotal:       req.SellTotal,
		Entity:          entity,

		SimulateAgreementID: req.SimulateAgreementID,
	}

	rateCtx.RequiresTempControl = requiresTempControl(entity)
	rateCtx.PartyName = partyName(entity, req.PartyType)

	applyCoordinates(rateCtx, entity)
	s.applyMode(ctx, rateCtx, entity)

	return rateCtx, nil
}

// applyMode reads the shipment's service model and equipment class from the
// mode profile that already governs it.
//
// A carrier running drayage and truckload publishes two tariffs on the same
// lane, and the mode is the only thing separating them. Rather than adding a
// second mode concept to rating, this reuses the one the rest of the product
// already resolves — the same profile that decides which capabilities apply.
//
// Every failure here leaves the mode blank, which is a real answer rather than
// a fallback: rules that name a mode stop matching, and rules that do not still
// price the shipment. Enforcement is opt-in, so an organization with no
// profiles configured must keep rating exactly as it did, and a capability
// service that is briefly unavailable must not stop shipments being saved.
func (s *Service) applyMode(
	ctx context.Context,
	rateCtx *RateContext,
	entity *shipment.Shipment,
) {
	if s.modeProfile == nil {
		return
	}

	policy, err := s.modeProfile.Resolve(ctx, &modeprofileports.ResolveModeProfileRequest{
		TenantInfo:     rateCtx.TenantInfo,
		CustomerID:     entity.CustomerID,
		ServiceTypeID:  entity.ServiceTypeID,
		ShipmentTypeID: entity.ShipmentTypeID,
		TractorTypeID:  entity.TractorTypeID,
		TrailerTypeID:  entity.TrailerTypeID,
		At:             rateCtx.AsOf,
	})
	if err != nil {
		if !errors.Is(err, modeprofileports.ErrNoModeProfileConfigured) {
			s.l.Warn("failed to resolve mode profile for rating",
				zap.String("shipmentId", entity.ID.String()),
				zap.Error(err),
			)
		}

		return
	}

	if policy == nil {
		return
	}

	rateCtx.ServiceModel = policy.ServiceModel
	rateCtx.EquipmentClass = policy.EquipmentClass
}

func partyFor(req *ratequoteports.RateShipmentRequest) (pulid.ID, error) {
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

func requiresTempControl(entity *shipment.Shipment) bool {
	return entity.TemperatureMin != nil || entity.TemperatureMax != nil
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

func ratedStops(entity *shipment.Shipment) int16 {
	count := entity.RatedStopCount()
	if count > math.MaxInt16 {
		return math.MaxInt16
	}
	return int16(count) //nolint:gosec // bounded by MaxInt16 above
}
