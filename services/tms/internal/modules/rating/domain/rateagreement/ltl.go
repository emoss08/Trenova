package rateagreement

import (
	"errors"

	"github.com/shopspring/decimal"
)

// ErrNoBreakForWeight is returned when no band covers the shipment's weight,
// which can only happen on a break set that failed validation.
var ErrNoBreakForWeight = errors.New("no weight break covers the shipment weight")

// poundsPerHundredweight is the divisor that turns pounds into the units a
// hundredweight rate multiplies.
var poundsPerHundredweight = decimal.NewFromInt(100)

// BreakResult is the outcome of pricing a weight against a set of bands.
type BreakResult struct {
	// UsedBreak is the band the charge was actually taken from, which is not
	// the band the shipment's weight falls in when bumping applied.
	UsedBreak *RateAgreementRuleBreak
	// RatedWeight is the weight the charge was computed on: the actual weight,
	// or the minimum of a heavier band when bumping to it came out cheaper.
	RatedWeight decimal.Decimal
	// ActualWeight is what the shipment really weighs.
	ActualWeight decimal.Decimal
	// Bumped records that the shipment was rated as if it were heavier.
	Bumped bool
	// ActualBreak is the band the real weight falls in, kept so a trace can
	// show what was given up.
	ActualBreak *RateAgreementRuleBreak
	// Charge is the resulting amount before discounts and guardrails.
	Charge decimal.Decimal
}

// ApplyDeficitRating prices a weight against a rule's bands, applying the
// deficit rule when it helps the shipper.
//
// Class tariffs get cheaper per hundredweight as freight gets heavier, which
// means a shipment near the top of a band can cost more than the same freight
// declared at the bottom of the next one up. Carriers have always let the
// shipper pay the cheaper of the two — it is called deficit rating, or bumping,
// or rating "as though" at the next break. Mid-market systems routinely skip
// it, and shippers routinely notice.
//
// Every heavier band is considered, not just the next one, because with three
// or more bands the cheapest is not always the adjacent one.
func ApplyDeficitRating(
	breaks []*RateAgreementRuleBreak,
	actualWeight decimal.Decimal,
	allowDeficit bool,
) (BreakResult, error) {
	sorted := SortedBreaks(breaks)
	if len(sorted) == 0 {
		return BreakResult{}, ErrNoBreakForWeight
	}

	actualIndex := -1
	for i, ruleBreak := range sorted {
		if ruleBreak.Contains(actualWeight) {
			actualIndex = i
			break
		}
	}

	if actualIndex < 0 {
		return BreakResult{}, ErrNoBreakForWeight
	}

	actualBreak := sorted[actualIndex]
	best := BreakResult{
		UsedBreak:    actualBreak,
		RatedWeight:  actualWeight,
		ActualWeight: actualWeight,
		ActualBreak:  actualBreak,
		Charge:       breakCharge(actualBreak, actualWeight),
	}

	if !allowDeficit {
		return best, nil
	}

	for _, heavier := range sorted[actualIndex+1:] {
		// Rating at a heavier band means paying as though the shipment weighed
		// that band's minimum, which is the least weight that qualifies for it.
		candidateCharge := breakCharge(heavier, heavier.FromWeight)
		if candidateCharge.GreaterThanOrEqual(best.Charge) {
			continue
		}

		best.UsedBreak = heavier
		best.RatedWeight = heavier.FromWeight
		best.Charge = candidateCharge
		best.Bumped = true
	}

	return best, nil
}

// breakCharge prices a weight at one band's rate, honouring that band's own
// minimum charge if it carries one.
func breakCharge(
	ruleBreak *RateAgreementRuleBreak,
	weight decimal.Decimal,
) decimal.Decimal {
	charge := ruleBreak.Rate.Mul(weight.Div(poundsPerHundredweight))

	if ruleBreak.MinCharge.Valid && charge.LessThan(ruleBreak.MinCharge.Decimal) {
		return ruleBreak.MinCharge.Decimal
	}

	return charge
}

// Density is weight over volume in pounds per cubic foot, which is what a
// density scale classifies.
//
// A zero or negative volume yields no density rather than an error: dimensions
// are frequently missing on entry, and a rule that wanted density has to fall
// back to the commodity's class rather than refuse the shipment.
func Density(weightPounds, cubicFeet decimal.Decimal) (decimal.Decimal, bool) {
	if cubicFeet.LessThanOrEqual(decimal.Zero) || weightPounds.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, false
	}

	return weightPounds.Div(cubicFeet), true
}
