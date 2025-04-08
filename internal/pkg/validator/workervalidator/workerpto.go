package workervalidator

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator"
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
		// TODO(Wolfred): allow transitioning Approved -> Rejected or
		// just keep it limited to Cancelled depending on your business logic.
	},
	worker.PTOStatusRejected:  {}, // No transitions allowed once Rejected.
	worker.PTOStatusCancelled: {}, // No transitions allowed once Cancelled.
}

type WorkerPTOValidatorParams struct {
	fx.In

	Repo repositories.WorkerRepository
}

type WorkerPTOValidator struct {
	repo repositories.WorkerRepository
}

func NewWorkerPTOValidator(p WorkerPTOValidatorParams) *WorkerPTOValidator {
	return &WorkerPTOValidator{
		repo: p.Repo,
	}
}

// ValidatePTO validates a worker's PTO and returns a MultiError if there are any validation errors
// This is a wrapper for the individual PTO checks
func (v *WorkerPTOValidator) Validate(ctx context.Context, valCtx *validator.ValidationContext, wrk *worker.Worker, pto *worker.WorkerPTO, multiErr *errors.MultiError, index int) {
	// Common PTO validation
	pto.Validate(ctx, multiErr, index)

	// Validate PTO overlaps
	if valCtx.IsUpdate {
		v.validatePTOStatusTransition(ctx, wrk, pto, multiErr, index)
		v.validatePTOOverlaps(wrk, multiErr)
	}

	// Validate ID
	if valCtx.IsCreate {
		v.validateID(wrk, multiErr)
	}
}

func (v *WorkerPTOValidator) validatePTOStatusTransition(ctx context.Context, wrk *worker.Worker, pto *worker.WorkerPTO, multiErr *errors.MultiError, index int) {
	oldPTO, err := v.repo.GetWorkerPTO(ctx, pto.ID, wrk.ID, wrk.BusinessUnitID, wrk.OrganizationID)
	if err != nil {
		return
	}

	// If there is no change in status, no need to validate transitions
	if oldPTO.Status == pto.Status {
		return
	}

	allowedTransitions, ok := validPTOStatusTransitions[oldPTO.Status]
	if !ok {
		multiErr.Add(
			fmt.Sprintf("pto[%d].status", index), errors.ErrInvalid,
			fmt.Sprintf("Invalid status transition from %s to %s", oldPTO.Status, pto.Status),
		)
		return
	}

	// Check if the new status is one of the allowed next states
	isAllowed := false
	for _, next := range allowedTransitions {
		if pto.Status == next {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		multiErr.Add(
			fmt.Sprintf("pto[%d].status", index), errors.ErrInvalid,
			fmt.Sprintf("Invalid status transition from %s to %s", oldPTO.Status, pto.Status),
		)
	}
}

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

			v.checkPTOOverlap(i, pto, otherPTO, multiErr)
		}
	}
}

// checkPTOOverlap checks if two PTO requests overlap and adds appropriate errors
func (v *WorkerPTOValidator) checkPTOOverlap(index int, pto, otherPTO *worker.WorkerPTO, multiErr *errors.MultiError) {
	startDate := time.Unix(otherPTO.StartDate, 0).Format("2006-01-02")
	endDate := time.Unix(otherPTO.EndDate, 0).Format("2006-01-02")
	dateRange := fmt.Sprintf("(%s to %s)", startDate, endDate)

	// Complete overlap (both dates fall within another request)
	if pto.StartDate >= otherPTO.StartDate && pto.EndDate <= otherPTO.EndDate {
		multiErr.Add(
			fmt.Sprintf("pto[%d].startDate", index),
			errors.ErrAlreadyExists,
			fmt.Sprintf("Start date falls within an existing PTO request %s", dateRange),
		)
		multiErr.Add(
			fmt.Sprintf("pto[%d].endDate", index),
			errors.ErrAlreadyExists,
			fmt.Sprintf("End date falls within an existing PTO request %s", dateRange),
		)
		return
	}

	// Start date overlaps with another request
	if pto.StartDate >= otherPTO.StartDate && pto.StartDate <= otherPTO.EndDate {
		multiErr.Add(
			fmt.Sprintf("pto[%d].startDate", index),
			errors.ErrAlreadyExists,
			fmt.Sprintf("Start date overlaps with an existing PTO request %s", dateRange),
		)
		return
	}

	// End date overlaps with another request
	if pto.EndDate >= otherPTO.StartDate && pto.EndDate <= otherPTO.EndDate {
		multiErr.Add(
			fmt.Sprintf("pto[%d].endDate", index),
			errors.ErrAlreadyExists,
			fmt.Sprintf("End date overlaps with an existing PTO request %s", dateRange),
		)
		return
	}

	// Another request falls completely within this request
	if otherPTO.StartDate >= pto.StartDate && otherPTO.EndDate <= pto.EndDate {
		multiErr.Add(
			fmt.Sprintf("pto[%d].startDate", index),
			errors.ErrAlreadyExists,
			fmt.Sprintf("Request overlaps with an existing PTO request %s", dateRange),
		)
		multiErr.Add(
			fmt.Sprintf("pto[%d].endDate", index),
			errors.ErrAlreadyExists,
			fmt.Sprintf("Request overlaps with an existing PTO request %s", dateRange),
		)
	}
}

func (v *WorkerPTOValidator) validateID(wrk *worker.Worker, multiErr *errors.MultiError) {
	// Loop through all of the PTOs and validate that the ID is not set
	for idx, pto := range wrk.PTO {
		if pto.ID.IsNotNil() {
			multiErr.Add(fmt.Sprintf("pto[%d].id", idx), errors.ErrInvalid, "ID cannot be set on create")
		}
	}
}
