// Package schedulingservice owns the working pattern: who is expected on which
// days, what they would rather work, and the swaps they arrange between
// themselves.
//
// The load-bearing decision is that the rota is derived on every read and
// never stored. A stored week would be wrong within the hour — time off gets
// approved, a leave case opens, dispatch puts a load on a Saturday — and the
// only way to keep it right would be to invalidate it from five other
// services. Composing it from the evidence instead means the board is correct
// by construction, and the cost is four grouped queries a week rather than a
// walk of the roster.
package schedulingservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const secondsPerDay = int64(86400)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.SchedulingRepository
	AuditService services.AuditService
}

type Service struct {
	l            *zap.Logger
	repo         repositories.SchedulingRepository
	auditService services.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.scheduling"),
		repo:         p.Repo,
		auditService: p.AuditService,
	}
}

// Deps is the constructor shape tests use to swap in fakes.
type Deps struct {
	Logger       *zap.Logger
	Repo         repositories.SchedulingRepository
	AuditService services.AuditService
}

func NewWithDeps(d Deps) *Service {
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		l:            logger.Named("service.scheduling"),
		repo:         d.Repo,
		auditService: d.AuditService,
	}
}

type auditParams struct {
	resource   permission.Resource
	resourceID string
	operation  permission.Operation
	userID     pulid.ID
	tenantInfo pagination.TenantInfo
	current    any
	previous   any
	comment    string
}

func (s *Service) audit(p auditParams) {
	if s.auditService == nil || p.userID.IsNil() {
		return
	}
	params := &services.LogActionParams{
		Resource:       p.resource,
		ResourceID:     p.resourceID,
		Operation:      p.operation,
		UserID:         p.userID,
		CurrentState:   jsonutils.MustToJSON(p.current),
		OrganizationID: p.tenantInfo.OrgID,
		BusinessUnitID: p.tenantInfo.BuID,
	}
	opts := []services.LogOption{auditservice.WithComment(p.comment)}
	if p.previous != nil {
		params.PreviousState = jsonutils.MustToJSON(p.previous)
		opts = append(opts, auditservice.WithDiff(p.previous, p.current))
	}
	if err := s.auditService.LogAction(params, opts...); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}
}

func (s *Service) ListTemplates(
	ctx context.Context,
	req *repositories.ListShiftTemplatesRequest,
) ([]*worker.ShiftTemplate, error) {
	return s.repo.ListTemplates(ctx, req)
}

