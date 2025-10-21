package workervalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/framework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Repo             repositories.WorkerRepository
	ProfileValidator *WorkerProfileValidator
}

type Validator struct {
	profileValidator *WorkerProfileValidator
	repo             repositories.WorkerRepository
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		profileValidator: p.ProfileValidator,
		repo:             p.Repo,
	}
}

func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	w *worker.Worker,
) *errortypes.MultiError {
	engine := framework.NewValidationEngine(framework.DefaultEngineConfig())

	engine.AddRule(framework.NewBusinessRule("worker_validation").
		WithStage(framework.ValidationStageBusinessRules).
		WithPriority(framework.ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			w.Validate(multiErr)
			return nil
		}),
	)

	engine.AddRule(framework.NewConcreteRule("worker_profile_validation").
		WithStage(framework.ValidationStageBusinessRules).
		WithPriority(framework.ValidationPriorityHigh).
		WithValidation(func(ctx context.Context, multiErr *errortypes.MultiError) error {
			v.profileValidator.Validate(ctx, w.Profile, multiErr)
			return nil
		}),
	)

	engine.AddRule(
		framework.NewBusinessRule("id_validation").
			WithStage(framework.ValidationStageBusinessRules).
			WithPriority(framework.ValidationPriorityHigh).
			WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
				if valCtx.IsCreate && w.ID.IsNotNil() {
					multiErr.Add("id", errortypes.ErrInvalid, "ID cannot be set on create")
				}
				return nil
			}),
	)

	return engine.Validate(ctx)
}
