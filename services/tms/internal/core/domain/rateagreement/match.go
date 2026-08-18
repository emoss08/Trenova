package rateagreement

import (
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// MatchFacts is everything about a shipment a rule can be checked against,
// excluding geography — the lane is settled by the database before any of this
// runs.
//
// It is a plain value with no behaviour so that matching stays a pure function
// of stored facts. That is what lets a quote be replayed months later and get
// the same answer, and what lets the whole predicate set be tested without a
// database.
type MatchFacts struct {
	RatingDate int64
	// Weekday is Sunday-zero, matching time.Weekday.
	Weekday int

	ServiceTypeID  pulid.ID
	ShipmentTypeID pulid.ID
	TractorTypeID  pulid.ID
	TrailerTypeID  pulid.ID

	CommodityIDs   []pulid.ID
	FreightClasses []commodity.FreightClass

	ServiceModel   modeprofile.ServiceModel
	EquipmentClass modeprofile.EquipmentClass

	Weight   decimal.Decimal
	Distance decimal.Decimal
	Stops    int16

	HasHazmat           bool
	RequiresTempControl bool
}

// Matches reports whether the rule applies to these facts, and when it does
// not, why.
//
// An empty applicability set means the rule does not care about that dimension,
// which is the convention detention policies and fuel surcharge programs
// already follow. Checks run cheapest first so a rule that is out of date or
// out of service costs almost nothing to reject.
func (rar *RateAgreementRule) Matches(facts *MatchFacts) (bool, ratetypes.RejectReason) {
	if rar.Status != RuleStatusActive {
		return false, ratetypes.RejectReasonRuleInactive
	}

	if !rar.IsEffectiveAt(facts.RatingDate) {
		return false, ratetypes.RejectReasonNotEffective
	}

	if !rar.MatchesDayOfWeek(facts.Weekday) {
		return false, ratetypes.RejectReasonDayOfWeekExcluded
	}

	if rar.HazmatOnly && !facts.HasHazmat {
		return false, ratetypes.RejectReasonHazmatRequired
	}

	if rar.TempControlOnly && !facts.RequiresTempControl {
		return false, ratetypes.RejectReasonTempControlRequired
	}

	if !matchesID(rar.ServiceTypeIDs, facts.ServiceTypeID) {
		return false, ratetypes.RejectReasonServiceTypeMismatch
	}

	if !matchesID(rar.ShipmentTypeIDs, facts.ShipmentTypeID) {
		return false, ratetypes.RejectReasonShipmentTypeMismatch
	}

	if !rar.matchesEquipment(facts) {
		return false, ratetypes.RejectReasonEquipmentMismatch
	}

	if len(rar.ServiceModels) > 0 && !slices.Contains(rar.ServiceModels, facts.ServiceModel) {
		return false, ratetypes.RejectReasonServiceModelMismatch
	}

	if !matchesAnyID(rar.CommodityIDs, facts.CommodityIDs) {
		return false, ratetypes.RejectReasonCommodityMismatch
	}

	if !matchesAnyClass(rar.FreightClasses, facts.FreightClasses) {
		return false, ratetypes.RejectReasonFreightClassMismatch
	}

	if !withinDecimalRange(rar.MinWeight, rar.MaxWeight, facts.Weight) {
		return false, ratetypes.RejectReasonWeightOutOfRange
	}

	if !withinDecimalRange(rar.MinDistance, rar.MaxDistance, facts.Distance) {
		return false, ratetypes.RejectReasonDistanceOutOfRange
	}

	if !withinStopRange(rar.MinStops, rar.MaxStops, facts.Stops) {
		return false, ratetypes.RejectReasonStopsOutOfRange
	}

	return true, ratetypes.RejectReasonNone
}

// matchesEquipment treats the tractor, trailer and equipment class filters as
// one dimension. A rule may narrow by any of them, and the shipment has to
// satisfy every one the rule actually names.
func (rar *RateAgreementRule) matchesEquipment(facts *MatchFacts) bool {
	if !matchesID(rar.TractorTypeIDs, facts.TractorTypeID) {
		return false
	}

	if !matchesID(rar.TrailerTypeIDs, facts.TrailerTypeID) {
		return false
	}

	if len(rar.EquipmentClasses) > 0 &&
		!slices.Contains(rar.EquipmentClasses, facts.EquipmentClass) {
		return false
	}

	return true
}

func matchesID(allowed []pulid.ID, value pulid.ID) bool {
	if len(allowed) == 0 {
		return true
	}

	return slices.Contains(allowed, value)
}

// matchesAnyID is satisfied when the shipment carries at least one of the
// values a rule names. A multi-commodity load rates on a commodity-specific
// rule as soon as one of its commodities qualifies, which is how carriers
// price mixed freight.
func matchesAnyID(allowed, values []pulid.ID) bool {
	if len(allowed) == 0 {
		return true
	}

	for _, value := range values {
		if slices.Contains(allowed, value) {
			return true
		}
	}

	return false
}

func matchesAnyClass(allowed, values []commodity.FreightClass) bool {
	if len(allowed) == 0 {
		return true
	}

	for _, value := range values {
		if slices.Contains(allowed, value) {
			return true
		}
	}

	return false
}

// withinDecimalRange treats both bounds as inclusive.
//
// Weight and distance ranges on a rule are written the way a person reads a
// contract — "5,000 to 10,000 pounds" includes both ends. This differs from the
// half-open weight breaks inside a rule, which are a rating mechanism rather
// than a stated eligibility range, and are documented as such where they are
// defined.
func withinDecimalRange(minimum, maximum decimal.NullDecimal, value decimal.Decimal) bool {
	if minimum.Valid && value.LessThan(minimum.Decimal) {
		return false
	}

	if maximum.Valid && value.GreaterThan(maximum.Decimal) {
		return false
	}

	return true
}

func withinStopRange(minimum, maximum *int16, value int16) bool {
	if minimum != nil && value < *minimum {
		return false
	}

	if maximum != nil && value > *maximum {
		return false
	}

	return true
}
