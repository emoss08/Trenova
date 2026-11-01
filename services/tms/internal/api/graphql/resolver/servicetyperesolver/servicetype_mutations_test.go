package servicetyperesolver

import (
	"context"
	"errors"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceTypeMutations_CheckTheServiceTypePermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("st_").String()
	input := gqlmodel.ServiceTypeInput{Status: domaintypes.StatusActive, Code: "EXP"}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"createServiceType", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateServiceType(ctx, input)
			return err
		}},
		{"updateServiceType", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateServiceType(ctx, id, input)
			return err
		}},
		{"patchServiceType", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).PatchServiceType(
				ctx,
				id,
				gqlmodel.ServiceTypePatchInput{},
			)
			return err
		}},
		{
			"bulkUpdateServiceTypeStatus",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).BulkUpdateServiceTypeStatus(
					ctx,
					gqlmodel.BulkUpdateServiceTypeStatusInput{
						ServiceTypeIds: []string{id},
						Status:         domaintypes.StatusInactive,
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
				permission.ResourceServiceType,
				call.operation,
			)
		})
	}
}

func TestBulkUpdateServiceTypeStatus_RefusesAnAPIKey(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}}}
	authCtx := resolvertest.UserAuthContext()
	authCtx.PrincipalType = authctx.PrincipalTypeAPIKey
	ctx := resolvertest.AuthedContext(t, authCtx)

	_, err := (&MutationResolver{deps}).BulkUpdateServiceTypeStatus(
		ctx,
		gqlmodel.BulkUpdateServiceTypeStatusInput{
			ServiceTypeIds: []string{pulid.MustNew("st_").String()},
			Status:         domaintypes.StatusInactive,
		},
	)

	var authzErr *errortypes.AuthorizationError
	require.True(t, errors.As(err, &authzErr), "%v", err)
}

func TestServiceTypeFromInput_MapsTheInputIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	description := "Expedited"
	version := 3

	entity := serviceTypeFromInput(&gqlmodel.ServiceTypeInput{
		Status:      domaintypes.StatusActive,
		Code:        "EXP",
		Description: &description,
		Version:     &version,
	}, authCtx)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, domaintypes.StatusActive, entity.Status)
	assert.Equal(t, "EXP", entity.Code)
	assert.Equal(t, "Expedited", entity.Description)
	assert.Empty(t, entity.Color)
	assert.Equal(t, int64(3), entity.Version)
}

func TestApplyServiceTypeInput_KeepsTheRecordsIdentity(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("st_")
	existing := &servicetype.ServiceType{
		ID:        id,
		Code:      "OLD",
		Color:     "#111111",
		CreatedAt: 1700000000,
	}

	applyServiceTypeInput(
		existing,
		&gqlmodel.ServiceTypeInput{Status: domaintypes.StatusInactive, Code: "NEW"},
	)

	assert.Equal(t, id, existing.ID)
	assert.Equal(t, int64(1700000000), existing.CreatedAt)
	assert.Equal(t, "NEW", existing.Code)
	assert.Empty(t, existing.Color)
	assert.Equal(t, domaintypes.StatusInactive, existing.Status)
}

func TestApplyServiceTypePatch_ChangesOnlyTheFieldsItCarries(t *testing.T) {
	t.Parallel()

	existing := &servicetype.ServiceType{
		Status:      domaintypes.StatusActive,
		Code:        "EXP",
		Description: "Expedited",
		Color:       "#2563eb",
	}
	description := "Volume"

	err := applyServiceTypePatch(existing, &gqlmodel.ServiceTypePatchInput{
		Description: graphql.OmittableOf(&description),
		Color:       graphql.OmittableOf[*string](nil),
	})
	require.NoError(t, err)

	assert.Equal(t, "Volume", existing.Description)
	assert.Empty(t, existing.Color)
	assert.Equal(t, "EXP", existing.Code)
	assert.Equal(t, domaintypes.StatusActive, existing.Status)
}

func TestApplyServiceTypePatch_RefusesToClearTheCode(t *testing.T) {
	t.Parallel()

	existing := &servicetype.ServiceType{Code: "EXP"}

	err := applyServiceTypePatch(existing, &gqlmodel.ServiceTypePatchInput{
		Code: graphql.OmittableOf[*string](nil),
	})

	resolvertest.RequireRequiredPatchError(t, err, "code")
	assert.Equal(t, "EXP", existing.Code)
}
