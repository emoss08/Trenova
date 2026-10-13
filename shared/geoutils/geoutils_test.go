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
		{
			name:      "same point",
			lat1:      32.7767,
			lon1:      -96.797,
			lat2:      32.7767,
			lon2:      -96.797,
			want:      0,
			tolerance: 0.001,
		},
		{
			name:      "dallas to houston",
			lat1:      32.7767,
			lon1:      -96.797,
			lat2:      29.7604,
			lon2:      -95.3698,
			want:      225,
			tolerance: 5,
		},
		{
			name:      "dallas to fort worth",
			lat1:      32.7767,
			lon1:      -96.797,
			lat2:      32.7555,
			lon2:      -97.3308,
			want:      31,
			tolerance: 2,
		},
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

func TestPlanningRoadMilesWidensTheStraightLine(t *testing.T) {
	straight := HaversineMiles(32.7767, -96.797, 29.7604, -95.3698)
	got := PlanningRoadMiles(32.7767, -96.797, 29.7604, -95.3698)
	if math.Abs(got-straight*RoadCircuityFactor) > 1e-9 {
		t.Fatalf("expected %v, got %v", straight*RoadCircuityFactor, got)
	}
}

func TestPlanningDriveSeconds(t *testing.T) {
	if got := PlanningDriveSeconds(100); got != 7200 {
		t.Fatalf("expected 100 miles at 50 mph to be 7200s, got %d", got)
	}
	if got := PlanningDriveSeconds(0); got != 0 {
		t.Fatalf("expected no drive for no miles, got %d", got)
	}
	if got := PlanningDriveSeconds(-5); got != 0 {
		t.Fatalf("expected no drive for negative miles, got %d", got)
	}
}
