// Package rateengine decides what a shipment costs, and records why.
//
// The engine takes a shipment, finds the contract rule that covers its lane,
// prices it, and writes a quote carrying every input it read, every candidate
// it considered and every step of the arithmetic. That record is not a
// by-product: it is what makes a rate defensible six months later, when the
// contract has been amended, the fuel index revised and the stops edited.
//
// Rating is a pure function of a stored context and stored rules. Nothing here
// reads the clock or the live shipment once the context is built, which is what
// lets a quote be replayed and get the same answer.
package rateengine

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rategeo"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// EngineVersion changes whenever the scoring or the pricing arithmetic changes.
//
// It is stored on every quote so that two runs disagreeing can be told apart:
// the same version with a different context means the shipment moved, and the
// same context with a different version means we did.
const EngineVersion = "1.0.0"

// linehaulLabel is what every path that produces the base charge calls it, so
// the trace reads the same whether the number came from a rate per mile, a
// matrix cell or a formula.
const linehaulLabel = "Linehaul"

// RateContext is everything a rating reads.
//
// It is assembled once, from the shipment and its relations, and then nothing
// else is consulted. Freezing the inputs this way is what makes a rating
// reproducible — stops get edited, mileage engines get upgraded and commodities
// get reclassified, and a replay against today's shipment would answer a
// question nobody asked.
type RateContext struct {
	TenantInfo pagination.TenantInfo

	PartyType rateagreement.PartyType
	PartyID   pulid.ID
	PartyName string

	// AsOf is the date the contract terms are read at. It is not when the
	// rating ran: a shipment picked up last week rates on last week's terms
	// however long the invoice takes to produce.
	AsOf int64

	Origin      rategeo.Place
	Destination rategeo.Place

	OriginLatitude       *float64
	OriginLongitude      *float64
	DestinationLatitude  *float64
	DestinationLongitude *float64

	Distance       decimal.Decimal
	DistanceSource string
	Weight         decimal.Decimal
	Pieces         int64
	Pallets        int64
	LinearFeet     decimal.Decimal
	CubicFeet      decimal.Decimal
	Stops          int16
	Hours          decimal.Decimal

	ServiceTypeID  pulid.ID
	ShipmentTypeID pulid.ID
	TractorTypeID  pulid.ID
	TrailerTypeID  pulid.ID
	ServiceModel   modeprofile.ServiceModel
	EquipmentClass modeprofile.EquipmentClass

	CommodityIDs   []pulid.ID
	FreightClasses []commodity.FreightClass

	HasHazmat           bool
	RequiresTempControl bool

	// BillingCurrency is the currency the organization bills this party in,
	// which a contract written in another currency is converted to at the end.
	BillingCurrency string

	// SellTotal is only set when rating the buy side, for the carrier
	// agreements written as a share of what the customer pays.
	SellTotal decimal.NullDecimal

	// Entity is the shipment itself, needed only by rules that delegate to a
	// formula template — the formula engine resolves its own variables from it.
	Entity any

	// SimulateAgreementID lets one agreement price this shipment whatever its
	// status, which is what a simulation of a draft contract needs.
	SimulateAgreementID *pulid.ID
}

// Weekday is the day of week the rating date falls on, Sunday-zero, matching
// both time.Weekday and the bitmask a rule stores.
func (rc *RateContext) Weekday() int {
	return int(time.Unix(rc.AsOf, 0).UTC().Weekday())
}

// MatchFacts projects the context onto the values a rule is checked against.
func (rc *RateContext) MatchFacts() *rateagreement.MatchFacts {
	return &rateagreement.MatchFacts{
		RatingDate:          rc.AsOf,
		Weekday:             rc.Weekday(),
		ServiceTypeID:       rc.ServiceTypeID,
		ShipmentTypeID:      rc.ShipmentTypeID,
		TractorTypeID:       rc.TractorTypeID,
		TrailerTypeID:       rc.TrailerTypeID,
		CommodityIDs:        rc.CommodityIDs,
		FreightClasses:      rc.FreightClasses,
		ServiceModel:        rc.ServiceModel,
		EquipmentClass:      rc.EquipmentClass,
		Weight:              rc.Weight,
		Distance:            rc.Distance,
		Stops:               rc.Stops,
		HasHazmat:           rc.HasHazmat,
		RequiresTempControl: rc.RequiresTempControl,
	}
}

