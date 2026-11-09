package hazmatsegregationruleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/reference/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*hazmatsegregationrule.HazmatSegregationRule]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*hazmatsegregationrule.HazmatSegregationRule]().
			WithModelName("HazmatSegregationRule").
			WithIntegrityChecker(p.Integrity).
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
) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *hazmatsegregationrule.HazmatSegregationRule,
) error {
	return v.validator.ValidateUpdate(ctx, entity)
}
