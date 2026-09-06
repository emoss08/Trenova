// Package timesheetservice owns the hours staff paid by the clock actually
// worked, the manager's sign-off on them, and the file payroll is run from.
//
// The load-bearing decision is that a timesheet's totals are frozen when it is
// submitted rather than recomputed on read. Everywhere else in this system a
// roll-up is derived, because a stored one goes stale. Here the roll-up IS the
// decision: it is what a manager approved and what payroll paid from.
// Recomputing it later would quietly change what somebody signed off, which is
// the one thing a wage record must never do.
package timesheetservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	daysInWeek       = 7
	secondsPerDay    = int64(86400)
	secondsPerMinute = int64(60)
	// workingDaysInWeek turns a week's overtime threshold into one standard
	// day, which is how a day of approved time off becomes minutes on a sheet.
	workingDaysInWeek = 5
)

// WorkerReader is the slice of the roster this service needs: who somebody is,
// so a contractor is not put on a clock they are not paid by.
type WorkerReader interface {
	GetByID(ctx context.Context, req repositories.GetWorkerByIDRequest) (*worker.Worker, error)
}

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.TimesheetRepository
	WorkerRepo   repositories.WorkerRepository
	AuditService services.AuditService
}

type Service struct {
	l            *zap.Logger
	repo         repositories.TimesheetRepository
	workerRepo   WorkerReader
	auditService services.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.timesheet"),
		repo:         p.Repo,
		workerRepo:   p.WorkerRepo,
		auditService: p.AuditService,
	}
}

// Deps is the constructor shape tests use to swap in fakes.
type Deps struct {
	Logger       *zap.Logger
	Repo         repositories.TimesheetRepository
	WorkerRepo   WorkerReader
	AuditService services.AuditService
}

func NewWithDeps(d Deps) *Service {
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		l:            logger.Named("service.timesheet"),
		repo:         d.Repo,
		workerRepo:   d.WorkerRepo,
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

// requireHourlyWorker refuses to put a contractor on a clock. A contractor
// invoices; recording their hours here would produce a wage record for
// somebody who is not owed wages, and a payroll file that says otherwise.
func (s *Service) requireHourlyWorker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) error {
	if s.workerRepo == nil {
		return nil
	}

	wrk, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:         workerID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if wrk.Type != worker.WorkerTypeEmployee {
		return errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalidOperation,
			"Only employees are paid by the clock — a contractor invoices instead",
		)
	}

	return nil
}

func (s *Service) ListEntries(
	ctx context.Context,
	req *repositories.ListTimeClockEntriesRequest,
) ([]*worker.TimeClockEntry, error) {
	return s.repo.ListEntries(ctx, req)
}

func (s *Service) ListTimesheets(
	ctx context.Context,
	req *repositories.ListTimesheetsRequest,
) ([]*worker.Timesheet, error) {
	return s.repo.ListTimesheets(ctx, req)
}

func (s *Service) GetTimesheet(
	ctx context.Context,
	req *repositories.GetTimesheetByIDRequest,
) (*worker.Timesheet, error) {
	return s.repo.GetTimesheetByID(ctx, req)
}

// OpenEntry is the punch a worker is currently on, or nil. It is what the
// clock button reads to know whether it says "in" or "out".
func (s *Service) OpenEntry(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.TimeClockEntry, error) {
	return s.repo.GetOpenEntry(ctx, tenantInfo, workerID)
}

// ClockRequest is a worker going on or coming off the clock.
type ClockRequest struct {
	WorkerID     pulid.ID
	At           int64
	Source       worker.TimeEntrySource
	BreakMinutes int32
	PayCodeID    pulid.ID
	Note         string
	TenantInfo   pagination.TenantInfo
	UserID       pulid.ID
}

