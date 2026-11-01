package hazardousmaterialresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHazardousMaterialMutations_CheckTheHazardousMaterialPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("hm_").String()
	input := gqlmodel.HazardousMaterialInput{
		Status: domaintypes.StatusActive,
		Name:   "Gasoline",
		Class:  hazardousmaterial.HazardousClass3,
	}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"createHazardousMaterial",
			permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).CreateHazardousMaterial(ctx, input)
				return err
			},
		},
		{
			"updateHazardousMaterial",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateHazardousMaterial(ctx, id, input)
				return err
			},
		},
		{
			"patchHazardousMaterial",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).PatchHazardousMaterial(
					ctx,
					id,
					gqlmodel.HazardousMaterialPatchInput{},
				)
				return err
			},
		},
		{
			"bulkUpdateHazardousMaterialStatus",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).BulkUpdateHazardousMaterialStatus(
					ctx,
					gqlmodel.BulkUpdateHazardousMaterialStatusInput{
						HazardousMaterialIds: []string{id},
						Status:               domaintypes.StatusInactive,
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
				permission.ResourceHazardousMaterial,
				call.operation,
			)
		})
	}
}

func TestBulkUpdateHazardousMaterialStatus_RefusesAnAPIKey(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}}}
	authCtx := resolvertest.UserAuthContext()
	authCtx.PrincipalType = authctx.PrincipalTypeAPIKey

	_, err := (&MutationResolver{deps}).BulkUpdateHazardousMaterialStatus(
		resolvertest.AuthedContext(t, authCtx),
		gqlmodel.BulkUpdateHazardousMaterialStatusInput{
			HazardousMaterialIds: []string{pulid.MustNew("hm_").String()},
			Status:               domaintypes.StatusInactive,
		},
	)

	var authzErr *errortypes.AuthorizationError
	require.True(t, errors.As(err, &authzErr), "%v", err)
}

func TestHazardousMaterialFromInput_MapsTheInputIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	unNumber := "UN1203"
	packingGroup := hazardousmaterial.PackingGroupII
	phone := "800-424-9300"

	entity := hazardousMaterialFromInput(&gqlmodel.HazardousMaterialInput{
		Status:                      domaintypes.StatusActive,
		Name:                        "Gasoline",
		Class:                       hazardousmaterial.HazardousClass3,
		UnNumber:                    &unNumber,
		PackingGroup:                &packingGroup,
		EmergencyContactPhoneNumber: &phone,
		PlacardRequired:             true,
	}, authCtx)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, "Gasoline", entity.Name)
	assert.Equal(t, hazardousmaterial.HazardousClass3, entity.Class)
	assert.Equal(t, "UN1203", entity.UNNumber)
	assert.Equal(t, hazardousmaterial.PackingGroupII, entity.PackingGroup)
	assert.Equal(t, phone, entity.EmergencyContactPhoneNumber)
	assert.True(t, entity.PlacardRequired)
	assert.False(t, entity.MarinePollutant)
	assert.Empty(t, entity.Code)
}

func TestApplyHazardousMaterialPatch_ChangesOnlyTheFieldsItCarries(t *testing.T) {
	t.Parallel()

	existing := &hazardousmaterial.HazardousMaterial{
		Status:          domaintypes.StatusActive,
		Name:            "Gasoline",
		UNNumber:        "UN1203",
		LabelCodes:      "3",
		PlacardRequired: true,
	}
	inactive := domaintypes.StatusInactive

	err := applyHazardousMaterialPatch(existing, &gqlmodel.HazardousMaterialPatchInput{
		Status:     graphql.OmittableOf(&inactive),
		LabelCodes: graphql.OmittableOf[*string](nil),
	})
	require.NoError(t, err)

	assert.Equal(t, domaintypes.StatusInactive, existing.Status)
	assert.Empty(t, existing.LabelCodes)
	assert.Equal(t, "UN1203", existing.UNNumber)
	assert.True(t, existing.PlacardRequired)
}

func TestApplyHazardousMaterialPatch_RefusesToClearTheClass(t *testing.T) {
	t.Parallel()

	err := applyHazardousMaterialPatch(
		&hazardousmaterial.HazardousMaterial{Class: hazardousmaterial.HazardousClass3},
		&gqlmodel.HazardousMaterialPatchInput{
			Class: graphql.OmittableOf[*hazardousmaterial.HazardousClass](nil),
		},
	)

	resolvertest.RequireRequiredPatchError(t, err, "class")
}
