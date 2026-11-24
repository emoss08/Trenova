package operationsboardreader

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/operationsboard"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var onTimeZone = func() *time.Location {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		panic(err)
	}

	return loc
}()

func onTimeDay(year int, month time.Month, date int) time.Time {
	return time.Date(year, month, date, 0, 0, 0, 0, onTimeZone)
}

func onTimeDays(rows map[time.Time][2]int64) []serviceDayRow {
	out := make([]serviceDayRow, 0, len(rows))
	for at, counts := range rows {
		out = append(out, serviceDayRow{
			Bucket:    at.UTC(),
			Delivered: counts[0],
			OnTime:    counts[1],
			Late:      counts[0] - counts[1],
		})
	}

	return out
}

func onTimeQuarter(t *testing.T, now time.Time) serviceWindow {
	t.Helper()
	first, last, ok := boardcatalog.PeriodThisQuarter.Days(now.Unix(), onTimeZone)
	require.True(t, ok)

	return newServiceWindow(first, last, now.Unix(), onTimeZone)
}

func TestTheQuarterIsReadInWholeWeeksFromTheMondayBeforeItStarts(t *testing.T) {
	t.Parallel()

	window := onTimeQuarter(t, onTimeDay(2026, time.November, 23).Add(10*time.Hour))

	assert.Equal(t, onTimeDay(2026, time.September, 28), window.start)
	assert.Equal(t, onTimeDay(2026, time.November, 23), window.readTo)
	assert.Equal(t, 9, window.weekCount())
	assert.Equal(t, onTimeDay(2026, time.July, 1), window.readFrom())
}

func TestTheLowestFullWeekIsTheSlipAndTheWeekStillRunningIsNot(t *testing.T) {
	t.Parallel()

	window := onTimeQuarter(t, onTimeDay(2026, time.November, 23).Add(10*time.Hour))
	weeks := foldServiceWeeks(onTimeDays(map[time.Time][2]int64{
		onTimeDay(2026, time.September, 29): {10, 10},
		onTimeDay(2026, time.October, 1):    {100, 94},
		onTimeDay(2026, time.November, 3):   {100, 88},
		onTimeDay(2026, time.November, 4):   {70, 62},
		onTimeDay(2026, time.November, 16):  {50, 46},
		onTimeDay(2026, time.November, 23):  {10, 5},
		onTimeDay(2026, time.August, 12):    {40, 30},
	}), &window)

	require.Len(t, weeks.weeks, 9)
	assert.Equal(t, 5, weeks.slip)
	assert.Equal(t, onTimeDay(2026, time.November, 2), weeks.weeks[5].Monday)
	assert.True(t, weeks.weeks[8].Running)
	assert.Equal(t, int64(330), weeks.delivered)
	assert.Equal(t, int64(295), weeks.onTime)
	assert.Equal(t, int64(50), weeks.prevDelivered)
	assert.Equal(t, int64(110), weeks.weeks[0].Delivered)
	assert.Equal(t, "0.92", weeks.latest().String())
}

func TestTheRunningWeekIsNeverTheSlipEvenWhenItIsLowest(t *testing.T) {
	t.Parallel()

	window := onTimeQuarter(t, onTimeDay(2026, time.October, 7).Add(10*time.Hour))
	weeks := foldServiceWeeks(onTimeDays(map[time.Time][2]int64{
		onTimeDay(2026, time.October, 6): {10, 1},
	}), &window)

	assert.Equal(t, noServiceWeek, weeks.slip)
	assert.Nil(t, weeks.latest())
}

func TestTheTrendDrawsTheSlipTheTargetAndTheGhost(t *testing.T) {
	t.Parallel()

	window := onTimeQuarter(t, onTimeDay(2026, time.November, 23).Add(10*time.Hour))
	weeks := foldServiceWeeks(onTimeDays(map[time.Time][2]int64{
		onTimeDay(2026, time.October, 6):   {100, 95},
		onTimeDay(2026, time.November, 3):  {100, 88},
		onTimeDay(2026, time.November, 23): {20, 19},
	}), &window)
	target := 92.0

	now := onTimeDay(2026, time.November, 23).Add(10 * time.Hour).Unix()
	read := onTimeRead(context.Background(), boardcatalog.PeriodThisQuarter, &weeks, &target, now)

	require.NotNil(t, read.Chart)
	assert.True(t, read.Chart.Ghost)
	assert.Equal(t, []int{5}, read.Chart.Marked)
	require.NotNil(t, read.Chart.Band)
	assert.Equal(t, 5, *read.Chart.Band.From)
	assert.True(t, read.Chart.X[5].Marked)
	require.NotNil(t, read.Chart.Target)
	assert.Equal(t, "0.92", read.Chart.Target.Value.String())
	assert.Nil(t, read.Chart.Series[0].Points[0])
	assert.Equal(t, "0.88", read.Figures["lowest"].Number.String())
	assert.Equal(t, "Nov 2", read.Figures["lowestWeek"].Text, "composes as the week of Nov 2")
	assert.Equal(t, "0.92", read.Figures["target"].Number.String())
	assert.Equal(t, "0.9182", read.Figures["average"].Number.String())
	assert.Equal(t, "220 deliveries, 9 weeks", read.Source)
	assert.True(t, onTimeFiguresComplete(t, operationsboard.MetricOnTimeByWeek, read))
}

