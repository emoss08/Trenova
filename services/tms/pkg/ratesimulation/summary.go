// Package ratesimulation summarizes what a pricing change would have done.
//
// It is shared rather than duplicated because two things ask the same question:
// a formula template backtest ("what would this expression have charged") and a
// rate agreement simulation ("what would this contract have charged"). The
// answer has to read the same either way, or two screens showing the same kind
// of number would disagree on what "changed" means.
package ratesimulation

import "github.com/shopspring/decimal"

var oneHundred = decimal.NewFromInt(100)

// deltaScale is how precisely a percentage move is reported. Four places is
// what the quote and backtest columns already carry.
const deltaScale = 4

// Delta is one shipment priced two ways.
type Delta struct {
	Before decimal.Decimal
	After  decimal.Decimal

	// Failed marks a shipment that could not be priced one way or the other.
	// Its amounts are not trusted and are left out of every total: folding a
	// zero in would understate the revenue the change is being judged against.
	Failed bool
}

// Amount is how far this shipment moved.
func (d Delta) Amount() decimal.Decimal {
	return d.After.Sub(d.Before)
}

// Percent is the move as a share of what the shipment was priced at.
//
// A shipment priced at nothing has no base to take a share of, so it reads zero
// rather than dividing by nothing.
func (d Delta) Percent() decimal.Decimal {
	if d.Before.IsZero() {
		return decimal.Zero
	}

	return d.Amount().Div(d.Before).Mul(oneHundred).Round(deltaScale)
}

// Summary is what a whole run came to.
type Summary struct {
	ShipmentCount  int `json:"shipmentCount"`
	EvaluatedCount int `json:"evaluatedCount"`
	ChangedCount   int `json:"changedCount"`
	IncreasedCount int `json:"increasedCount"`
	DecreasedCount int `json:"decreasedCount"`
	ErrorCount     int `json:"errorCount"`

	BeforeTotal decimal.Decimal `json:"beforeTotal"`
	AfterTotal  decimal.Decimal `json:"afterTotal"`

	TotalDelta    decimal.Decimal `json:"totalDelta"`
	TotalDeltaPct decimal.Decimal `json:"totalDeltaPct"`

	// MaxIncrease and MaxDecrease are the largest single moves in each
	// direction. They are what somebody scans for: the shipment that will
	// produce the phone call.
	MaxIncrease decimal.Decimal `json:"maxIncrease"`
	MaxDecrease decimal.Decimal `json:"maxDecrease"`
}

// Accumulator builds a Summary one shipment at a time.
//
// It streams rather than taking a slice because a simulation walks a year of
// shipments, which is far too many to hold at once. Its zero value is ready to
// use.
type Accumulator struct {
	summary Summary
}

// Add folds one shipment into the running summary.
func (a *Accumulator) Add(delta Delta) {
	a.summary.ShipmentCount++

	if delta.Failed {
		a.summary.ErrorCount++
		return
	}

	a.summary.EvaluatedCount++
	a.summary.BeforeTotal = a.summary.BeforeTotal.Add(delta.Before)
	a.summary.AfterTotal = a.summary.AfterTotal.Add(delta.After)

	amount := delta.Amount()

	switch amount.Sign() {
	case 1:
		a.summary.ChangedCount++
		a.summary.IncreasedCount++

		if amount.GreaterThan(a.summary.MaxIncrease) {
			a.summary.MaxIncrease = amount
		}
	case -1:
		a.summary.ChangedCount++
		a.summary.DecreasedCount++

		if amount.LessThan(a.summary.MaxDecrease) {
			a.summary.MaxDecrease = amount
		}
	}
}

// Summary closes the run and returns what it came to.
func (a *Accumulator) Summary() Summary {
	summary := a.summary

	summary.TotalDelta = summary.AfterTotal.Sub(summary.BeforeTotal)

	if !summary.BeforeTotal.IsZero() {
		summary.TotalDeltaPct = summary.TotalDelta.
			Div(summary.BeforeTotal).
			Mul(oneHundred).
			Round(deltaScale)
	}

	return summary
}
