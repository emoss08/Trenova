package fuelboardreader

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelboard"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chicago(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	return loc
}

func day(loc *time.Location, month time.Month, d int) int64 {
	return time.Date(2026, month, d, 0, 0, 0, 0, loc).Unix()
}

func spend(at int64, minor int64, miles int64) dayRow {
	return dayRow{
		DayAt: at, Purchases: 2, Gallons: decimal.NewFromInt(minor / 400),
		AmountMinor: minor, Miles: decimal.NewFromInt(miles), Currencies: 1, Currency: "USD",
	}
}

func TestCostPerMileRunsSevenDaysAgainstTheMonthBefore(t *testing.T) {
	t.Parallel()

	loc := chicago(t)
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, loc).Unix()
	window, ok := windowOf(boardcatalog.PeriodThisMonth, now, loc)
	require.True(t, ok)
	rows := []dayRow{
		spend(day(loc, 10, 1), 60000, 1000),
		spend(day(loc, 10, 2), 60000, 1000),
		spend(day(loc, 10, 31), 60000, 1000),
		spend(day(loc, 11, 1), 70000, 1000),
		spend(day(loc, 11, 3), 80000, 1000),
	}
	read := costRead(context.Background(), &window, rows)

	require.NotNil(t, read.Chart)
	assert.Len(t, read.Chart.X, 3, "Nov 1 to today")
	now3 := read.Chart.Series[0].Points
	assert.Equal(t, "0.650", now3[1].StringFixed(3), "Oct 31 and Nov 1 over two days' miles")
	assert.Equal(t, "0.700", now3[2].StringFixed(3))
	was := read.Chart.Series[1].Points
	assert.Equal(t, "0.600", was[0].StringFixed(3), "Oct 1 against Nov 1")
	assert.True(t, read.Chart.Ghost)
	assert.Equal(t, "October", read.Chart.Ref.Label)
	assert.Equal(t, "0.600", read.Chart.Ref.Value.StringFixed(3))

	assert.Equal(t, "0.750", read.Figures[fuelboard.FigureCostPerMile].Number.StringFixed(3))
	assert.Equal(t, "0.600",
		read.Figures[fuelboard.FigurePreviousCostPerMile].Number.StringFixed(3))
	assert.Equal(t, "1500.00", read.Figures[fuelboard.FigureFuelSpend].Number.StringFixed(2))
	assert.Equal(t, "2000", read.Figures[fuelboard.FigureMilesRun].Number.String())
	assert.Equal(t, boardcatalog.ToneWarn, read.Changes[fuelboard.FigureCostPerMile].Tone)
	assert.Equal(t, "Fuel cost per mile, November", read.Title)
}

func TestCostPerMileWithholdsOnlyTheMonthAndDaysBoughtInAnotherCurrency(t *testing.T) {
	t.Parallel()

	loc := chicago(t)
	window, _ := windowOf(boardcatalog.PeriodThisMonth, day(loc, 11, 3)+3600, loc)
	canadian := spend(day(loc, 10, 23), 60000, 1000)
	canadian.Currency = "CAD"
	rows := []dayRow{
		spend(day(loc, 10, 2), 60000, 1000),
		canadian,
		spend(day(loc, 10, 24), 60000, 1000),
		spend(day(loc, 11, 1), 70000, 1000),
	}
	read := costRead(context.Background(), &window, rows)

	assert.Equal(t, "0.700", read.Figures[fuelboard.FigureCostPerMile].Number.StringFixed(3))
	assert.Nil(t, read.Figures[fuelboard.FigurePreviousCostPerMile], "October holds CAD")
	assert.Nil(t, read.Chart.Ref)
	assert.Empty(t, read.Changes)
	assert.Equal(t, "0.700", read.Chart.Series[0].Points[0].StringFixed(3), "Oct 26 to Nov 1")
	days := &costDays{byDay: map[int64]*dayRow{}, currency: "USD"}
	for idx := range rows {
		days.byDay[rows[idx].DayAt] = &rows[idx]
	}
	assert.Nil(t, days.running(time.Unix(day(loc, 10, 29), 0).In(loc)), "its run holds Oct 23")
	assert.Contains(t, read.Source, "October holds fuel bought in another currency")

	both := costRead(context.Background(), &window, []dayRow{{
		DayAt: day(loc, 11, 1), AmountMinor: 100, Miles: decimal.NewFromInt(10), Currencies: 2,
		Currency: "CAD",
	}})
	assert.Nil(t, both.Figures[fuelboard.FigureCostPerMile])
	assert.Nil(t, both.Figures[fuelboard.FigureFuelSpend])
}

