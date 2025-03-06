package workervalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/compliancevalidator"
	"go.uber.org/fx"
)

type WorkerProfileValidatorParams struct {
	fx.In

	CompValidator       *compliancevalidator.Validator
	ShipmentControlRepo repositories.ShipmentControlRepository
}

type WorkerProfileValidator struct {
	compValidator *compliancevalidator.Validator
	scp           repositories.ShipmentControlRepository
}

func NewWorkerProfileValidator(p WorkerProfileValidatorParams) *WorkerProfileValidator {
	return &WorkerProfileValidator{
		compValidator: p.CompValidator,
		scp:           p.ShipmentControlRepo,
	}
}

func (v *WorkerProfileValidator) Validate(ctx context.Context, valCtx *validator.ValidationContext, wp *worker.WorkerProfile, multiErr *errors.MultiError) {
	wp.Validate(ctx, multiErr)


	v.compValidator.ValidateWorkerCompliance(ctx, wp, multiErr)

	if valCtx.IsCreate {
		v.validateID(wp, multiErr)
	}
}

func (v *WorkerProfileValidator) validateID(wp *worker.WorkerProfile, multiErr *errors.MultiError) {
	if wp.ID.IsNotNil() {
		multiErr.Add("profile.id", errors.ErrInvalid, "ID cannot be set on create")
	}
}
