package formulatypes

import (
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/shopspring/decimal"
)

const (
	DefaultRoundingPrecision int32 = 2
	MaxRoundingPrecision     int32 = 4
)

// ChargePolicy is everything a template does to a raw evaluation before the
// number is billable: clamp it to its guardrails, then round it. It travels
// with the template, its snapshots, a Studio preview, and a saved scenario, so
// every one of them lands on the same cents.
type ChargePolicy struct {
	MinCharge         decimal.NullDecimal
	MaxCharge         decimal.NullDecimal
	RoundingMode      ratetypes.RoundingMode
	RoundingPrecision int32
}

// Normalized fills in the default rounding for a policy that never set one.
// An empty mode means "nobody chose", and what nobody chose is half-up to the
// cent; a precision of zero next to a chosen mode is a real choice and kept.
func (p ChargePolicy) Normalized() ChargePolicy {
	if p.RoundingMode == "" {
		p.RoundingMode = ratetypes.RoundingModeHalfUp
		p.RoundingPrecision = DefaultRoundingPrecision
	}
	return p
}
