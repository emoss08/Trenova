/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package workervalidator

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"go.uber.org/fx"
)

var validPTOStatusTransitions = map[worker.PTOStatus][]worker.PTOStatus{
	worker.PTOStatusRequested: {
		worker.PTOStatusApproved,
		worker.PTOStatusRejected,
		worker.PTOStatusCancelled,
	},
	worker.PTOStatusApproved: {
		worker.PTOStatusCancelled,
	},
	worker.PTOStatusRejected:  {}, // No transitions allowed once Rejected.
	worker.PTOStatusCancelled: {}, // No transitions allowed once Cancelled.
}

// WorkerPTOValidatorParams defines the dependencies required for initializing the WorkerPTOValidator.
// This includes the worker repository and validation engine factory.
type WorkerPTOValidatorParams struct {
	fx.In

	Repo                    repositories.WorkerRepository
	ValidationEngineFactory framework.ValidationEngineFactory
}

// WorkerPTOValidator is a validator for worker PTOs.
// It validates worker PTOs, including status transitions and overlaps.
type WorkerPTOValidator struct {
	repo repositories.WorkerRepository
	vef  framework.ValidationEngineFactory
}

// NewWorkerPTOValidator initializes a new WorkerPTOValidator with the provided dependencies.
//
// Parameters:
//   - p: WorkerPTOValidatorParams containing dependencies.
//
// Returns:
//   - *WorkerPTOValidator: A new WorkerPTOValidator instance.
func NewWorkerPTOValidator(p WorkerPTOValidatorParams) *WorkerPTOValidator {
	return &WorkerPTOValidator{
		repo: p.Repo,
		vef:  p.ValidationEngineFactory,
	}
}

// ValidatePTO validates a worker's PTO and returns a MultiError if there are any validation errors
// This is a wrapper for the individual PTO checks
//
// Parameters:
//   - ctx: The context of the request.
//   - valCtx: The validation context.
//   - wrk: The worker to validate.
//   - pto: The PTO to validate.
//   - multiErr: The MultiError to add validation errors to.
//   - index: The index of the PTO in the worker's PTO array.
func (v *WorkerPTOValidator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	wrk *worker.Worker,
	pto *worker.WorkerPTO,
	multiErr *errors.MultiError,
	idx int,
) {
	engine := v.vef.CreateEngine().
		ForField("pto").
		AtIndex(idx).
		WithParent(multiErr)

	// * Basic validation rules (field presence, format, etc.)
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageBasic,
			framework.ValidationPriorityHigh,
			func(ctx context.Context, multiErr *errors.MultiError) error {
				pto.Validate(ctx, multiErr)
				return nil
			},
		),
	)

	// * Status transition validation
	if valCtx.IsUpdate {
		engine.AddRule(
			framework.NewValidationRule(
				framework.ValidationStageBusinessRules,
				framework.ValidationPriorityHigh,
				func(ctx context.Context, multiErr *errors.MultiError) error {
					v.validatePTOStatusTransition(ctx, wrk, pto, multiErr)
					return nil
				},
			),
		)

		// * PTO overlap validation
		engine.AddRule(
			framework.NewValidationRule(
				framework.ValidationStageBusinessRules,
				framework.ValidationPriorityMedium,
				func(_ context.Context, multiErr *errors.MultiError) error {
					v.validatePTOOverlaps(wrk, multiErr)
					return nil
				},
			),
		)
	}

	// * ID validation on create
	if valCtx.IsCreate {
		engine.AddRule(
			framework.NewValidationRule(
				framework.ValidationStageBusinessRules,
				framework.ValidationPriorityHigh,
				func(_ context.Context, multiErr *errors.MultiError) error {
					if pto.ID.IsNotNil() {
						multiErr.Add("id", errors.ErrInvalid, "ID cannot be set on create")
					}
					return nil
				},
			),
		)
	}

	// * Execute validation rules and add errors to the provided multiErr
	engine.ValidateInto(ctx, multiErr)
}

