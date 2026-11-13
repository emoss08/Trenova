package dispatchmarket_test

import (
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chicagoGrid(t *testing.T) (dispatchmarket.Grid, *time.Location) {
	t.Helper()
	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	return dispatchmarket.GridIn(chicago), chicago
}

func TestGrid_WithNoTimeZoneIsTheUTCGrid(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, time.November, 12, 15, 30, 0, 0, time.UTC).Unix()
	for _, grid := range []dispatchmarket.Grid{
		{},
		dispatchmarket.GridIn(nil),
		dispatchmarket.GridNamed(""),
		dispatchmarket.GridNamed("Not/AZone"),
	} {
		assert.Equal(t, dispatchmarket.WeekOf(at), grid.Week(at))
		assert.Equal(t, dispatchmarket.SlotOf(at), grid.Slot(at))
		assert.Equal(t, dispatchmarket.SlotStart(at), grid.SlotStart(at))
		assert.Equal(t, dispatchmarket.SlotStart(at)+dispatchmarket.SlotSeconds, grid.SlotEnd(at))
		assert.Equal(t, "UTC", grid.Timezone())
	}
}

func TestGrid_KeepsALocalHourInTheSameSlotAcrossDaylightSaving(t *testing.T) {
	t.Parallel()

	grid, chicago := chicagoGrid(t)
	summer := time.Date(2026, time.July, 14, 18, 0, 0, 0, chicago).Unix()
	winter := time.Date(2026, time.December, 15, 18, 0, 0, 0, chicago).Unix()

	assert.Equal(t, grid.Slot(summer), grid.Slot(winter),
		"six in the evening on a Tuesday is one slot of the week all year")
	assert.NotEqual(t, dispatchmarket.SlotOf(summer), dispatchmarket.SlotOf(winter),
		"which the UTC grid splits across two")
	assert.Equal(t, int16(6+4), grid.Slot(summer), "Tuesday's fifth slot, 16:00 to 20:00")
}

func TestGrid_SlotsStartOnLocalFourHourMarksAndWeeksOnLocalMonday(t *testing.T) {
	t.Parallel()

	grid, chicago := chicagoGrid(t)
	at := time.Date(2026, time.November, 12, 15, 30, 0, 0, chicago).Unix()

	start := time.Unix(grid.SlotStart(at), 0).In(chicago)
	assert.Equal(t, time.Date(2026, time.November, 12, 12, 0, 0, 0, chicago), start)
	assert.Equal(
		t,
		time.Date(2026, time.November, 12, 16, 0, 0, 0, chicago).Unix(),
		grid.SlotEnd(at),
	)

	sunday := time.Date(2026, time.November, 15, 23, 59, 0, 0, chicago).Unix()
	monday := time.Date(2026, time.November, 16, 0, 0, 0, 0, chicago).Unix()
	assert.Equal(t, grid.Week(at), grid.Week(sunday))
	assert.Equal(t, grid.Week(at)+1, grid.Week(monday))
	assert.Equal(t, int16(0), grid.Slot(monday))
	assert.Equal(t, dispatchmarket.SlotsPerWeek-1, grid.Slot(sunday))
}

func TestGrid_ADayThatChangesClocksHasOneLongOrShortSlot(t *testing.T) {
	t.Parallel()

	grid, chicago := chicagoGrid(t)
	fallBack := time.Date(2026, time.November, 1, 0, 0, 0, 0, chicago).Unix()
	springForward := time.Date(2026, time.March, 8, 0, 0, 0, 0, chicago).Unix()

	long := grid.Starts(fallBack, 7)
	assert.Equal(t, int64(5*3600), long[1]-long[0], "the night the clocks go back")
	short := grid.Starts(springForward, 7)
	assert.Equal(t, int64(3*3600), short[1]-short[0], "the night they go forward")
	for _, starts := range [][]int64{long, short} {
		for i, start := range starts {
			local := time.Unix(start, 0).In(chicago)
			assert.Equal(t, (i*4)%24, local.Hour(), "every slot still starts on a four-hour mark")
			assert.Equal(t, start, grid.SlotStart(start))
			if i > 0 {
				assert.Equal(t, start, grid.SlotEnd(starts[i-1]))
				assert.Equal(t, starts[i-1], grid.SlotStart(start-1))
			}
		}
		assert.Equal(t, grid.Day(starts[0])+1, grid.Day(starts[6]), "six slots are one local day")
	}
}

func TestGrid_DayStartIsLocalMidnightAndWallDayNamesTheSameDay(t *testing.T) {
	t.Parallel()

	grid, chicago := chicagoGrid(t)
	at := time.Date(2026, time.November, 12, 15, 30, 0, 0, chicago).Unix()

	assert.Equal(
		t,
		time.Date(2026, time.November, 12, 0, 0, 0, 0, chicago).Unix(),
		grid.DayStart(at, 0),
	)
	assert.Equal(
		t,
		time.Date(2026, time.November, 9, 0, 0, 0, 0, chicago).Unix(),
		grid.DayStart(at, 3),
	)
	assert.Equal(t, grid.Day(at), dispatchmarket.WallDay(grid.Week(at), grid.Slot(at)))
	assert.Equal(t,
		time.Date(2026, time.November, 12, 0, 0, 0, 0, time.UTC).Unix()/dispatchmarket.DaySeconds,
		grid.Day(at), "a day number is the local calendar date")
}
