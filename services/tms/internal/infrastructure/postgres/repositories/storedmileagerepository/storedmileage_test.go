package storedmileagerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/storedmileage"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestDedupeUpsertEntitiesKeepsLatestByPolicyKey(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	profileID := pulid.MustNew("dp_")
	older := &storedmileage.StoredMileage{
		ID:                pulid.MustNew("smg_"),
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		RouteHash:         "route|hash",
		DistanceUnits:     "Miles",
		RoutingType:       "Practical",
		Method:            "PostalCode",
		DistanceProfileID: profileID,
		HazmatSignature:   "hazmat|class",
		Distance:          10,
		LastCalculatedAt:  100,
	}
	latest := &storedmileage.StoredMileage{
		ID:                pulid.MustNew("smg_"),
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		RouteHash:         older.RouteHash,
		DistanceUnits:     older.DistanceUnits,
		RoutingType:       older.RoutingType,
		Method:            older.Method,
		DistanceProfileID: profileID,
		HazmatSignature:   older.HazmatSignature,
		Distance:          20,
		LastCalculatedAt:  200,
	}

	result := dedupeUpsertEntities([]*storedmileage.StoredMileage{older, nil, latest})

	require.Len(t, result, 1)
	require.Equal(t, latest.ID, result[0].ID)
	require.Equal(t, float64(20), result[0].Distance)
}
