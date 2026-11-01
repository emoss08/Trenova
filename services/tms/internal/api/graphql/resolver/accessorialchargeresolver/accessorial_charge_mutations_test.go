package accessorialchargeresolver

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessorialChargeMutations_CheckTheAccessorialChargePermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("acc_").String()
	input := gqlmodel.AccessorialChargeInput{
		Status:      domaintypes.StatusActive,
		Code:        "DET",
		Description: "Detention",
		Method:      accessorialcharge.MethodFlat,
		Amount:      "75.00",
	}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"createAccessorialCharge",
			permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).CreateAccessorialCharge(ctx, input)
				return err
			},
		},
		{
			"updateAccessorialCharge",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateAccessorialCharge(ctx, id, input)
				return err
			},
		},
		{
			"patchAccessorialCharge",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).PatchAccessorialCharge(
					ctx,
					id,
					gqlmodel.AccessorialChargePatchInput{},
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
				permission.ResourceAccessorialCharge,
				call.operation,
			)
		})
	}
}

func TestAccessorialChargeFromInput_MapsTheInputIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	rateUnit := accessorialcharge.RateUnitMile
	version := 3

	entity, err := accessorialChargeFromInput(&gqlmodel.AccessorialChargeInput{
		Status:      domaintypes.StatusInactive,
		Code:        "FSC",
		Description: "Fuel surcharge",
		Method:      accessorialcharge.MethodPerUnit,
		RateUnit:    &rateUnit,
		Amount:      "0.42",
		Version:     &version,
	}, authCtx)
	require.NoError(t, err)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.True(t, entity.ID.IsNil())
	assert.Equal(t, domaintypes.StatusInactive, entity.Status)
	assert.Equal(t, "FSC", entity.Code)
	assert.Equal(t, "Fuel surcharge", entity.Description)
	assert.Equal(t, accessorialcharge.MethodPerUnit, entity.Method)
	assert.Equal(t, accessorialcharge.RateUnitMile, entity.RateUnit)
	assert.True(t, decimal.RequireFromString("0.42").Equal(entity.Amount))
	assert.Equal(t, int64(3), entity.Version)
}

func TestApplyAccessorialChargeInput_RejectsAnAmountThatIsNotADecimal(t *testing.T) {
	t.Parallel()

	err := applyAccessorialChargeInput(
		&accessorialcharge.AccessorialCharge{},
		&gqlmodel.AccessorialChargeInput{Amount: "ten"},
	)

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "amount", valErr.Field)
}

func TestApplyAccessorialChargeInput_KeepsWhatTheInputDoesNotCarry(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("acc_")
	existing := &accessorialcharge.AccessorialCharge{
		ID:        id,
		RateUnit:  accessorialcharge.RateUnitHour,
		Version:   7,
		CreatedAt: 1_700_000_000,
	}

	err := applyAccessorialChargeInput(existing, &gqlmodel.AccessorialChargeInput{
		Status:      domaintypes.StatusActive,
		Code:        "LUMP",
		Description: "Lumper",
		Method:      accessorialcharge.MethodFlat,
		Amount:      "100",
	})
	require.NoError(t, err)

	assert.Equal(t, id, existing.ID)
	assert.Equal(t, int64(7), existing.Version)
	assert.Equal(t, int64(1_700_000_000), existing.CreatedAt)
	assert.Empty(t, existing.RateUnit)
}

func TestApplyAccessorialChargePatch_ChangesOnlyTheFieldsItCarries(t *testing.T) {
	t.Parallel()

	existing := &accessorialcharge.AccessorialCharge{
		Status:      domaintypes.StatusActive,
		Code:        "DET",
		Description: "Detention",
		Method:      accessorialcharge.MethodPerUnit,
		RateUnit:    accessorialcharge.RateUnitHour,
		Amount:      decimal.RequireFromString("50"),
		Version:     2,
	}
	inactive := domaintypes.StatusInactive
	amount := "65.5"

	err := applyAccessorialChargePatch(existing, &gqlmodel.AccessorialChargePatchInput{
		Status:   graphql.OmittableOf(&inactive),
		RateUnit: graphql.OmittableOf[*accessorialcharge.RateUnit](nil),
		Amount:   graphql.OmittableOf(&amount),
	})
	require.NoError(t, err)

	assert.Equal(t, domaintypes.StatusInactive, existing.Status)
	assert.Equal(t, "DET", existing.Code)
	assert.Equal(t, "Detention", existing.Description)
	assert.Equal(t, accessorialcharge.MethodPerUnit, existing.Method)
	assert.Empty(t, existing.RateUnit)
	assert.True(t, decimal.RequireFromString("65.5").Equal(existing.Amount))
	assert.Equal(t, int64(2), existing.Version)
}

func TestApplyAccessorialChargePatch_RefusesToClearARequiredField(t *testing.T) {
	t.Parallel()

	err := applyAccessorialChargePatch(
		&accessorialcharge.AccessorialCharge{Code: "DET"},
		&gqlmodel.AccessorialChargePatchInput{Code: graphql.OmittableOf[*string](nil)},
	)

	resolvertest.RequireRequiredPatchError(t, err, "code")
}
