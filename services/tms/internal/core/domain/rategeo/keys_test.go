package rategeo_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/rategeo"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScopeKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scope  rategeo.Scope
		want   string
		wantOK bool
	}{
		{
			name:   "any",
			scope:  rategeo.Scope{Type: rategeo.ScopeTypeAny},
			want:   rategeo.KeyAny,
			wantOK: true,
		},
		{
			name:   "location",
			scope:  rategeo.Scope{Type: rategeo.ScopeTypeLocation, Value: "loc_1"},
			want:   "LOC:loc_1",
			wantOK: true,
		},
		{
			name:   "zip5 strips the plus four",
			scope:  rategeo.Scope{Type: rategeo.ScopeTypeZip5, Value: "60601-1234"},
			want:   "Z5:60601",
			wantOK: true,
		},
		{
			name:   "zip3 truncates",
			scope:  rategeo.Scope{Type: rategeo.ScopeTypeZip3, Value: "60601"},
			want:   "Z3:606",
			wantOK: true,
		},
		{
			name:   "zip3 rejects a postal code that is too short",
			scope:  rategeo.Scope{Type: rategeo.ScopeTypeZip3, Value: "60"},
			wantOK: false,
		},
		{
			name: "city state folds case and inner whitespace",
			scope: rategeo.Scope{
				Type:  rategeo.ScopeTypeCityState,
				Value: "ust_il",
				City:  "  saint   louis ",
			},
			want:   "CS:ust_il|SAINT LOUIS",
			wantOK: true,
		},
		{
			name:   "zone",
			scope:  rategeo.Scope{Type: rategeo.ScopeTypeZone, Value: "rzn_1"},
			want:   "ZN:rzn_1",
			wantOK: true,
		},
		{
			name:   "state",
			scope:  rategeo.Scope{Type: rategeo.ScopeTypeState, Value: "ust_il"},
			want:   "ST:ust_il",
			wantOK: true,
		},
		{
			name:   "country upper cases",
			scope:  rategeo.Scope{Type: rategeo.ScopeTypeCountry, Value: "usa"},
			want:   "CT:USA",
			wantOK: true,
		},
		{
			name:   "radius has no key",
			scope:  rategeo.Scope{Type: rategeo.ScopeTypeRadius},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := tt.scope.Key()
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestPlaceMatchKeysCoversEveryScopeARuleCouldStore(t *testing.T) {
	t.Parallel()

	stateID := pulid.MustNew("ust_")
	locationID := pulid.MustNew("loc_")
	zoneID := pulid.MustNew("rzn_")

	place := rategeo.Place{
		LocationID:  locationID,
		PostalCode:  "60601-1234",
		City:        "Chicago",
		StateID:     stateID,
		CountryISO3: "USA",
		ZoneIDs:     []pulid.ID{zoneID},
	}

	keys := place.MatchKeys()

	assert.Equal(t, []string{
		"LOC:" + locationID.String(),
		"Z5:60601",
		"Z3:606",
		"CS:" + stateID.String() + "|CHICAGO",
		"ZN:" + zoneID.String(),
		"ST:" + stateID.String(),
		"CT:USA",
		rategeo.KeyAny,
	}, keys)
}

// A rule keyed at any granularity must be reachable from the place it covers.
// This is the invariant the whole lane key scheme rests on: if a scope can be
// stored, the matching place has to produce that exact string.
func TestEveryKeyedScopeIsReachableFromItsPlace(t *testing.T) {
	t.Parallel()

	stateID := pulid.MustNew("ust_")
	locationID := pulid.MustNew("loc_")
	zoneID := pulid.MustNew("rzn_")

	place := rategeo.Place{
		LocationID:  locationID,
		PostalCode:  "60601",
		City:        "Chicago",
		StateID:     stateID,
		CountryISO3: "USA",
		ZoneIDs:     []pulid.ID{zoneID},
	}

	scopes := []rategeo.Scope{
		{Type: rategeo.ScopeTypeAny},
		{Type: rategeo.ScopeTypeCountry, Value: "USA"},
		{Type: rategeo.ScopeTypeState, Value: stateID.String()},
		{Type: rategeo.ScopeTypeZone, Value: zoneID.String()},
		{Type: rategeo.ScopeTypeCityState, Value: stateID.String(), City: "Chicago"},
		{Type: rategeo.ScopeTypeZip3, Value: "60601"},
		{Type: rategeo.ScopeTypeZip5, Value: "60601"},
		{Type: rategeo.ScopeTypeLocation, Value: locationID.String()},
	}

	keys := place.MatchKeys()

	for _, scope := range scopes {
		key, ok := scope.Key()
		require.True(t, ok, "scope %s should be keyable", scope.Type)
		assert.Contains(t, keys, key, "place should match a %s rule", scope.Type)
	}
}

func TestPlaceMatchKeysAlwaysEndsWithAny(t *testing.T) {
	t.Parallel()

	empty := rategeo.Place{}
	keys := empty.MatchKeys()

	require.Len(t, keys, 1)
	assert.Equal(t, rategeo.KeyAny, keys[0])
}

func TestLaneKeyCandidatesIsTheCrossProduct(t *testing.T) {
	t.Parallel()

	candidates := rategeo.LaneKeyCandidates(
		[]string{"Z3:606", "ANY"},
		[]string{"ST:ust_ga", "ANY"},
	)

	assert.Equal(t, []string{
		"Z3:606>ST:ust_ga",
		"Z3:606>ANY",
		"ANY>ST:ust_ga",
		"ANY>ANY",
	}, candidates)
}

// Specificity has to be a total order over scope types, because it is the first
// thing that separates two rules that both match. A tie between two different
// granularities would make the winner depend on row order.
func TestScopeWeightsAreStrictlyOrdered(t *testing.T) {
	t.Parallel()

	ordered := []rategeo.ScopeType{
		rategeo.ScopeTypeAny,
		rategeo.ScopeTypeCountry,
		rategeo.ScopeTypeState,
		rategeo.ScopeTypeZone,
		rategeo.ScopeTypeRadius,
		rategeo.ScopeTypeCityState,
		rategeo.ScopeTypeZip3,
		rategeo.ScopeTypeZip5,
		rategeo.ScopeTypeLocation,
	}

	for i := 1; i < len(ordered); i++ {
		assert.Greater(t,
			ordered[i].Weight(), ordered[i-1].Weight(),
			"%s must outrank %s", ordered[i], ordered[i-1],
		)
	}
}

func TestLaneSpecificityPrefersTheNarrowerLane(t *testing.T) {
	t.Parallel()

	zipToZip := rategeo.LaneSpecificity(rategeo.ScopeTypeZip5, rategeo.ScopeTypeZip5)
	zipToState := rategeo.LaneSpecificity(rategeo.ScopeTypeZip5, rategeo.ScopeTypeState)
	stateToState := rategeo.LaneSpecificity(rategeo.ScopeTypeState, rategeo.ScopeTypeState)
	anyToAny := rategeo.LaneSpecificity(rategeo.ScopeTypeAny, rategeo.ScopeTypeAny)

	assert.Greater(t, zipToZip, zipToState)
	assert.Greater(t, zipToState, stateToState)
	assert.Greater(t, stateToState, anyToAny)
	assert.Equal(t, int32(0), anyToAny)
}
