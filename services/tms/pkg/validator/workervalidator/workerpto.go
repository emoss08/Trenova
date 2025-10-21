package workervalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/framework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type WorkerPTOValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type WorkerPTOValidator struct {
	factory *framework.TenantedValidatorFactory[*worker.WorkerPTO]
}

func NewWorkerPTOValidator(p WorkerPTOValidatorParams) *WorkerPTOValidator {
	factory := framework.NewTenantedValidatorFactory[*worker.WorkerPTO](
		func(ctx context.Context) (*bun.DB, error) {
			return p.DB.DB(ctx)
		},
	).
		WithModelName("WorkerPTO").
		WithCustomRules(func(entity *worker.WorkerPTO, vc *validator.ValidationContext) []framework.ValidationRule {
			var rules []framework.ValidationRule

			if vc.IsCreate {
				rules = append(rules, framework.NewBusinessRule("id_validation").
					WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
						if entity.ID.IsNotNil() {
							multiErr.Add("id", errortypes.ErrInvalid, "ID cannot be set on create")
						}
						return nil
					}),
				)
			}

			return rules
		})

	return &WorkerPTOValidator{
		factory: factory,
	}
}

func (v *WorkerPTOValidator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	entity *worker.WorkerPTO,
) *errortypes.MultiError {
	return v.factory.Validate(ctx, entity, valCtx)
}
