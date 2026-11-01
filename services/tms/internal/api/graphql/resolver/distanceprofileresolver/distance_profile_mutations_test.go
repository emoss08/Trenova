package distanceprofileresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestDistanceProfileMutations_CheckTheDistanceProfilePermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("dp_").String()
	input := gqlmodel.DistanceProfileInput{Name: "Practical", Status: "Active", Provider: "PCMiler"}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"createDistanceProfile", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateDistanceProfile(ctx, input)
			return err
		}},
		{"updateDistanceProfile", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateDistanceProfile(ctx, id, input)
			return err
		}},
		{"deleteDistanceProfile", permission.OpDelete, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).DeleteDistanceProfile(ctx, id)
			return err
		}},
		{
			"setDefaultDistanceProfile",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).SetDefaultDistanceProfile(ctx, id)
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
				permission.ResourceDistanceProfile,
				call.operation,
			)
		})
	}
}

func TestDistanceProfileFromInput_MapsTheProfileIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	routing := "Shortest"
	version := 2

	entity := distanceProfileFromInput(&gqlmodel.DistanceProfileInput{
		Name:                "Shortest",
		Status:              "Active",
		IsDefault:           true,
		Provider:            "PCMiler",
		Region:              "NA",
		RoutingType:         &routing,
		LocationGranularity: "Zip",
		HighwayOnly:         true,
		Version:             &version,
	}, authCtx)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, integration.TypePCMiler, entity.Provider)
	assert.Equal(t, "Shortest", entity.RoutingType)
	assert.Empty(t, entity.DistanceUnits)
	assert.True(t, entity.IsDefault)
	assert.True(t, entity.HighwayOnly)
	assert.Equal(t, int64(2), entity.Version)
}

func TestApplyDistanceProfileInput_KeepsTheRecordsIdentity(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("dp_")
	existing := &distanceprofile.DistanceProfile{ID: id, CreatedAt: 1_700_000_000, Name: "Old"}

	applyDistanceProfileInput(
		existing,
		&gqlmodel.DistanceProfileInput{Name: "New", Status: "Inactive"},
	)

	assert.Equal(t, id, existing.ID)
	assert.Equal(t, int64(1_700_000_000), existing.CreatedAt)
	assert.Equal(t, "New", existing.Name)
	assert.Equal(t, "Inactive", existing.Status)
}
