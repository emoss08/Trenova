package ptoledgerservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func date(year int, month time.Month, day int, loc *time.Location) int64 {
	return time.Date(year, month, day, 0, 0, 0, 0, loc).Unix()
}

func monthlyRule() pto.PTOPolicyRule {
	return pto.PTOPolicyRule{
		PTOType:           worker.PTOTypeVacation,
		AccrualMethod:     pto.PTOAccrualMethodMonthly,
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
		YearBasis:   pto.PTOYearBasisCalendarYear,
		WaitingDays: 90,
		HireDate:    date(2026, time.January, 15, loc),
		Loc:         loc,
		AsOf:        date(2026, time.July, 10, loc),
	})

	assert.Equal(t, []string{"M:2026-05", "M:2026-06", "M:2026-07"}, keys(plan))
	for _, e := range plan {
		assert.Equal(t, pto.PTOLedgerEntryAccrual, e.EntryType)
		assert.True(t, e.NominalDays.Equal(decimal.RequireFromString("1.25")))
		assert.False(t, e.Deferred)
	}
}

func TestScheduleSkipsPeriodsAlreadyPosted(t *testing.T) {
	t.Parallel()
	loc := time.UTC

	plan := Schedule(CalcInput{
		Rule:           monthlyRule(),
		YearBasis:      pto.PTOYearBasisCalendarYear,
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
		if e.EntryType == pto.PTOLedgerEntryAccrual {
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
		YearBasis:       pto.PTOYearBasisCalendarYear,
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
		YearBasis:      pto.PTOYearBasisCalendarYear,
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
	rule := pto.PTOPolicyRule{
		PTOType:           worker.PTOTypeSick,
		AccrualMethod:     pto.PTOAccrualMethodFixedAnnualGrant,
		AccrualAmountDays: decimal.NewFromInt(5),
	}

	plan := Schedule(CalcInput{
		Rule:           rule,
		YearBasis:      pto.PTOYearBasisHireAnniversary,
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
	rule := pto.PTOPolicyRule{
		PTOType:             worker.PTOTypeVacation,
		AccrualMethod:       pto.PTOAccrualMethodFixedAnnualGrant,
		AccrualAmountDays:   decimal.NewFromInt(10),
		CarryoverCapDays:    decimal.NewNullDecimal(decimal.NewFromInt(5)),
		CarryoverExpiryDays: 90,
	}

	plan := Schedule(CalcInput{
		Rule:           rule,
		YearBasis:      pto.PTOYearBasisCalendarYear,
		HireDate:       date(2024, time.June, 1, loc),
		Loc:            loc,
		AsOf:           date(2026, time.May, 1, loc),
		LookbackMonths: 6,
	})

	require.Len(t, plan, 3)
	assert.Equal(t, "C:2026", plan[0].PeriodKey)
	assert.True(t, plan[0].Deferred)
	assert.Equal(t, pto.PTOLedgerEntryExpiry, plan[0].EntryType)
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
	rule := pto.PTOPolicyRule{
		PTOType:           worker.PTOTypeVacation,
		AccrualMethod:     pto.PTOAccrualMethodMonthly,
		AccrualAmountDays: decimal.NewFromInt(1),
		CarryoverCapDays:  decimal.NewNullDecimal(decimal.NewFromInt(5)),
	}

	plan := Schedule(CalcInput{
		Rule:            rule,
		YearBasis:       pto.PTOYearBasisCalendarYear,
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
		YearBasis:   pto.PTOYearBasisCalendarYear,
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
	rule := pto.PTOPolicyRule{
		PTOType:           worker.PTOTypeVacation,
		AccrualMethod:     pto.PTOAccrualMethodMonthly,
		AccrualAmountDays: decimal.NewFromInt(2),
		MaxBalanceDays:    decimal.NewNullDecimal(decimal.NewFromInt(6)),
		CarryoverCapDays:  decimal.NewNullDecimal(decimal.NewFromInt(3)),
	}

	plan := []PlannedEntry{
		{
			EntryType:   pto.PTOLedgerEntryAccrual,
			PeriodKey:   "M:2026-11",
			NominalDays: decimal.NewFromInt(2),
			EffectiveAt: date(2026, time.November, 1, loc),
		},
		{
			EntryType:   pto.PTOLedgerEntryAccrual,
			PeriodKey:   "M:2026-12",
			NominalDays: decimal.NewFromInt(2),
			EffectiveAt: date(2026, time.December, 1, loc),
		},
		{
			EntryType:   pto.PTOLedgerEntryExpiry,
			PeriodKey:   "C:2027",
			Deferred:    true,
			EffectiveAt: date(2027, time.January, 1, loc),
		},
		{
			EntryType:   pto.PTOLedgerEntryAccrual,
			PeriodKey:   "M:2027-01",
			NominalDays: decimal.NewFromInt(2),
			EffectiveAt: date(2027, time.January, 1, loc),
		},
	}

	delta := ProjectedAccrual(plan, decimal.NewFromInt(4), decimal.Zero, &rule)
	assert.True(t, delta.Equal(decimal.NewFromInt(1)),
		"4 → 6 (capped) → 6 (capped) → 3 (carryover cap) → 5; delta %s", delta)
}

func TestRolloversPostedWithTheGrantLeaveNothingForAProjectionToReplay(t *testing.T) {
	t.Parallel()
	loc := time.UTC
	rule := pto.PTOPolicyRule{
		PTOType:           worker.PTOTypeSick,
		AccrualMethod:     pto.PTOAccrualMethodFixedAnnualGrant,
		AccrualAmountDays: decimal.NewFromInt(5),
		CarryoverCapDays:  decimal.NewNullDecimal(decimal.Zero),
	}
	hire := date(2023, time.October, 28, loc)
	seededAt := date(2026, time.October, 27, loc)

	bal := &pto.WorkerPTOBalance{BalanceDays: decimal.NewFromInt(2)}
	for _, planned := range Schedule(CalcInput{
		Rule:           rule,
		YearBasis:      pto.PTOYearBasisCalendarYear,
		HireDate:       hire,
		Loc:            loc,
		AsOf:           seededAt,
		LookbackMonths: 12,
	}) {
		if planned.Deferred {
			if amount, posts := ResolveExpiry(bal, planned, &rule); posts {
				bal.Apply(&pto.WorkerPTOLedgerEntry{
					EntryType:  planned.EntryType,
					AmountDays: amount,
				})
			}
			AdvanceCursor(bal, &planned, &rule)
			continue
		}
		bal.Apply(&pto.WorkerPTOLedgerEntry{
			EntryType:  planned.EntryType,
			AmountDays: planned.NominalDays,
		})
		AdvanceCursor(bal, &planned, &rule)
	}

	require.Equal(t, "C:2026", bal.LastRolloverKey, "this year's rollover is on the balance")
	assert.True(
		t,
		decimal.NewFromInt(5).Equal(bal.BalanceDays),
		"2 forfeited at year end, 5 granted",
	)

	ahead := Schedule(CalcInput{
		Rule:            rule,
		YearBasis:       pto.PTOYearBasisCalendarYear,
		HireDate:        hire,
		Loc:             loc,
		AsOf:            date(2026, time.November, 8, loc),
		LastAccrualKey:  bal.LastAccrualPeriodKey,
		LastRolloverKey: bal.LastRolloverKey,
	})
	assert.Empty(t, ahead, "nothing falls between now and the request")
	assert.True(t, ProjectedAccrual(ahead, bal.BalanceDays, bal.CarriedDays, &rule).IsZero())
}

func TestRolloverDue_OrdersByYearThenCarryoverBeforeExpiry(t *testing.T) {
	t.Parallel()

	assert.True(t, RolloverDue("C:2026", ""))
	assert.True(t, RolloverDue("X:2026", "C:2026"), "the carried days expire after the carryover")
	assert.False(t, RolloverDue("C:2026", "X:2026"))
	assert.False(t, RolloverDue("C:2026", "C:2026"))
	assert.True(
		t,
		RolloverDue("C:2027", "X:2026"),
		"a new year's carryover follows last year's expiry",
	)
	assert.False(t, RolloverDue("X:2025", "C:2026"))
}

func TestScheduleExpiresCarriedDaysAfterTheCarryoverWasPosted(t *testing.T) {
	t.Parallel()
	loc := time.UTC
	rule := pto.PTOPolicyRule{
		PTOType:             worker.PTOTypeVacation,
		AccrualMethod:       pto.PTOAccrualMethodFixedAnnualGrant,
		AccrualAmountDays:   decimal.NewFromInt(10),
		CarryoverCapDays:    decimal.NewNullDecimal(decimal.NewFromInt(5)),
		CarryoverExpiryDays: 90,
	}
	in := CalcInput{
		Rule:            rule,
		YearBasis:       pto.PTOYearBasisCalendarYear,
		HireDate:        date(2024, time.June, 1, loc),
		Loc:             loc,
		AsOf:            date(2026, time.April, 2, loc),
		LastAccrualKey:  "A:2026",
		LastRolloverKey: "C:2026",
		LookbackMonths:  6,
	}

	assert.Equal(t, []string{"X:2026"}, keys(Schedule(in)),
		"the daily run that posted January's carryover still expires the carried days in April")

	in.AsOf = date(2027, time.January, 2, loc)
	in.LastRolloverKey = "X:2026"
	in.LookbackMonths = 3
	assert.Equal(t, []string{"C:2027", "A:2027"}, keys(Schedule(in)),
		"last year's expiry does not hold back this year's carryover")
}
