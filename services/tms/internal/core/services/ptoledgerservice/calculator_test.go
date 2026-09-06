package ptoledgerservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func date(year int, month time.Month, day int, loc *time.Location) int64 {
	return time.Date(year, month, day, 0, 0, 0, 0, loc).Unix()
}

func monthlyRule() worker.PTOPolicyRule {
	return worker.PTOPolicyRule{
		PTOType:           worker.PTOTypeVacation,
		AccrualMethod:     worker.PTOAccrualMethodMonthly,
		AccrualAmountDays: decimal.RequireFromString("1.25"),
	}
}

func keys(plan []PlannedEntry) []string {
	out := make([]string, 0, len(plan))
	for _, e := range plan {
		out = append(out, e.PeriodKey)
	}
	return out
}

func TestScheduleMonthlyStartsAfterWaitingPeriod(t *testing.T) {
	t.Parallel()
	loc := time.UTC

	plan := Schedule(CalcInput{
		Rule:        monthlyRule(),
		YearBasis:   worker.PTOYearBasisCalendarYear,
		WaitingDays: 90,
		HireDate:    date(2026, time.January, 15, loc),
		Loc:         loc,
		AsOf:        date(2026, time.July, 10, loc),
	})

	assert.Equal(t, []string{"M:2026-05", "M:2026-06", "M:2026-07"}, keys(plan))
	for _, e := range plan {
		assert.Equal(t, worker.PTOLedgerEntryAccrual, e.EntryType)
		assert.True(t, e.NominalDays.Equal(decimal.RequireFromString("1.25")))
		assert.False(t, e.Deferred)
	}
}

func TestScheduleSkipsPeriodsAlreadyPosted(t *testing.T) {
	t.Parallel()
	loc := time.UTC

	plan := Schedule(CalcInput{
		Rule:           monthlyRule(),
		YearBasis:      worker.PTOYearBasisCalendarYear,
		HireDate:       date(2025, time.March, 1, loc),
		Loc:            loc,
		AsOf:           date(2026, time.March, 15, loc),
		LastAccrualKey: "M:2026-01",
		LookbackMonths: 24,
	})

	assert.Equal(t, []string{"C:2026", "M:2026-02", "M:2026-03"}[1:], keys(filterAccruals(plan)))
}

func filterAccruals(plan []PlannedEntry) []PlannedEntry {
	out := make([]PlannedEntry, 0, len(plan))
	for _, e := range plan {
		if e.EntryType == worker.PTOLedgerEntryAccrual {
			out = append(out, e)
		}
	}
	return out
}

func TestScheduleStopsAtTermination(t *testing.T) {
	t.Parallel()
	loc := time.UTC
	termination := date(2026, time.April, 20, loc)

	plan := Schedule(CalcInput{
		Rule:            monthlyRule(),
		YearBasis:       worker.PTOYearBasisCalendarYear,
		HireDate:        date(2026, time.January, 1, loc),
		TerminationDate: &termination,
		Loc:             loc,
		AsOf:            date(2026, time.September, 1, loc),
	})

	assert.Equal(
		t,
		[]string{"M:2026-01", "M:2026-02", "M:2026-03", "M:2026-04"},
		keys(filterAccruals(plan)),
	)
}

func TestScheduleBoundsCatchUpToLookback(t *testing.T) {
	t.Parallel()
	loc := time.UTC

	plan := Schedule(CalcInput{
		Rule:           monthlyRule(),
		YearBasis:      worker.PTOYearBasisCalendarYear,
		HireDate:       date(2019, time.January, 1, loc),
		Loc:            loc,
		AsOf:           date(2026, time.March, 1, loc),
		LookbackMonths: 3,
	})

	accruals := keys(filterAccruals(plan))
	require.NotEmpty(t, accruals)
	assert.Equal(t, "M:2026-01", accruals[0])
	assert.Equal(t, "M:2026-03", accruals[len(accruals)-1])
}

func TestScheduleAnnualGrantOnHireAnniversaryHandlesLeapDay(t *testing.T) {
	t.Parallel()
	loc := time.UTC
	rule := worker.PTOPolicyRule{
		PTOType:           worker.PTOTypeSick,
		AccrualMethod:     worker.PTOAccrualMethodFixedAnnualGrant,
		AccrualAmountDays: decimal.NewFromInt(5),
	}

	plan := Schedule(CalcInput{
		Rule:           rule,
		YearBasis:      worker.PTOYearBasisHireAnniversary,
		HireDate:       date(2024, time.February, 29, loc),
		Loc:            loc,
		AsOf:           date(2026, time.March, 5, loc),
		LookbackMonths: 24,
	})

	accruals := filterAccruals(plan)
	require.Len(t, accruals, 2)
	assert.Equal(t, "A:2025", accruals[0].PeriodKey)
	assert.Equal(t, date(2025, time.February, 28, loc), accruals[0].EffectiveAt)
	assert.Equal(t, "A:2026", accruals[1].PeriodKey)
	assert.Equal(t, date(2026, time.February, 28, loc), accruals[1].EffectiveAt)
}

