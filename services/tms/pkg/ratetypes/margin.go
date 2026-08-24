package ratetypes

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// marginPercentScale is how precisely a margin percentage is carried.
//
// It matches the four decimal places the quote's margin_percent column stores,
// so a percentage that cleared a floor in memory cannot fail it after a round
// trip through the database.
const marginPercentScale = 4

var oneHundred = decimal.NewFromInt(100)

// Margin is what is left of the sell price once the carrier is paid.
//
// It goes negative when a load was bought for more than it sold for, which is
// exactly the case a margin floor exists to catch. Clamping it at zero would
// hide the loss from the guardrail meant to find it.
func Margin(sell, buy decimal.Decimal) decimal.Decimal {
	return sell.Sub(buy)
}

// MarginPercent is the margin as a share of what the customer pays.
//
// Revenue is the denominator because that is how the industry states it and how
// contracts are written. A shipment with no sell price has no revenue to take a
// share of, so it reads zero rather than dividing by nothing — a made-up number
// would let a guardrail fire on something meaningless.
func MarginPercent(sell, buy decimal.Decimal) decimal.Decimal {
	if sell.IsZero() {
		return decimal.Zero
	}

	return Margin(sell, buy).
		Div(sell).
		Mul(oneHundred).
		Round(marginPercentScale)
}

// MarginInputs is one comparison of a sell price against a buy price, together
// with whatever the contracts on either side agreed to allow.
type MarginInputs struct {
	Sell decimal.Decimal
	Buy  decimal.Decimal

	// Floor is the least margin the sell side agreed to accept, as a
	// percentage. Absent when the contract never named one.
	Floor decimal.NullDecimal

	// MaxPayPct is the most the buy side agreed to pay, as a percentage of what
	// the load sold for. The same guardrail written from the other end.
	MaxPayPct decimal.NullDecimal
}

// MarginVerdict is what a shipment's two sides came to, and whether that is
// allowed.
type MarginVerdict struct {
	Amount  decimal.Decimal `json:"amount"`
	Percent decimal.Decimal `json:"percent"`
	// PayPercent is the buy price as a share of the sell price.
	PayPercent decimal.Decimal `json:"payPercent"`

	FloorApplies   bool `json:"floorApplies"`
	BelowFloor     bool `json:"belowFloor"`
	CeilingApplies bool `json:"ceilingApplies"`
	// AbovePayCeiling means the carrier is being paid a larger share of the
	// sell price than the buy side contract allows.
	AbovePayCeiling bool `json:"abovePayCeiling"`

	// Explanation says what was breached and by how much, in the words the
	// advisory or the block will carry. Empty when nothing was breached.
	Explanation string `json:"explanation,omitempty"`
}

// Breached reports whether either guardrail was crossed.
func (v MarginVerdict) Breached() bool {
	return v.BelowFloor || v.AbovePayCeiling
}

// EvaluateMargin compares a shipment's two sides against its contracts.
//
// Both guardrails are checked even when the first is already breached, because
// somebody being told only half of what is wrong will fix half of it and
// resubmit.
//
// Neither is checked at all when the sell side has no price yet. Rating the buy
// side first is ordinary — a broker quotes cost before quoting the customer —
// and firing a guardrail against a sell price of nothing would block every one
// of those.
func EvaluateMargin(in *MarginInputs) MarginVerdict {
	verdict := MarginVerdict{
		Amount:  Margin(in.Sell, in.Buy),
		Percent: MarginPercent(in.Sell, in.Buy),
	}

	if in.Sell.IsZero() {
		return verdict
	}

	verdict.PayPercent = in.Buy.Div(in.Sell).Mul(oneHundred).Round(marginPercentScale)

	reasons := make([]string, 0, 2)

	if in.Floor.Valid {
		verdict.FloorApplies = true

		if verdict.Percent.LessThan(in.Floor.Decimal) {
			verdict.BelowFloor = true
			reasons = append(reasons, fmt.Sprintf(
				"margin of %s%% is below the %s%% this contract requires",
				trim(verdict.Percent), trim(in.Floor.Decimal),
			))
		}
	}

	if in.MaxPayPct.Valid {
		verdict.CeilingApplies = true

		if verdict.PayPercent.GreaterThan(in.MaxPayPct.Decimal) {
			verdict.AbovePayCeiling = true
			reasons = append(reasons, fmt.Sprintf(
				"carrier pay of %s%% of the sell price is above the %s%% this contract allows",
				trim(verdict.PayPercent), trim(in.MaxPayPct.Decimal),
			))
		}
	}

	verdict.Explanation = strings.Join(reasons, "; ")

	return verdict
}

// trim prints a percentage without the trailing zeros the storage scale adds,
// so an advisory reads "15%" rather than "15.0000%".
func trim(value decimal.Decimal) string {
	return value.RoundBank(marginPercentScale).String()
}