func (s *Service) GetTemplate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.ShiftTemplate, error) {
	return s.repo.GetTemplateByID(ctx, &repositories.GetShiftTemplateByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
}

// CreateTemplateRequest is a new working pattern.
type CreateTemplateRequest struct {
	Entity     *worker.ShiftTemplate
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
}

func (s *Service) CreateTemplate(
	ctx context.Context,
	req *CreateTemplateRequest,
) (*worker.ShiftTemplate, error) {
	entity := req.Entity
	entity.OrganizationID = req.TenantInfo.OrgID
	entity.BusinessUnitID = req.TenantInfo.BuID
	entity.Normalise()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.repo.CreateTemplate(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(auditParams{
		resource:   permission.ResourceShiftTemplate,
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     req.UserID,
		tenantInfo: req.TenantInfo,
		current:    created,
		comment:    "Shift created",
	})

	return created, nil
}

// UpdateTemplateRequest edits a working pattern.
type UpdateTemplateRequest struct {
	Entity     *worker.ShiftTemplate
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
}

func (s *Service) UpdateTemplate(
	ctx context.Context,
	req *UpdateTemplateRequest,
) (*worker.ShiftTemplate, error) {
	entity := req.Entity
	entity.OrganizationID = req.TenantInfo.OrgID
	entity.BusinessUnitID = req.TenantInfo.BuID
	entity.Normalise()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	original, err := s.repo.GetTemplateByID(ctx, &repositories.GetShiftTemplateByIDRequest{
		ID:         entity.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	// Retiring a pattern people are still on would empty their rota rather
	// than move them, so the assignments have to be ended first.
	if original.Status == domaintypes.StatusActive && entity.Status == domaintypes.StatusInactive {
		count, cErr := s.repo.CountTemplateAssignments(ctx, req.TenantInfo, entity.ID)
		if cErr != nil {
			return nil, cErr
		}
		if count > 0 {
			return nil, errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalidOperation,
				fmt.Sprintf(
					"%d worker(s) are still on this shift — move or end their assignments first",
					count,
				),
			)
		}
	}

	updated, err := s.repo.UpdateTemplate(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(auditParams{
		resource:   permission.ResourceShiftTemplate,
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenantInfo: req.TenantInfo,
		current:    updated,
		previous:   original,
		comment:    "Shift updated",
	})

	return updated, nil
}

func (s *Service) ListAssignments(
	ctx context.Context,
	req *repositories.ListShiftAssignmentsRequest,
) ([]*worker.WorkerShiftAssignment, error) {
	return s.repo.ListAssignments(ctx, req)
}

func (s *Service) GetAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.WorkerShiftAssignment, error) {
	return s.repo.GetAssignmentByID(ctx, &repositories.GetShiftAssignmentByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
}

func (s *Service) CountTemplateAssignments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	templateID pulid.ID,
) (int, error) {
	return s.repo.CountTemplateAssignments(ctx, tenantInfo, templateID)
}

func (s *Service) CountTemplateAssignmentsByIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	templateIDs []pulid.ID,
) (map[pulid.ID]int, error) {
	return s.repo.CountTemplateAssignmentsByIDs(
		ctx,
		&repositories.CountShiftTemplateAssignmentsRequest{
			TenantInfo:  tenantInfo,
			TemplateIDs: templateIDs,
		},
	)
}

// AssignShiftRequest puts a worker on a pattern from a date.
type AssignShiftRequest struct {
	Entity     *worker.WorkerShiftAssignment
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
}

// AssignShift moves a worker onto a pattern. The assignment in force is ended
// the day before rather than left open: two open assignments would put a
// worker on two patterns at once and the rota would silently pick one.
func (s *Service) AssignShift(
	ctx context.Context,
	req *AssignShiftRequest,
) (*worker.WorkerShiftAssignment, error) {
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
	// An offset beyond the cycle wraps to a week the pattern never reaches, so
	// it is a typo rather than a rotation.
	if entity.CycleOffsetWeeks >= template.CycleWeeks && template.CycleWeeks > 1 {
		return nil, errortypes.NewValidationError(
			"cycleOffsetWeeks",
			errortypes.ErrInvalid,
			fmt.Sprintf("This shift rotates over %d weeks, so the offset is 0 to %d",
				template.CycleWeeks, template.CycleWeeks-1),
		)
	}

	if err = s.closeOpenAssignment(ctx, closeAssignmentParams{
		tenantInfo: req.TenantInfo,
		workerID:   entity.WorkerID,
		endAt:      entity.EffectiveFrom - secondsPerDay,
		userID:     req.UserID,
	}); err != nil {
		return nil, err
	}

	created, err := s.repo.CreateAssignment(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(auditParams{
		resource:   permission.ResourceWorkerSchedule,
		resourceID: created.ID.String(),
		operation:  permission.OpAssign,
		userID:     req.UserID,
		tenantInfo: req.TenantInfo,
		current:    created,
		comment:    "Worker put on a shift",
	})

	return created, nil
}

type closeAssignmentParams struct {
	tenantInfo pagination.TenantInfo
	workerID   pulid.ID
	endAt      int64
	userID     pulid.ID
}

func (s *Service) closeOpenAssignment(ctx context.Context, p closeAssignmentParams) error {
	open, err := s.repo.ListAssignments(ctx, &repositories.ListShiftAssignmentsRequest{
		TenantInfo: p.tenantInfo,
		WorkerID:   p.workerID,
		ActiveAt:   p.endAt + secondsPerDay,
	})
	if err != nil {
		return err
	}

	for _, assignment := range open {
		if assignment.EffectiveTo != nil {
			continue
		}
		// An assignment that never took effect is replaced rather than ended
		// on a date before it began, which would not validate.
		if assignment.EffectiveFrom > p.endAt {
			return errortypes.NewValidationError(
				"effectiveFrom",
				errortypes.ErrInvalidOperation,
				"This worker already has a shift starting on or after that date",
			)
		}
		previous := *assignment
		endAt := p.endAt
		assignment.EffectiveTo = &endAt
		if _, err = s.repo.UpdateAssignment(ctx, assignment); err != nil {
			return err
		}
		s.audit(auditParams{
			resource:   permission.ResourceWorkerSchedule,
			resourceID: assignment.ID.String(),
			operation:  permission.OpUnassign,
			userID:     p.userID,
			tenantInfo: p.tenantInfo,
			current:    assignment,
			previous:   &previous,
			comment:    "Shift assignment ended",
		})
	}

	return nil
}

// EndAssignmentRequest takes a worker off a pattern from a date.
type EndAssignmentRequest struct {
	ID          pulid.ID
	EffectiveTo int64
	TenantInfo  pagination.TenantInfo
	UserID      pulid.ID
}

func (s *Service) EndAssignment(
	ctx context.Context,
	req *EndAssignmentRequest,
) (*worker.WorkerShiftAssignment, error) {
	assignment, err := s.repo.GetAssignmentByID(ctx, &repositories.GetShiftAssignmentByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if assignment.EffectiveTo != nil {
		return nil, errortypes.NewValidationError(
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
		return nil, multiErr
	}

	updated, err := s.repo.UpdateAssignment(ctx, assignment)
	if err != nil {
		return nil, err
	}

	s.audit(auditParams{
		resource:   permission.ResourceWorkerSchedule,
		resourceID: updated.ID.String(),
		operation:  permission.OpUnassign,
		userID:     req.UserID,
		tenantInfo: req.TenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    "Shift assignment ended",
	})

	return updated, nil
}

func (s *Service) ListPreferences(
	ctx context.Context,
	req *repositories.ListAvailabilityPreferencesRequest,
) ([]*worker.WorkerAvailabilityPreference, error) {
	return s.repo.ListPreferences(ctx, req)
}

// SetPreferenceRequest states what a worker would rather work on one weekday.
type SetPreferenceRequest struct {
	Entity     *worker.WorkerAvailabilityPreference
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
}

func (s *Service) SetPreference(
	ctx context.Context,
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

	saved, err := s.repo.UpsertPreference(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(auditParams{
		resource:   permission.ResourceWorkerSchedule,
		resourceID: saved.ID.String(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenantInfo: req.TenantInfo,
		current:    saved,
		comment:    "Availability preference set",
	})

	return saved, nil
}

func (s *Service) ListSwaps(
	ctx context.Context,
	req *repositories.ListShiftSwapsRequest,
) ([]*worker.ShiftSwapRequest, error) {
	return s.repo.ListSwaps(ctx, req)
}

func (s *Service) GetSwap(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.ShiftSwapRequest, error) {
	return s.repo.GetSwapByID(ctx, &repositories.GetShiftSwapByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
}

// ProposeSwapRequest is one driver asking another to take a day.
type ProposeSwapRequest struct {
	Entity     *worker.ShiftSwapRequest
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
}

func (s *Service) ProposeSwap(
	ctx context.Context,
	req *ProposeSwapRequest,
) (*worker.ShiftSwapRequest, error) {
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

	// A swap for a day that has already been worked cannot change who worked
	// it, and would let the board be rewritten after the fact.
	if entity.ShiftDate < startOfDayUTC(timeutils.NowUnix()) {
		return nil, errortypes.NewValidationError(
			"shiftDate",
			errortypes.ErrInvalid,
			"That day has already passed",
		)
	}

	created, err := s.repo.CreateSwap(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(auditParams{
		resource:   permission.ResourceShiftSwap,
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     req.UserID,
		tenantInfo: req.TenantInfo,
		current:    created,
		comment:    "Shift swap proposed",
	})

	return created, nil
}

// TransitionSwapRequest moves a swap along. ActorWorkerID is set when the
// answer comes from a driver rather than the office: the counterparty is the
// only person who can accept or decline, and the requester is the only person
// who can withdraw.
type TransitionSwapRequest struct {
	ID            pulid.ID
	Status        worker.ShiftSwapStatus
	Note          string
	ActorWorkerID pulid.ID
	TenantInfo    pagination.TenantInfo
	UserID        pulid.ID
}

// TransitionSwap is the one path a swap changes state by. Every caller — the
// driver portal, the manager queue, a withdrawal — goes through it so the
// state machine is enforced in one place rather than four.
func (s *Service) TransitionSwap(
	ctx context.Context,
	req *TransitionSwapRequest,
) (*worker.ShiftSwapRequest, error) {
	swap, err := s.repo.GetSwapByID(ctx, &repositories.GetShiftSwapByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if !swap.Status.CanTransitionTo(req.Status) {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			fmt.Sprintf("A %s swap cannot be %s", swap.Status, req.Status),
		)
	}
	if err = authoriseSwapActor(swap, req); err != nil {
		return nil, err
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
		return nil, multiErr
	}

	updated, err := s.repo.UpdateSwap(ctx, swap)
	if err != nil {
		return nil, err
	}

	s.audit(auditParams{
		resource:   permission.ResourceShiftSwap,
		resourceID: updated.ID.String(),
		operation:  swapOperation(req.Status),
		userID:     req.UserID,
		tenantInfo: req.TenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    fmt.Sprintf("Shift swap %s", req.Status),
	})

	return updated, nil
}

// authoriseSwapActor keeps a driver from answering on somebody else's behalf.
// The office is not checked here — a manager reaching this code has already
// been through the permission gate for the decision.
func authoriseSwapActor(swap *worker.ShiftSwapRequest, req *TransitionSwapRequest) error {
	if req.ActorWorkerID.IsNil() {
		return nil
	}

	switch req.Status {
	case worker.SwapAccepted, worker.SwapDeclined:
		if swap.CounterpartyWorkerID != req.ActorWorkerID {
			return errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalidOperation,
				"Only the driver a swap was offered to can answer it",
			)
		}
	case worker.SwapWithdrawn:
		if swap.RequestingWorkerID != req.ActorWorkerID {
			return errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalidOperation,
				"Only the driver who proposed a swap can withdraw it",
			)
		}
	case worker.SwapApproved, worker.SwapRejected:
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"A swap is decided by the office, not by a driver",
		)
	case worker.SwapProposed:
	}

	return nil
}

func swapOperation(status worker.ShiftSwapStatus) permission.Operation {
	switch status {
	case worker.SwapApproved:
		return permission.OpApprove
	case worker.SwapRejected, worker.SwapDeclined:
		return permission.OpReject
	case worker.SwapWithdrawn:
		return permission.OpCancel
	case worker.SwapAccepted, worker.SwapProposed:
		return permission.OpUpdate
	default:
		return permission.OpUpdate
	}
}

func startOfDayUTC(at int64) int64 {
	if at <= 0 {
		return at
	}
	t := time.Unix(at, 0).UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix()
}
