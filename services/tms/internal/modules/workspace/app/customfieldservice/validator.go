package customfieldservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield/customfielddb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/keelcheck"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*customfield.CustomFieldDefinition]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*customfield.CustomFieldDefinition]().
			WithModelName("Custom Field Definition").
			WithUniquenessChecker(
				keelcheck.NewChecker(p.DB, customfielddb.CustomFieldDefinitionsSchema()),
			).
			WithCompositeUniqueFields(
				"resource_type_name",
				"Custom field with this name already exists for this resource type",
				tenantedvalidation.CompositeField[*customfield.CustomFieldDefinition]{
					FieldName:     "resourceType",
					Column:        "resource_type",
					CaseSensitive: true,
					GetValue:      func(e *customfield.CustomFieldDefinition) any { return e.ResourceType },
				},
				tenantedvalidation.CompositeField[*customfield.CustomFieldDefinition]{
					FieldName:     "name",
					Column:        "name",
					CaseSensitive: true,
					GetValue:      func(e *customfield.CustomFieldDefinition) any { return e.Name },
				},
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *customfield.CustomFieldDefinition,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *customfield.CustomFieldDefinition,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
