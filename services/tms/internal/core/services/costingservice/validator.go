package costingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/costingcontrol"
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
	controlValidator  *validationframework.TenantedValidator[*costingcontrol.CostingControl]
	categoryValidator *validationframework.TenantedValidator[*costingcontrol.CostCategory]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		controlValidator: validationframework.
			NewTenantedValidatorBuilder[*costingcontrol.CostingControl]().
			WithModelName("CostingControl").
			WithUniquenessChecker(
				validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() }),
			).
			WithReferenceChecker(
				validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return p.DB.DB() }),
			).
			WithCustomRule(createGLActualsRule()).
			Build(),
		categoryValidator: validationframework.
			NewTenantedValidatorBuilder[*costingcontrol.CostCategory]().
			WithModelName("CostCategory").
			WithUniquenessChecker(
				validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() }),
			).
			WithReferenceChecker(
				validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return p.DB.DB() }),
			).
			Build(),
	}
}

func createGLActualsRule() validationframework.TenantedRule[*costingcontrol.CostingControl] {
	return validationframework.NewTenantedRule[*costingcontrol.CostingControl]("gl_actuals").
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *costingcontrol.CostingControl,
			_ *validationframework.TenantedValidationContext,
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
) *errortypes.MultiError {
	return v.controlValidator.ValidateUpdate(ctx, entity)
}

func (v *Validator) ValidateCategoryUpdate(
	ctx context.Context,
	entity *costingcontrol.CostCategory,
) *errortypes.MultiError {
	return v.categoryValidator.ValidateUpdate(ctx, entity)
}
