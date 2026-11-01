package locationcategoryresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func locationCategoryInput() gqlmodel.LocationCategoryInput {
	return gqlmodel.LocationCategoryInput{
		Name:                "Cold storage DC",
		Type:                locationcategory.CategoryDistributionCenter,
		RequiresAppointment: true,
	}
}

func TestLocationCategoryMutations_CheckTheLocationCategoryPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("lc_").String()
	input := locationCategoryInput()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"createLocationCategory",
			permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).CreateLocationCategory(ctx, input)
				return err
			},
		},
		{
			"updateLocationCategory",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateLocationCategory(ctx, id, input)
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

			resolvertest.RequireRefused(
				t,
				err,
				engine,
				permission.ResourceLocationCategory,
				call.operation,
			)
		})
	}
}

func TestLocationCategoryFromInput_MapsTheInputIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	facility := locationcategory.FacilityTypeCrossDock
	color := "#14b8a6"
	input := locationCategoryInput()
	input.FacilityType = &facility
	input.Color = &color
	input.HasRestroom = true

	entity := locationCategoryFromInput(&input, authCtx)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, "Cold storage DC", entity.Name)
	assert.Equal(t, locationcategory.CategoryDistributionCenter, entity.Type)
	assert.Equal(t, locationcategory.FacilityTypeCrossDock, entity.FacilityType)
	assert.Equal(t, color, entity.Color)
	assert.True(t, entity.RequiresAppointment)
	assert.True(t, entity.HasRestroom)
	assert.False(t, entity.AllowsOvernight)
}

func TestApplyLocationCategoryInput_ClearsAnOmittedFacilityType(t *testing.T) {
	t.Parallel()

	existing := &locationcategory.LocationCategory{
		FacilityType: locationcategory.FacilityTypeCrossDock,
		Version:      8,
	}
	input := locationCategoryInput()

	applyLocationCategoryInput(existing, &input)

	assert.Empty(t, existing.FacilityType)
	assert.Equal(t, int64(8), existing.Version)
}
