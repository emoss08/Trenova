package customfielddefinitionresolver

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomFieldDefinitionMutations_CheckTheDefinitionPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("cfd_").String()
	input := gqlmodel.CustomFieldDefinitionInput{
		ResourceType: "trailer",
		Name:         "door",
		Label:        "Door",
	}

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"createCustomFieldDefinition",
			permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).CreateCustomFieldDefinition(ctx, input)
				return err
			},
		},
		{
			"updateCustomFieldDefinition",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateCustomFieldDefinition(ctx, id, input)
				return err
			},
		},
		{
			"patchCustomFieldDefinition",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).PatchCustomFieldDefinition(
					ctx,
					id,
					gqlmodel.CustomFieldDefinitionPatchInput{},
				)
				return err
			},
		},
		{
			"deleteCustomFieldDefinition",
			permission.OpDelete,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).DeleteCustomFieldDefinition(ctx, id)
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
				permission.ResourceCustomFieldDefinition,
				call.operation,
			)
		})
	}
}

func TestCustomFieldDefinitionFromInput_MapsOptionsRulesAndAttributes(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	id := pulid.MustNew("cfd_")
	maxLength := 20
	pattern := "^[a-z]+$"
	help := "How the doors open"
	description := "Two doors"
	version := 3

	entity := customFieldDefinitionFromInput(&gqlmodel.CustomFieldDefinitionInput{
		ResourceType: "trailer",
		Name:         "door_type",
		Label:        "Door type",
		FieldType:    customfield.FieldType("select"),
		IsRequired:   true,
		IsActive:     true,
		DisplayOrder: 2,
		Options: []*gqlmodel.CustomFieldSelectOptionInput{
			{Value: "swing", Label: "Swing", Description: &description},
			nil,
			{Value: "roll", Label: "Roll-up"},
		},
		ValidationRules: &gqlmodel.CustomFieldValidationRulesInput{
			MaxLength: &maxLength,
			Pattern:   &pattern,
		},
		DefaultValue: "swing",
		UIAttributes: &gqlmodel.CustomFieldUIAttributesInput{HelpText: &help},
		Version:      &version,
	}, id, authCtx)

	assert.Equal(t, id, entity.ID)
	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, []customfield.SelectOption{
		{Value: "swing", Label: "Swing", Description: "Two doors"},
		{Value: "roll", Label: "Roll-up"},
	}, entity.Options)
	require.NotNil(t, entity.ValidationRules)
	assert.Equal(t, &maxLength, entity.ValidationRules.MaxLength)
	assert.Nil(t, entity.ValidationRules.MinLength)
	assert.Equal(t, &pattern, entity.ValidationRules.Pattern)
	assert.Equal(t, "swing", entity.DefaultValue)
	require.NotNil(t, entity.UIAttributes)
	assert.Equal(t, "How the doors open", entity.UIAttributes.HelpText)
	assert.Equal(t, int64(3), entity.Version)
}

func TestCustomFieldDefinitionFromInput_LeavesAbsentRulesAndAttributesUnset(t *testing.T) {
	t.Parallel()

	entity := customFieldDefinitionFromInput(
		&gqlmodel.CustomFieldDefinitionInput{Name: "notes", Label: "Notes"},
		pulid.Nil,
		resolvertest.UserAuthContext(),
	)

	assert.Nil(t, entity.ValidationRules)
	assert.Nil(t, entity.UIAttributes)
	assert.Empty(t, entity.Options)
}

func TestApplyCustomFieldDefinitionPatch_DeactivatesWithoutTouchingTheRest(t *testing.T) {
	t.Parallel()

	existing := &customfield.CustomFieldDefinition{
		Label:        "Door type",
		IsActive:     true,
		IsRequired:   true,
		DisplayOrder: 2,
		Color:        "#2563eb",
	}
	inactive := false

	require.NoError(
		t,
		applyCustomFieldDefinitionPatch(existing, &gqlmodel.CustomFieldDefinitionPatchInput{
			IsActive: graphql.OmittableOf(&inactive),
		}),
	)

	assert.False(t, existing.IsActive)
	assert.True(t, existing.IsRequired)
	assert.Equal(t, "Door type", existing.Label)
	assert.Equal(t, 2, existing.DisplayOrder)
	assert.Equal(t, "#2563eb", existing.Color)
}

func TestApplyCustomFieldDefinitionPatch_RefusesToClearTheLabel(t *testing.T) {
	t.Parallel()

	existing := &customfield.CustomFieldDefinition{Label: "Door type"}

	err := applyCustomFieldDefinitionPatch(existing, &gqlmodel.CustomFieldDefinitionPatchInput{
		Label: graphql.OmittableOf[*string](nil),
	})

	resolvertest.RequireRequiredPatchError(t, err, "label")
	assert.Equal(t, "Door type", existing.Label)
}
