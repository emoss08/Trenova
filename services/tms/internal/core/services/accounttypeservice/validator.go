package accounttypeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accounttype"
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
	validator *validationframework.TenantedValidator[*accounttype.AccountType]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*accounttype.AccountType]().
			WithModelName("AccountType").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerScoped(p.DB)).
			WithReferenceChecker(validationframework.NewBunReferenceCheckerScoped(p.DB)).
			WithUniqueField(
				"code",
				"code",
				"Account type with this code already exists in your organization",
				func(a *accounttype.AccountType) any { return a.Code },
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *accounttype.AccountType,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *accounttype.AccountType,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
