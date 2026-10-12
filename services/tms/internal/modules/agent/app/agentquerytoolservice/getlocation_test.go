package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubLocations struct {
	locationports.LocationRepository

	entity *location.Location
	last   locationports.GetLocationByIDRequest
}

func (s *stubLocations) GetByID(
	_ context.Context,
	req locationports.GetLocationByIDRequest,
) (*location.Location, error) {
	s.last = req

	return s.entity, nil
}

func TestGetLocation_ReadsTheLocationInTheActorsTenant(t *testing.T) {
	t.Parallel()

	latitude, longitude := 32.7801, -96.8005
	entity := &location.Location{
		ID:        pulid.MustNew("loc_"),
		Code:      "DAL-YD",
		Name:      "Dallas yard",
		City:      "Dallas",
		Timezone:  "America/Chicago",
		Latitude:  &latitude,
		Longitude: &longitude,
	}
	repo := &stubLocations{entity: entity}
	params := agentParams(
		map[string]any{"locationId": entity.ID.String()},
		permission.SensitivityRestricted,
	)

	result, err := newGetLocationTool(repo, &fakePermissions{allowed: true}).Query(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, entity.ID, repo.last.ID)
	assert.Equal(t, params.OrganizationID, repo.last.TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, repo.last.TenantInfo.BuID)
	document := encodedDocument(t, result)
	assert.Equal(t, "Dallas yard", document["name"])
	assert.Equal(t, "America/Chicago", document["timezone"])
	assert.InDelta(t, latitude, document["latitude"], 0.0001)
}

func TestGetLocation_IsGatedOnTheLocationResource(t *testing.T) {
	t.Parallel()

	tool := newGetLocationTool(nil, nil).(*getTool)
	assert.Equal(t, permission.ResourceLocation, tool.spec.resource)
	assert.Equal(t, "location", tool.spec.entity)
	assert.Contains(t, tool.spec.summary, "list_locations")
}
