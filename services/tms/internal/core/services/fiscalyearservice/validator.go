package fiscalyearservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
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
	validator *validationframework.TenantedValidator[*fiscalyear.FiscalYear]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.
			NewTenantedValidatorBuilder[*fiscalyear.FiscalYear]().
			WithModelName("FiscalYear").
			WithCustomRule(createDateValidationRule()).
			WithCustomRule(createStatusConsistencyRule()).
			WithCustomRule(createCurrentYearRule(p.DB)).
			WithCustomRule(createOverlappingYearsRule(p.DB)).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *fiscalyear.FiscalYear,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *fiscalyear.FiscalYear,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}
