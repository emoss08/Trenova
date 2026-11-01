package documentpacketruleresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentpacketrule"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func packetRuleInput(documentTypeID string) gqlmodel.DocumentPacketRuleInput {
	return gqlmodel.DocumentPacketRuleInput{
		ResourceType:          "Trailer",
		DocumentTypeID:        documentTypeID,
		Required:              true,
		AllowMultiple:         false,
		DisplayOrder:          2,
		ExpirationRequired:    true,
		ExpirationWarningDays: 14,
	}
}

func TestDocumentPacketRuleMutations_CheckTheDocumentTypePermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("dpr_").String()
	input := packetRuleInput(pulid.MustNew("dt_").String())

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"createDocumentPacketRule",
			permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).CreateDocumentPacketRule(ctx, input)
				return err
			},
		},
		{
			"updateDocumentPacketRule",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateDocumentPacketRule(ctx, id, input)
				return err
			},
		},
		{
			"deleteDocumentPacketRule",
			permission.OpDelete,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).DeleteDocumentPacketRule(ctx, id)
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
				permission.ResourceDocumentType,
				call.operation,
			)
		})
	}
}

func TestDocumentPacketRuleFromInput_MapsTheRuleIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	documentTypeID := pulid.MustNew("dt_")
	input := packetRuleInput(documentTypeID.String())

	entity, err := documentPacketRuleFromInput(&input, authCtx)

	require.NoError(t, err)
	assert.Equal(t, &documentpacketrule.DocumentPacketRule{
		OrganizationID:        authCtx.OrganizationID,
		BusinessUnitID:        authCtx.BusinessUnitID,
		ResourceType:          "Trailer",
		DocumentTypeID:        documentTypeID,
		Required:              true,
		DisplayOrder:          2,
		ExpirationRequired:    true,
		ExpirationWarningDays: 14,
	}, entity)
}

func TestApplyDocumentPacketRuleInput_KeepsTheRecordsIdentityAndTakesTheVersion(t *testing.T) {
	t.Parallel()

	ruleID := pulid.MustNew("dpr_")
	existing := &documentpacketrule.DocumentPacketRule{ID: ruleID, CreatedAt: 100, Version: 3}
	input := packetRuleInput(pulid.MustNew("dt_").String())
	version := 3
	input.Version = &version

	require.NoError(t, applyDocumentPacketRuleInput(existing, &input))

	assert.Equal(t, ruleID, existing.ID)
	assert.Equal(t, int64(100), existing.CreatedAt)
	assert.Equal(t, int64(3), existing.Version)
	assert.Equal(t, 14, existing.ExpirationWarningDays)
}

func TestApplyDocumentPacketRuleInput_NeedsADocumentType(t *testing.T) {
	t.Parallel()

	input := packetRuleInput("")

	err := applyDocumentPacketRuleInput(&documentpacketrule.DocumentPacketRule{}, &input)

	var validationErr *errortypes.Error
	require.True(t, errors.As(err, &validationErr), "%v", err)
	assert.Equal(t, "documentTypeId", validationErr.Field)
}
