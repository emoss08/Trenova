package rateagreement

import (
	"slices"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// SameTerms reports whether two rules state the same contract terms.
//
// It compares everything a person edits and nothing the system derives:
// identity, tenancy, lane key, specificity, lineage and timestamps are all
// consequences of the compared fields or of who saved them, and including any
// would make every save look like a change. Decimals compare by value rather
// than representation, because "2.55" and "2.550000" are the same rate however
// the column returned it.
//
// This is what decides whether a save amends a lane: an edit that changes no
// term leaves the rule untouched, keeping its history free of amendments that
// amended nothing.
func (rar *RateAgreementRule) SameTerms(other *RateAgreementRule) bool {
	if rar == nil || other == nil {
		return rar == other
	}

	return rar.sameLane(other) &&
		rar.sameApplicability(other) &&
		rar.samePricing(other) &&
		rar.sameWindow(other) &&
		sameBreakTerms(rar.Breaks, other.Breaks)
}

func (rar *RateAgreementRule) sameLane(other *RateAgreementRule) bool {
	return rar.Label == other.Label &&
		rar.Status == other.Status &&
		rar.Direction == other.Direction &&
		rar.OriginScopeType == other.OriginScopeType &&
		rar.OriginScopeValue == other.OriginScopeValue &&
		rar.OriginCity == other.OriginCity &&
		rar.DestinationScopeType == other.DestinationScopeType &&
		rar.DestinationScopeValue == other.DestinationScopeValue &&
		rar.DestinationCity == other.DestinationCity &&
		sameFloatPtr(rar.OriginRadiusMeters, other.OriginRadiusMeters) &&
		sameFloatPtr(rar.DestinationRadiusMeters, other.DestinationRadiusMeters) &&
		sameFloatPtr(rar.OriginLatitude, other.OriginLatitude) &&
		sameFloatPtr(rar.OriginLongitude, other.OriginLongitude) &&
		sameFloatPtr(rar.DestinationLatitude, other.DestinationLatitude) &&
		sameFloatPtr(rar.DestinationLongitude, other.DestinationLongitude)
}

func (rar *RateAgreementRule) sameApplicability(other *RateAgreementRule) bool {
	return sameIDSet(rar.ServiceTypeIDs, other.ServiceTypeIDs) &&
		sameIDSet(rar.ShipmentTypeIDs, other.ShipmentTypeIDs) &&
		sameIDSet(rar.TractorTypeIDs, other.TractorTypeIDs) &&
		sameIDSet(rar.TrailerTypeIDs, other.TrailerTypeIDs) &&
		sameIDSet(rar.CommodityIDs, other.CommodityIDs) &&
		sameEnumSet(rar.FreightClasses, other.FreightClasses) &&
		sameEnumSet(rar.ServiceModels, other.ServiceModels) &&
		sameEnumSet(rar.EquipmentClasses, other.EquipmentClasses) &&
		sameNullDecimal(rar.MinWeight, other.MinWeight) &&
		sameNullDecimal(rar.MaxWeight, other.MaxWeight) &&
		sameNullDecimal(rar.MinDistance, other.MinDistance) &&
		sameNullDecimal(rar.MaxDistance, other.MaxDistance) &&
		sameInt16Ptr(rar.MinStops, other.MinStops) &&
		sameInt16Ptr(rar.MaxStops, other.MaxStops) &&
		rar.DaysOfWeek == other.DaysOfWeek &&
		rar.HazmatOnly == other.HazmatOnly &&
		rar.TempControlOnly == other.TempControlOnly
}

func (rar *RateAgreementRule) samePricing(other *RateAgreementRule) bool {
	return samePulidPtr(rar.FormulaTemplateID, other.FormulaTemplateID) &&
		samePulidPtr(rar.RateMatrixID, other.RateMatrixID) &&
		sameNullDecimal(rar.Rate, other.Rate) &&
		rar.Currency == other.Currency &&
		rar.FreightClassSource == other.FreightClassSource &&
		rar.FixedFreightClass == other.FixedFreightClass &&
		samePulidPtr(rar.DensityScaleID, other.DensityScaleID) &&
		sameNullDecimal(rar.DiscountPercent, other.DiscountPercent) &&
		sameNullDecimal(rar.AbsoluteMinCharge, other.AbsoluteMinCharge) &&
		rar.AllowDeficitRating == other.AllowDeficitRating &&
		sameNullDecimal(rar.MinCharge, other.MinCharge) &&
		sameNullDecimal(rar.MaxCharge, other.MaxCharge) &&
		sameNullDecimal(rar.MinBillableDistance, other.MinBillableDistance) &&
		rar.RoundingMode == other.RoundingMode
}

func (rar *RateAgreementRule) sameWindow(other *RateAgreementRule) bool {
	return rar.Priority == other.Priority &&
		rar.EffectiveFrom == other.EffectiveFrom &&
		sameInt64Ptr(rar.EffectiveTo, other.EffectiveTo)
}

func sameBreakTerms(a, b []*RateAgreementRuleBreak) bool {
	sortedA := SortedBreaks(a)
	sortedB := SortedBreaks(b)

	if len(sortedA) != len(sortedB) {
		return false
	}

	for i := range sortedA {
		left, right := sortedA[i], sortedB[i]
		if !left.FromWeight.Equal(right.FromWeight) ||
			!sameNullDecimal(left.ToWeight, right.ToWeight) ||
			!left.Rate.Equal(right.Rate) ||
			!sameNullDecimal(left.MinCharge, right.MinCharge) ||
			left.Label != right.Label {
			return false
		}
	}

	return true
}

func sameNullDecimal(a, b decimal.NullDecimal) bool {
	if a.Valid != b.Valid {
		return false
	}

	return !a.Valid || a.Decimal.Equal(b.Decimal)
}

func samePulidPtr(a, b *pulid.ID) bool {
	aNil := a == nil || a.IsNil()
	bNil := b == nil || b.IsNil()

	if aNil || bNil {
		return aNil == bNil
	}

	return *a == *b
}

func sameFloatPtr(a, b *float64) bool {
	if a == nil || b == nil {
		return (a == nil) == (b == nil)
	}

	return *a == *b
}

func sameInt16Ptr(a, b *int16) bool {
	if a == nil || b == nil {
		return (a == nil) == (b == nil)
	}

	return *a == *b
}

func sameInt64Ptr(a, b *int64) bool {
	if a == nil || b == nil {
		return (a == nil) == (b == nil)
	}

	return *a == *b
}

func sameIDSet(a, b []pulid.ID) bool {
	if len(a) != len(b) {
		return false
	}

	return slices.Equal(a, b)
}

func sameEnumSet[T ~string](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	return slices.Equal(a, b)
}