// ClockIn puts a worker on the clock. Being on it twice is refused rather than
// deduplicated: two open punches is somebody being paid twice for the same hour,
// and the database will not hold them either.
func (s *Service) ClockIn(
	ctx context.Context,
	req *ClockRequest,
) (*worker.TimeClockEntry, error) {
	if err := s.requireHourlyWorker(ctx, req.TenantInfo, req.WorkerID); err != nil {
		return nil, err
	}

	open, err := s.repo.GetOpenEntry(ctx, req.TenantInfo, req.WorkerID)
	if err != nil {
		return nil, err
	}
	if open != nil {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalidOperation,
			"Already on the clock — clock out first",
		)
	}

	at := req.At
	if at <= 0 {
		at = timeutils.NowUnix()
	}
	// A punch dated into the future would sit on a week nobody can approve yet
	// and would grow on its own until somebody closed it.
	if at > timeutils.NowUnix()+secondsPerMinute {
		return nil, errortypes.NewValidationError(
			"clockedInAt",
			errortypes.ErrInvalid,
			"A punch cannot be dated in the future",
		)
	}

	sheet, err := s.openSheetFor(ctx, req.TenantInfo, req.WorkerID, at)
	if err != nil {
		return nil, err
	}
	if !sheet.Status.IsEditable() {
		return nil, errortypes.NewValidationError(
			"clockedInAt",
			errortypes.ErrInvalidOperation,
			fmt.Sprintf("That week has already been %s", sheet.Status),
		)
	}

	source := req.Source
	if source == "" {
		source = worker.TimeEntrySourceClock
	}

	entity := &worker.TimeClockEntry{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		WorkerID:       req.WorkerID,
		TimesheetID:    sheet.ID,
		Source:         source,
		ClockedInAt:    at,
		PayCodeID:      req.PayCodeID,
		Note:           req.Note,
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.repo.CreateEntry(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(auditParams{
		resource:   permission.ResourceTimesheet,
		resourceID: created.ID.String(),
		operation:  permission.OpCreate,
		userID:     req.UserID,
		tenantInfo: req.TenantInfo,
		current:    created,
		comment:    "Clocked in",
	})

	return created, nil
}

// ClockOut closes the punch the worker is on and rolls the week up again.
func (s *Service) ClockOut(
	ctx context.Context,
	req *ClockRequest,
) (*worker.TimeClockEntry, error) {
	open, err := s.repo.GetOpenEntry(ctx, req.TenantInfo, req.WorkerID)
	if err != nil {
		return nil, err
	}
	if open == nil {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalidOperation,
			"Not on the clock",
		)
	}

	at := req.At
	if at <= 0 {
		at = timeutils.NowUnix()
	}

	previous := *open
	open.ClockedOutAt = &at
	open.BreakMinutes = req.BreakMinutes
	if req.Note != "" {
		open.Note = req.Note
	}

	multiErr := errortypes.NewMultiError()
	open.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updated, err := s.repo.UpdateEntry(ctx, open)
	if err != nil {
		return nil, err
	}

	if err = s.recalculate(ctx, req.TenantInfo, updated.TimesheetID); err != nil {
		return nil, err
	}

	s.audit(auditParams{
		resource:   permission.ResourceTimesheet,
		resourceID: updated.ID.String(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenantInfo: req.TenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    "Clocked out",
	})

	return updated, nil
}

// RecordEntryRequest is a period somebody is adding or correcting by hand.
type RecordEntryRequest struct {
	ID           pulid.ID
	WorkerID     pulid.ID
	ClockedInAt  int64
	ClockedOutAt int64
	BreakMinutes int32
	PayCodeID    pulid.ID
	Note         string
	// Reason is why the record was changed. A wage record altered by somebody
	// with no reason recorded is not a record anybody can defend.
	Reason     string
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
}

// RecordEntry adds or corrects a period by hand. It is refused once the week
// has been handed over: the totals a manager is looking at have to be the ones
// they are being asked to sign.
func (s *Service) RecordEntry(
	ctx context.Context,
	req *RecordEntryRequest,
) (*worker.TimeClockEntry, error) {
	if req.Reason == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Changing somebody's hours needs a reason",
		)
	}

	var (
		entity   *worker.TimeClockEntry
		previous *worker.TimeClockEntry
	)

	if req.ID.IsNil() {
		if err := s.requireHourlyWorker(ctx, req.TenantInfo, req.WorkerID); err != nil {
			return nil, err
		}
		entity = &worker.TimeClockEntry{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			WorkerID:       req.WorkerID,
			Source:         worker.TimeEntrySourceManual,
		}
	} else {
		existing, err := s.repo.GetEntryByID(ctx, &repositories.GetTimeClockEntryByIDRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}
		snapshot := *existing
		previous = &snapshot
		entity = existing
	}

	// A corrected entry may move weeks. The week it is leaving has to be as
	// open as the one it is joining, or a submitted total would lose a punch
	// out from under the manager reading it.
	if previous != nil && !previous.TimesheetID.IsNil() {
		if err := s.requireEditableSheet(ctx, req.TenantInfo, previous.TimesheetID); err != nil {
			return nil, err
		}
	}

	sheet, err := s.openSheetFor(ctx, req.TenantInfo, entity.WorkerID, req.ClockedInAt)
	if err != nil {
		return nil, err
	}
	if !sheet.Status.IsEditable() {
		return nil, errortypes.NewValidationError(
			"clockedInAt",
			errortypes.ErrInvalidOperation,
			fmt.Sprintf("That week has already been %s", sheet.Status),
		)
	}

	closedAt := req.ClockedOutAt
	entity.TimesheetID = sheet.ID
	entity.ClockedInAt = req.ClockedInAt
	entity.ClockedOutAt = &closedAt
	entity.BreakMinutes = req.BreakMinutes
	entity.PayCodeID = req.PayCodeID
	entity.Note = req.Note
	entity.EditedByID = req.UserID
	entity.EditReason = req.Reason

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	var saved *worker.TimeClockEntry
	if req.ID.IsNil() {
		saved, err = s.repo.CreateEntry(ctx, entity)
	} else {
		saved, err = s.repo.UpdateEntry(ctx, entity)
	}
	if err != nil {
		return nil, err
	}

	if err = s.recalculate(ctx, req.TenantInfo, sheet.ID); err != nil {
		return nil, err
	}
	if previous != nil && !previous.TimesheetID.IsNil() && previous.TimesheetID != sheet.ID {
		if err = s.recalculate(ctx, req.TenantInfo, previous.TimesheetID); err != nil {
			return nil, err
		}
	}

	operation := permission.OpCreate
	if !req.ID.IsNil() {
		operation = permission.OpUpdate
	}
	s.audit(auditParams{
		resource:   permission.ResourceTimesheet,
		resourceID: saved.ID.String(),
		operation:  operation,
		userID:     req.UserID,
		tenantInfo: req.TenantInfo,
		current:    saved,
		previous:   previous,
		comment:    req.Reason,
	})

	return saved, nil
}

