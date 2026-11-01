package locationresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/places/domain/geofence"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLocationInput() gqlmodel.LocationInput {
	return gqlmodel.LocationInput{
		Status:             domaintypes.StatusActive,
		Name:               "Columbus DC",
		LocationCategoryID: pulid.MustNew("lc_").String(),
		StateID:            pulid.MustNew("us_").String(),
		AddressLine1:       "1 Main St",
		City:               "Columbus",
		PostalCode:         "43004",
		GeofenceType:       geofence.TypeAuto,
	}
}

func TestLocationMutations_CheckTheLocationPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("loc_").String()
	input := newLocationInput()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"createLocation", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateLocation(ctx, input)
			return err
		}},
		{"updateLocation", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateLocation(ctx, id, input)
			return err
		}},
		{"patchLocation", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).PatchLocation(
				ctx,
				id,
				gqlmodel.LocationPatchInput{},
			)
			return err
		}},
		{
			"bulkUpdateLocationStatus",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).BulkUpdateLocationStatus(
					ctx,
					gqlmodel.BulkUpdateLocationStatusInput{
						LocationIds: []string{id},
						Status:      domaintypes.StatusInactive,
					},
				)
				return err
			},
		},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(t, err, engine, permission.ResourceLocation, call.operation)
		})
	}
}

func TestBulkUpdateLocationStatus_RefusesAnAPIKey(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}}}
	authCtx := resolvertest.UserAuthContext()
	authCtx.PrincipalType = authctx.PrincipalTypeAPIKey
	ctx := resolvertest.AuthedContext(t, authCtx)

	_, err := (&MutationResolver{deps}).BulkUpdateLocationStatus(
		ctx,
		gqlmodel.BulkUpdateLocationStatusInput{
			LocationIds: []string{pulid.MustNew("loc_").String()},
			Status:      domaintypes.StatusInactive,
		},
	)

	var authzErr *errortypes.AuthorizationError
	require.True(t, errors.As(err, &authzErr), "%v", err)
}

func TestLocationFromInput_MapsTheInputIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	input := newLocationInput()
	timezone := "America/New_York"
	radius := 300.0
	version := 2
	input.Timezone = &timezone
	input.GeofenceType = geofence.TypeDraw
	input.GeofenceRadiusMeters = &radius
	input.GeofenceVertices = []*gqlmodel.GeofenceVertexInput{
		{Latitude: 39.9, Longitude: -82.9},
		nil,
		{Latitude: 39.91, Longitude: -82.91},
	}
	input.Version = &version

	entity, err := locationFromInput(&input, authCtx)
	require.NoError(t, err)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, pulid.ID(input.LocationCategoryID), entity.LocationCategoryID)
	assert.Equal(t, pulid.ID(input.StateID), entity.StateID)
	assert.Empty(t, entity.Code)
	assert.Equal(t, "America/New_York", entity.Timezone)
	assert.Equal(t, geofence.TypeDraw, entity.GeofenceType)
	assert.Equal(t, &radius, entity.GeofenceRadiusMeters)
	assert.Equal(t, []geofence.Vertex{
		{Latitude: 39.9, Longitude: -82.9},
		{Latitude: 39.91, Longitude: -82.91},
	}, entity.GeofenceVertices)
	assert.Equal(t, int64(2), entity.Version)
}

func TestApplyLocationInput_KeepsTheSavedCodeWhenTheInputLeavesItBlank(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("loc_")
	existing := &location.Location{ID: id, Code: "CMH-001"}
	input := newLocationInput()
	blank := ""
	input.Code = &blank

	require.NoError(t, applyLocationInput(existing, &input))

	assert.Equal(t, id, existing.ID)
	assert.Equal(t, "CMH-001", existing.Code)
}

func TestApplyLocationInput_NamesAMissingCategory(t *testing.T) {
	t.Parallel()

	input := newLocationInput()
	input.LocationCategoryID = ""

	err := applyLocationInput(&location.Location{}, &input)

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "locationCategoryId", valErr.Field)
}

func TestApplyLocationPatch_ChangesOnlyTheFieldsItCarries(t *testing.T) {
	t.Parallel()

	stateID := pulid.MustNew("us_")
	existing := &location.Location{
		Status:       domaintypes.StatusActive,
		Name:         "Columbus DC",
		Description:  "Main yard",
		StateID:      stateID,
		AddressLine2: "Dock 4",
		Timezone:     "America/New_York",
	}
	inactive := domaintypes.StatusInactive

	err := applyLocationPatch(existing, &gqlmodel.LocationPatchInput{
		Status:       graphql.OmittableOf(&inactive),
		AddressLine2: graphql.OmittableOf[*string](nil),
	})
	require.NoError(t, err)

	assert.Equal(t, domaintypes.StatusInactive, existing.Status)
	assert.Empty(t, existing.AddressLine2)
	assert.Equal(t, "Columbus DC", existing.Name)
	assert.Equal(t, "Main yard", existing.Description)
	assert.Equal(t, stateID, existing.StateID)
	assert.Equal(t, "America/New_York", existing.Timezone)
}

func TestApplyLocationPatch_RefusesToClearTheState(t *testing.T) {
	t.Parallel()

	stateID := pulid.MustNew("us_")
	existing := &location.Location{StateID: stateID}

	err := applyLocationPatch(existing, &gqlmodel.LocationPatchInput{
		StateID: graphql.OmittableOf[*string](nil),
	})

	resolvertest.RequireRequiredPatchError(t, err, "stateId")
	assert.Equal(t, stateID, existing.StateID)
}
