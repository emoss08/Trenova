package equipmentmanufacturerresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestEquipmentManufacturerMutations_CheckTheEquipmentManufacturerPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("em_").String()
	status := domaintypes.StatusActive
	input := gqlmodel.EquipmentManufacturerInput{Name: "Freightliner", Status: &status}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"createEquipmentManufacturer",
			permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).CreateEquipmentManufacturer(ctx, input)
				return err
			},
		},
		{
			"updateEquipmentManufacturer",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateEquipmentManufacturer(ctx, id, input)
				return err
			},
		},
		{
			"patchEquipmentManufacturer",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).PatchEquipmentManufacturer(
					ctx,
					id,
					gqlmodel.EquipmentManufacturerPatchInput{},
				)
				return err
			},
		},
		{
			"bulkUpdateEquipmentManufacturerStatus",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).BulkUpdateEquipmentManufacturerStatus(
					ctx,
					gqlmodel.BulkUpdateEquipmentManufacturerStatusInput{
						EquipmentManufacturerIds: []string{id},
						Status:                   domaintypes.StatusInactive,
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
				permission.ResourceEquipmentManufacturer,
				call.operation,
			)
		})
	}
}

func TestBulkUpdateEquipmentManufacturerStatus_RefusesAnAPIKey(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}}}
	authCtx := resolvertest.UserAuthContext()
	authCtx.PrincipalType = authctx.PrincipalTypeAPIKey

	_, err := (&MutationResolver{deps}).BulkUpdateEquipmentManufacturerStatus(
		resolvertest.AuthedContext(t, authCtx),
		gqlmodel.BulkUpdateEquipmentManufacturerStatusInput{
			EquipmentManufacturerIds: []string{pulid.MustNew("em_").String()},
			Status:                   domaintypes.StatusInactive,
		},
	)

	var authzErr *errortypes.AuthorizationError
	require.True(t, errors.As(err, &authzErr), "%v", err)
}
