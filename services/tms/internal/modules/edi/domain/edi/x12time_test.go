package edi_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestX12DateTimeUnix(t *testing.T) {
	t.Parallel()

	newYork, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	denver, err := time.LoadLocation("America/Denver")
	require.NoError(t, err)

	for name, tc := range map[string]struct {
		value edi.X12DateTime
		local *time.Location
		want  time.Time
	}{
		"universal": {
			value: edi.X12DateTime{Date: "20260707", Time: "1200", TimeCode: "UT"},
			want:  time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC),
		},
		"eastern time follows daylight saving": {
			value: edi.X12DateTime{Date: "20260707", Time: "1200", TimeCode: "ET"},
			want:  time.Date(2026, 7, 7, 12, 0, 0, 0, newYork),
		},
		"eastern standard is fixed in summer": {
			value: edi.X12DateTime{Date: "20260707", Time: "1200", TimeCode: "ES"},
			want:  time.Date(2026, 7, 7, 17, 0, 0, 0, time.UTC),
		},
		"pacific daylight": {
			value: edi.X12DateTime{Date: "20260107", Time: "0800", TimeCode: "pd"},
			want:  time.Date(2026, 1, 7, 15, 0, 0, 0, time.UTC),
		},
		"newfoundland standard half hour": {
			value: edi.X12DateTime{Date: "20260107", Time: "0800", TimeCode: "NS"},
			want:  time.Date(2026, 1, 7, 11, 30, 0, 0, time.UTC),
		},
		"local time uses the stop zone": {
			value: edi.X12DateTime{Date: "20260107", Time: "080000", TimeCode: "LT"},
			local: denver,
			want:  time.Date(2026, 1, 7, 8, 0, 0, 0, denver),
		},
		"no code uses the stop zone": {
			value: edi.X12DateTime{Date: "260107", Time: "0800"},
			local: denver,
			want:  time.Date(2026, 1, 7, 8, 0, 0, 0, denver),
		},
		"local time without a zone is read as UTC": {
			value: edi.X12DateTime{Date: "20260107", Time: "08001599", TimeCode: "LT"},
			want:  time.Date(2026, 1, 7, 8, 0, 15, 0, time.UTC),
		},
		"date only": {
			value: edi.X12DateTime{Date: "20260107", TimeCode: "CT"},
			want:  time.Date(2026, 1, 7, 6, 0, 0, 0, time.UTC),
		},
	} {
		got, ok := tc.value.Unix(tc.local)
		require.True(t, ok, name)
		assert.Equal(t, tc.want.Unix(), got, name)
	}

	_, ok := edi.X12DateTime{}.Unix(time.UTC)
	assert.False(t, ok)
	_, ok = edi.X12DateTime{Date: "2026-01-07"}.Unix(time.UTC)
	assert.False(t, ok)
}

func TestX12DateTimeIsLocal(t *testing.T) {
	t.Parallel()

	assert.True(t, edi.X12DateTime{Date: "20260107"}.IsLocal())
	assert.True(t, edi.X12DateTime{Date: "20260107", TimeCode: "LT"}.IsLocal())
	assert.True(t, edi.X12DateTime{Date: "20260107", TimeCode: "ZZ"}.IsLocal())
	assert.False(t, edi.X12DateTime{Date: "20260107", TimeCode: "CT"}.IsLocal())
	assert.True(t, edi.IsKnownX12TimeCode(" ut "))
	assert.False(t, edi.IsKnownX12TimeCode("ZZ"))
}

func TestParseX12Coordinates(t *testing.T) {
	t.Parallel()

	lat := edi.ParseX12Latitude("413200", "N")
	require.NotNil(t, lat)
	assert.InDelta(t, 41.5333, *lat, 0.0001)

	lon := edi.ParseX12Longitude("0880500", "W")
	require.NotNil(t, lon)
	assert.InDelta(t, -88.0833, *lon, 0.0001)

	decimal := edi.ParseX12Latitude("-33.8688", "")
	require.NotNil(t, decimal)
	assert.InDelta(t, -33.8688, *decimal, 0.00001)

	south := edi.ParseX12Latitude("33.8688", "s")
	require.NotNil(t, south)
	assert.InDelta(t, -33.8688, *south, 0.00001)

	for _, bad := range []string{"", "12", "9100000", "416000", "41A200", "NaN.0"} {
		assert.Nil(t, edi.ParseX12Latitude(bad, "N"), bad)
	}
	assert.Nil(t, edi.ParseX12Longitude("1810000", "E"))
}
