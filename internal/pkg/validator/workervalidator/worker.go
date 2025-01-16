package workervalidator

import (
	"context"

	"github.com/trenova-app/transport/internal/core/domain"
	"github.com/trenova-app/transport/internal/core/domain/worker"
	"github.com/trenova-app/transport/internal/core/ports/repositories"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/validator"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Repo             repositories.WorkerRepository
	HazmatExpRepo    repositories.HazmatExpirationRepository
	ProfileValidator *WorkerProfileValidator
	PTOValidator     *WorkerPTOValidator
}

type Validator struct {
	repo             repositories.WorkerRepository
	hazExpRepo       repositories.HazmatExpirationRepository
	profileValidator *WorkerProfileValidator
	ptoValidator     *WorkerPTOValidator
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		repo:             p.Repo,
		profileValidator: p.ProfileValidator,
		ptoValidator:     p.PTOValidator,
		hazExpRepo:       p.HazmatExpRepo,
	}
}

// Validate validates a worker and returns a MultiError if there are any validation errors
func (v *Validator)Validate(ctx context.Context, valCtx *validator.ValidationContext, wrk *worker.Worker) *errors.MultiError {
	multiErr := errors.NewMultiError()

	// Basic worker validation
	wrk.Validate(ctx, multiErr)

	// Worker profile validation
	if wrk.Profile != nil {
		v.profileValidator.Validate(ctx, valCtx, wrk.Profile, multiErr)
	}

	// Validate PTO
	for idx, pto := range wrk.PTO {
		v.ptoValidator.Validate(ctx, valCtx, wrk, pto, multiErr, idx)
	}

	// Check if the worker is eligible for assignment
	if err := v.CheckAssignmentEligibility(wrk); err != nil {
		multiErr.Add("assignmentEligibility", errors.ErrInvalid, err.Error())
	}

	// Validate ID
	v.validateID(wrk, valCtx, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
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

func (v *Validator) validateID(wrk *worker.Worker, valCtx *validator.ValidationContext, multiErr *errors.MultiError) {
	if valCtx.IsCreate && wrk.ID.IsNotNil() {
		multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
	}
}
