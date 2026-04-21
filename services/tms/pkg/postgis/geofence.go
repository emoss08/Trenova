package postgis

import (
	"fmt"

	"github.com/paulmach/orb"
)

type Vertex struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func NormalizeVertices(vertices []Vertex) []Vertex {
	if len(vertices) == 0 {
		return nil
	}

	normalized := make([]Vertex, 0, len(vertices))
	normalized = append(normalized, vertices...)

	if len(normalized) > 1 && normalized[0] == normalized[len(normalized)-1] {
		normalized = normalized[:len(normalized)-1]
	}

	return normalized
}

func RectangleGeometry(vertices []Vertex) (*Geometry, error) {
	normalized := NormalizeVertices(vertices)
	if len(normalized) < 4 {
		return nil, fmt.Errorf("rectangle geofences require four corners")
	}

	minLon, maxLon := normalized[0].Longitude, normalized[0].Longitude
	minLat, maxLat := normalized[0].Latitude, normalized[0].Latitude

	for _, vertex := range normalized[1:] {
		if vertex.Longitude < minLon {
			minLon = vertex.Longitude
		}
		if vertex.Longitude > maxLon {
			maxLon = vertex.Longitude
		}
		if vertex.Latitude < minLat {
			minLat = vertex.Latitude
		}
		if vertex.Latitude > maxLat {
			maxLat = vertex.Latitude
		}
	}

	if minLon == maxLon || minLat == maxLat {
		return nil, fmt.Errorf("rectangle geofences require a non-zero width and height")
	}

	return &Geometry{
		Geometry: orb.Polygon{
			{
				{minLon, minLat},
				{maxLon, minLat},
				{maxLon, maxLat},
				{minLon, maxLat},
				{minLon, minLat},
			},
		},
	}, nil
}

func PolygonGeometry(vertices []Vertex) (*Geometry, error) {
	normalized := NormalizeVertices(vertices)
	if len(normalized) < 3 {
		return nil, fmt.Errorf("drawn geofences require at least three points")
	}

	ring := make(orb.Ring, 0, len(normalized)+1)
	for _, vertex := range normalized {
		ring = append(ring, orb.Point{vertex.Longitude, vertex.Latitude})
	}
	ring = append(ring, ring[0])

	return &Geometry{Geometry: orb.Polygon{ring}}, nil
}

func VerticesFromGeometry(geometry *Geometry) ([]Vertex, error) {
	if geometry == nil || geometry.Geometry == nil {
		return nil, nil
	}

	polygon, ok := geometry.Geometry.(orb.Polygon)
	if !ok {
		return nil, fmt.Errorf("expected polygon geometry, got %T", geometry.Geometry)
	}
	if len(polygon) == 0 {
		return nil, nil
	}

	vertices := make([]Vertex, 0, len(polygon[0]))
	for _, point := range polygon[0] {
		vertices = append(vertices, Vertex{
			Latitude:  point.Lat(),
			Longitude: point.Lon(),
		})
	}

	if len(vertices) > 1 && vertices[0] == vertices[len(vertices)-1] {
		vertices = vertices[:len(vertices)-1]
	}

	return vertices, nil
}

func CirclePolygonExpression(longitude, latitude, radius float64) (string, []any) {
	return "ST_Buffer(ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography, ?)::geometry",
		[]any{longitude, latitude, radius}
}
