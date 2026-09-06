package ptoledgerservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func utcDay(y int, m time.Month, d int) int64 {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix()
}

func keysOf(plan []PlannedEntry) []string {
	keys := make([]string, 0, len(plan))
	for _, entry := range plan {
		keys = append(keys, entry.PeriodKey)
	}
	return keys
}

func TestSchedule_PerPayPeriodFollowsTheSettlementCalendar(t *testing.T) {
	rule := worker.PTOPolicyRule{
		PTOType:           worker.PTOTypeVacation,
		AccrualMethod:     worker.PTOAccrualMethodPerPayPeriod,
		AccrualAmountDays: decimal.RequireFromString("0.5"),
	}
	// Biweekly periods closing on Saturday (end day 6): period end is the
	// midnight after the most recent Saturday, i.e. a Sunday 00:00 UTC.
	in := CalcInput{
		Rule:      rule,
		YearBasis: worker.PTOYearBasisCalendarYear,
		HireDate:  utcDay(2026, time.January, 5),
		Loc:       time.UTC,
		AsOf:      utcDay(2026, time.March, 1),
		PayPeriod: &PayPeriodSpec{Frequency: tenant.PayPeriodFrequencyBiweekly, EndDayOfWeek: 6},
	}

	plan := Schedule(in)
	keys := keysOf(plan)
	require.NotEmpty(t, keys)
	for _, entry := range plan {
		assert.Equal(t, worker.PTOLedgerEntryAccrual, entry.EntryType)
		assert.True(t, entry.NominalDays.Equal(decimal.RequireFromString("0.5")))
		day := time.Unix(entry.EffectiveAt, 0).UTC()
		assert.Equal(t, time.Sunday, day.Weekday(), "period ends are the midnight after the Saturday close")
		assert.True(t, entry.EffectiveAt > in.HireDate)
		assert.True(t, entry.EffectiveAt <= in.AsOf)
	}
	for i := 1; i < len(plan); i++ {
		assert.Equal(t, int64(14*86400), plan[i].EffectiveAt-plan[i-1].EffectiveAt, "biweekly spacing")
	}

	in.LastAccrualKey = keys[len(keys)-2]
	assert.Equal(t, keys[len(keys)-1:], keysOf(Schedule(in)), "the cursor skips already-posted periods")

	in.PayPeriod = nil
	assert.Empty(t, Schedule(in), "no settlement calendar, nothing to post")

	weekly := in
	weekly.PayPeriod = &PayPeriodSpec{Frequency: tenant.PayPeriodFrequencyWeekly, EndDayOfWeek: 5}
	weekly.LastAccrualKey = ""
	weeklyPlan := Schedule(weekly)
	require.Greater(t, len(weeklyPlan), len(plan))
	assert.Equal(t, time.Saturday, time.Unix(weeklyPlan[0].EffectiveAt, 0).UTC().Weekday())
}

func TestSchedule_TenureTiersRaiseAccrualAndCap(t *testing.T) {
	rule := worker.PTOPolicyRule{
		PTOType:           worker.PTOTypeVacation,
		AccrualMethod:     worker.PTOAccrualMethodMonthly,
		AccrualAmountDays: decimal.NewFromInt(1),
		MaxBalanceDays:    decimal.NewNullDecimal(decimal.NewFromInt(10)),
		Tiers: []worker.PTOAccrualTier{
			{MinMonths: 12, AccrualAmountDays: decimal.NewFromInt(2), MaxBalanceDays: decimal.NewNullDecimal(decimal.NewFromInt(20))},
		},
	}
	in := CalcInput{
		Rule:           rule,
		YearBasis:      worker.PTOYearBasisCalendarYear,
		HireDate:       utcDay(2025, time.March, 1),
		Loc:            time.UTC,
		AsOf:           utcDay(2026, time.April, 15),
		LookbackMonths: 6,
	}

	plan := Schedule(in)
	require.NotEmpty(t, plan)
	byKey := map[string]PlannedEntry{}
	for _, entry := range plan {
		byKey[entry.PeriodKey] = entry
	}
	before := byKey["M:2026-02"]
	assert.True(t, before.NominalDays.Equal(decimal.NewFromInt(1)), "eleven months in: base amount")
	assert.True(t, before.MaxBalanceDays.Decimal.Equal(decimal.NewFromInt(10)))
	after := byKey["M:2026-03"]
	assert.True(t, after.NominalDays.Equal(decimal.NewFromInt(2)), "twelve months in: tier amount")
	assert.True(t, after.MaxBalanceDays.Decimal.Equal(decimal.NewFromInt(20)))

	projected := ProjectedAccrual(plan, decimal.NewFromInt(9), decimal.Zero, rule)
	assert.True(t, projected.GreaterThan(decimal.NewFromInt(1)),
		"the raised cap lets the projection grow past the base cap")
}
