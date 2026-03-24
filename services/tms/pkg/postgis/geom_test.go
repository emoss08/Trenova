package postgis

import (
	"encoding/binary"
	"encoding/hex"
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/ewkb"
	"github.com/paulmach/orb/encoding/wkb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeWKB(t *testing.T, lon, lat float64, byteOrder ...binary.ByteOrder) []byte {
	t.Helper()
	opts := []binary.ByteOrder{binary.LittleEndian}
	if len(byteOrder) > 0 {
		opts = byteOrder
	}
	data, err := wkb.Marshal(orb.Point{lon, lat}, opts[0])
	require.NoError(t, err)
	return data
}

func makeEWKB(t *testing.T, lon, lat float64, srid int, byteOrder ...binary.ByteOrder) []byte {
	t.Helper()
	data, err := ewkb.Marshal(orb.Point{lon, lat}, srid, byteOrder...)
	require.NoError(t, err)
	return data
}

func TestNewPoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		lon  float64
		lat  float64
	}{
		{"zero values", 0, 0},
		{"positive coordinates", 77.5946, 12.9716},
		{"negative coordinates", -73.9857, 40.7484},
		{"boundary values", 180, 90},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := NewPoint(tt.lon, tt.lat)
			assert.Equal(t, tt.lon, p.Lon())
			assert.Equal(t, tt.lat, p.Lat())
		})
	}
}

func TestPoint_Scan(t *testing.T) {
	t.Parallel()

	t.Run("nil and empty inputs", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			src  any
		}{
			{"nil input", nil},
			{"empty byte slice", []byte{}},
			{"empty hex string", ""},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				p := &Point{}
				err := p.Scan(tt.src)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("valid byte inputs", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			data []byte
			lon  float64
			lat  float64
		}{
			{"WKB little-endian", makeWKB(t, -73.9857, 40.7484), -73.9857, 40.7484},
			{"WKB big-endian", makeWKB(t, -73.9857, 40.7484, binary.BigEndian), -73.9857, 40.7484},
			{"EWKB 4326 little-endian", makeEWKB(t, 2.3522, 48.8566, 4326), 2.3522, 48.8566},
			{
				"EWKB 4326 big-endian",
				makeEWKB(t, 2.3522, 48.8566, 4326, binary.BigEndian),
				2.3522,
				48.8566,
			},
			{"EWKB 3857", makeEWKB(t, 139.6917, 35.6895, 3857), 139.6917, 35.6895},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				p := &Point{}
				err := p.Scan(tt.data)
				require.NoError(t, err)
				assert.InDelta(t, tt.lon, p.Lon(), 1e-10)
				assert.InDelta(t, tt.lat, p.Lat(), 1e-10)
			})
		}
	})

	t.Run("valid hex string inputs", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			hex  string
			lon  float64
			lat  float64
		}{
			{"WKB hex LE", hex.EncodeToString(makeWKB(t, -122.4194, 37.7749)), -122.4194, 37.7749},
			{
				"EWKB hex LE",
				hex.EncodeToString(makeEWKB(t, -122.4194, 37.7749, 4326)),
				-122.4194,
				37.7749,
			},
			{
				"EWKB hex BE",
				hex.EncodeToString(makeEWKB(t, 151.2093, -33.8688, 4326, binary.BigEndian)),
				151.2093,
				-33.8688,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				p := &Point{}
				err := p.Scan(tt.hex)
				require.NoError(t, err)
				assert.InDelta(t, tt.lon, p.Lon(), 1e-10)
				assert.InDelta(t, tt.lat, p.Lat(), 1e-10)
			})
		}
	})

	t.Run("error cases", func(t *testing.T) {
		t.Parallel()

		t.Run("unsupported type", func(t *testing.T) {
			t.Parallel()
			p := &Point{}
			err := p.Scan(42)
			assert.ErrorContains(t, err, "unsupported scan type for Point")
		})

		t.Run("invalid hex string", func(t *testing.T) {
			t.Parallel()
			p := &Point{}
			err := p.Scan("not-valid-hex!!!")
			assert.ErrorContains(t, err, "failed to decode hex string")
		})

		t.Run("too short data", func(t *testing.T) {
			t.Parallel()
			p := &Point{}
			err := p.Scan([]byte{0x01, 0x02})
			assert.ErrorContains(t, err, "too short")
		})

		t.Run("invalid byte order", func(t *testing.T) {
			t.Parallel()
			p := &Point{}
			data := make([]byte, 21)
			data[0] = 0x05
			err := p.Scan(data)
			assert.ErrorContains(t, err, "invalid byte order")
		})

		t.Run("non-point geometry", func(t *testing.T) {
			t.Parallel()
			lineData, err := wkb.Marshal(orb.LineString{{0, 0}, {1, 1}})
			require.NoError(t, err)
			p := &Point{}
			err = p.Scan(lineData)
			assert.ErrorContains(t, err, "expected Point geometry")
		})
	})
}