// DeleteEntryRequest removes a period that should never have been recorded.
type DeleteEntryRequest struct {
	ID         pulid.ID
	Reason     string
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
}

func (s *Service) DeleteEntry(ctx context.Context, req *DeleteEntryRequest) error {
	if req.Reason == "" {
		return errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Removing somebody's hours needs a reason",
		)
	}

	entry, err := s.repo.GetEntryByID(ctx, &repositories.GetTimeClockEntryByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}

	if !entry.TimesheetID.IsNil() {
		if err = s.requireEditableSheet(ctx, req.TenantInfo, entry.TimesheetID); err != nil {
			return err
		}
	}

	if err = s.repo.DeleteEntry(ctx, &repositories.GetTimeClockEntryByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return err
	}

	if !entry.TimesheetID.IsNil() {
		if err = s.recalculate(ctx, req.TenantInfo, entry.TimesheetID); err != nil {
			return err
		}
	}

	s.audit(auditParams{
		resource:   permission.ResourceTimesheet,
		resourceID: req.ID.String(),
		operation:  permission.OpDelete,
		userID:     req.UserID,
		tenantInfo: req.TenantInfo,
		current:    entry,
		comment:    req.Reason,
	})

	return nil
}

func (s *Service) requireEditableSheet(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	timesheetID pulid.ID,
) error {
	sheet, err := s.repo.GetTimesheetByID(ctx, &repositories.GetTimesheetByIDRequest{
		ID:         timesheetID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if !sheet.Status.IsEditable() {
		return errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalidOperation,
			fmt.Sprintf("That week has already been %s", sheet.Status),
		)
	}
	return nil
}

// openSheetFor is the week a moment belongs to, opening it if nobody has yet.
// The week starts on the same Sunday the rota starts on, so a timesheet and a
// rota row line up without anybody converting between them.
func (s *Service) openSheetFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	at int64,
) (*worker.Timesheet, error) {
	periodStart := worker.StartOfWeekUTC(at, time.UTC)

	return s.repo.GetOrCreateTimesheet(ctx, &worker.Timesheet{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		WorkerID:       workerID,
		Status:         worker.TimesheetOpen,
		PeriodStart:    periodStart,
		PeriodEnd:      periodStart + daysInWeek*secondsPerDay,
	})
}

