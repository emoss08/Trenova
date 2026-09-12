package invoicerunservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func profile(mutate func(*customer.CustomerBillingProfile)) *customer.CustomerBillingProfile {
	p := &customer.CustomerBillingProfile{
		InvoiceDelivery:       customer.InvoiceDeliveryConsolidated,
		BillingCycle:          customer.BillingCycleMonthly,
		BillingCycleAnchorDay: 1,
		BillingCycleTimezone:  "America/Denver",
	}
	if mutate != nil {
		mutate(p)
	}

	return p
}

func at(t *testing.T, zone, value string) int64 {
	t.Helper()
	loc, err := time.LoadLocation(zone)
	require.NoError(t, err)
	parsed, err := time.ParseInLocation("2006-01-02 15:04", value, loc)
	require.NoError(t, err)

	return parsed.Unix()
}

func TestPeriodsDueIgnoresNonStatementCustomers(t *testing.T) {
	t.Parallel()

	perShipment := profile(func(p *customer.CustomerBillingProfile) {
		p.InvoiceDelivery = customer.InvoiceDeliveryPerShipment
		p.BillingCycle = customer.BillingCycleImmediate
	})
	assert.Empty(t, PeriodsDue(perShipment, at(t, "UTC", "2026-03-15 12:00")))
	assert.Empty(t, PeriodsDue(nil, 0))
}

func TestPeriodsDueReturnsNothingBeforeThePeriodCloses(t *testing.T) {
	t.Parallel()

	// Mid-March, monthly on the 1st, last billed through 1 March: the period is
	// still open, so there is nothing to bill.
	lastBilled := at(t, "America/Denver", "2026-03-01 00:00")
	p := profile(func(p *customer.CustomerBillingProfile) {
		p.LastBilledPeriodEnd = &lastBilled
	})

	assert.Empty(t, PeriodsDue(p, at(t, "America/Denver", "2026-03-15 12:00")))
}

func TestPeriodsDueBillsTheClosedPeriod(t *testing.T) {
	t.Parallel()

	lastBilled := at(t, "America/Denver", "2026-03-01 00:00")
	p := profile(func(p *customer.CustomerBillingProfile) {
		p.LastBilledPeriodEnd = &lastBilled
	})

	periods := PeriodsDue(p, at(t, "America/Denver", "2026-04-02 09:00"))
	require.Len(t, periods, 1)
	assert.Equal(t, lastBilled, periods[0].Start)
	assert.Equal(t, at(t, "America/Denver", "2026-04-01 00:00"), periods[0].End)
}

func TestPeriodsDueCatchesUpAfterAnOutage(t *testing.T) {
	t.Parallel()

	// Six months without a successful run. Returning only the latest period would
	// silently lose five months of revenue, and the next tick would look healthy.
	lastBilled := at(t, "America/Denver", "2026-01-01 00:00")
	p := profile(func(p *customer.CustomerBillingProfile) {
		p.LastBilledPeriodEnd = &lastBilled
	})

	periods := PeriodsDue(p, at(t, "America/Denver", "2026-07-02 09:00"))
	require.Len(t, periods, 6)
	assert.Equal(t, lastBilled, periods[0].Start)
	assert.Equal(t, at(t, "America/Denver", "2026-07-01 00:00"), periods[5].End)

	// The periods are contiguous: no gap, no overlap.
	for i := 1; i < len(periods); i++ {
		assert.Equal(t, periods[i-1].End, periods[i].Start)
	}
}

func TestPeriodsDueIsBoundedForANeverBilledProfile(t *testing.T) {
	t.Parallel()

	// No LastBilledPeriodEnd means the customer was only just switched to
	// statement billing; it must not emit a run for every period since the epoch.
	p := profile(nil)
	periods := PeriodsDue(p, at(t, "America/Denver", "2026-04-02 09:00"))
	assert.Empty(t, periods, "the current period has not closed yet")
}

