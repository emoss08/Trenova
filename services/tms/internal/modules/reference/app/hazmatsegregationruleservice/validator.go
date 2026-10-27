package hazmatsegregationruleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazmatsegregationrule/hazmatsegregationruledb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator validationframework.TenantedEntityValidator[*hazmatsegregationrule.HazmatSegregationRule]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := validationframework.NewKeelChecker(
		p.DB,
		hazmatsegregationruledb.HazmatSegregationRulesSchema(),
	)

	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*hazmatsegregationrule.HazmatSegregationRule]().
			WithModelName("HazmatSegregationRule").
			WithUniquenessChecker(checker).
			WithReferenceChecker(checker).
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
