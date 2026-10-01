package userservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator *validationframework.TenantedValidator[*tenant.User]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*tenant.User]().
			WithModelName("User").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerScoped(p.DB)).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *tenant.User,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *tenant.User,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