// LaneKeys is every lane a rule could have been written against to reach this
// shipment.
func (rc *RateContext) LaneKeys() []string {
	return rategeo.LaneKeyCandidates(
		rc.Origin.MatchKeys(),
		rc.Destination.MatchKeys(),
	)
}

// Inputs renders the context for the trace.
func (rc *RateContext) Inputs() ratetypes.Inputs {
	return ratetypes.Inputs{
		RatingDate:            rc.AsOf,
		Weekday:               rc.Weekday(),
		PartyType:             string(rc.PartyType),
		PartyID:               rc.PartyID.String(),
		PartyName:             rc.PartyName,
		OriginLocationID:      rc.Origin.LocationID.String(),
		OriginCity:            rc.Origin.City,
		OriginStateID:         rc.Origin.StateID.String(),
		OriginPostalCode:      rc.Origin.PostalCode,
		OriginZoneIDs:         idStrings(rc.Origin.ZoneIDs),
		DestinationLocationID: rc.Destination.LocationID.String(),
		DestinationCity:       rc.Destination.City,
		DestinationStateID:    rc.Destination.StateID.String(),
		DestinationPostalCode: rc.Destination.PostalCode,
		DestinationZoneIDs:    idStrings(rc.Destination.ZoneIDs),
		Distance:              rc.Distance,
		DistanceSource:        rc.DistanceSource,
		Weight:                rc.Weight,
		Pieces:                rc.Pieces,
		LinearFeet:            rc.LinearFeet,
		CubicFeet:             rc.CubicFeet,
		Stops:                 rc.Stops,
		ServiceTypeID:         rc.ServiceTypeID.String(),
		ShipmentTypeID:        rc.ShipmentTypeID.String(),
		TractorTypeID:         rc.TractorTypeID.String(),
		TrailerTypeID:         rc.TrailerTypeID.String(),
		ServiceModel:          string(rc.ServiceModel),
		EquipmentClass:        string(rc.EquipmentClass),
		CommodityIDs:          idStrings(rc.CommodityIDs),
		FreightClasses:        classStrings(rc.FreightClasses),
		HasHazmat:             rc.HasHazmat,
		RequiresTempControl:   rc.RequiresTempControl,
	}
}

// Hash fingerprints the inputs.
//
// Two ratings with the same hash and the same engine version have to produce
// the same answer, which makes this both a cheap way to skip re-rating an
// unchanged shipment and the signal that tells a changed shipment apart from a
// changed engine.
//
// The fields are written in a fixed order with explicit separators, so a value
// ending where another begins can never collide with a different split of the
// same characters.
func (rc *RateContext) Hash() string {
	var b strings.Builder

	write := func(value string) {
		b.WriteString(value)
		b.WriteByte('\x1f')
	}

	write(string(rc.PartyType))
	write(rc.PartyID.String())
	write(strconv.FormatInt(rc.AsOf, 10))
	write(strings.Join(rc.Origin.MatchKeys(), ","))
	write(strings.Join(rc.Destination.MatchKeys(), ","))
	write(rc.Distance.String())
	write(rc.Weight.String())
	write(strconv.FormatInt(rc.Pieces, 10))
	write(rc.LinearFeet.String())
	write(rc.CubicFeet.String())
	write(strconv.Itoa(int(rc.Stops)))
	write(rc.ServiceTypeID.String())
	write(rc.ShipmentTypeID.String())
	write(rc.TractorTypeID.String())
	write(rc.TrailerTypeID.String())
	write(string(rc.ServiceModel))
	write(string(rc.EquipmentClass))
	write(strings.Join(idStrings(rc.CommodityIDs), ","))
	write(strings.Join(classStrings(rc.FreightClasses), ","))
	write(strconv.FormatBool(rc.HasHazmat))
	write(strconv.FormatBool(rc.RequiresTempControl))
	write(rc.BillingCurrency)

	sum := sha256.Sum256([]byte(b.String()))

	return hex.EncodeToString(sum[:])
}

func idStrings(ids []pulid.ID) []string {
	if len(ids) == 0 {
		return nil
	}

	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}

	return out
}

func classStrings(classes []commodity.FreightClass) []string {
	if len(classes) == 0 {
		return nil
	}

	out := make([]string, 0, len(classes))
	for _, class := range classes {
		out = append(out, class.String())
	}

	return out
}
