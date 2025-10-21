package distancehelpers

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateHaversineDistance(t *testing.T) {
	tests := []struct {
		name      string
		lat1      float64
		lon1      float64
		lat2      float64
		lon2      float64
		expected  float64
		tolerance float64 // Allow small margin of error
	}{
		{
			name:      "New York to Los Angeles",
			lat1:      40.7128, // New York
			lon1:      -74.0060,
			lat2:      34.0522, // Los Angeles
			lon2:      -118.2437,
			expected:  2451.0, // ~2,451 miles
			tolerance: 10.0,
		},
		{
			name:      "Chicago to Miami",
			lat1:      41.8781, // Chicago
			lon1:      -87.6298,
			lat2:      25.7617, // Miami
			lon2:      -80.1918,
			expected:  1188.0, // ~1,188 miles
			tolerance: 10.0,
		},
		{
			name:      "Dallas to Houston",
			lat1:      32.7767, // Dallas
			lon1:      -96.7970,
			lat2:      29.7604, // Houston
			lon2:      -95.3698,
			expected:  225.0, // ~225 miles
			tolerance: 5.0,
		},
		{
			name:      "San Francisco to Seattle",
			lat1:      37.7749, // San Francisco
			lon1:      -122.4194,
			lat2:      47.6062, // Seattle
			lon2:      -122.3321,
			expected:  680.0, // ~680 miles
			tolerance: 10.0,
		},
		{
			name:      "Same location (zero distance)",
			lat1:      40.7128,
			lon1:      -74.0060,
			lat2:      40.7128,
			lon2:      -74.0060,
			expected:  0.0,
			tolerance: 0.001,
		},
		{
			name:      "Very short distance (1 degree lat)",
			lat1:      40.0,
			lon1:      -74.0,
			lat2:      41.0,
			lon2:      -74.0,
			expected:  69.0, // ~69 miles (1 degree latitude ≈ 69 miles)
			tolerance: 2.0,
		},
		{
			name:      "Equator crossing",
			lat1:      5.0,
			lon1:      0.0,
			lat2:      -5.0,
			lon2:      0.0,
			expected:  690.0, // ~690 miles (10 degrees latitude)
			tolerance: 10.0,
		},
		{
			name:      "Boston to Philadelphia",
			lat1:      42.3601, // Boston
			lon1:      -71.0589,
			lat2:      39.9526, // Philadelphia
			lon2:      -75.1652,
			expected:  270.0, // ~270 miles
			tolerance: 5.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateHaversineDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)

			// Check if the result is within tolerance
			diff := math.Abs(result - tt.expected)
			assert.LessOrEqual(t, diff, tt.tolerance,
				"Distance %.2f miles outside tolerance (expected %.2f ± %.2f, diff: %.2f)",
				result, tt.expected, tt.tolerance, diff)

			// Log the result for reference
			t.Logf(
				"Distance: %.2f miles (expected: %.2f ± %.2f)",
				result,
				tt.expected,
				tt.tolerance,
			)
		})
	}
}

func TestCalculateHaversineDistance_Symmetry(t *testing.T) {
	// Distance from A to B should equal distance from B to A
	tests := []struct {
		name string
		lat1 float64
		lon1 float64
		lat2 float64
		lon2 float64
	}{
		{
			name: "NY to LA",
			lat1: 40.7128,
			lon1: -74.0060,
			lat2: 34.0522,
			lon2: -118.2437,
		},
		{
			name: "Chicago to Miami",
			lat1: 41.8781,
			lon1: -87.6298,
			lat2: 25.7617,
			lon2: -80.1918,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distanceAB := CalculateHaversineDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			distanceBA := CalculateHaversineDistance(tt.lat2, tt.lon2, tt.lat1, tt.lon1)

			diff := math.Abs(distanceAB - distanceBA)
			assert.LessOrEqual(t, diff, 0.001,
				"Distance is not symmetric: A->B = %.6f, B->A = %.6f",
				distanceAB, distanceBA)
		})
	}
}

func TestCalculateHaversineDistance_PositiveResult(t *testing.T) {
	// Distance should always be positive or zero
	tests := []struct {
		name string
		lat1 float64
		lon1 float64
		lat2 float64
		lon2 float64
	}{
		{"North to South", 45.0, 0.0, -45.0, 0.0},
		{"East to West", 0.0, 90.0, 0.0, -90.0},
		{"Random points", 12.34, 56.78, -12.34, -56.78},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateHaversineDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)

			assert.GreaterOrEqual(t, result, 0.0,
				"Distance should be positive or zero, got %.6f", result)
		})
	}
}

func BenchmarkCalculateHaversineDistance(b *testing.B) {
	lat1, lon1 := 40.7128, -74.0060  // New York
	lat2, lon2 := 34.0522, -118.2437 // Los Angeles

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateHaversineDistance(lat1, lon1, lat2, lon2)
	}
}

func BenchmarkCalculateHaversineDistance_ShortDistance(b *testing.B) {
	lat1, lon1 := 40.7128, -74.0060
	lat2, lon2 := 40.7580, -73.9855 // ~5 miles away

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateHaversineDistance(lat1, lon1, lat2, lon2)
	}
}

func TestEarthRadiusMiles(t *testing.T) {
	// Verify the Earth's radius constant is correct
	const expectedRadius = 3958.8

	assert.Equal(t, expectedRadius, EarthRadiusMiles,
		"EarthRadiusMiles constant should be %.1f", expectedRadius)
}

// Example demonstrates basic usage of CalculateHaversineDistance
func ExampleCalculateHaversineDistance() {
	// Calculate distance between New York and Los Angeles
	nyLat, nyLon := 40.7128, -74.0060
	laLat, laLon := 34.0522, -118.2437

	distance := CalculateHaversineDistance(nyLat, nyLon, laLat, laLon)

	// distance is approximately 2451 miles
	_ = distance
}
