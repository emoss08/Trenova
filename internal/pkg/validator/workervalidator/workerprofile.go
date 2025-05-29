package workervalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/compliancevalidator"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"go.uber.org/fx"
)

// WorkerProfileValidatorParams defines the dependencies required for initializing the WorkerProfileValidator.
// This includes the compliance validator, shipment control repository, and validation engine factory.
type WorkerProfileValidatorParams struct {
	fx.In

	CompValidator           *compliancevalidator.Validator
	ShipmentControlRepo     repositories.ShipmentControlRepository
	ValidationEngineFactory framework.ValidationEngineFactory
}

// WorkerProfileValidator is a validator for worker profiles.
// It validates worker profiles, including compliance and shipment control settings and the validation engine.
type WorkerProfileValidator struct {
	compValidator *compliancevalidator.Validator
	scp           repositories.ShipmentControlRepository
	vef           framework.ValidationEngineFactory
}

// NewWorkerProfileValidator initializes a new WorkerProfileValidator with the provided dependencies.
//
// Parameters:
//   - p: WorkerProfileValidatorParams containing dependencies.
//
// Returns:
//   - *WorkerProfileValidator: A new WorkerProfileValidator instance.
func NewWorkerProfileValidator(p WorkerProfileValidatorParams) *WorkerProfileValidator {
	return &WorkerProfileValidator{
		compValidator: p.CompValidator,
		scp:           p.ShipmentControlRepo,
		vef:           p.ValidationEngineFactory,
	}
}

// Validate validates a worker profile and returns a MultiError if there are any validation errors
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - wp: The worker profile to validate.
//   - multiErr: The MultiError to add validation errors to.
func (v *WorkerProfileValidator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	wp *worker.WorkerProfile,
	multiErr *errors.MultiError,
) {
	engine := v.vef.CreateEngine().
		ForField("profile").
		WithParent(multiErr)

	// Basic validation rules
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBasic,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				wp.Validate(ctx, multiErr)
				return nil
			},
		),
	)

	// Compliance validation
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageCompliance,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				// Use the compliance validator with the current multiErr
				v.compValidator.Validate(ctx, wp, multiErr)
				return nil
			},
		),
	)

	// ID validation for create operations
	if valCtx.IsCreate {
		engine.AddRule(
			framework.NewValidationRule(
				framework.ValidationStageBusinessRules,
				framework.ValidationPriorityHigh,
				func(_ context.Context, multiErr *errors.MultiError) error {
					if wp.ID.IsNotNil() {
						multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
					}
					return nil
				},
			),
		)
	}

	// Execute validation rules and add errors to the provided multiErr
	engine.ValidateInto(ctx, multiErr)
}
