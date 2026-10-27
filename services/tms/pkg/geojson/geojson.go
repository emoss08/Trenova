package geojson

import (
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel/postgis"
	orbgeojson "github.com/paulmach/orb/geojson"
)

func Map(g *postgis.Geometry) (map[string]any, error) {
	raw, err := String(g)
	if err != nil || raw == "" {
		return nil, err
	}

	var out map[string]any
	if err = sonic.UnmarshalString(raw, &out); err != nil {
		return nil, fmt.Errorf("decode geometry json: %w", err)
	}

	return out, nil
}

func String(g *postgis.Geometry) (string, error) {
	if g == nil || g.Geometry == nil {
		return "", nil
	}

	raw, err := sonic.Marshal(orbgeojson.NewGeometry(g.Geometry))
	if err != nil {
		return "", fmt.Errorf("marshal geometry: %w", err)
	}

	return string(raw), nil
}

func Parse(raw []byte) (*postgis.Geometry, error) {
	g, err := orbgeojson.UnmarshalGeometry(raw)
	if err != nil {
		return nil, fmt.Errorf("parse geometry json: %w", err)
	}

	return &postgis.Geometry{Geometry: g.Geometry()}, nil
}
