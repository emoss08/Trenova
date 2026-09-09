package storedmileage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHazmatSignatureNormalizesValues(t *testing.T) {
	signature := HazmatSignature([]string{"Flammable", "Explosives", "Flammable", " "})

	require.Equal(t, "Explosives,Flammable", signature)
}

func TestConvertDistance(t *testing.T) {
	require.InDelta(t, 1.609344, ConvertDistance(1, "Miles", "Kilometers"), 0.000001)
	require.InDelta(t, 1, ConvertDistance(1.609344, "Kilometers", "Miles"), 0.000001)
	require.Equal(t, 10.0, ConvertDistance(10, "Miles", "Miles"))
}

func TestHasJurisdictionBreakdown(t *testing.T) {
	var nilMileage *StoredMileage
	require.False(t, nilMileage.HasJurisdictionBreakdown())

	require.False(t, (&StoredMileage{}).HasJurisdictionBreakdown())
	require.False(t, (&StoredMileage{JurisdictionDistances: []JurisdictionDistance{}}).HasJurisdictionBreakdown())

	withBreakdown := &StoredMileage{
		JurisdictionDistances: []JurisdictionDistance{
			{Country: "US", Code: "TX", Distance: 120.5},
		},
	}
	require.True(t, withBreakdown.HasJurisdictionBreakdown())
}
