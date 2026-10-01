package servicetypeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/servicetype"
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
	validator *validationframework.TenantedValidator[*servicetype.ServiceType]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*servicetype.ServiceType]().
			WithModelName("ServiceType").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerScoped(p.DB)).
			WithReferenceChecker(validationframework.NewBunReferenceCheckerScoped(p.DB)).
			WithUniqueField(
				"code",
				"code",
				"Service type with this code already exists in your organization",
				func(st *servicetype.ServiceType) any { return st.Code },
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *servicetype.ServiceType,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *servicetype.ServiceType,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
