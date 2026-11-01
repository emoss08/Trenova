package servicefailurereasoncoderesolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func newReasonCodeInput() gqlmodel.ServiceFailureReasonCodeInput {
	return gqlmodel.ServiceFailureReasonCodeInput{
		Code:      "WX",
		Label:     "Weather",
		Category:  servicefailure.ReasonCategory("Weather"),
		AppliesTo: servicefailure.ReasonCodeAppliesTo("Both"),
		SortOrder: 10,
	}
}

func TestServiceFailureReasonCodeMutations_CheckTheReasonCodePermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("sfrc_").String()
	input := newReasonCodeInput()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"createServiceFailureReasonCode",
			permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).CreateServiceFailureReasonCode(ctx, input)
				return err
			},
		},
		{
			"updateServiceFailureReasonCode",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateServiceFailureReasonCode(ctx, id, input)
				return err
			},
		},
		{
			"activateServiceFailureReasonCode",
			permission.OpArchive,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ActivateServiceFailureReasonCode(ctx, id)
				return err
			},
		},
		{
			"archiveServiceFailureReasonCode",
			permission.OpArchive,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ArchiveServiceFailureReasonCode(ctx, id)
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
				permission.ResourceServiceFailureReasonCode,
				call.operation,
			)
		})
	}
}

func TestServiceFailureReasonCodeFromInput_IsActiveUnlessTheInputSaysOtherwise(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	input := newReasonCodeInput()

	created := serviceFailureReasonCodeFromInput(&input, authCtx)
	assert.True(t, created.Active)
	assert.Equal(t, authCtx.OrganizationID, created.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, created.BusinessUnitID)
	assert.Equal(t, int32(10), created.SortOrder)

	inactive := false
	input.Active = &inactive
	assert.False(t, serviceFailureReasonCodeFromInput(&input, authCtx).Active)
}

func TestApplyServiceFailureReasonCodeInput_KeepsTheArchiveHistory(t *testing.T) {
	t.Parallel()

	archivedAt := int64(1_700_000_000)
	archivedBy := pulid.MustNew("usr_")
	existing := &servicefailure.ReasonCode{
		ID:           pulid.MustNew("sfrc_"),
		Active:       false,
		ArchivedAt:   &archivedAt,
		ArchivedByID: &archivedBy,
	}
	input := newReasonCodeInput()
	note := "Severe weather"
	input.DefaultNote = &note
	input.ExternalMap = map[string]any{"edi": "WX01"}

	applyServiceFailureReasonCodeInput(existing, &input)

	assert.False(t, existing.Active)
	assert.Equal(t, &archivedAt, existing.ArchivedAt)
	assert.Equal(t, &archivedBy, existing.ArchivedByID)
	assert.Equal(t, "Severe weather", existing.DefaultNote)
	assert.Empty(t, existing.DefaultStatusCode)
	assert.Equal(t, map[string]any{"edi": "WX01"}, existing.ExternalMap)
}