func TestPeriodsDueHandlesDSTSpringForward(t *testing.T) {
	t.Parallel()

	// Denver springs forward on 8 March 2026. A boundary built by adding 86400
	// would drift an hour and put the 8th's freight in the wrong day.
	lastBilled := at(t, "America/Denver", "2026-03-06 00:00")
	p := profile(func(p *customer.CustomerBillingProfile) {
		p.BillingCycle = customer.BillingCycleDaily
		p.LastBilledPeriodEnd = &lastBilled
	})

	periods := PeriodsDue(p, at(t, "America/Denver", "2026-03-10 06:00"))
	require.GreaterOrEqual(t, len(periods), 3)

	loc, err := time.LoadLocation("America/Denver")
	require.NoError(t, err)
	for _, period := range periods {
		start := time.Unix(period.Start, 0).In(loc)
		assert.Equal(t, 0, start.Hour(), "every boundary stays at local midnight")
		assert.Equal(t, 0, start.Minute())
	}
}

func TestPeriodsDueHandlesDSTFallBack(t *testing.T) {
	t.Parallel()

	// Denver falls back on 1 November 2026.
	lastBilled := at(t, "America/Denver", "2026-10-30 00:00")
	p := profile(func(p *customer.CustomerBillingProfile) {
		p.BillingCycle = customer.BillingCycleDaily
		p.LastBilledPeriodEnd = &lastBilled
	})

	periods := PeriodsDue(p, at(t, "America/Denver", "2026-11-04 06:00"))
	loc, err := time.LoadLocation("America/Denver")
	require.NoError(t, err)
	for _, period := range periods {
		start := time.Unix(period.Start, 0).In(loc)
		assert.Equal(t, 0, start.Hour())
	}
}

func TestPeriodsDueWeeklyAnchorsOnTheWeekday(t *testing.T) {
	t.Parallel()

	// Anchor 1 is Monday.
	lastBilled := at(t, "America/Denver", "2026-03-02 00:00")
	p := profile(func(p *customer.CustomerBillingProfile) {
		p.BillingCycle = customer.BillingCycleWeekly
		p.BillingCycleAnchorDay = 1
		p.LastBilledPeriodEnd = &lastBilled
	})

	periods := PeriodsDue(p, at(t, "America/Denver", "2026-03-20 09:00"))
	require.NotEmpty(t, periods)

	loc, err := time.LoadLocation("America/Denver")
	require.NoError(t, err)
	for _, period := range periods {
		assert.Equal(t, time.Monday, time.Unix(period.Start, 0).In(loc).Weekday())
	}
}

func TestPeriodsDueMonthlyAnchorNeverShiftsInFebruary(t *testing.T) {
	t.Parallel()

	// The anchor is capped at 28 by validation, so a monthly boundary always
	// exists in every month.
	lastBilled := at(t, "America/Denver", "2026-01-28 00:00")
	p := profile(func(p *customer.CustomerBillingProfile) {
		p.BillingCycleAnchorDay = 28
		p.LastBilledPeriodEnd = &lastBilled
	})

	periods := PeriodsDue(p, at(t, "America/Denver", "2026-04-01 09:00"))
	require.NotEmpty(t, periods)

	loc, err := time.LoadLocation("America/Denver")
	require.NoError(t, err)
	for _, period := range periods {
		assert.Equal(t, 28, time.Unix(period.Start, 0).In(loc).Day())
	}
}

func TestPeriodsDueRespectsTheCustomersZone(t *testing.T) {
	t.Parallel()

	// 1 April 00:00 Pacific is 07:00 UTC. Evaluated in UTC the period would close
	// eight hours early and take the 31st's late deliveries with it.
	lastBilled := at(t, "America/Los_Angeles", "2026-03-01 00:00")
	p := profile(func(p *customer.CustomerBillingProfile) {
		p.BillingCycleTimezone = "America/Los_Angeles"
		p.LastBilledPeriodEnd = &lastBilled
	})

	periods := PeriodsDue(p, at(t, "America/Los_Angeles", "2026-04-01 09:00"))
	require.Len(t, periods, 1)
	assert.Equal(t, at(t, "America/Los_Angeles", "2026-04-01 00:00"), periods[0].End)
}
