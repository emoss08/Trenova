package rategeo_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScopeKeys_KeysEveryScopeInOneCall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		scope rategeo.Scope
		want  string
	}{
		{"any", rategeo.Scope{Type: rategeo.ScopeTypeAny}, "ANY"},
		{"location", rategeo.Scope{Type: rategeo.ScopeTypeLocation, Value: "loc_1"}, "LOC:loc_1"},
		{
			"zip5 strips the plus four",
			rategeo.Scope{Type: rategeo.ScopeTypeZip5, Value: "60601-1234"},
			"Z5:60601",
		},
		{"zip3 truncates", rategeo.Scope{Type: rategeo.ScopeTypeZip3, Value: "60601"}, "Z3:606"},
		{
			"zip3 rejects a postal code that is too short",
			rategeo.Scope{Type: rategeo.ScopeTypeZip3, Value: "60"},
			"",
		},
		{
			"city state folds case and inner whitespace",
			rategeo.Scope{Type: rategeo.ScopeTypeCityState, Value: "ust_il", City: "  saint   louis "},
			"CS:ust_il|SAINT LOUIS",
		},
		{
			"city state needs a city",
			rategeo.Scope{Type: rategeo.ScopeTypeCityState, Value: "ust_il"},
			"",
		},
		{"zone", rategeo.Scope{Type: rategeo.ScopeTypeZone, Value: "rzn_1"}, "ZN:rzn_1"},
		{"state", rategeo.Scope{Type: rategeo.ScopeTypeState, Value: "ust_il"}, "ST:ust_il"},
		{"country upper cases", rategeo.Scope{Type: rategeo.ScopeTypeCountry, Value: "usa"}, "CT:USA"},
		{"radius has no key", rategeo.Scope{Type: rategeo.ScopeTypeRadius}, ""},
		{"a location with no value has no key", rategeo.Scope{Type: rategeo.ScopeTypeLocation}, ""},
	}

	scopes := make([]rategeo.Scope, 0, len(tests))
	for _, tt := range tests {
		scopes = append(scopes, tt.scope)
	}

	keys, err := rategeo.ScopeKeys(t.Context(), scopes)
	require.NoError(t, err)
	require.Len(t, keys, len(tests))
	for i, tt := range tests {
		assert.Equal(t, tt.want, keys[i], tt.name)
	}
}

func TestMatchLane_CoversEveryScopeARuleCouldStore(t *testing.T) {
	t.Parallel()

	stateID := pulid.MustNew("ust_")
	locationID := pulid.MustNew("loc_")
	zoneID := pulid.MustNew("rzn_")

	origin := rategeo.Place{
		LocationID:  locationID,
		PostalCode:  "60601-1234",
		City:        "Chicago",
		StateID:     stateID,
		CountryISO3: "USA",
		ZoneIDs:     []pulid.ID{zoneID},
	}
	destination := rategeo.Place{StateID: stateID}

	lane, err := rategeo.MatchLane(t.Context(), &origin, &destination)
	require.NoError(t, err)

	assert.Equal(t, []string{
		"LOC:" + locationID.String(),
		"Z5:60601",
		"Z3:606",
		"CS:" + stateID.String() + "|CHICAGO",
		"ZN:" + zoneID.String(),
		"ST:" + stateID.String(),
		"CT:USA",
		"ANY",
	}, lane.OriginKeys)
	assert.Equal(t, []string{"ST:" + stateID.String(), "ANY"}, lane.DestinationKeys)
	require.Len(t, lane.Candidates, len(lane.OriginKeys)*len(lane.DestinationKeys))
	assert.Equal(t, "LOC:"+locationID.String()+">ST:"+stateID.String(), lane.Candidates[0])
	assert.Equal(t, "ANY>ANY", lane.Candidates[len(lane.Candidates)-1])
}

func TestMatchLane_ReachesEveryKeyedScopeOfItsPlace(t *testing.T) {
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

	lane, err := rategeo.MatchLane(t.Context(), &place, &rategeo.Place{})
	require.NoError(t, err)
	keys, err := rategeo.ScopeKeys(t.Context(), scopes)
	require.NoError(t, err)

	for i, scope := range scopes {
		require.NotEmpty(t, keys[i], "scope %s should be keyable", scope.Type)
		assert.Contains(t, lane.OriginKeys, keys[i], "place should match a %s rule", scope.Type)
	}
}

func TestMatchLane_AnEmptyPlaceMatchesOnlyAny(t *testing.T) {
	t.Parallel()

	lane, err := rategeo.MatchLane(t.Context(), &rategeo.Place{}, &rategeo.Place{})
	require.NoError(t, err)

	assert.Equal(t, []string{"ANY"}, lane.OriginKeys)
	assert.Equal(t, []string{"ANY>ANY"}, lane.Candidates)
}
