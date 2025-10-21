package postgis

import (
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkb"
)

// Point wraps orb.Point and implements sql.Scanner and driver.Valuer
// for PostGIS geography/geometry types
type Point struct {
	orb.Point
}

func NewPoint(lon, lat float64) *Point {
	return &Point{Point: orb.Point{lon, lat}}
}

func (p *Point) Scan(src interface{}) error {
	if src == nil {
		return nil
	}

	var data []byte
	var err error

	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		// PostGIS returns hex-encoded EWKB string
		data, err = hex.DecodeString(v)
		if err != nil {
			return fmt.Errorf("failed to decode hex string: %w", err)
		}
	default:
		return fmt.Errorf("unsupported scan type for Point: %T", src)
	}

	if len(data) == 0 {
		return nil
	}

	// PostGIS returns EWKB (Extended Well-Known Binary) with SRID
	// We need to strip the SRID to get standard WKB that orb can parse
	wkbData, err := stripEWKBHeader(data)
	if err != nil {
		return fmt.Errorf("failed to process EWKB: %w", err)
	}

	// Decode WKB using orb
	geom, err := wkb.Unmarshal(wkbData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal WKB: %w", err)
	}

	point, ok := geom.(orb.Point)
	if !ok {
		return fmt.Errorf("expected Point geometry, got %T", geom)
	}

	p.Point = point
	return nil
}

func (p *Point) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil //nolint:nilnil // this is a valid value for a driver.Value
	}

	return wkb.Marshal(p.Point)
}

func stripEWKBHeader(data []byte) ([]byte, error) {
	if len(data) < 5 {
		return nil, fmt.Errorf("invalid EWKB data: too short (%d bytes)", len(data))
	}

	byteOrder := data[0]
	if byteOrder != 0 && byteOrder != 1 {
		return nil, fmt.Errorf("invalid byte order: %d", byteOrder)
	}

	var geomType uint32
	if byteOrder == 0 {
		geomType = binary.BigEndian.Uint32(data[1:5])
	} else {
		geomType = binary.LittleEndian.Uint32(data[1:5])
	}

	hasSRID := (geomType & 0x20000000) != 0

	if !hasSRID {
		return data, nil
	}

	if len(data) < 9 {
		return nil, ErrInvalidEWKB
	}

	geomType &= ^uint32(0x20000000)

	wkbData := make([]byte, len(data)-4)
	wkbData[0] = byteOrder

	if byteOrder == 0 {
		binary.BigEndian.PutUint32(wkbData[1:5], geomType)
	} else {
		binary.LittleEndian.PutUint32(wkbData[1:5], geomType)
	}

	copy(wkbData[5:], data[9:])

	return wkbData, nil
}

func (p *Point) Lon() float64 {
	if p == nil {
		return 0
	}
	return p.Point.Lon()
}

func (p *Point) Lat() float64 {
	if p == nil {
		return 0
	}
	return p.Point.Lat()
}

func (p *Point) IsValid() bool {
	if p == nil {
		return false
	}
	lon, lat := p.Lon(), p.Lat()
	return lon >= -180 && lon <= 180 && lat >= -90 && lat <= 90
}

func (p *Point) String() string {
	if p == nil {
		return "Point(nil)"
	}
	return fmt.Sprintf("POINT(%.6f %.6f)", p.Lon(), p.Lat())
}