func TestPoint_Value(t *testing.T) {
	t.Parallel()

	t.Run("nil pointer returns nil", func(t *testing.T) {
		t.Parallel()
		var p *Point
		v, err := p.Value()
		assert.NoError(t, err)
		assert.Nil(t, v)
	})

	t.Run("zero point", func(t *testing.T) {
		t.Parallel()
		p := NewPoint(0, 0)
		v, err := p.Value()
		require.NoError(t, err)
		require.NotNil(t, v)

		data, ok := v.([]byte)
		require.True(t, ok)
		geom, err := wkb.Unmarshal(data)
		require.NoError(t, err)
		pt, ok := geom.(orb.Point)
		require.True(t, ok)
		assert.Equal(t, 0.0, pt.Lon())
		assert.Equal(t, 0.0, pt.Lat())
	})

	t.Run("valid point", func(t *testing.T) {
		t.Parallel()
		p := NewPoint(-73.9857, 40.7484)
		v, err := p.Value()
		require.NoError(t, err)
		require.NotNil(t, v)

		data, ok := v.([]byte)
		require.True(t, ok)
		geom, err := wkb.Unmarshal(data)
		require.NoError(t, err)
		pt, ok := geom.(orb.Point)
		require.True(t, ok)
		assert.InDelta(t, -73.9857, pt.Lon(), 1e-10)
		assert.InDelta(t, 40.7484, pt.Lat(), 1e-10)
	})
}

func TestPoint_ScanValueRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		lon  float64
		lat  float64
	}{
		{"origin", 0, 0},
		{"New York", -73.9857, 40.7484},
		{"Sydney", 151.2093, -33.8688},
		{"boundary max", 180, 90},
		{"boundary min", -180, -90},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			original := NewPoint(tt.lon, tt.lat)
			v, err := original.Value()
			require.NoError(t, err)

			restored := &Point{}
			err = restored.Scan(v)
			require.NoError(t, err)

			assert.InDelta(t, tt.lon, restored.Lon(), 1e-10)
			assert.InDelta(t, tt.lat, restored.Lat(), 1e-10)
		})
	}
}

