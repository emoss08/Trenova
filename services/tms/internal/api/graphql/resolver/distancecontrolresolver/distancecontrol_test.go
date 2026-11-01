package distancecontrolresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/places/domain/distancecontrol"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistanceControlResolvers_CheckTheDistanceControlPermission(t *testing.T) {
	t.Parallel()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"distanceControl", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).DistanceControl(ctx)
			return err
		}},
		{"updateDistanceControl", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateDistanceControl(
				ctx,
				gqlmodel.UpdateDistanceControlInput{},
			)
			return err
		}},
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
				permission.ResourceDistanceControl,
				call.operation,
			)
		})
	}
}

func TestDistanceControlFromInput_MapsEveryFieldOntoTheStoredControl(t *testing.T) {
	t.Parallel()

	existing := &distancecontrol.DistanceControl{
		ID:             pulid.MustNew("x_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Version:        3,
		CreatedAt:      100,
		UpdatedAt:      200,
	}
	v4 := pulid.MustNew("x_")
	v5 := pulid.MustNew("x_")
	v6 := pulid.MustNew("x_")
	v7 := pulid.MustNew("x_")
	v8 := pulid.MustNew("x_")
	v9 := pulid.MustNew("x_")
	v10 := pulid.MustNew("x_")
	v11 := pulid.MustNew("x_")

	entity, err := distanceControlFromInput(existing, &gqlmodel.UpdateDistanceControlInput{
		Version:                                      3,
		StoreMileage:                                 true,
		StoredDistanceUnits:                          "storedDistanceUnits",
		PostalCodeFallbackToCity:                     true,
		AutoCreateStoredMileage:                      true,
		LoadedMoveDistanceProfileID:                  v4.String(),
		EmptyMoveDistanceProfileID:                   v5.String(),
		PayDistanceProfileID:                         v6.String(),
		BillingDistanceProfileID:                     v7.String(),
		FuelDistanceProfileID:                        v8.String(),
		EtaOutOfRouteDistanceProfileID:               v9.String(),
		DistanceCalculatorShortestDistanceProfileID:  v10.String(),
		DistanceCalculatorPracticalDistanceProfileID: v11.String(),
		CaptureJurisdictionMiles:                     true,
	})
	require.NoError(t, err)

	assert.Equal(t, existing.ID, entity.ID)
	assert.Equal(t, existing.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, existing.OrganizationID, entity.OrganizationID)
	assert.Equal(t, int64(100), entity.CreatedAt)
	assert.Equal(t, int64(3), entity.Version)
	assert.True(t, entity.StoreMileage)
	assert.Equal(t, "storedDistanceUnits", entity.StoredDistanceUnits)
	assert.True(t, entity.PostalCodeFallbackToCity)
	assert.True(t, entity.AutoCreateStoredMileage)
	assert.Equal(t, v4, entity.LoadedMoveDistanceProfileID)
	assert.Equal(t, v5, entity.EmptyMoveDistanceProfileID)
	assert.Equal(t, v6, entity.PayDistanceProfileID)
	assert.Equal(t, v7, entity.BillingDistanceProfileID)
	assert.Equal(t, v8, entity.FuelDistanceProfileID)
	assert.Equal(t, v9, entity.EtaOutOfRouteDistanceProfileID)
	assert.Equal(t, v10, entity.DistanceCalculatorShortestDistanceProfileID)
	assert.Equal(t, v11, entity.DistanceCalculatorPracticalDistanceProfileID)
	assert.True(t, entity.CaptureJurisdictionMiles)
}

func TestDistanceControlFromInput_NamesAMalformedField(t *testing.T) {
	t.Parallel()

	_, err := distanceControlFromInput(
		&distancecontrol.DistanceControl{},
		&gqlmodel.UpdateDistanceControlInput{
			LoadedMoveDistanceProfileID:                  "not-an-id",
			EmptyMoveDistanceProfileID:                   pulid.MustNew("x_").String(),
			PayDistanceProfileID:                         pulid.MustNew("x_").String(),
			BillingDistanceProfileID:                     pulid.MustNew("x_").String(),
			FuelDistanceProfileID:                        pulid.MustNew("x_").String(),
			EtaOutOfRouteDistanceProfileID:               pulid.MustNew("x_").String(),
			DistanceCalculatorShortestDistanceProfileID:  pulid.MustNew("x_").String(),
			DistanceCalculatorPracticalDistanceProfileID: pulid.MustNew("x_").String(),
		},
	)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.NotEmpty(t, multiErr.Errors)
	assert.Equal(t, "loadedMoveDistanceProfileId", multiErr.Errors[0].Field)
}
