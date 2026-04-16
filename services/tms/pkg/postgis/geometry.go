package postgis

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkb"
	"github.com/paulmach/orb/geojson"
)

type Geometry struct {
	orb.Geometry
}

func (g *Geometry) Scan(src any) error {
	if src == nil {
		return nil
	}

	var data []byte
	var err error

	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data, err = hex.DecodeString(v)
		if err != nil {
			return fmt.Errorf("failed to decode hex string: %w", err)
		}
	default:
		return fmt.Errorf("unsupported scan type for Geometry: %T", src)
	}

	if len(data) == 0 {
		return nil
	}

	wkbData, err := stripEWKBHeader(data)
	if err != nil {
		return fmt.Errorf("failed to process EWKB: %w", err)
	}

	geom, err := wkb.Unmarshal(wkbData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal WKB: %w", err)
	}

	g.Geometry = geom
	return nil
}

func (g *Geometry) Value() (driver.Value, error) {
	if g == nil || g.Geometry == nil {
		return nil, nil //nolint:nilnil // valid for driver.Value
	}

	return wkb.Marshal(g.Geometry)
}

func (g *Geometry) GeoJSON() (map[string]any, error) {
	if g == nil || g.Geometry == nil {
		return nil, nil
	}

	raw, err := sonic.Marshal(geojson.NewGeometry(g.Geometry))
	if err != nil {
		return nil, fmt.Errorf("marshal geometry: %w", err)
	}

	var result map[string]any
	if err = sonic.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("unmarshal geometry json: %w", err)
	}

	return result, nil
}

func (g *Geometry) GeoJSONString() (string, error) {
	if g == nil || g.Geometry == nil {
		return "", nil
	}

	raw, err := sonic.Marshal(geojson.NewGeometry(g.Geometry))
	if err != nil {
		return "", fmt.Errorf("marshal geometry: %w", err)
	}

	return string(raw), nil
}
