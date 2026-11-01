package commodityresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommodityMutations_CheckTheCommodityPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("com_").String()
	input := gqlmodel.CommodityInput{
		Status:      domaintypes.StatusActive,
		Name:        "Steel coils",
		Description: "Hot rolled",
	}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"createCommodity", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateCommodity(ctx, input)
			return err
		}},
		{"updateCommodity", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateCommodity(ctx, id, input)
			return err
		}},
		{"patchCommodity", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).PatchCommodity(
				ctx,
				id,
				gqlmodel.CommodityPatchInput{},
			)
			return err
		}},
		{
			"bulkUpdateCommodityStatus",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).BulkUpdateCommodityStatus(
					ctx,
					gqlmodel.BulkUpdateCommodityStatusInput{
						CommodityIds: []string{id},
						Status:       domaintypes.StatusInactive,
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

			resolvertest.RequireRefused(
				t,
				err,
				engine,
				permission.ResourceCommodity,
				call.operation,
			)
		})
	}
}

func TestBulkUpdateCommodityStatus_RefusesAnAPIKey(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}}}
	authCtx := resolvertest.UserAuthContext()
	authCtx.PrincipalType = authctx.PrincipalTypeAPIKey
	ctx := resolvertest.AuthedContext(t, authCtx)

	_, err := (&MutationResolver{deps}).BulkUpdateCommodityStatus(
		ctx,
		gqlmodel.BulkUpdateCommodityStatusInput{
			CommodityIds: []string{pulid.MustNew("com_").String()},
			Status:       domaintypes.StatusInactive,
		},
	)

	var authzErr *errortypes.AuthorizationError
	require.True(t, errors.As(err, &authzErr), "%v", err)
}

func TestCommodityFromInput_MapsTheInputIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	hazmatID := pulid.MustNew("hm_")
	hazmatRaw := hazmatID.String()
	minTemp, maxTemp := -10, 4
	weight, length := 42.5, 48.0
	freightClass := commodity.FreightClass70
	instructions := "Keep upright"
	version := 2

	entity, err := commodityFromInput(&gqlmodel.CommodityInput{
		Status:              domaintypes.StatusActive,
		Name:                "Frozen fish",
		Description:         "Palletized",
		HazardousMaterialID: &hazmatRaw,
		MinTemperature:      &minTemp,
		MaxTemperature:      &maxTemp,
		WeightPerUnit:       &weight,
		LengthPerUnit:       &length,
		FreightClass:        &freightClass,
		LoadingInstructions: &instructions,
		Stackable:           true,
		Version:             &version,
	}, authCtx)
	require.NoError(t, err)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, hazmatID, entity.HazardousMaterialID)
	assert.Equal(t, &minTemp, entity.MinTemperature)
	assert.Equal(t, &maxTemp, entity.MaxTemperature)
	assert.Equal(t, &weight, entity.WeightPerUnit)
	assert.Equal(t, &length, entity.LengthPerUnit)
	assert.Nil(t, entity.WidthPerUnit)
	assert.Equal(t, commodity.FreightClass70, entity.FreightClass)
	assert.Equal(t, "Keep upright", entity.LoadingInstructions)
	assert.True(t, entity.Stackable)
	assert.False(t, entity.Fragile)
	assert.Equal(t, int64(2), entity.Version)
}

func TestApplyCommodityInput_NamesAMalformedHazardousMaterial(t *testing.T) {
	t.Parallel()

	raw := "not-an-id"
	err := applyCommodityInput(&commodity.Commodity{}, &gqlmodel.CommodityInput{
		HazardousMaterialID: &raw,
	})

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "hazardousMaterialId", valErr.Field)
}

func TestApplyCommodityPatch_ChangesOnlyTheFieldsItCarries(t *testing.T) {
	t.Parallel()

	hazmatID := pulid.MustNew("hm_")
	weight := 10.0
	existing := &commodity.Commodity{
		Status:              domaintypes.StatusActive,
		Name:                "Paint",
		Description:         "Latex",
		HazardousMaterialID: hazmatID,
		WeightPerUnit:       &weight,
		FreightClass:        commodity.FreightClass85,
		Fragile:             true,
	}
	inactive := domaintypes.StatusInactive

	err := applyCommodityPatch(existing, &gqlmodel.CommodityPatchInput{
		Status:              graphql.OmittableOf(&inactive),
		HazardousMaterialID: graphql.OmittableOf[*string](nil),
		FreightClass:        graphql.OmittableOf[*commodity.FreightClass](nil),
	})
	require.NoError(t, err)

	assert.Equal(t, domaintypes.StatusInactive, existing.Status)
	assert.True(t, existing.HazardousMaterialID.IsNil())
	assert.Empty(t, existing.FreightClass)
	assert.Equal(t, &weight, existing.WeightPerUnit)
	assert.Equal(t, "Paint", existing.Name)
	assert.True(t, existing.Fragile)
}

func TestApplyCommodityPatch_RefusesToClearFragile(t *testing.T) {
	t.Parallel()

	err := applyCommodityPatch(
		&commodity.Commodity{},
		&gqlmodel.CommodityPatchInput{Fragile: graphql.OmittableOf[*bool](nil)},
	)

	resolvertest.RequireRequiredPatchError(t, err, "fragile")
}
