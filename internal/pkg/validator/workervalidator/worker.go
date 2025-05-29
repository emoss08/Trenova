package workervalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Repo                    repositories.WorkerRepository
	HazmatExpRepo           repositories.HazmatExpirationRepository
	ProfileValidator        *WorkerProfileValidator
	PTOValidator            *WorkerPTOValidator
	ValidationEngineFactory framework.ValidationEngineFactory
}

type Validator struct {
	repo             repositories.WorkerRepository
	hazExpRepo       repositories.HazmatExpirationRepository
	profileValidator *WorkerProfileValidator
	ptoValidator     *WorkerPTOValidator
	vef              framework.ValidationEngineFactory
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		repo:             p.Repo,
		profileValidator: p.ProfileValidator,
		ptoValidator:     p.PTOValidator,
		hazExpRepo:       p.HazmatExpRepo,
		vef:              p.ValidationEngineFactory,
	}
}

// Validate validates a worker and returns a MultiError if there are any validation errors
func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	wrk *worker.Worker,
) *errors.MultiError {
	engine := v.vef.CreateEngine()

	// Basic validation rules (field presence, format, etc.)
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBasic,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				wrk.Validate(ctx, multiErr)
				return nil
			},
		),
	)

	// Worker profile validation
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBusinessRules,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				if wrk.Profile != nil {
					v.profileValidator.Validate(ctx, valCtx, wrk.Profile, multiErr)
				}
				return nil
			},
		),
	)

	// Validate PTO
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBusinessRules,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				for idx, pto := range wrk.PTO {
					v.ptoValidator.Validate(ctx, valCtx, wrk, pto, multiErr, idx)
				}
				return nil
			},
		),
	)

	// Check if the worker is eligible for assignment
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBusinessRules,
			framework.ValidationPriorityHigh,
			func(_ context.Context, multiErr *errors.MultiError) error {
				if err := v.CheckAssignmentEligibility(wrk); err != nil {
					multiErr.Add("assignmentEligibility", errors.ErrInvalid, err.Error())
				}
				return nil
			},
		),
	)

	return engine.Validate(ctx)
}

func (v *Validator) CheckAssignmentEligibility(wrk *worker.Worker) error {
	if wrk.Profile == nil {
		return ErrWorkerProfileRequired
	}

	// Worker must be active
	if wrk.Status != domain.StatusActive {
		wrk.CanBeAssigned = false
		wrk.AssignmentBlocked = "Worker is not active"
		return nil
	}

	// Check if profile indicates qualified status
	if !wrk.Profile.IsQualified {
		wrk.CanBeAssigned = false
		wrk.AssignmentBlocked = wrk.Profile.DisqualificationReason
		return nil
	}

	// Check document compliance
	// TODO(Wolfred): While not smart we may want to allow organizations to override this
	// I'll wait for someone to complain about it before I make it configurable
	if wrk.Profile.ComplianceStatus != worker.ComplianceStatusCompliant {
		wrk.CanBeAssigned = false
		wrk.AssignmentBlocked = "Worker is not compliant with required documentation"
		return nil
	}

	// If all checks pass, then the worker is eligible for assignment
	wrk.CanBeAssigned = true
	wrk.AssignmentBlocked = ""

	return nil
}
