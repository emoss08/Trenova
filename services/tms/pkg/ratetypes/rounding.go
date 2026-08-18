// Package ratetypes holds the value types the rating domain and the rating
// engine both depend on.
//
// It exists for the same reason formulatypes does: the domain packages describe
// what is stored and the service packages describe what is computed, and a
// handful of types belong to both. Putting them here keeps the domain free of
// any dependency on the engine.
package ratetypes

import "github.com/shopspring/decimal"

// RoundingMode says how a computed charge is reduced to its billable precision.
//
// Contracts specify this and they do not all specify it the same way. A tariff
// that says "rounded up to the nearest cent" and one that says "rounded to the
// nearest cent" disagree on half the shipments they price, and the difference
// is a dispute rather than a rounding detail.
type RoundingMode string

const (
	RoundingModeHalfUp   = RoundingMode("HalfUp")
	RoundingModeHalfEven = RoundingMode("HalfEven")
	RoundingModeUp       = RoundingMode("Up")
	RoundingModeDown     = RoundingMode("Down")
	RoundingModeNone     = RoundingMode("None")
)

func (rm RoundingMode) String() string {
	return string(rm)
}

func (rm RoundingMode) IsValid() bool {
	switch rm {
	case RoundingModeHalfUp,
		RoundingModeHalfEven,
		RoundingModeUp,
		RoundingModeDown,
		RoundingModeNone:
		return true
	default:
		return false
	}
}

// Round applies the mode at the given number of decimal places.
//
// An unset mode rounds half up, which is what a contract means when it says
// nothing: it is the convention every North American tariff assumes.
func (rm RoundingMode) Round(value decimal.Decimal, places int32) decimal.Decimal {
	switch rm {
	case RoundingModeNone:
		return value
	case RoundingModeHalfEven:
		return value.RoundBank(places)
	case RoundingModeUp:
		return value.RoundCeil(places)
	case RoundingModeDown:
		return value.RoundFloor(places)
	case RoundingModeHalfUp:
		return value.Round(places)
	default:
		return value.Round(places)
	}
}
