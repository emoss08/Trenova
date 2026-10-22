package formula

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/shopspring/decimal"
)

func NegativeChargeMessage(amount string) rules.Message {
	return rules.Message{
		Template: "The formula came to {0}, and a charge cannot be negative. " +
			"Keep it at zero or above with max(..., 0), or set a minimum charge",
		Args: []any{amount},
	}
}

func ApplyChargePolicy(
	policy formulatypes.ChargePolicy,
	rawAmount decimal.Decimal,
) (decimal.Decimal, *formulatemplatetypes.GuardrailResult, *formulatemplatetypes.RoundingResult, error) {
	policy = policy.Normalized()

	clamped, guardrail := ApplyGuardrailBounds(policy.MinCharge, policy.MaxCharge, rawAmount)
	rounded := policy.RoundingMode.Round(clamped, policy.RoundingPrecision)

	rounding := &formulatemplatetypes.RoundingResult{
		Mode:            policy.RoundingMode.String(),
		Precision:       policy.RoundingPrecision,
		Applied:         !rounded.Equal(clamped),
		UnroundedAmount: clamped,
	}

	if rounded.IsNegative() {
		return rounded, guardrail, rounding, NegativeChargeMessage(
			rounded.StringFixed(policy.RoundingPrecision),
		).ValidationError(MainScope, errortypes.ErrInvalid)
	}

	return rounded, guardrail, rounding, nil
}

func ApplyGuardrailBounds(
	minCharge, maxCharge decimal.NullDecimal,
	rawAmount decimal.Decimal,
) (decimal.Decimal, *formulatemplatetypes.GuardrailResult) {
	if !minCharge.Valid && !maxCharge.Valid {
		return rawAmount, nil
	}

	guardrail := &formulatemplatetypes.GuardrailResult{RawAmount: rawAmount}
	if minCharge.Valid {
		lowerBound := minCharge.Decimal
		guardrail.MinCharge = &lowerBound
	}
	if maxCharge.Valid {
		upperBound := maxCharge.Decimal
		guardrail.MaxCharge = &upperBound
	}

	amount := rawAmount
	switch {
	case minCharge.Valid && rawAmount.LessThan(minCharge.Decimal):
		amount = minCharge.Decimal
		guardrail.Applied = true
		guardrail.Bound = "min"
	case maxCharge.Valid && rawAmount.GreaterThan(maxCharge.Decimal):
		amount = maxCharge.Decimal
		guardrail.Applied = true
		guardrail.Bound = "max"
	}

	return amount, guardrail
}
