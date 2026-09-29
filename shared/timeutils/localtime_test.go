package timeutils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustZone(t *testing.T, name string) *time.Location {
	t.Helper()

	loc, err := time.LoadLocation(name)
	require.NoError(t, err)

	return loc
}

func TestParseLocalDateTime_ReadsWallClockInTheZone(t *testing.T) {
	t.Parallel()

	chicago := mustZone(t, "America/Chicago")
	want := time.Date(2026, time.October, 1, 8, 0, 0, 0, chicago).Unix()

	for _, value := range []string{
		"2026-10-01T08:00",
		"2026-10-01T08:00:00",
		" 2026-10-01T08:00 ",
		"2026-10-01 08:00",
	} {
		got, err := ParseLocalDateTime(value, chicago)
		require.NoError(t, err, value)
		assert.Equal(t, want, got, value)
	}
}

func TestParseLocalDateTime_KeepsAnExplicitOffset(t *testing.T) {
	t.Parallel()

	got, err := ParseLocalDateTime("2026-10-01T08:00:00-05:00", mustZone(t, "Asia/Tokyo"))
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, time.October, 1, 13, 0, 0, 0, time.UTC).Unix(), got)
}

func TestParseLocalDateTime_DefaultsToUTCWithoutAZone(t *testing.T) {
	t.Parallel()

	got, err := ParseLocalDateTime("2026-10-01T08:00", nil)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, time.October, 1, 8, 0, 0, 0, time.UTC).Unix(), got)
}

func TestParseLocalDateTime_RefusesWhatIsNotADateTime(t *testing.T) {
	t.Parallel()

	chicago := mustZone(t, "America/Chicago")
	for _, value := range []string{"", "1791591001", "2026-10-01", "tomorrow 8am", "2026-13-01T08:00"} {
		_, err := ParseLocalDateTime(value, chicago)
		require.Error(t, err, value)
		assert.ErrorIs(t, err, ErrNotLocalDateTime, value)
	}
}

func TestParseLocalDateTime_RefusesATimeTheClocksSkip(t *testing.T) {
	t.Parallel()

	_, err := ParseLocalDateTime("2026-03-08T02:30", mustZone(t, "America/Chicago"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSkippedLocalTime)
	assert.Contains(t, err.Error(), "America/Chicago")
}

func TestFormatLocalDateTime_RoundTrips(t *testing.T) {
	t.Parallel()

	chicago := mustZone(t, "America/Chicago")
	at := time.Date(2026, time.October, 1, 8, 0, 0, 0, chicago).Unix()
	assert.Equal(t, "2026-10-01T08:00", FormatLocalDateTime(at, chicago))

	withSeconds := at + 15
	formatted := FormatLocalDateTime(withSeconds, chicago)
	assert.Equal(t, "2026-10-01T08:00:15", formatted)

	back, err := ParseLocalDateTime(formatted, chicago)
	require.NoError(t, err)
	assert.Equal(t, withSeconds, back)

	assert.Equal(t, "2026-10-01T13:00", FormatLocalDateTime(at, nil))
}

func TestResolveZone_TakesTheFirstZoneThatResolves(t *testing.T) {
	t.Parallel()

	loc, name := ResolveZone("", "Not/AZone", "America/Denver", "America/Chicago")
	assert.Equal(t, "America/Denver", name)
	assert.Equal(t, "America/Denver", loc.String())

	loc, name = ResolveZone("", "  ")
	assert.Equal(t, "UTC", name)
	assert.Equal(t, time.UTC, loc)

	loc, name = ResolveZone()
	assert.Equal(t, "UTC", name)
	assert.Equal(t, time.UTC, loc)
}
