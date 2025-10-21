package organizationvalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validator/framework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct{}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{}
}

func (v *Validator) Validate(
	ctx context.Context,
	entity *tenant.Organization,
) *errortypes.MultiError {
	engine := framework.NewValidationEngine(framework.DefaultEngineConfig())

	engine.AddRule(
		framework.NewConcreteRule("basic_validation").
			WithStage(framework.ValidationStageBasic).
			WithPriority(framework.ValidationPriorityHigh).
			WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
				entity.Validate(multiErr)
				return nil
			}),
	)

	return engine.Validate(ctx)
}
