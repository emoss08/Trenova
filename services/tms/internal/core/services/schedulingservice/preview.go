package schedulingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

// ShiftAssignmentPlan is what putting a worker on a pattern would do: the
// assignment made, and the open one it ends the day before.
type ShiftAssignmentPlan struct {
	Created  *worker.WorkerShiftAssignment
	Template *worker.ShiftTemplate
	Ended    []*worker.WorkerShiftAssignment
	EndedAt  int64
}

func (s *Service) PreviewAssignShift(
	ctx context.Context,
	req *AssignShiftRequest,
) (*ShiftAssignmentPlan, error) {
	return s.planAssignShift(ctx, req)
}

func (s *Service) planAssignShift(
	ctx context.Context,
	req *AssignShiftRequest,
) (*ShiftAssignmentPlan, error) {
	entity := req.Entity
	entity.OrganizationID = req.TenantInfo.OrgID
	entity.BusinessUnitID = req.TenantInfo.BuID
	entity.AssignedByID = req.UserID
	entity.EffectiveFrom = startOfDayUTC(entity.EffectiveFrom)

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	template, err := s.repo.GetTemplateByID(ctx, &repositories.GetShiftTemplateByIDRequest{
		ID:         entity.ShiftTemplateID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if template.Status != domaintypes.StatusActive {
		return nil, errortypes.NewValidationError(
			"shiftTemplateId",
			errortypes.ErrInvalidOperation,
			"That shift has been retired",
		)
	}
	if entity.CycleOffsetWeeks >= template.CycleWeeks && template.CycleWeeks > 1 {
		return nil, errortypes.NewValidationError(
			"cycleOffsetWeeks",
			errortypes.ErrInvalid,
			"This shift rotates over {0} weeks, so the offset is 0 to {1}",
			template.CycleWeeks,
			template.CycleWeeks-1,
		)
	}

	endAt := entity.EffectiveFrom - secondsPerDay
	ended, err := s.openAssignmentsToClose(ctx, &closeAssignmentParams{
		tenantInfo: req.TenantInfo,
		workerID:   entity.WorkerID,
		endAt:      endAt,
		userID:     req.UserID,
	})
	if err != nil {
		return nil, err
	}

	return &ShiftAssignmentPlan{
		Created:  entity,
		Template: template,
		Ended:    ended,
		EndedAt:  endAt,
	}, nil
}

// openAssignmentsToClose is the open assignments a new one replaces, as they
// stand; closeOpenAssignment ends them.
func (s *Service) openAssignmentsToClose(
	ctx context.Context,
	p *closeAssignmentParams,
) ([]*worker.WorkerShiftAssignment, error) {
	open, err := s.repo.ListAssignments(ctx, &repositories.ListShiftAssignmentsRequest{
		TenantInfo: p.tenantInfo,
		WorkerID:   p.workerID,
		ActiveAt:   p.endAt + secondsPerDay,
	})
	if err != nil {
		return nil, err
	}

	toClose := make([]*worker.WorkerShiftAssignment, 0, len(open))
	for _, assignment := range open {
		if assignment.EffectiveTo != nil {
			continue
		}
		if assignment.EffectiveFrom > p.endAt {
			return nil, errortypes.NewValidationError(
				"effectiveFrom",
				errortypes.ErrInvalidOperation,
				"This worker already has a shift starting on or after that date",
			)
		}
		toClose = append(toClose, assignment)
	}

	return toClose, nil
}

// AssignmentChange is an assignment before and after a change to it.
type AssignmentChange struct {
	Before *worker.WorkerShiftAssignment
	After  *worker.WorkerShiftAssignment
}

func (s *Service) PreviewEndAssignment(
	ctx context.Context,
	req *EndAssignmentRequest,
) (*AssignmentChange, error) {
	before, after, err := s.planEndAssignment(ctx, req)
	if err != nil {
		return nil, err
	}

	return &AssignmentChange{Before: before, After: after}, nil
}

func (s *Service) planEndAssignment(
	ctx context.Context,
	req *EndAssignmentRequest,
) (before, after *worker.WorkerShiftAssignment, err error) {
	assignment, err := s.repo.GetAssignmentByID(ctx, &repositories.GetShiftAssignmentByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}
	if assignment.EffectiveTo != nil {
		return nil, nil, errortypes.NewValidationError(
			"effectiveTo",
			errortypes.ErrInvalidOperation,
			"That assignment has already ended",
		)
	}

	previous := *assignment
	endAt := startOfDayUTC(req.EffectiveTo)
	assignment.EffectiveTo = &endAt

	multiErr := errortypes.NewMultiError()
	assignment.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, nil, multiErr
	}

	return &previous, assignment, nil
}

// PreferenceChange is a worker's preference for a weekday as it stands and as
// setting it would leave it; Before is nil when none is on file.
type PreferenceChange struct {
	Before *worker.WorkerAvailabilityPreference
	After  *worker.WorkerAvailabilityPreference
}

func (s *Service) PreviewSetPreference(
	ctx context.Context,
	req *SetPreferenceRequest,
) (*PreferenceChange, error) {
	entity, err := s.planSetPreference(req)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.ListPreferences(ctx, &repositories.ListAvailabilityPreferencesRequest{
		TenantInfo: req.TenantInfo,
		WorkerID:   entity.WorkerID,
	})
	if err != nil {
		return nil, err
	}

	change := &PreferenceChange{After: entity}
	for _, current := range existing {
		if current == nil || current.DayOfWeek != entity.DayOfWeek {
			continue
		}
		change.Before = current
		after := *current
		after.Preference = entity.Preference
		after.Note = entity.Note
		change.After = &after

		break
	}

	return change, nil
}

func (s *Service) planSetPreference(
	req *SetPreferenceRequest,
) (*worker.WorkerAvailabilityPreference, error) {
	entity := req.Entity
	entity.OrganizationID = req.TenantInfo.OrgID
	entity.BusinessUnitID = req.TenantInfo.BuID

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return entity, nil
}

func (s *Service) PreviewProposeSwap(
	req *ProposeSwapRequest,
) (*worker.ShiftSwapRequest, error) {
	return s.planProposeSwap(req)
}

func (s *Service) planProposeSwap(req *ProposeSwapRequest) (*worker.ShiftSwapRequest, error) {
	entity := req.Entity
	entity.OrganizationID = req.TenantInfo.OrgID
	entity.BusinessUnitID = req.TenantInfo.BuID
	entity.Status = worker.SwapProposed
	entity.ShiftDate = startOfDayUTC(entity.ShiftDate)
	if entity.CounterpartyShiftDate != nil {
		counterDate := startOfDayUTC(*entity.CounterpartyShiftDate)
		entity.CounterpartyShiftDate = &counterDate
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if entity.ShiftDate < startOfDayUTC(timeutils.NowUnix()) {
		return nil, errortypes.NewValidationError(
			"shiftDate",
			errortypes.ErrInvalid,
			"That day has already passed",
		)
	}

	return entity, nil
}

// SwapChange is a swap before and after a transition.
type SwapChange struct {
	Before *worker.ShiftSwapRequest
	After  *worker.ShiftSwapRequest
}

func (s *Service) PreviewTransitionSwap(
	ctx context.Context,
	req *TransitionSwapRequest,
) (*SwapChange, error) {
	before, after, err := s.planTransitionSwap(ctx, req)
	if err != nil {
		return nil, err
	}

	return &SwapChange{Before: before, After: after}, nil
}

func (s *Service) planTransitionSwap(
	ctx context.Context,
	req *TransitionSwapRequest,
) (before, after *worker.ShiftSwapRequest, err error) {
	swap, err := s.repo.GetSwapByID(ctx, &repositories.GetShiftSwapByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}

	if !swap.Status.CanTransitionTo(req.Status) {
		return nil, nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"A {0} swap cannot be {1}", swap.Status, req.Status,
		)
	}
	if err = authoriseSwapActor(swap, req); err != nil {
		return nil, nil, err
	}

	previous := *swap
	now := timeutils.NowUnix()
	swap.Status = req.Status
	if req.Note != "" {
		swap.ResponseNote = req.Note
	}

	switch req.Status {
	case worker.SwapAccepted, worker.SwapDeclined, worker.SwapWithdrawn:
		swap.RespondedAt = &now
	case worker.SwapApproved, worker.SwapRejected:
		swap.DecidedAt = &now
		swap.DecidedByID = req.UserID
	case worker.SwapProposed:
	}

	multiErr := errortypes.NewMultiError()
	swap.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, nil, multiErr
	}

	return &previous, swap, nil
}
