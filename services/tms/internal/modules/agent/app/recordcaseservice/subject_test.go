package recordcaseservice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readAt(t *testing.T, now string) *subjectRead {
	t.Helper()
	zone, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	at, err := time.ParseInLocation("2006-01-02 15:04", now, zone)
	require.NoError(t, err)

	return &subjectRead{zone: zone, now: at.Unix()}
}

func at(t *testing.T, read *subjectRead, value string) int64 {
	t.Helper()
	parsed, err := time.ParseInLocation("2006-01-02 15:04", value, read.zone)
	require.NoError(t, err)

	return parsed.Unix()
}

func TestClock_NamesTheDayOnlyWhenItIsNotToday(t *testing.T) {
	read := readAt(t, "2026-11-09 06:43")

	assert.Equal(t, "2:00 PM", read.clock(at(t, read, "2026-11-09 14:00")))
	assert.Equal(t, "Tue 7:00 AM", read.clock(at(t, read, "2026-11-10 07:00")))
	assert.Equal(t, "Nov 26, 9:15 AM", read.clock(at(t, read, "2026-11-26 09:15")))
}

func TestWindow_WritesASharedMeridiemOnce(t *testing.T) {
	read := readAt(t, "2026-11-09 06:43")

	assert.Equal(t, "2:00–3:00 PM",
		read.window(at(t, read, "2026-11-09 14:00"), at(t, read, "2026-11-09 15:00")))
	assert.Equal(t, "11:30 AM–12:30 PM",
		read.window(at(t, read, "2026-11-09 11:30"), at(t, read, "2026-11-09 12:30")))
	assert.Equal(t, "Tue 7:00–8:00 AM",
		read.window(at(t, read, "2026-11-10 07:00"), at(t, read, "2026-11-10 08:00")))
	assert.Equal(t, "2:00 PM", read.window(at(t, read, "2026-11-09 14:00"), 0))
}

func TestSpan_SaysMinutesAsAPersonDoes(t *testing.T) {
	assert.Equal(t, "45m", span(45))
	assert.Equal(t, "2h", span(120))
	assert.Equal(t, "2h 10m", span(130))
	assert.Equal(t, "4h 05m", span(245))
	assert.Equal(t, "0m", span(-5))
}

func TestFraction_StaysBetweenNothingAndAll(t *testing.T) {
	assert.InDelta(t, 0.2, fraction(132, 660), 0.001)
	assert.InDelta(t, 0.0, fraction(-10, 660), 0.001)
	assert.InDelta(t, 1.0, fraction(900, 660), 0.001)
	assert.InDelta(t, 0.0, fraction(10, 0), 0.001)
}

func TestProjector_FitsEveryPointInsideTheSketchWithNorthUp(t *testing.T) {
	project := projector([]geoPoint{
		{lat: 32.7767, lon: -96.7970},
		{lat: 41.8781, lon: -87.6298},
	})

	westX, southY := project(32.7767, -96.7970)
	eastX, northY := project(41.8781, -87.6298)

	assert.InDelta(t, routePadding, westX, 0.0001)
	assert.InDelta(t, 1-routePadding, eastX, 0.0001)
	assert.InDelta(t, 1-routePadding, southY, 0.0001)
	assert.InDelta(t, routePadding, northY, 0.0001)
}

func TestProjector_CentresPointsThatShareACoordinate(t *testing.T) {
	project := projector([]geoPoint{{lat: 35, lon: -90}, {lat: 35, lon: -90}})

	x, y := project(35, -90)

	assert.InDelta(t, 0.5, x, 0.0001)
	assert.InDelta(t, 0.5, y, 0.0001)
}