func TestScheduleEmitsCarryoverAndExpiryBeforeAccrualOnYearStart(t *testing.T) {
	t.Parallel()
	loc := time.UTC
	rule := worker.PTOPolicyRule{
		PTOType:             worker.PTOTypeVacation,
		AccrualMethod:       worker.PTOAccrualMethodFixedAnnualGrant,
		AccrualAmountDays:   decimal.NewFromInt(10),
		CarryoverCapDays:    decimal.NewNullDecimal(decimal.NewFromInt(5)),
		CarryoverExpiryDays: 90,
	}

	plan := Schedule(CalcInput{
		Rule:           rule,
		YearBasis:      worker.PTOYearBasisCalendarYear,
		HireDate:       date(2024, time.June, 1, loc),
		Loc:            loc,
		AsOf:           date(2026, time.May, 1, loc),
		LookbackMonths: 6,
	})

	require.Len(t, plan, 3)
	assert.Equal(t, "C:2026", plan[0].PeriodKey)
	assert.True(t, plan[0].Deferred)
	assert.Equal(t, worker.PTOLedgerEntryExpiry, plan[0].EntryType)
	assert.Equal(t, "A:2026", plan[1].PeriodKey)
	assert.Equal(
		t,
		plan[0].EffectiveAt,
		plan[1].EffectiveAt,
		"carryover cap applies before the grant",
	)
	assert.Equal(t, "X:2026", plan[2].PeriodKey)
	assert.Equal(t, date(2026, time.April, 1, loc), plan[2].EffectiveAt)
}

func TestScheduleSkipsRolloverAlreadyApplied(t *testing.T) {
	t.Parallel()
	loc := time.UTC
	rule := worker.PTOPolicyRule{
		PTOType:           worker.PTOTypeVacation,
		AccrualMethod:     worker.PTOAccrualMethodMonthly,
		AccrualAmountDays: decimal.NewFromInt(1),
		CarryoverCapDays:  decimal.NewNullDecimal(decimal.NewFromInt(5)),
	}

	plan := Schedule(CalcInput{
		Rule:            rule,
		YearBasis:       worker.PTOYearBasisCalendarYear,
		HireDate:        date(2025, time.January, 1, loc),
		Loc:             loc,
		AsOf:            date(2026, time.February, 1, loc),
		LastAccrualKey:  "M:2026-01",
		LastRolloverKey: "C:2026",
		LookbackMonths:  3,
	})

	assert.Equal(t, []string{"M:2026-02"}, keys(plan))
}

func TestScheduleReturnsNothingBeforeEligibility(t *testing.T) {
	t.Parallel()
	loc := time.UTC

	plan := Schedule(CalcInput{
		Rule:        monthlyRule(),
		YearBasis:   worker.PTOYearBasisCalendarYear,
		WaitingDays: 180,
		HireDate:    date(2026, time.May, 1, loc),
		Loc:         loc,
		AsOf:        date(2026, time.August, 1, loc),
	})

	assert.Empty(t, plan)
}

func TestProjectedAccrualRespectsCapAndCarryover(t *testing.T) {
	t.Parallel()
	loc := time.UTC
	rule := worker.PTOPolicyRule{
		PTOType:           worker.PTOTypeVacation,
		AccrualMethod:     worker.PTOAccrualMethodMonthly,
		AccrualAmountDays: decimal.NewFromInt(2),
		MaxBalanceDays:    decimal.NewNullDecimal(decimal.NewFromInt(6)),
		CarryoverCapDays:  decimal.NewNullDecimal(decimal.NewFromInt(3)),
	}

	plan := []PlannedEntry{
		{
			EntryType:   worker.PTOLedgerEntryAccrual,
			PeriodKey:   "M:2026-11",
			NominalDays: decimal.NewFromInt(2),
			EffectiveAt: date(2026, time.November, 1, loc),
		},
		{
			EntryType:   worker.PTOLedgerEntryAccrual,
			PeriodKey:   "M:2026-12",
			NominalDays: decimal.NewFromInt(2),
			EffectiveAt: date(2026, time.December, 1, loc),
		},
		{
			EntryType:   worker.PTOLedgerEntryExpiry,
			PeriodKey:   "C:2027",
			Deferred:    true,
			EffectiveAt: date(2027, time.January, 1, loc),
		},
		{
			EntryType:   worker.PTOLedgerEntryAccrual,
			PeriodKey:   "M:2027-01",
			NominalDays: decimal.NewFromInt(2),
			EffectiveAt: date(2027, time.January, 1, loc),
		},
	}

	delta := ProjectedAccrual(plan, decimal.NewFromInt(4), decimal.Zero, rule)
	assert.True(t, delta.Equal(decimal.NewFromInt(1)),
		"4 → 6 (capped) → 6 (capped) → 3 (carryover cap) → 5; delta %s", delta)
}
