package documenttypeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*documenttype.DocumentType]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*documenttype.DocumentType]().
			WithModelName("Document Type").
			WithIntegrityChecker(p.Integrity).
			WithUniqueField(
				"name",
				"name",
				"Document type with this name already exists in your organization",
				func(e *documenttype.DocumentType) any { return e.Name },
			).
			WithCustomRule(createSystemDocumentTypeProtectionRule()).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *documenttype.DocumentType,
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *documenttype.DocumentType,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
