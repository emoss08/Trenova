package invoiceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
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
	validator *validationframework.TenantedValidator[*invoice.Invoice]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*invoice.Invoice]().
			WithModelName("Invoice").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *invoice.Invoice,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *invoice.Invoice,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
