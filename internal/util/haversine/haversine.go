// CREDIT: https://github.com/umahmood/haversine/
package haversine

import (
	"math"
)

const (
	EarthRadiusMi = 3958 // Radius of the Earth in miles.
	EarthRadiusKm = 6371 // Radius of the Earth in kilometers.
)

// Coord represents a geographic coordinate using latitude and longitude.
type Coord struct {
	Lat float64 // Latitude in decimal degrees.
	Lon float64 // Longitude in decimal degrees.
}

// degreesToRadians converts an angle from degrees to radians.
// The function is unexported as it's an internal helper function.
func degreesToRadians(deg float64) float64 {
	return deg * math.Pi / 180
}

// Distance calculates the shortest path between two coordinates (p and q)
// on the surface of the Earth and returns the distance in both miles and kilometers.
// This uses the Haversine formula to calculate the great-circle distance between
// two points, which is the shortest over the earth's surface.
//
// Parameters:
//   - p: Origin coordinate as a Coord struct.
//   - q: Destination coordinate as a Coord struct.
//
// Returns:
//   - Distance in miles (float64).
//   - Distance in kilometers (float64).
func Distance(p, q Coord) (float64, float64) {
	lat1 := degreesToRadians(p.Lat)
	lon1 := degreesToRadians(p.Lon)
	lat2 := degreesToRadians(q.Lat)
	lon2 := degreesToRadians(q.Lon)

	diffLat := lat2 - lat1
	diffLon := lon2 - lon1

	a := math.Pow(math.Sin(diffLat/2), 2) + math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(diffLon/2), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Calculate the distance in miles and kilometers
	var mi, km float64
	mi = c * EarthRadiusMi
	km = c * EarthRadiusKm

	return mi, km
}
