package geoutils

import "math"

const (
	earthRadiusMiles   = 3958.7613
	RoadCircuityFactor = 1.2
)

func HaversineMiles(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	return earthRadiusMiles * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

const earthRadiusMeters = 6371008.8

// Point is a place on the earth in decimal degrees.
type Point struct {
	Latitude  float64
	Longitude float64
}

// HaversineMeters is the great-circle distance between two points.
func HaversineMeters(a, b Point) float64 {
	return HaversineMiles(a.Latitude, a.Longitude, b.Latitude, b.Longitude) *
		earthRadiusMeters / earthRadiusMiles
}

// PolygonContains reports whether p lies inside the polygon whose vertices are
// given in order, by casting a ray east from p and counting the edges it
// crosses. A polygon smaller than a triangle contains nothing. The polygon is
// treated as planar, which holds for anything the size of a yard or a site.
func PolygonContains(polygon []Point, p Point) bool {
	if len(polygon) < 3 {
		return false
	}

	inside := false
	for i, j := 0, len(polygon)-1; i < len(polygon); j, i = i, i+1 {
		a, b := polygon[i], polygon[j]
		if (a.Latitude > p.Latitude) == (b.Latitude > p.Latitude) {
			continue
		}
		crossing := (b.Longitude-a.Longitude)*(p.Latitude-a.Latitude)/
			(b.Latitude-a.Latitude) + a.Longitude
		if p.Longitude < crossing {
			inside = !inside
		}
	}

	return inside
}
