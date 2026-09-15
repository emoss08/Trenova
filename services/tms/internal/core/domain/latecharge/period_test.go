package latecharge_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/latecharge"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

const day = int64(86400)

func TestOverdueStart(t *testing.T) {
	due := int64(1_700_000_000)
	assert.Equal(t, due, latecharge.OverdueStart(due, 0))
	assert.Equal(t, due+10*day, latecharge.OverdueStart(due, 10))
	assert.Equal(t, due, latecharge.OverdueStart(due, -3), "a negative grace period counts as none")
}

func TestPeriodsElapsed(t *testing.T) {
	start := int64(1_700_000_000)
	tests := []struct {
		name string
		asOf int64
		want int
	}{
		{name: "before overdue", asOf: start - 1, want: 0},
		{name: "the instant overdue starts", asOf: start, want: 1},
		{name: "last second of period one", asOf: start + 30*day - 1, want: 1},
		{name: "first second of period two", asOf: start + 30*day, want: 2},
		{name: "day 61 is period three", asOf: start + 61*day, want: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, latecharge.PeriodsElapsed(start, tt.asOf))
		})
	}
}

func TestPeriodBounds(t *testing.T) {
	start := int64(1_700_000_000)
	s1, e1 := latecharge.PeriodBounds(start, 1)
	assert.Equal(t, start, s1)
	assert.Equal(t, start+30*day-1, e1)

	s2, e2 := latecharge.PeriodBounds(start, 2)
	assert.Equal(t, e1+1, s2, "periods are contiguous")
	assert.Equal(t, start+60*day-1, e2)

	s0, _ := latecharge.PeriodBounds(start, 0)
	assert.Equal(t, start, s0, "an index below one is treated as the first period")
}

func TestChargeMinor(t *testing.T) {
	rate := decimal.RequireFromString("1.5")
	assert.Equal(t, int64(1800), latecharge.ChargeMinor(120_000, rate), "1.5% of 1,200.00")
	assert.Equal(t, int64(0), latecharge.ChargeMinor(0, rate))
	assert.Equal(t, int64(0), latecharge.ChargeMinor(-500, rate))
	assert.Equal(t, int64(0), latecharge.ChargeMinor(120_000, decimal.Zero))

	// 2.5% of 0.25 is 0.625 minor units: half to even rounds down to 0 at .5
	// boundaries only, so 0.625 becomes 1.
	assert.Equal(t, int64(1), latecharge.ChargeMinor(25, decimal.RequireFromString("2.5")))
	// 1% of 2.50 is exactly 2.5 minor units: half to even gives 2.
	assert.Equal(t, int64(2), latecharge.ChargeMinor(250, decimal.NewFromInt(1)))
	// 1% of 3.50 is 3.5: half to even gives 4.
	assert.Equal(t, int64(4), latecharge.ChargeMinor(350, decimal.NewFromInt(1)))
}

func TestPendingPeriods(t *testing.T) {
	start := int64(1_700_000_000)
	assert.Nil(t, latecharge.PendingPeriods(start, start-1, nil), "nothing is due before overdue starts")
	assert.Equal(t, []int{1}, latecharge.PendingPeriods(start, start, nil))
	assert.Equal(t, []int{1, 2, 3}, latecharge.PendingPeriods(start, start+61*day, nil))
	assert.Equal(t, []int{3}, latecharge.PendingPeriods(start, start+61*day, []int{1, 2}),
		"already-assessed periods are skipped")
	assert.Empty(t, latecharge.PendingPeriods(start, start+61*day, []int{1, 2, 3}))
	assert.Equal(t, []int{2}, latecharge.PendingPeriods(start, start+61*day, []int{1, 3}),
		"a gap left by an earlier run is filled")
}
