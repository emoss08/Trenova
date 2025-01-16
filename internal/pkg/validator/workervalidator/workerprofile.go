package workervalidator

import (
	"context"

	"github.com/trenova-app/transport/internal/core/domain/worker"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/validator"
	"github.com/trenova-app/transport/internal/pkg/validator/compliancevalidator"
	"go.uber.org/fx"
)

type WorkerProfileValidatorParams struct {
	fx.In

	CompValidator *compliancevalidator.Validator
}

type WorkerProfileValidator struct {
	compValidator *compliancevalidator.Validator
}

func NewWorkerProfileValidator(p WorkerProfileValidatorParams) *WorkerProfileValidator {
	return &WorkerProfileValidator{
		compValidator: p.CompValidator,
	}
}

func (v *WorkerProfileValidator) Validate(ctx context.Context, valCtx *validator.ValidationContext, wp *worker.WorkerProfile, multiErr *errors.MultiError) {
	wp.Validate(ctx, multiErr)

	// Validate DOT Compliance
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