// recalculate re-rolls an open week from its entries. It does nothing once the
// week has been submitted: from there the totals are the record of what was
// approved, and moving them would change a signed-off wage record.
func (s *Service) recalculate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	timesheetID pulid.ID,
) error {
	if timesheetID.IsNil() {
		return nil
	}

	sheet, err := s.repo.GetTimesheetByID(ctx, &repositories.GetTimesheetByIDRequest{
		ID:         timesheetID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if !sheet.Status.IsEditable() {
		return nil
	}

	if _, err = s.repo.AttachEntries(
		ctx,
		tenantInfo,
		sheet.ID,
		sheet.WorkerID,
		sheet.PeriodStart,
		sheet.PeriodEnd,
	); err != nil {
		return err
	}

	entries, err := s.repo.ListEntries(ctx, &repositories.ListTimeClockEntriesRequest{
		TenantInfo:  tenantInfo,
		TimesheetID: sheet.ID,
	})
	if err != nil {
		return err
	}

	var worked int32
	for _, entry := range entries {
		worked += entry.PaidMinutes()
	}

	paidLeave, err := s.paidLeaveMinutes(ctx, tenantInfo, sheet)
	if err != nil {
		return err
	}

	sheet.ApplyTotals(worked, paidLeave, int32(len(entries))) //nolint:gosec // a week of punches
	_, err = s.repo.UpdateTimesheet(ctx, sheet)

	return err
}

// paidLeaveMinutes is the approved time off falling inside the week, converted
// at one standard day per day off. It is read from the time-off record rather
// than typed onto the sheet, so a week cannot say somebody was here when the
// same system says they were signed off.
func (s *Service) paidLeaveMinutes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sheet *worker.Timesheet,
) (int32, error) {
	dayMinutes := sheet.OvertimeThresholdMinutes / workingDaysInWeek
	if dayMinutes <= 0 {
		return 0, nil
	}

	ranges, err := s.repo.ApprovedLeaveRanges(
		ctx,
		tenantInfo,
		sheet.WorkerID,
		sheet.PeriodStart,
		sheet.PeriodEnd,
	)
	if err != nil {
		return 0, err
	}

	var days int32
	for _, span := range ranges {
		days += overlapDays(span.StartsAt, span.EndsAt, sheet.PeriodStart, sheet.PeriodEnd)
	}

	return days * dayMinutes, nil
}

// overlapDays is how many whole days of a span fall inside a window. Time off
// is stored inclusively — a single day off has the same start and end — so the
// last day counts.
func overlapDays(start, end, windowStart, windowEnd int64) int32 {
	from := max(startOfDayUTC(start), windowStart)
	to := min(startOfDayUTC(end), windowEnd-secondsPerDay)
	if to < from {
		return 0
	}

	return int32((to-from)/secondsPerDay) + 1 //nolint:gosec // at most a week
}

func startOfDayUTC(at int64) int64 {
	if at <= 0 {
		return at
	}
	t := time.Unix(at, 0).UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix()
}
