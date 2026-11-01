package documenttyperesolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func documentTypeInput() gqlmodel.DocumentTypeInput {
	return gqlmodel.DocumentTypeInput{
		Code:                   "POD",
		Name:                   "Proof of delivery",
		DocumentClassification: documenttype.ClassificationRegulatory,
		DocumentCategory:       documenttype.CategoryShipment,
	}
}

func TestDocumentTypeMutations_CheckTheDocumentTypePermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("dt_").String()
	input := documentTypeInput()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"createDocumentType", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateDocumentType(ctx, input)
			return err
		}},
		{"updateDocumentType", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateDocumentType(ctx, id, input)
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
				permission.ResourceDocumentType,
				call.operation,
			)
		})
	}
}

func TestDocumentTypeFromInput_MapsTheInputIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	color := "#f97316"
	version := 1
	input := documentTypeInput()
	input.Color = &color
	input.Version = &version

	entity := documentTypeFromInput(&input, authCtx)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, "POD", entity.Code)
	assert.Equal(t, "Proof of delivery", entity.Name)
	assert.Empty(t, entity.Description)
	assert.Equal(t, color, entity.Color)
	assert.Equal(t, documenttype.ClassificationRegulatory, entity.DocumentClassification)
	assert.Equal(t, documenttype.CategoryShipment, entity.DocumentCategory)
	assert.Equal(t, int64(1), entity.Version)
	assert.False(t, entity.IsSystem)
}

func TestApplyDocumentTypeInput_KeepsWhatOnlyTheServerSets(t *testing.T) {
	t.Parallel()

	existing := &documenttype.DocumentType{IsSystem: true, RequiresSignature: true, Version: 4}
	input := documentTypeInput()

	applyDocumentTypeInput(existing, &input)

	assert.True(t, existing.IsSystem)
	assert.True(t, existing.RequiresSignature)
	assert.Equal(t, int64(4), existing.Version)
}
