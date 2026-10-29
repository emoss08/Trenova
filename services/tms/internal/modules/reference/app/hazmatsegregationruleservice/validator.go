package hazmatsegregationruleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazmatsegregationrule/hazmatsegregationruledb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/keelcheck"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*hazmatsegregationrule.HazmatSegregationRule]
}

func NewValidator(p ValidatorParams) *Validator {
	checker := keelcheck.NewChecker(
		p.DB,
		hazmatsegregationruledb.HazmatSegregationRulesSchema(),
	)

	return &Validator{
		validator: tenantedvalidation.NewBuilder[*hazmatsegregationrule.HazmatSegregationRule]().
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
