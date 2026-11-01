package distanceoverrideresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDistanceOverrideInput() gqlmodel.DistanceOverrideInput {
	return gqlmodel.DistanceOverrideInput{
		OriginLocationID:      pulid.MustNew("loc_").String(),
		DestinationLocationID: pulid.MustNew("loc_").String(),
		Distance:              412.5,
	}
}

func TestDistanceOverrideMutations_CheckTheDistanceOverridePermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("do_").String()
	input := newDistanceOverrideInput()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"createDistanceOverride",
			permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).CreateDistanceOverride(ctx, input)
				return err
			},
		},
		{
			"updateDistanceOverride",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateDistanceOverride(ctx, id, input)
				return err
			},
		},
		{
			"deleteDistanceOverride",
			permission.OpDelete,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).DeleteDistanceOverride(ctx, id)
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
				permission.ResourceDistanceOverride,
				call.operation,
			)
		})
	}
}

func TestDistanceOverrideFromInput_KeepsTheStopsInRouteOrder(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	id := pulid.MustNew("do_")
	first, second := pulid.MustNew("loc_"), pulid.MustNew("loc_")
	customer := pulid.MustNew("cus_")
	customerRaw := customer.String()
	input := newDistanceOverrideInput()
	input.CustomerID = &customerRaw
	input.IntermediateStops = []*gqlmodel.DistanceOverrideStopInput{
		{LocationID: first.String()},
		nil,
		{LocationID: second.String()},
	}

	entity, err := distanceOverrideFromInput(&input, id, authCtx)
	require.NoError(t, err)

	assert.Equal(t, id, entity.ID)
	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, customer, entity.CustomerID)
	assert.InDelta(t, 412.5, entity.Distance, 0)
	require.Len(t, entity.IntermediateStops, 2)
	assert.Equal(t, first, entity.IntermediateStops[0].LocationID)
	assert.Equal(t, second, entity.IntermediateStops[1].LocationID)
}

func TestDistanceOverrideFromInput_NamesAMissingStopLocation(t *testing.T) {
	t.Parallel()

	input := newDistanceOverrideInput()
	input.IntermediateStops = []*gqlmodel.DistanceOverrideStopInput{{LocationID: ""}}

	_, err := distanceOverrideFromInput(&input, pulid.Nil, resolvertest.UserAuthContext())

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "intermediateStops[0].locationId", valErr.Field)
}
