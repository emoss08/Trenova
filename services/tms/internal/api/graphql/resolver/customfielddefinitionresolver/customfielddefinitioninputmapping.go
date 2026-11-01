package customfielddefinitionresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
)

func customFieldDefinitionFromInput(
	input *gqlmodel.CustomFieldDefinitionInput,
	id pulid.ID,
	authCtx *authctx.AuthContext,
) *customfield.CustomFieldDefinition {
	entity := &customfield.CustomFieldDefinition{
		ID:              id,
		OrganizationID:  authCtx.OrganizationID,
		BusinessUnitID:  authCtx.BusinessUnitID,
		ResourceType:    input.ResourceType,
		Name:            input.Name,
		Label:           input.Label,
		Description:     gqlscalar.StringValue(input.Description),
		FieldType:       input.FieldType,
		IsRequired:      input.IsRequired,
		IsActive:        input.IsActive,
		DisplayOrder:    input.DisplayOrder,
		Color:           gqlscalar.StringValue(input.Color),
		Options:         selectOptionsFromInput(input.Options),
		ValidationRules: validationRulesFromInput(input.ValidationRules),
		DefaultValue:    input.DefaultValue,
		UIAttributes:    uiAttributesFromInput(input.UIAttributes),
	}
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	return entity
}

func selectOptionsFromInput(
	input []*gqlmodel.CustomFieldSelectOptionInput,
) []customfield.SelectOption {
	options := make([]customfield.SelectOption, 0, len(input))
	for _, option := range input {
		if option == nil {
			continue
		}
		options = append(options, customfield.SelectOption{
			Value:       option.Value,
			Label:       option.Label,
			Color:       gqlscalar.StringValue(option.Color),
			Description: gqlscalar.StringValue(option.Description),
		})
	}

	return options
}

func validationRulesFromInput(
	input *gqlmodel.CustomFieldValidationRulesInput,
) *customfield.ValidationRules {
	if input == nil {
		return nil
	}

	return &customfield.ValidationRules{
		MinLength: input.MinLength,
		MaxLength: input.MaxLength,
		Min:       input.Min,
		Max:       input.Max,
		Pattern:   input.Pattern,
	}
}

func uiAttributesFromInput(input *gqlmodel.CustomFieldUIAttributesInput) *customfield.UIAttributes {
	if input == nil {
		return nil
	}

	return &customfield.UIAttributes{
		Placeholder: gqlscalar.StringValue(input.Placeholder),
		HelpText:    gqlscalar.StringValue(input.HelpText),
		Width:       gqlscalar.StringValue(input.Width),
	}
}

func applyCustomFieldDefinitionPatch(
	entity *customfield.CustomFieldDefinition,
	input *gqlmodel.CustomFieldDefinitionPatchInput,
) error {
	if err := gqlscalar.ApplyRequired("label", "Label", input.Label, &entity.Label); err != nil {
		return err
	}
	gqlscalar.ApplyClearable(input.Description, &entity.Description)
	if err := gqlscalar.ApplyRequired(
		"isRequired",
		"Required",
		input.IsRequired,
		&entity.IsRequired,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"isActive",
		"Active",
		input.IsActive,
		&entity.IsActive,
	); err != nil {
		return err
	}
	if err := gqlscalar.ApplyRequired(
		"displayOrder",
		"Display order",
		input.DisplayOrder,
		&entity.DisplayOrder,
	); err != nil {
		return err
	}
	gqlscalar.ApplyClearable(input.Color, &entity.Color)

	return gqlscalar.ApplyRequiredInt("version", "Version", input.Version, &entity.Version)
}
