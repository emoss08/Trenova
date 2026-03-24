package timeutils_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
)

func TestNowUnix(t *testing.T) {
	t.Parallel()

	t.Run("returns current unix timestamp", func(t *testing.T) {
		t.Parallel()
		before := time.Now().Unix()
		result := timeutils.NowUnix()
		after := time.Now().Unix()

		assert.GreaterOrEqual(t, result, before)
		assert.LessOrEqual(t, result, after)
	})

	t.Run("returns positive value", func(t *testing.T) {
		t.Parallel()
		result := timeutils.NowUnix()
		assert.Greater(t, result, int64(0))
	})

	t.Run("returns reasonable timestamp", func(t *testing.T) {
		t.Parallel()
		result := timeutils.NowUnix()
		year2020 := int64(1577836800)
		year2100 := int64(4102444800)

		assert.Greater(t, result, year2020)
		assert.Less(t, result, year2100)
	})
}

func TestWithDefaultDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		val      time.Duration
		def      time.Duration
		expected time.Duration
	}{
		{"returns val when non-zero", 5 * time.Second, 10 * time.Second, 5 * time.Second},
		{"returns def when val is zero", 0, 10 * time.Second, 10 * time.Second},
		{"returns negative val", -5 * time.Second, 10 * time.Second, -5 * time.Second},
		{"returns zero def when val is zero", 0, 0, 0},
		{"returns val when def is zero", 5 * time.Second, 0, 5 * time.Second},
		{"handles nanoseconds", time.Nanosecond, time.Hour, time.Nanosecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := timeutils.WithDefaultDuration(tt.val, tt.def)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseTimeRFC3339(t *testing.T) {
	t.Parallel()

	t.Run("parses RFC3339 format", func(t *testing.T) {
		t.Parallel()
		result, ok := timeutils.ParseTimeRFC3339("2024-01-15T10:30:00Z")
		assert.True(t, ok)
		assert.Equal(t, 2024, result.Year())
		assert.Equal(t, time.January, result.Month())
		assert.Equal(t, 15, result.Day())
		assert.Equal(t, 10, result.Hour())
		assert.Equal(t, 30, result.Minute())
	})

	t.Run("parses RFC3339 with timezone offset", func(t *testing.T) {
		t.Parallel()
		result, ok := timeutils.ParseTimeRFC3339("2024-01-15T10:30:00+05:00")
		assert.True(t, ok)
		assert.Equal(t, 2024, result.Year())
	})

	t.Run("parses RFC3339Nano format", func(t *testing.T) {
		t.Parallel()
		result, ok := timeutils.ParseTimeRFC3339("2024-01-15T10:30:00.123456789Z")
		assert.True(t, ok)
		assert.Equal(t, 2024, result.Year())
		assert.Equal(t, 123456789, result.Nanosecond())
	})

	t.Run("parses unix timestamp in seconds", func(t *testing.T) {
		t.Parallel()
		result, ok := timeutils.ParseTimeRFC3339("1705315800")
		assert.True(t, ok)
		assert.Equal(t, int64(1705315800), result.Unix())
	})

	t.Run("parses unix timestamp in milliseconds", func(t *testing.T) {
		t.Parallel()
		result, ok := timeutils.ParseTimeRFC3339("1705315800123")
		assert.True(t, ok)
		assert.Equal(t, int64(1705315800), result.Unix())
		assert.Equal(t, 123000000, result.Nanosecond())
	})

	t.Run("returns false for empty string", func(t *testing.T) {
		t.Parallel()
		result, ok := timeutils.ParseTimeRFC3339("")
		assert.False(t, ok)
		assert.True(t, result.IsZero())
	})

	t.Run("returns false for invalid format", func(t *testing.T) {
		t.Parallel()
		result, ok := timeutils.ParseTimeRFC3339("not-a-date")
		assert.False(t, ok)
		assert.True(t, result.IsZero())
	})

	t.Run("returns false for partial date", func(t *testing.T) {
		t.Parallel()
		result, ok := timeutils.ParseTimeRFC3339("2024-01-15")
		assert.False(t, ok)
		assert.True(t, result.IsZero())
	})
}

func TestDayBoundariesUnix(t *testing.T) {
	t.Parallel()

	t.Run("computes UTC day boundaries", func(t *testing.T) {
		t.Parallel()

		ts := time.Date(2026, time.January, 15, 15, 30, 0, 0, time.UTC).Unix()
		start, err := timeutils.DayStartUnix(ts, "UTC")
		assert.NoError(t, err)
		end, err := timeutils.DayEndUnix(ts, "UTC")
		assert.NoError(t, err)

		assert.Equal(t, time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC).Unix(), start)
		assert.Equal(t, time.Date(2026, time.January, 15, 23, 59, 59, 0, time.UTC).Unix(), end)
	})

	t.Run("computes day boundaries in America/New_York across DST start", func(t *testing.T) {
		t.Parallel()

		loc, err := time.LoadLocation("America/New_York")
		assert.NoError(t, err)

		ts := time.Date(2024, time.March, 10, 18, 0, 0, 0, time.UTC).Unix()
		start, err := timeutils.DayStartUnix(ts, "America/New_York")
		assert.NoError(t, err)
		end, err := timeutils.DayEndUnix(ts, "America/New_York")
		assert.NoError(t, err)

		assert.Equal(t, time.Date(2024, time.March, 10, 0, 0, 0, 0, loc).Unix(), start)
		assert.Equal(t, time.Date(2024, time.March, 10, 23, 59, 59, 0, loc).Unix(), end)
	})

	t.Run("returns error for invalid timezone", func(t *testing.T) {
		t.Parallel()

		_, err := timeutils.DayStartUnix(time.Now().Unix(), "Mars/Olympus")
		assert.Error(t, err)

		_, err = timeutils.DayEndUnix(time.Now().Unix(), "Mars/Olympus")
		assert.Error(t, err)
	})
}