func TestStripEWKBHeader(t *testing.T) {
	t.Parallel()

	t.Run("WKB passthrough little-endian", func(t *testing.T) {
		t.Parallel()
		data := makeWKB(t, 10.0, 20.0)
		result, err := stripEWKBHeader(data)
		require.NoError(t, err)
		assert.Equal(t, data, result)
	})

	t.Run("WKB passthrough big-endian", func(t *testing.T) {
		t.Parallel()
		data := makeWKB(t, 10.0, 20.0, binary.BigEndian)
		result, err := stripEWKBHeader(data)
		require.NoError(t, err)
		assert.Equal(t, data, result)
	})

	t.Run("EWKB strips SRID little-endian", func(t *testing.T) {
		t.Parallel()
		ewkbData := makeEWKB(t, 10.0, 20.0, 4326)
		wkbData := makeWKB(t, 10.0, 20.0)

		result, err := stripEWKBHeader(ewkbData)
		require.NoError(t, err)
		assert.Equal(t, wkbData, result)
	})

	t.Run("EWKB strips SRID big-endian", func(t *testing.T) {
		t.Parallel()
		ewkbData := makeEWKB(t, 10.0, 20.0, 4326, binary.BigEndian)
		wkbData := makeWKB(t, 10.0, 20.0, binary.BigEndian)

		result, err := stripEWKBHeader(ewkbData)
		require.NoError(t, err)
		assert.Equal(t, wkbData, result)
	})

	t.Run("EWKB strips different SRID", func(t *testing.T) {
		t.Parallel()
		ewkbData := makeEWKB(t, 10.0, 20.0, 3857)
		wkbData := makeWKB(t, 10.0, 20.0)

		result, err := stripEWKBHeader(ewkbData)
		require.NoError(t, err)
		assert.Equal(t, wkbData, result)
	})

	t.Run("error on too short data", func(t *testing.T) {
		t.Parallel()
		_, err := stripEWKBHeader([]byte{0x01})
		assert.ErrorContains(t, err, "too short")
	})

	t.Run("error on empty data", func(t *testing.T) {
		t.Parallel()
		_, err := stripEWKBHeader([]byte{})
		assert.ErrorContains(t, err, "too short")
	})

	t.Run("error on invalid byte order", func(t *testing.T) {
		t.Parallel()
		data := []byte{0x03, 0x00, 0x00, 0x00, 0x01}
		_, err := stripEWKBHeader(data)
		assert.ErrorContains(t, err, "invalid byte order")
	})

	t.Run("error on EWKB with SRID flag but too short for SRID", func(t *testing.T) {
		t.Parallel()
		data := []byte{
			0x01,
			0x01, 0x00, 0x00, 0x20,
			0xE6, 0x10, 0x00,
		}
		_, err := stripEWKBHeader(data)
		assert.ErrorIs(t, err, ErrInvalidEWKB)
	})
}

func TestPoint_Lon(t *testing.T) {
	t.Parallel()

	t.Run("nil returns zero", func(t *testing.T) {
		t.Parallel()
		var p *Point
		assert.Equal(t, 0.0, p.Lon())
	})

	t.Run("returns longitude", func(t *testing.T) {
		t.Parallel()
		p := NewPoint(-73.9857, 40.7484)
		assert.Equal(t, -73.9857, p.Lon())
	})

	t.Run("zero point", func(t *testing.T) {
		t.Parallel()
		p := NewPoint(0, 0)
		assert.Equal(t, 0.0, p.Lon())
	})
}

func TestPoint_Lat(t *testing.T) {
	t.Parallel()

	t.Run("nil returns zero", func(t *testing.T) {
		t.Parallel()
		var p *Point
		assert.Equal(t, 0.0, p.Lat())
	})

	t.Run("returns latitude", func(t *testing.T) {
		t.Parallel()
		p := NewPoint(-73.9857, 40.7484)
		assert.Equal(t, 40.7484, p.Lat())
	})

	t.Run("zero point", func(t *testing.T) {
		t.Parallel()
		p := NewPoint(0, 0)
		assert.Equal(t, 0.0, p.Lat())
	})
}

func TestPoint_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		point *Point
		valid bool
	}{
		{"nil pointer", nil, false},
		{"origin", NewPoint(0, 0), true},
		{"valid NYC", NewPoint(-73.9857, 40.7484), true},
		{"max boundary", NewPoint(180, 90), true},
		{"min boundary", NewPoint(-180, -90), true},
		{"lon too high", NewPoint(180.1, 0), false},
		{"lon too low", NewPoint(-180.1, 0), false},
		{"lat too high", NewPoint(0, 90.1), false},
		{"lat too low", NewPoint(0, -90.1), false},
		{"both out of range", NewPoint(200, 100), false},
		{"lon at positive boundary", NewPoint(180, 45), true},
		{"lat at negative boundary", NewPoint(45, -90), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.valid, tt.point.IsValid())
		})
	}
}

func TestPoint_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		point    *Point
		expected string
	}{
		{"nil pointer", nil, "Point(nil)"},
		{"zero point", NewPoint(0, 0), "POINT(0.000000 0.000000)"},
		{"valid point", NewPoint(-73.9857, 40.7484), "POINT(-73.985700 40.748400)"},
		{"precision truncation", NewPoint(1.123456789, 2.987654321), "POINT(1.123457 2.987654)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.point.String())
		})
	}
}
