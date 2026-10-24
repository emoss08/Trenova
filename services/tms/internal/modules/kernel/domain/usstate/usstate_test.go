package usstate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegionForStateAbbreviation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		abbreviation string
		wantRegion   Region
		wantOK       bool
	}{
		{
			name:         "northeast",
			abbreviation: "NY",
			wantRegion:   RegionNortheast,
			wantOK:       true,
		},
		{
			name:         "midwest",
			abbreviation: "IL",
			wantRegion:   RegionMidwest,
			wantOK:       true,
		},
		{
			name:         "south",
			abbreviation: "TX",
			wantRegion:   RegionSouth,
			wantOK:       true,
		},
		{
			name:         "west",
			abbreviation: "CA",
			wantRegion:   RegionWest,
			wantOK:       true,
		},
		{
			name:         "unknown",
			abbreviation: "PR",
			wantOK:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotRegion, gotOK := RegionForStateAbbreviation(tt.abbreviation)

			assert.Equal(t, tt.wantOK, gotOK)
			assert.Equal(t, tt.wantRegion, gotRegion)
		})
	}
}
