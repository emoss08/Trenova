package fleetcoderesolver

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFleetCodeMutations_CheckTheFleetCodePermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("fc_").String()
	input := gqlmodel.FleetCodeInput{
		Status:    domaintypes.StatusActive,
		Code:      "EAST",
		ManagerID: pulid.MustNew("usr_").String(),
	}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"createFleetCode", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateFleetCode(ctx, input)
			return err
		}},
		{"updateFleetCode", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateFleetCode(ctx, id, input)
			return err
		}},
		{"patchFleetCode", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).PatchFleetCode(
				ctx,
				id,
				gqlmodel.FleetCodePatchInput{},
			)
			return err
		}},
		{
			"bulkUpdateFleetCodeStatus",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).BulkUpdateFleetCodeStatus(
					ctx,
					gqlmodel.BulkUpdateFleetCodeStatusInput{
						FleetCodeIds: []string{id},
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
				permission.ResourceFleetCode,
				call.operation,
			)
		})
	}
}

func TestFleetCodeFromInput_MapsTheInputIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	managerID := pulid.MustNew("usr_")
	description := "Eastern region"
	revenue, mileage := 125000.0, 9000.0
	color := "#0ea5e9"
	version := 5

	entity, err := fleetCodeFromInput(&gqlmodel.FleetCodeInput{
		Status:      domaintypes.StatusActive,
		Code:        "EAST",
		Description: &description,
		ManagerID:   managerID.String(),
		RevenueGoal: &revenue,
		MileageGoal: &mileage,
		Color:       &color,
		Version:     &version,
	}, authCtx)
	require.NoError(t, err)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, managerID, entity.ManagerID)
	assert.Equal(t, "Eastern region", entity.Description)
	assert.Equal(t, &revenue, entity.RevenueGoal)
	assert.Nil(t, entity.DeadheadGoal)
	assert.Equal(t, &mileage, entity.MileageGoal)
	assert.Equal(t, "#0ea5e9", entity.Color)
	assert.Equal(t, int64(5), entity.Version)
}

func TestApplyFleetCodeInput_RequiresAManager(t *testing.T) {
	t.Parallel()

	err := applyFleetCodeInput(&fleetcode.FleetCode{}, &gqlmodel.FleetCodeInput{Code: "EAST"})

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "managerId", valErr.Field)
}

func TestApplyFleetCodePatch_ChangesOnlyTheFieldsItCarries(t *testing.T) {
	t.Parallel()

	revenue := 50.0
	managerID := pulid.MustNew("usr_")
	existing := &fleetcode.FleetCode{
		Status:      domaintypes.StatusActive,
		Code:        "EAST",
		ManagerID:   managerID,
		RevenueGoal: &revenue,
		Color:       "#fff",
	}
	inactive := domaintypes.StatusInactive

	err := applyFleetCodePatch(existing, &gqlmodel.FleetCodePatchInput{
		Status:      graphql.OmittableOf(&inactive),
		RevenueGoal: graphql.OmittableOf[*float64](nil),
	})
	require.NoError(t, err)

	assert.Equal(t, domaintypes.StatusInactive, existing.Status)
	assert.Nil(t, existing.RevenueGoal)
	assert.Equal(t, managerID, existing.ManagerID)
	assert.Equal(t, "#fff", existing.Color)
}

func TestApplyFleetCodePatch_RefusesToClearTheManager(t *testing.T) {
	t.Parallel()

	err := applyFleetCodePatch(
		&fleetcode.FleetCode{ManagerID: pulid.MustNew("usr_")},
		&gqlmodel.FleetCodePatchInput{ManagerID: graphql.OmittableOf[*string](nil)},
	)

	resolvertest.RequireRequiredPatchError(t, err, "managerId")
}
