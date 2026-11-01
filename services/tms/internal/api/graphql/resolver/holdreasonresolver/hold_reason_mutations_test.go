package holdreasonresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func holdReasonInput() gqlmodel.HoldReasonInput {
	return gqlmodel.HoldReasonInput{
		Type:            holdreason.HoldTypeCompliance,
		Code:            "MISSING_POD",
		Label:           "Missing proof of delivery",
		Active:          true,
		DefaultSeverity: holdreason.HoldSeverityBlocking,
		SortOrder:       20,
	}
}

func TestHoldReasonMutations_CheckTheHoldReasonPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("hr_").String()
	input := holdReasonInput()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"createHoldReason", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateHoldReason(ctx, input)
			return err
		}},
		{"updateHoldReason", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateHoldReason(ctx, id, input)
			return err
		}},
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
				permission.ResourceHoldReason,
				call.operation,
			)
		})
	}
}

func TestHoldReasonFromInput_MapsTheInputIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	description := "Billing waits for a signed POD"
	version := 6
	input := holdReasonInput()
	input.Description = &description
	input.DefaultBlocksBilling = true
	input.DefaultVisibleToCustomer = true
	input.ExternalMap = map[string]any{"edi": "POD"}
	input.Version = &version

	entity := holdReasonFromInput(&input, authCtx)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, holdreason.HoldTypeCompliance, entity.Type)
	assert.Equal(t, "MISSING_POD", entity.Code)
	assert.Equal(t, description, entity.Description)
	assert.True(t, entity.Active)
	assert.Equal(t, holdreason.HoldSeverityBlocking, entity.DefaultSeverity)
	assert.True(t, entity.DefaultBlocksBilling)
	assert.False(t, entity.DefaultBlocksDispatch)
	assert.True(t, entity.DefaultVisibleToCustomer)
	assert.Equal(t, int32(20), entity.SortOrder)
	assert.Equal(t, map[string]any{"edi": "POD"}, entity.ExternalMap)
	assert.Equal(t, int64(6), entity.Version)
}

func TestApplyHoldReasonInput_KeepsTheRecordIdentity(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("hr_")
	existing := &holdreason.HoldReason{ID: id, Version: 2, CreatedAt: 1_700_000_000}
	input := holdReasonInput()

	applyHoldReasonInput(existing, &input)

	assert.Equal(t, id, existing.ID)
	assert.Equal(t, int64(2), existing.Version)
	assert.Equal(t, int64(1_700_000_000), existing.CreatedAt)
	assert.Equal(t, "MISSING_POD", existing.Code)
}