func (v *WorkerPTOValidator) validatePTOStatusTransition(
	ctx context.Context,
	wrk *worker.Worker,
	pto *worker.WorkerPTO,
	multiErr *errors.MultiError,
) {
	oldPTO, err := v.repo.GetWorkerPTO(ctx, &repositories.GetWorkerPTORequest{
		PtoID:    pto.ID,
		WorkerID: wrk.ID,
		BuID:     wrk.BusinessUnitID,
		OrgID:    wrk.OrganizationID,
	})
	if err != nil {
		return
	}

	// * If there is no change in status, no need to validate transitions
	if oldPTO.Status == pto.Status {
		return
	}

	allowedTransitions, ok := validPTOStatusTransitions[oldPTO.Status]
	if !ok {
		multiErr.Add(
			"status", errors.ErrInvalid,
			fmt.Sprintf("Invalid status transition from %s to %s", oldPTO.Status, pto.Status),
		)
		return
	}

	// * Check if the new status is one of the allowed next states
	isAllowed := slices.Contains(allowedTransitions, pto.Status)

	if !isAllowed {
		multiErr.Add(
			"status", errors.ErrInvalid,
			fmt.Sprintf("Invalid status transition from %s to %s", oldPTO.Status, pto.Status),
		)
	}
}

// validatePTOOverlaps checks if any PTO requests overlap and adds appropriate errors
//
// Parameters:
//   - wrk: The worker to validate.
//   - multiErr: The MultiError to add validation errors to.
func (v *WorkerPTOValidator) validatePTOOverlaps(wrk *worker.Worker, multiErr *errors.MultiError) {
	if len(wrk.PTO) <= 1 {
		return
	}

	for i, pto := range wrk.PTO {
		if pto.IsInvalid() {
			continue
		}

		for j, otherPTO := range wrk.PTO {
			if i == j || otherPTO.IsInvalid() {
				continue
			}

			v.checkPTOOverlap(pto, otherPTO, multiErr)
		}
	}
}

// checkPTOOverlap checks if two PTO requests overlap and adds appropriate errors
//
// Parameters:
//   - index: The index of the PTO in the worker's PTO array.
//   - pto: The PTO to check for overlap.
//   - otherPTO: The other PTO to check against.
//   - multiErr: The MultiError to add validation errors to.
func (v *WorkerPTOValidator) checkPTOOverlap(
	pto, otherPTO *worker.WorkerPTO,
	multiErr *errors.MultiError,
) {
	startDate := time.Unix(otherPTO.StartDate, 0).Format("2006-01-02")
	endDate := time.Unix(otherPTO.EndDate, 0).Format("2006-01-02")
	dateRange := fmt.Sprintf("(%s to %s)", startDate, endDate)

	// Complete overlap (both dates fall within another request)
	if pto.StartDate >= otherPTO.StartDate && pto.EndDate <= otherPTO.EndDate {
		multiErr.Add(
			"startDate",
			errors.ErrAlreadyExists,
			fmt.Sprintf("Start date falls within an existing PTO request %s", dateRange),
		)
		multiErr.Add(
			"endDate",
			errors.ErrAlreadyExists,
			fmt.Sprintf("End date falls within an existing PTO request %s", dateRange),
		)
		return
	}

	// Start date overlaps with another request
	if pto.StartDate >= otherPTO.StartDate && pto.StartDate <= otherPTO.EndDate {
		multiErr.Add(
			"startDate",
			errors.ErrAlreadyExists,
			fmt.Sprintf("Start date overlaps with an existing PTO request %s", dateRange),
		)
		return
	}

	// End date overlaps with another request
	if pto.EndDate >= otherPTO.StartDate && pto.EndDate <= otherPTO.EndDate {
		multiErr.Add(
			"endDate",
			errors.ErrAlreadyExists,
			fmt.Sprintf("End date overlaps with an existing PTO request %s", dateRange),
		)
		return
	}

	// Another request falls completely within this request
	if otherPTO.StartDate >= pto.StartDate && otherPTO.EndDate <= pto.EndDate {
		multiErr.Add(
			"startDate",
			errors.ErrAlreadyExists,
			fmt.Sprintf("Request overlaps with an existing PTO request %s", dateRange),
		)
		multiErr.Add(
			"endDate",
			errors.ErrAlreadyExists,
			fmt.Sprintf("Request overlaps with an existing PTO request %s", dateRange),
		)
	}
}
