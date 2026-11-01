package dataentrycontrolresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataEntryControlResolvers_CheckTheDataEntryControlPermission(t *testing.T) {
	t.Parallel()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"dataEntryControl", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).DataEntryControl(ctx)
			return err
		}},
		{
			"updateDataEntryControl",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateDataEntryControl(
					ctx,
					gqlmodel.UpdateDataEntryControlInput{},
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
				permission.ResourceDataEntryControl,
				call.operation,
			)
		})
	}
}

func TestDataEntryControlFromInput_MapsEveryFieldOntoTheStoredControl(t *testing.T) {
	t.Parallel()

	existing := &dataentrycontrol.DataEntryControl{
		ID:             pulid.MustNew("x_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Version:        3,
		CreatedAt:      100,
		UpdatedAt:      200,
	}

	entity, err := dataEntryControlFromInput(existing, &gqlmodel.UpdateDataEntryControlInput{
		Version:   3,
		CodeCase:  dataentrycontrol.CaseFormat("TitleCase"),
		NameCase:  dataentrycontrol.CaseFormat("TitleCase"),
		EmailCase: dataentrycontrol.CaseFormat("TitleCase"),
		CityCase:  dataentrycontrol.CaseFormat("TitleCase"),
	})
	require.NoError(t, err)

	assert.Equal(t, existing.ID, entity.ID)
	assert.Equal(t, existing.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, existing.OrganizationID, entity.OrganizationID)
	assert.Equal(t, int64(100), entity.CreatedAt)
	assert.Equal(t, int64(3), entity.Version)
	assert.Equal(t, dataentrycontrol.CaseFormat("TitleCase"), entity.CodeCase)
	assert.Equal(t, dataentrycontrol.CaseFormat("TitleCase"), entity.NameCase)
	assert.Equal(t, dataentrycontrol.CaseFormat("TitleCase"), entity.EmailCase)
	assert.Equal(t, dataentrycontrol.CaseFormat("TitleCase"), entity.CityCase)
}
