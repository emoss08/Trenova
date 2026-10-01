package hazmatsegregationruleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
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
	validator *validationframework.TenantedValidator[*hazmatsegregationrule.HazmatSegregationRule]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*hazmatsegregationrule.HazmatSegregationRule]().
			WithModelName("HazmatSegregationRule").
			WithUniquenessChecker(
				validationframework.NewBunUniquenessCheckerScoped(p.DB),
			).
			WithReferenceChecker(
				validationframework.NewBunReferenceCheckerScoped(p.DB),
			).
			WithUniqueField(
				"name",
				"name",
				"Hazmat segregation rule with this name already exists in your organization",
				func(entity *hazmatsegregationrule.HazmatSegregationRule) any { return entity.Name },
			).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *hazmatsegregationrule.HazmatSegregationRule,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *hazmatsegregationrule.HazmatSegregationRule,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
