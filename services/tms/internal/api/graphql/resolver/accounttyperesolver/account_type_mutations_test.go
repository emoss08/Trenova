package accounttyperesolver

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountTypeMutations_CheckTheAccountTypePermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("at_").String()
	input := gqlmodel.AccountTypeInput{
		Status:   domaintypes.StatusActive,
		Code:     "AST",
		Name:     "Assets",
		Category: accounttype.CategoryAsset,
	}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"createAccountType", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateAccountType(ctx, input)
			return err
		}},
		{"updateAccountType", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateAccountType(ctx, id, input)
			return err
		}},
		{"patchAccountType", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).PatchAccountType(
				ctx,
				id,
				gqlmodel.AccountTypePatchInput{},
			)
			return err
		}},
		{
			"bulkUpdateAccountTypeStatus",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).BulkUpdateAccountTypeStatus(
					ctx,
					gqlmodel.BulkUpdateAccountTypeStatusInput{
						AccountTypeIds: []string{id},
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
				permission.ResourceAccountType,
				call.operation,
			)
		})
	}
}

func TestAccountTypeFromInput_MapsTheInputIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	description := "Things the company owns"
	color := "#22c55e"
	version := 4

	entity := accountTypeFromInput(&gqlmodel.AccountTypeInput{
		Status:      domaintypes.StatusActive,
		Code:        "AST",
		Name:        "Assets",
		Description: &description,
		Category:    accounttype.CategoryAsset,
		Color:       &color,
		Version:     &version,
	}, authCtx)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, "AST", entity.Code)
	assert.Equal(t, "Assets", entity.Name)
	assert.Equal(t, description, entity.Description)
	assert.Equal(t, accounttype.CategoryAsset, entity.Category)
	assert.Equal(t, color, entity.Color)
	assert.Equal(t, int64(4), entity.Version)
}

func TestApplyAccountTypeInput_KeepsTheSystemFlag(t *testing.T) {
	t.Parallel()

	existing := &accounttype.AccountType{IsSystem: true, Version: 2}

	applyAccountTypeInput(existing, &gqlmodel.AccountTypeInput{
		Status:   domaintypes.StatusActive,
		Code:     "REV",
		Name:     "Revenue",
		Category: accounttype.CategoryRevenue,
	})

	assert.True(t, existing.IsSystem)
	assert.Equal(t, int64(2), existing.Version)
	assert.Empty(t, existing.Description)
}

func TestApplyAccountTypePatch_ChangesOnlyTheFieldsItCarries(t *testing.T) {
	t.Parallel()

	existing := &accounttype.AccountType{
		Status:      domaintypes.StatusActive,
		Code:        "AST",
		Name:        "Assets",
		Description: "Owned",
		Color:       "#fff",
		Category:    accounttype.CategoryAsset,
	}
	inactive := domaintypes.StatusInactive

	err := applyAccountTypePatch(existing, &gqlmodel.AccountTypePatchInput{
		Status: graphql.OmittableOf(&inactive),
		Color:  graphql.OmittableOf[*string](nil),
	})
	require.NoError(t, err)

	assert.Equal(t, domaintypes.StatusInactive, existing.Status)
	assert.Equal(t, "Owned", existing.Description)
	assert.Empty(t, existing.Color)
	assert.Equal(t, "Assets", existing.Name)
}

func TestApplyAccountTypePatch_RefusesToClearTheName(t *testing.T) {
	t.Parallel()

	err := applyAccountTypePatch(
		&accounttype.AccountType{Name: "Assets"},
		&gqlmodel.AccountTypePatchInput{Name: graphql.OmittableOf[*string](nil)},
	)

	resolvertest.RequireRequiredPatchError(t, err, "name")
}