func tractor(code string, minor int64, miles int64, purchases int64) tractorRow {
	return tractorRow{
		TractorID: pulid.MustNew("tr_"), Code: code, Purchases: purchases,
		AmountMinor: minor, Miles: decimal.NewFromInt(miles), Currencies: 1, Currency: "USD",
	}
}

func TestTractorsArePlacedAgainstTheRangeMostOfTheFleetRunsIn(t *testing.T) {
	t.Parallel()

	loc := chicago(t)
	window, _ := windowOf(boardcatalog.PeriodThisMonth, day(loc, 11, 20), loc)
	rows := []tractorRow{
		tractor("1211", 100000, 1000, 8),
		tractor("1100", 10000, 1000, 8),
		tractor("1131", 70000, 300, 1),
	}
	for idx := range 10 {
		code := "10" + strconv.Itoa(idx)
		rows = append(rows, tractor(code, int64(40000+2000*idx), 1000, 6))
	}
	read := tractorsRead(context.Background(), &window, rows)

	assert.Equal(t, "0.402", read.Figures[fuelboard.FigureUsualLow].Number.StringFixed(3))
	assert.Equal(t, "0.578", read.Figures[fuelboard.FigureUsualHigh].Number.StringFixed(3))
	assert.Equal(t, "2", read.Figures[fuelboard.FigureTractorsAbove].Number.String(), "1211, 109")
	assert.Equal(t, "2", read.Figures[fuelboard.FigureTractorsBelow].Number.String(), "1100, 100")
	assert.Equal(t, "1211", read.Figures[fuelboard.FigureHighestTractor].Text)
	assert.Equal(t, "1.000", read.Figures[fuelboard.FigureHighestCostPerMile].Number.StringFixed(3))
	assert.Equal(t, "1100", read.Figures[fuelboard.FigureLowestTractor].Text)
	assert.Equal(t, "0.100", read.Figures[fuelboard.FigureLowestCostPerMile].Number.StringFixed(3))

	require.NotNil(t, read.Chart)
	assert.Equal(t, "0.402", read.Chart.Band.Low.StringFixed(3))
	assert.Equal(t, "0.578", read.Chart.Band.High.StringFixed(3))
	byCode := map[string]boardcatalog.ChartRow{}
	for _, row := range read.Chart.Rows {
		byCode[row.Label] = row
	}
	require.Len(t, byCode, 13)
	assert.Equal(t, "1211", byCode["1211"].Dots[0].Label)
	assert.True(t, byCode["1100"].Dots[0].Marked)
	assert.True(t, byCode["1131"].Dots[0].Ghost, "one purchase over 300 miles cannot place it")
	assert.False(t, byCode["1131"].Dots[0].Marked)
	assert.False(t, byCode["104"].Dots[0].Marked, "inside the usual range")
	assert.Equal(t, "1211", read.Chart.Rows[0].Label, "the furthest above is drawn first")
}

func TestOnlyAHandfulOutsideTheRangeAreNamed(t *testing.T) {
	t.Parallel()

	loc := chicago(t)
	window, _ := windowOf(boardcatalog.PeriodThisMonth, day(loc, 11, 20), loc)
	rows := make([]tractorRow, 0, 60)
	for idx := range 60 {
		rows = append(rows, tractor("T"+strconv.Itoa(idx), int64(30000+1000*idx), 1000, 6))
	}
	read := tractorsRead(context.Background(), &window, rows)

	named := 0
	for _, row := range read.Chart.Rows {
		if row.Dots[0].Marked {
			named++
			assert.NotEmpty(t, row.Dots[0].Label)
		}
	}
	assert.Equal(t, 6, named, "three each side")
	assert.Equal(t, "6", read.Figures[fuelboard.FigureTractorsAbove].Number.String())
	assert.Equal(t, "6", read.Figures[fuelboard.FigureTractorsBelow].Number.String())
}

func TestNoMilesLeavesTheTractorsUnplaced(t *testing.T) {
	t.Parallel()

	loc := chicago(t)
	window, _ := windowOf(boardcatalog.PeriodLastMonth, day(loc, 11, 20), loc)
	read := tractorsRead(context.Background(), &window, nil)

	assert.Nil(t, read.Chart)
	assert.Nil(t, read.Figures[fuelboard.FigureFleetCostPerMile])
	assert.Equal(t, "No tractor ran miles in October", read.Source)
}
