package sequenceconfigresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSequenceConfigOperations_CheckTheSequenceConfigPermission(t *testing.T) {
	t.Parallel()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"sequenceConfigs", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).SequenceConfigs(ctx)
			return err
		}},
		{"updateSequenceConfigs", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).UpdateSequenceConfigs(
				ctx,
				gqlmodel.SequenceConfigDocumentInput{},
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
				permission.ResourceSequenceConfig,
				call.operation,
			)
		})
	}
}

func TestSequenceConfigDocumentFromInput_StampsEveryConfigWithTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	id := pulid.MustNew("sc_")
	raw := id.String()
	version := 5

	doc, err := sequenceConfigDocumentFromInput(&gqlmodel.SequenceConfigDocumentInput{
		Configs: []*gqlmodel.SequenceConfigInput{
			{
				ID:             &raw,
				SequenceType:   sequenceconfig.SequenceTypeLocationCode,
				Prefix:         "LOC",
				YearDigits:     2,
				SequenceDigits: 3,
				UseSeparators:  true,
				SeparatorChar:  "-",
				LocationCodeStrategy: &gqlmodel.LocationCodeStrategyInput{
					Components: []sequenceconfig.LocationCodeComponent{
						sequenceconfig.LocationCodeComponentName,
						sequenceconfig.LocationCodeComponentCity,
					},
					ComponentWidth: 3,
					SequenceDigits: 4,
					Separator:      "-",
					Casing:         sequenceconfig.LocationCodeCasingUpper,
					FallbackPrefix: "LOC",
				},
				Version: &version,
			},
			nil,
			{SequenceType: sequenceconfig.SequenceTypeInvoice, Prefix: "INV", SequenceDigits: 6},
		},
	}, authCtx)
	require.NoError(t, err)

	assert.Equal(t, authCtx.OrganizationID, doc.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, doc.BusinessUnitID)
	require.Len(t, doc.Configs, 2)
	for _, cfg := range doc.Configs {
		assert.Equal(t, authCtx.OrganizationID, cfg.OrganizationID)
		assert.Equal(t, authCtx.BusinessUnitID, cfg.BusinessUnitID)
	}
	assert.Equal(t, id, doc.Configs[0].ID)
	assert.Equal(t, int16(2), doc.Configs[0].YearDigits)
	assert.Equal(t, int64(5), doc.Configs[0].Version)
	assert.Equal(t, &sequenceconfig.LocationCodeStrategy{
		Components: []sequenceconfig.LocationCodeComponent{
			sequenceconfig.LocationCodeComponentName,
			sequenceconfig.LocationCodeComponentCity,
		},
		ComponentWidth: 3,
		SequenceDigits: 4,
		Separator:      "-",
		Casing:         sequenceconfig.LocationCodeCasingUpper,
		FallbackPrefix: "LOC",
	}, doc.Configs[0].LocationCodeStrategy)
	assert.True(t, doc.Configs[1].ID.IsNil())
	assert.Nil(t, doc.Configs[1].LocationCodeStrategy)
	assert.Equal(t, int16(6), doc.Configs[1].SequenceDigits)
}

func TestSequenceConfigDocumentFromInput_NamesAMalformedConfigID(t *testing.T) {
	t.Parallel()

	bad := "nope"
	_, err := sequenceConfigDocumentFromInput(&gqlmodel.SequenceConfigDocumentInput{
		Configs: []*gqlmodel.SequenceConfigInput{{ID: &bad}},
	}, resolvertest.UserAuthContext())

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "configs[0].id", valErr.Field)
}
