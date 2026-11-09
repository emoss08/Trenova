package detentionrollup_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/rollupread"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/detentionrollup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func figureNames() []string {
	return []string{
		detentionrollup.Stops, detentionrollup.DetainedStops, detentionrollup.PastFree,
		detentionrollup.DockMinutes, detentionrollup.OpenMinutes, detentionrollup.EarlyMinutes,
		detentionrollup.LateMinutes, detentionrollup.Billable, detentionrollup.DriverPay,
		detentionrollup.Net, detentionrollup.Waived, detentionrollup.Disputes,
		detentionrollup.Suppressed, detentionrollup.NoticesDue, detentionrollup.NoticesKept,
	}
}

func TestEveryTableHoldsTheColumnsItsReadsSum(t *testing.T) {
	figures := append([]string{detentionrollup.Bucket}, figureNames()...)
	for _, b := range detentionrollup.RecoveryBuckets {
		figures = append(
			figures,
			detentionrollup.RecoveryCount(b),
			detentionrollup.RecoveryGross(b),
			detentionrollup.RecoveryBillable(b),
		)
	}
	for _, split := range detentionrollup.Splits {
		figures = append(figures, detentionrollup.SplitStops(split),
			detentionrollup.SplitDetained(split), detentionrollup.SplitMinutes(split))
	}
	facility := append([]string{detentionrollup.Location, detentionrollup.DwellTotal}, figures...)
	for i := range detentionrollup.DwellBounds {
		facility = append(facility, detentionrollup.DwellBin(i))
	}
	for i := range detentionrollup.FreeBounds {
		facility = append(facility, detentionrollup.FreeBin(i))
	}
	customer := append([]string{detentionrollup.Bucket, detentionrollup.Customer}, figureNames()...)
	policy := []string{
		detentionrollup.Bucket, detentionrollup.Location, detentionrollup.Policy,
		detentionrollup.Stops, detentionrollup.FirstClockStart,
	}
	arrivals := []string{detentionrollup.Bucket}
	for h := range detentionrollup.HoursPerDay {
		arrivals = append(
			arrivals,
			detentionrollup.ArrivalsAt(h),
			detentionrollup.DetainedArrivalsAt(h),
		)
	}

	cases := map[string]struct {
		table *rollupread.Table
		names []string
	}{
		"daily":            {detentionrollup.DailyTable, figures},
		"facility daily":   {detentionrollup.FacilityDailyTable, facility},
		"facility monthly": {detentionrollup.FacilityMonthlyTable, facility},
		"customer daily":   {detentionrollup.CustomerDailyTable, customer},
		"customer monthly": {detentionrollup.CustomerMonthlyTable, customer},
		"policy daily":     {detentionrollup.PolicyDailyTable, policy},
		"policy monthly":   {detentionrollup.PolicyMonthlyTable, policy},
		"arrival daily":    {detentionrollup.ArrivalDailyTable, arrivals},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			for _, column := range c.names {
				assert.True(t, c.table.Has(column), "%s has no column %s", name, column)
			}
		})
	}
}

func TestTheDailyAndMonthlyRollupsOfAFigureHoldTheSameColumns(t *testing.T) {
	pairs := map[string][2]*rollupread.Table{
		"facility": {detentionrollup.FacilityDailyTable, detentionrollup.FacilityMonthlyTable},
		"customer": {detentionrollup.CustomerDailyTable, detentionrollup.CustomerMonthlyTable},
		"policy":   {detentionrollup.PolicyDailyTable, detentionrollup.PolicyMonthlyTable},
	}
	for name, pair := range pairs {
		daily, monthly := pair[0].Ref.Table.Columns, pair[1].Ref.Table.Columns
		require.Len(t, monthly, len(daily), name)
		for i := range daily {
			assert.Equal(t, daily[i].Name, monthly[i].Name,
				"%s: a window unions the two, column by column", name)
		}
	}
}

func TestDwellBinsStayWithinHalfAnHourUpToTwelveHours(t *testing.T) {
	bounds := detentionrollup.DwellBounds
	for i := 1; i < len(bounds); i++ {
		require.Greater(t, bounds[i], bounds[i-1], "bounds rise")
		if bounds[i] <= 720 {
			assert.LessOrEqual(t, bounds[i]-bounds[i-1], 30.0,
				"a bin up to twelve hours spans at most half an hour")
		}
	}
}
