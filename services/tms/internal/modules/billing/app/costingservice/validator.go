package costingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
)

type Validator struct {
	controlValidator  tenantedvalidation.EntityValidator[*costingcontrol.CostingControl]
	categoryValidator tenantedvalidation.EntityValidator[*costingcontrol.CostCategory]
}

func NewValidator() *Validator {
	return &Validator{
		controlValidator: tenantedvalidation.NewBuilder[*costingcontrol.CostingControl]().
			WithModelName("CostingControl").
			WithCustomRule(createGLActualsRule()).
			Build(),
		categoryValidator: tenantedvalidation.NewBuilder[*costingcontrol.CostCategory]().
			WithModelName("CostCategory").
			Build(),
	}
}

func createGLActualsRule() validationframework.Rule[*costingcontrol.CostingControl] {
	return validationframework.NewRule[*costingcontrol.CostingControl]("gl_actuals").
		OnUpdate().
		WithStage(validationframework.StageBusinessRules).
		WithValidation(func(
			_ context.Context,
			entity *costingcontrol.CostingControl,
			_ *validationframework.Context,
			multiErr *errortypes.MultiError,
		) error {
			if !entity.GLActualsEnabled {
				return nil
			}

			if entity.GLRollingMonths < 1 || entity.GLRollingMonths > 12 {
				multiErr.Add(
					"glRollingMonths",
					errortypes.ErrInvalid,
					"GL rolling months must be between 1 and 12 when GL actuals are enabled",
				)
			}

			return nil
		})
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *costingcontrol.CostingControl,
) error {
	return v.controlValidator.ValidateUpdate(ctx, entity)
}

func (v *Validator) ValidateCategoryUpdate(
	ctx context.Context,
	entity *costingcontrol.CostCategory,
) error {
	return v.categoryValidator.ValidateUpdate(ctx, entity)
}
