package documenttypeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator *validationframework.TenantedValidator[*documenttype.DocumentType]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*documenttype.DocumentType]().
			WithModelName("Document Type").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() })).
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
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *documenttype.DocumentType,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
