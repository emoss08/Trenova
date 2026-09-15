package latecharge

import (
	"github.com/shopspring/decimal"
)

const (
	secondsPerDay = int64(86400)
	// PeriodDays is the length of one late-charge period. A charge is assessed
	// once per period on whatever is still open when the run looks.
	PeriodDays    = int64(30)
	periodSeconds = PeriodDays * secondsPerDay
)

var oneHundred = decimal.NewFromInt(100)

// OverdueStart is the instant an invoice becomes chargeable: its due date plus
// the customer's grace period.
func OverdueStart(dueDate int64, graceDays int) int64 {
	if graceDays < 0 {
		graceDays = 0
	}

	return dueDate + int64(graceDays)*secondsPerDay
}

// PeriodsElapsed is how many periods are assessable at asOf. A period counts
// from the moment it begins, so the first is assessable the instant the grace
// period ends, and the second thirty days later.
func PeriodsElapsed(overdueStart, asOf int64) int {
	if asOf < overdueStart {
		return 0
	}

	return int((asOf-overdueStart)/periodSeconds) + 1
}

// PeriodBounds is the inclusive window of period index (1-based).
func PeriodBounds(overdueStart int64, index int) (start, end int64) {
	if index < 1 {
		index = 1
	}
	start = overdueStart + int64(index-1)*periodSeconds
	end = start + periodSeconds - 1

	return start, end
}

// ChargeMinor is the late charge on an open balance at a percentage rate,
// rounded half to even to the minor unit.
func ChargeMinor(openMinor int64, ratePercent decimal.Decimal) int64 {
	if openMinor <= 0 || ratePercent.LessThanOrEqual(decimal.Zero) {
		return 0
	}

	return decimal.NewFromInt(openMinor).Mul(ratePercent).Div(oneHundred).RoundBank(0).IntPart()
}

// PendingPeriods lists the period indexes assessable at asOf that have not
// been assessed yet, in order.
func PendingPeriods(overdueStart, asOf int64, assessed []int) []int {
	elapsed := PeriodsElapsed(overdueStart, asOf)
	if elapsed == 0 {
		return nil
	}
	done := make(map[int]struct{}, len(assessed))
	for _, idx := range assessed {
		done[idx] = struct{}{}
	}
	pending := make([]int, 0, elapsed)
	for idx := 1; idx <= elapsed; idx++ {
		if _, ok := done[idx]; ok {
			continue
		}
		pending = append(pending, idx)
	}

	return pending
}
