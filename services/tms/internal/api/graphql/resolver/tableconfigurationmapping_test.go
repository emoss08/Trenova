package resolver

import (
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTableConfigurationFixture() *tableconfiguration.TableConfiguration {
	return &tableconfiguration.TableConfiguration{
		Name:        "My shipments",
		Description: "Open loads",
		Resource:    "shipment",
		TableConfig: &tableconfiguration.TableConfig{PageSize: 25, Density: "compact"},
		Visibility:  tableconfiguration.VisibilityPrivate,
		IsDefault:   true,
	}
}

func TestApplyTableConfigurationPatch_AbsentLeavesUnchanged(t *testing.T) {
	t.Parallel()

	entity := newTableConfigurationFixture()
	require.NoError(
		t,
		applyTableConfigurationPatch(entity, gqlmodel.TableConfigurationPatchInput{}),
	)
	assert.Equal(t, newTableConfigurationFixture(), entity)
}

func TestApplyTableConfigurationPatch_NullClearsOptionalFields(t *testing.T) {
	t.Parallel()

	entity := newTableConfigurationFixture()
	require.NoError(t, applyTableConfigurationPatch(entity, gqlmodel.TableConfigurationPatchInput{
		Description: graphql.OmittableOf[*string](nil),
		IsDefault:   graphql.OmittableOf[*bool](nil),
	}))
	assert.Empty(t, entity.Description)
	assert.False(t, entity.IsDefault)
	assert.Equal(t, "My shipments", entity.Name)
	assert.Equal(t, "shipment", entity.Resource)
}

func TestApplyTableConfigurationPatch_NullRejectedOnRequiredFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		field string
		input gqlmodel.TableConfigurationPatchInput
	}{
		{
			field: "name",
			input: gqlmodel.TableConfigurationPatchInput{
				Name: graphql.OmittableOf[*string](nil),
			},
		},
		{
			field: "resource",
			input: gqlmodel.TableConfigurationPatchInput{
				Resource: graphql.OmittableOf[*string](nil),
			},
		},
		{
			field: "tableConfig",
			input: gqlmodel.TableConfigurationPatchInput{
				TableConfig: graphql.OmittableOf[map[string]any](nil),
			},
		},
		{
			field: "visibility",
			input: gqlmodel.TableConfigurationPatchInput{
				Visibility: graphql.OmittableOf[*tableconfiguration.Visibility](nil),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			t.Parallel()

			entity := newTableConfigurationFixture()
			err := applyTableConfigurationPatch(entity, tc.input)
			requireRequiredPatchError(t, err, tc.field)
			assert.Equal(t, newTableConfigurationFixture(), entity)
		})
	}
}

func TestApplyTableConfigurationPatch_ValuesAreSet(t *testing.T) {
	t.Parallel()

	name := "Team board"
	description := "Shared with dispatch"
	resource := "tractor"
	visibility := tableconfiguration.VisibilityShared
	isDefault := false
	entity := newTableConfigurationFixture()
	require.NoError(t, applyTableConfigurationPatch(entity, gqlmodel.TableConfigurationPatchInput{
		Name:        graphql.OmittableOf(&name),
		Description: graphql.OmittableOf(&description),
		Resource:    graphql.OmittableOf(&resource),
		TableConfig: graphql.OmittableOf(map[string]any{
			"pageSize":    50,
			"density":     "comfortable",
			"columnOrder": []any{"code", "status"},
		}),
		Visibility: graphql.OmittableOf(&visibility),
		IsDefault:  graphql.OmittableOf(&isDefault),
	}))
	assert.Equal(t, name, entity.Name)
	assert.Equal(t, description, entity.Description)
	assert.Equal(t, resource, entity.Resource)
	assert.Equal(t, visibility, entity.Visibility)
	assert.False(t, entity.IsDefault)
	require.NotNil(t, entity.TableConfig)
	assert.Equal(t, 50, entity.TableConfig.PageSize)
	assert.Equal(t, "comfortable", entity.TableConfig.Density)
	assert.Equal(t, []string{"code", "status"}, entity.TableConfig.ColumnOrder)
}

func TestApplyTableConfigurationPatch_EmptyTableConfigIsAccepted(t *testing.T) {
	t.Parallel()

	entity := newTableConfigurationFixture()
	require.NoError(t, applyTableConfigurationPatch(entity, gqlmodel.TableConfigurationPatchInput{
		TableConfig: graphql.OmittableOf(map[string]any{}),
	}))
	require.NotNil(t, entity.TableConfig)
	assert.Equal(t, tableconfiguration.TableConfig{}, *entity.TableConfig)
}
