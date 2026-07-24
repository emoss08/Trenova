package geoutils

import (
	"math"
	"testing"
)

func TestHaversineMiles(t *testing.T) {
	tests := []struct {
		name                   string
		lat1, lon1, lat2, lon2 float64
		want                   float64
		tolerance              float64
	}{
		{name: "same point", lat1: 32.7767, lon1: -96.797, lat2: 32.7767, lon2: -96.797, want: 0, tolerance: 0.001},
		{name: "dallas to houston", lat1: 32.7767, lon1: -96.797, lat2: 29.7604, lon2: -95.3698, want: 225, tolerance: 5},
		{name: "dallas to fort worth", lat1: 32.7767, lon1: -96.797, lat2: 32.7555, lon2: -97.3308, want: 31, tolerance: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HaversineMiles(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			if math.Abs(got-tt.want) > tt.tolerance {
				t.Fatalf("HaversineMiles() = %f, want %f ± %f", got, tt.want, tt.tolerance)
			}
		})
	}
}
