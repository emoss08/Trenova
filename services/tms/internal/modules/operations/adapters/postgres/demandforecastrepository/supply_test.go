package demandforecastrepository

import (
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/emoss08/trenova/internal/modules/operations/domain/driverready"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartRow_ReadyAtReadsClocksInMillisecondsAndTimeOffInSeconds(t *testing.T) {
	t.Parallel()

	const now = int64(1_800_000_000)
	ms := func(hours int64) *int64 {
		v := hours * 3_600_000
		return &v
	}
	at := func(v int64) *int64 { return &v }

	plain := startRow{FreeAt: now + 3600}
	assert.Equal(
		t,
		now+3600,
		plain.readyAt(now, nil),
		"no clocks and no time off: when the work ends",
	)

	spent := startRow{
		FreeAt: now, ClocksAt: at(now - 300),
		DriveMs: ms(1), ShiftMs: ms(5), CycleMs: ms(40), CycleTomorrowMs: ms(0),
	}
	assert.Equal(t, now+driverready.ResetSeconds, spent.readyAt(now, nil),
		"an hour of drive time left is a ten-hour rest away from a load")

	rested := startRow{
		FreeAt: now, ClocksAt: at(now - 300),
		DriveMs: ms(11), ShiftMs: ms(14), CycleMs: ms(40),
	}
	assert.Equal(t, now, rested.readyAt(now, nil))

	partial := startRow{FreeAt: now, ClocksAt: at(now - 300), DriveMs: ms(0)}
	assert.Equal(t, now, partial.readyAt(now, nil), "half a set of clocks judges nothing")

	away := startRow{FreeAt: now, TimeOffUntil: at(now + 3*86_400)}
	assert.Equal(t, now+3*86_400, away.readyAt(now, nil), "approved time off holds the driver out")

	leave := startRow{
		FreeAt: now, TimeOffUntil: at(now + 86_400),
		LeaveFrom: at(now - 86_400), LeaveUntil: at(now + 5*86_400),
	}
	assert.False(t, leave.onOpenLeave())
	assert.Equal(t, now+5*86_400, leave.readyAt(now, nil), "the later of time off and leave")

	open := startRow{FreeAt: now, LeaveFrom: at(now - 86_400)}
	assert.True(t, open.onOpenLeave(), "a leave with no end takes the driver off the board")
}

func TestStartRow_ReadyAtWaitsForTheNextDayOnTheDriversPattern(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	saturday := time.Date(2026, time.November, 14, 15, 0, 0, 0, chicago)
	weekdays, start, weekly, offset := "0111110", int16(6*60), int16(1), int16(0)
	row := startRow{
		FreeAt:          saturday.Unix(),
		ShiftDays:       &weekdays,
		ShiftCycleWeeks: &weekly,
		ShiftStart:      &start,
		ShiftOffset:     &offset,
	}

	assert.Equal(t,
		time.Date(2026, time.November, 16, 6, 0, 0, 0, chicago).Unix(),
		row.readyAt(saturday.Unix(), chicago),
		"free on a Saturday, a Monday to Friday driver is back at Monday's shift start")

	tuesday := time.Date(2026, time.November, 17, 15, 0, 0, 0, chicago).Unix()
	row.FreeAt = tuesday
	assert.Equal(t, tuesday, row.readyAt(tuesday, chicago), "a working day changes nothing")

	unrostered := startRow{FreeAt: saturday.Unix()}
	assert.Equal(t, saturday.Unix(), unrostered.readyAt(saturday.Unix(), chicago),
		"a driver on no pattern is free when their work ends")
}
