package billingcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
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
	validator *validationframework.TenantedValidator[*tenant.BillingControl]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*tenant.BillingControl]().
			WithModelName("BillingControl").
			WithUniquenessChecker(
				validationframework.NewBunUniquenessCheckerLazy(
					func() bun.IDB { return p.DB.DB() },
				),
			).
			WithReferenceChecker(
				validationframework.NewBunReferenceCheckerLazy(
					func() bun.IDB { return p.DB.DB() },
				),
			).
			Build(),
	}
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *tenant.BillingControl,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
