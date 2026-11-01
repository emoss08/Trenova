package documentcontrolresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentControlResolvers_CheckTheDocumentControlPermission(t *testing.T) {
	t.Parallel()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"documentControl", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).DocumentControl(ctx)
			return err
		}},
		{"updateDocumentControl", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateDocumentControl(
				ctx,
				gqlmodel.UpdateDocumentControlInput{},
			)
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
				permission.ResourceDocumentControl,
				call.operation,
			)
		})
	}
}

func TestDocumentControlFromInput_MapsEveryFieldOntoTheStoredControl(t *testing.T) {
	t.Parallel()

	existing := &documentcontrol.DocumentControl{
		ID:             pulid.MustNew("x_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Version:        3,
		CreatedAt:      100,
		UpdatedAt:      200,
	}

	entity, err := documentControlFromInput(existing, &gqlmodel.UpdateDocumentControlInput{
		Version:                         3,
		EnableDocumentIntelligence:      true,
		EnableOcr:                       true,
		EnableAutoClassification:        true,
		EnableAutoDocumentTypeAssociate: true,
		EnableAutoCreateDocumentTypes:   true,
		EnableShipmentDraftExtraction:   true,
		EnableAiAssistedClassification:  true,
		EnableAiAssistedExtraction:      true,
		ShipmentDraftAllowedResources:   []string{"shipmentDraftAllowedResources"},
		EnableFullTextIndexing:          true,
		EnableCapture:                   true,
		CaptureAutoFileCoverSheets:      true,
		CaptureRetentionDays:            15,
		CaptureMinAgentVersion:          "captureMinAgentVersion",
		CaptureAllowAutoUpdate:          true,
	})
	require.NoError(t, err)

	assert.Equal(t, existing.ID, entity.ID)
	assert.Equal(t, existing.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, existing.OrganizationID, entity.OrganizationID)
	assert.Equal(t, int64(100), entity.CreatedAt)
	assert.Equal(t, int64(3), entity.Version)
	assert.True(t, entity.EnableDocumentIntelligence)
	assert.True(t, entity.EnableOCR)
	assert.True(t, entity.EnableAutoClassification)
	assert.True(t, entity.EnableAutoDocumentTypeAssociate)
	assert.True(t, entity.EnableAutoCreateDocumentTypes)
	assert.True(t, entity.EnableShipmentDraftExtraction)
	assert.True(t, entity.EnableAIAssistedClassification)
	assert.True(t, entity.EnableAIAssistedExtraction)
	assert.Equal(t, []string{"shipmentDraftAllowedResources"}, entity.ShipmentDraftAllowedResources)
	assert.True(t, entity.EnableFullTextIndexing)
	assert.True(t, entity.EnableCapture)
	assert.True(t, entity.CaptureAutoFileCoverSheets)
	assert.Equal(t, 15, entity.CaptureRetentionDays)
	assert.Equal(t, "captureMinAgentVersion", entity.CaptureMinAgentVersion)
	assert.True(t, entity.CaptureAllowAutoUpdate)
}
