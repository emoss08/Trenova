package agenttoolservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTractorStatusUpdater struct {
	request *repositories.BulkUpdateTractorStatusRequest
}

func (f *fakeTractorStatusUpdater) BulkUpdateStatus(
	_ context.Context,
	req *repositories.BulkUpdateTractorStatusRequest,
) ([]*tractor.Tractor, error) {
	f.request = req

	return nil, nil
}

type fakeTrailerStatusUpdater struct {
	request *repositories.BulkUpdateTrailerStatusRequest
}

func (f *fakeTrailerStatusUpdater) BulkUpdateStatus(
	_ context.Context,
	req *repositories.BulkUpdateTrailerStatusRequest,
) ([]*trailer.Trailer, error) {
	f.request = req

	return nil, nil
}

func TestUpdateTractorStatus_PassesTheIdsAndStatusThrough(t *testing.T) {
	t.Parallel()

	tractors := &fakeTractorStatusUpdater{}
	tool := newUpdateTractorStatusTool(tractors)

	id := pulid.MustNew("trc_").String()
	require.NoError(t, tool.Execute(t.Context(), executeParams(map[string]any{
		"tractorIds": []any{id},
		"status":     "OutOfService",
	})))

	require.NotNil(t, tractors.request)
	require.Len(t, tractors.request.TractorIDs, 1)
	assert.Equal(t, id, tractors.request.TractorIDs[0].String())
	assert.Equal(t, domaintypes.EquipmentStatusOOS, tractors.request.Status)
}

// The tenant travels with the write and comes from the actor, never from the
// model's arguments.
func TestUpdateTractorStatus_CarriesTheActorTenant(t *testing.T) {
	t.Parallel()

	tractors := &fakeTractorStatusUpdater{}
	params := executeParams(map[string]any{
		"tractorIds": []any{pulid.MustNew("trc_").String()},
		"status":     "Available",
	})

	require.NoError(t, newUpdateTractorStatusTool(tractors).Execute(t.Context(), params))

	assert.Equal(t, params.OrganizationID, tractors.request.TenantInfo.OrgID)
	assert.Equal(t, params.BusinessUnitID, tractors.request.TenantInfo.BuID)
	assert.Equal(t, params.Actor.UserID, tractors.request.TenantInfo.UserID)
}

/*
A status outside the four is refused, and the refusal names the four.

A model that sends "Maintenance" or "out_of_service" has to be told what would
work, because the prompt tells it a refusal listing alternatives is a
correction. Coercing the value would move equipment to a status nobody asked
for; dropping the filter silently is worse still.
*/
func TestUpdateTractorStatus_RefusesAStatusOutsideTheSet(t *testing.T) {
	t.Parallel()

	tractors := &fakeTractorStatusUpdater{}

	err := newUpdateTractorStatusTool(tractors).Execute(t.Context(), executeParams(map[string]any{
		"tractorIds": []any{pulid.MustNew("trc_").String()},
		"status":     "Maintenance",
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "AtMaintenance")
	assert.Contains(t, err.Error(), "OutOfService")
	assert.Nil(t, tractors.request, "nothing may be written on a refused argument")
}

// The cap is the tool's, not the schema's: maxItems is a hint to the model and
// a model that ignores it must not be able to move a whole yard in one call.
func TestUpdateTractorStatus_RefusesMoreIdsThanTheCap(t *testing.T) {
	t.Parallel()

	ids := make([]any, 0, maxEquipmentPerStatusChange+1)
	for range maxEquipmentPerStatusChange + 1 {
		ids = append(ids, pulid.MustNew("trc_").String())
	}

	tractors := &fakeTractorStatusUpdater{}
	err := newUpdateTractorStatusTool(tractors).Execute(t.Context(), executeParams(map[string]any{
		"tractorIds": ids,
		"status":     "Sold",
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "smaller calls")
	assert.Nil(t, tractors.request)
}

func TestUpdateTractorStatus_RefusesAnIdThatIsNotOne(t *testing.T) {
	t.Parallel()

	tractors := &fakeTractorStatusUpdater{}
	err := newUpdateTractorStatusTool(tractors).Execute(t.Context(), executeParams(map[string]any{
		"tractorIds": []any{"unit 4471"},
		"status":     "Available",
	}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid id")
	assert.Nil(t, tractors.request)
}

func TestUpdateTrailerStatus_PassesTheIdsAndStatusThrough(t *testing.T) {
	t.Parallel()

	trailers := &fakeTrailerStatusUpdater{}
	id := pulid.MustNew("trl_").String()

	require.NoError(t, newUpdateTrailerStatusTool(trailers).Execute(
		t.Context(),
		executeParams(map[string]any{
			"trailerIds": []any{id},
			"status":     "AtMaintenance",
		}),
	))

	require.Len(t, trailers.request.TrailerIDs, 1)
	assert.Equal(t, id, trailers.request.TrailerIDs[0].String())
	assert.Equal(t, domaintypes.EquipmentStatusAtMaintenance, trailers.request.Status)
}

// A status change is an update, not a deletion, and it can be set back. Saying
// so is what lets an organization raise the tool's tier.
func TestEquipmentStatusTools_DeclareTheirAuthorization(t *testing.T) {
	t.Parallel()

	tractorTool := newUpdateTractorStatusTool(&fakeTractorStatusUpdater{})
	assert.Equal(t, permission.ResourceTractor, tractorTool.PermissionResource())
	assert.Equal(t, permission.OpUpdate, tractorTool.PermissionOperation())
	assert.True(t, tractorTool.Reversible())

	trailerTool := newUpdateTrailerStatusTool(&fakeTrailerStatusUpdater{})
	assert.Equal(t, permission.ResourceTrailer, trailerTool.PermissionResource())
	assert.Equal(t, permission.OpUpdate, trailerTool.PermissionOperation())
	assert.True(t, trailerTool.Reversible())
}

// Every status the schema offers has to be one the parser accepts, or the model
// is being handed a value that fails on arrival.
func TestEquipmentStatusSchema_OffersOnlyStatusesTheToolAccepts(t *testing.T) {
	t.Parallel()

	schema := newUpdateTractorStatusTool(&fakeTractorStatusUpdater{}).ParamSchema()
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	status, ok := properties["status"].(map[string]any)
	require.True(t, ok)
	values, ok := status["enum"].([]string)
	require.True(t, ok)
	require.NotEmpty(t, values)

	for _, value := range values {
		_, err := domaintypes.EquipmentStatusFromString(value)
		assert.NoError(t, err, "schema offers %q", value)
	}
}

// The descriptions are the only place a model learns that a unit number is not
// an id, which is the mistake it makes when a person names a truck by its
// number.
func TestEquipmentStatusTools_PointAtTheListToolForIds(t *testing.T) {
	t.Parallel()

	assert.Contains(t,
		newUpdateTractorStatusTool(&fakeTractorStatusUpdater{}).Description(),
		"list_tractors")
	assert.True(t, strings.Contains(
		newUpdateTrailerStatusTool(&fakeTrailerStatusUpdater{}).Description(),
		"list_trailers"))
}