func TestNoTargetIsNoLineAndNoFigure(t *testing.T) {
	t.Parallel()

	window := onTimeQuarter(t, onTimeDay(2026, time.November, 23).Add(10*time.Hour))
	weeks := foldServiceWeeks(nil, &window)

	read := onTimeRead(
		context.Background(), boardcatalog.PeriodThisQuarter, &weeks, nil,
		onTimeDay(2026, time.November, 23).Unix(),
	)

	assert.Nil(t, read.Chart.Target)
	assert.Nil(t, read.Figures["target"])
	assert.Nil(t, read.Figures["average"])
	assert.Nil(t, read.Figures["lowest"])
	assert.Empty(t, read.Chart.Marked)
}

func TestLateCustomersRankTheTopFourAndGatherTheRest(t *testing.T) {
	t.Parallel()

	northline := pulid.MustNew("cus_")
	week := lateWeek{
		monday: onTimeDay(2026, time.November, 2),
		last:   onTimeDay(2026, time.November, 8),
	}
	read := lateRead(context.Background(), week, []lateCustomerRow{
		{CustomerID: northline, Name: "Northline Grocers", Late: 17, Delivered: 60},
		{CustomerID: pulid.MustNew("cus_"), Name: "Acme Foods", Late: 7, Delivered: 40},
		{CustomerID: pulid.MustNew("cus_"), Name: "Harbor Paper Co.", Late: 4, Delivered: 30},
		{CustomerID: pulid.MustNew("cus_"), Name: "Delta Tile Supply", Late: 2, Delivered: 20},
	}, lateTotalsRow{Delivered: 272, Late: 32, Customers: 7}, onTimeDay(2026, time.November, 23).Unix())

	require.Len(t, read.Chart.Rows, 5)
	assert.True(t, read.Chart.Rows[0].Marked)
	assert.Equal(t, northline, read.Chart.Rows[0].Ref.ID)
	assert.Equal(t, "3 other customers", read.Chart.Rows[4].Label)
	assert.Nil(t, read.Chart.Rows[4].Ref)
	assert.True(t, read.Chart.Rows[4].Rest, "the others are drawn lighter")
	assert.False(t, read.Chart.Rows[0].Rest)
	assert.True(t, decimal.NewFromInt(2).Equal(read.Chart.Rows[4].Value.Number))
	assert.True(t, decimal.NewFromInt(32).Equal(read.Figures["late"].Number))
	assert.True(t, decimal.NewFromInt(17).Equal(read.Figures["top"].Number))
	assert.Equal(t, "Northline Grocers", read.Figures["topCustomer"].Text)
	assert.Equal(t, "Late in the week of Nov 2, by customer", read.Title)
	assert.Equal(t, "32 late deliveries, Nov 2 to Nov 8", read.Source)
	assert.True(t, onTimeFiguresComplete(t, operationsboard.MetricLateByCustomer, read))
}

func TestAWeekWithNothingLateHasNoRowsAndNoLeader(t *testing.T) {
	t.Parallel()

	week := lateWeek{
		monday: onTimeDay(2026, time.November, 9),
		last:   onTimeDay(2026, time.November, 15),
	}
	read := lateRead(
		context.Background(), week, nil, lateTotalsRow{Delivered: 80},
		onTimeDay(2027, time.January, 4).Unix(),
	)

	assert.Empty(t, read.Chart.Rows)
	assert.True(t, decimal.Zero.Equal(read.Figures["late"].Number))
	assert.Nil(t, read.Figures["top"])
	assert.Nil(t, read.Figures["topCustomer"])
	assert.Equal(t, "Late in the week of Nov 9, 2026, by customer", read.Title, "another year")
}

func onTimeFiguresComplete(
	t *testing.T,
	name string,
	read *boardcatalog.MetricRead,
) bool {
	t.Helper()
	metric, ok := boardcatalog.New(boardcatalog.Entries{Metrics: operationsboard.Metrics}).
		MetricNamed(name)
	require.True(t, ok)
	for _, figure := range metric.Figures {
		if _, present := read.Figures[figure.Name]; !present {
			return false
		}
	}

	return len(read.Figures) == len(metric.Figures)
}
