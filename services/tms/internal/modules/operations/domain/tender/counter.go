package tender

import (
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/shopspring/decimal"
)

const (
	MaxCounterNoteLength = 500
)

// MaxCounterRate bounds a counter-offer well inside what the rate columns
// hold, so a slipped key cannot store a rate nobody meant.
var MaxCounterRate = decimal.NewFromInt(10_000_000)

// CounterTerms is everything a counter-offer is judged against: what the load
// sells for, what the carrier would be paid at its counter, and the
// organization's floor and appetite for taking counters on its own.
type CounterTerms struct {
	Sell       decimal.Decimal
	Buy        decimal.Decimal
	Floor      decimal.Decimal
	AutoAccept bool
}

// CounterVerdict is what a counter-offer comes to: the margin it leaves,
// whether that margin is below the floor, and whether a person has to decide
// it, or it may be taken without one.
type CounterVerdict struct {
	MarginPercent decimal.NullDecimal
	BelowFloor    bool
	NeedsApproval bool
	AutoAccept    bool
	Explanation   string
}

// EvaluateCounter judges a counter-offer against the margin floor.
//
// A counter that leaves less margin than the floor needs somebody with the
// authority to approve it. A load with no sell price has no margin to measure,
// so its counters always wait for a person rather than being waved through on
// a guess. Only a counter that clears the floor is taken on its own, and only
// when the organization has asked for that.
func EvaluateCounter(terms CounterTerms) CounterVerdict {
	if terms.Sell.IsZero() {
		return CounterVerdict{
			NeedsApproval: true,
			Explanation:   "the load has no sell price, so the margin cannot be checked",
		}
	}

	margin := ratetypes.EvaluateMargin(&ratetypes.MarginInputs{
		Sell:  terms.Sell,
		Buy:   terms.Buy,
		Floor: decimal.NewNullDecimal(terms.Floor),
	})
	verdict := CounterVerdict{
		MarginPercent: decimal.NewNullDecimal(margin.Percent),
		BelowFloor:    margin.BelowFloor,
	}
	if margin.BelowFloor {
		verdict.NeedsApproval = true
		verdict.Explanation = "the counter leaves a margin of " + percentLabel(margin.Percent) +
			", below the " + percentLabel(terms.Floor) + " floor"
		return verdict
	}
	verdict.AutoAccept = terms.AutoAccept
	return verdict
}

func percentLabel(value decimal.Decimal) string {
	return value.Round(2).String() + "%"
}
