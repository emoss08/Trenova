package timesheetservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// TransitionTimesheetRequest moves a week along.
type TransitionTimesheetRequest struct {
	ID     pulid.ID
	Status worker.TimesheetStatus
	Note   string
	// ActorWorkerID is set when the request comes from the worker whose week it
	// is: they may hand their own week over and nothing else.
	ActorWorkerID pulid.ID
	TenantInfo    pagination.TenantInfo
	UserID        pulid.ID
}

// Transition is the one path a week changes state by. Submitting freezes the
// totals — everything after that reads the frozen numbers, because they are
// what somebody was asked to sign.
func (s *Service) Transition(
	ctx context.Context,
	req *TransitionTimesheetRequest,
) (*worker.Timesheet, error) {
	sheet, err := s.repo.GetTimesheetByID(ctx, &repositories.GetTimesheetByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if !sheet.Status.CanTransitionTo(req.Status) {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"A {0} timesheet cannot be {1}", sheet.Status, req.Status,
		)
	}
	// A worker hands their own week over and nothing else. Letting them
	// approve it would make the sign-off meaningless.
	if !req.ActorWorkerID.IsNil() {
		if sheet.WorkerID != req.ActorWorkerID {
			return nil, errortypes.NewValidationError(
				"id",
				errortypes.ErrInvalidOperation,
				"That is not your timesheet",
			)
		}
		if req.Status != worker.TimesheetSubmitted {
			return nil, errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalidOperation,
				"A timesheet is approved by a manager, not by the person who worked it",
			)
		}
	}

	// The last roll-up before the numbers are frozen. After this the totals
	// are the record rather than a summary of one.
	if req.Status == worker.TimesheetSubmitted {
		if err = s.recalculate(ctx, req.TenantInfo, sheet.ID); err != nil {
			return nil, err
		}
		if sheet, err = s.repo.GetTimesheetByID(ctx, &repositories.GetTimesheetByIDRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		}); err != nil {
			return nil, err
		}
		// An empty week is a week nobody worked, and approving one would put a
		// zero-hour line in the payroll file for somebody who was simply away.
		if sheet.TotalMinutes() == 0 {
			return nil, errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalidOperation,
				"There are no hours on this week yet",
			)
		}
	}

	previous := *sheet
	now := timeutils.NowUnix()
	sheet.Status = req.Status
	if req.Note != "" {
		sheet.DecisionNote = req.Note
	}

	switch req.Status {
	case worker.TimesheetSubmitted:
		sheet.SubmittedAt = &now
		sheet.SubmittedByID = req.UserID
		// A resubmission is a fresh decision, so the old one is cleared rather
		// than left to read as an approval of numbers that have since changed.
		sheet.ApprovedAt = nil
		sheet.ApprovedByID = pulid.Nil
	case worker.TimesheetApproved:
		sheet.ApprovedAt = &now
		sheet.ApprovedByID = req.UserID
	case worker.TimesheetRejected, worker.TimesheetOpen:
		sheet.ApprovedAt = nil
		sheet.ApprovedByID = pulid.Nil
	case worker.TimesheetLocked:
	}

	multiErr := errortypes.NewMultiError()
	sheet.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updated, err := s.repo.UpdateTimesheet(ctx, sheet)
	if err != nil {
		return nil, err
	}

	s.audit(auditParams{
		resource:   permission.ResourceTimesheet,
		resourceID: updated.ID.String(),
		operation:  timesheetOperation(req.Status),
		userID:     req.UserID,
		tenantInfo: req.TenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    fmt.Sprintf("Timesheet %s", req.Status),
	})

	return updated, nil
}

func timesheetOperation(status worker.TimesheetStatus) permission.Operation {
	switch status {
	case worker.TimesheetSubmitted:
		return permission.OpSubmit
	case worker.TimesheetApproved:
		return permission.OpApprove
	case worker.TimesheetRejected:
		return permission.OpReject
	case worker.TimesheetLocked:
		return permission.OpLock
	case worker.TimesheetOpen:
		return permission.OpReopen
	default:
		return permission.OpUpdate
	}
}

// PayrollExportRow is one line of the file payroll is run from: a worker, and
// the hours they are owed for, split the way the sheet was approved.
type PayrollExportRow struct {
	WorkerID         pulid.ID `json:"workerId"`
	WorkerName       string   `json:"workerName"`
	PeriodStart      int64    `json:"periodStart"`
	PeriodEnd        int64    `json:"periodEnd"`
	RegularMinutes   int32    `json:"regularMinutes"`
	OvertimeMinutes  int32    `json:"overtimeMinutes"`
	PaidLeaveMinutes int32    `json:"paidLeaveMinutes"`
}

// GenerateExportRequest is a payroll run over one period.
type GenerateExportRequest struct {
	PeriodStart int64
	PeriodEnd   int64
	Note        string
	TenantInfo  pagination.TenantInfo
	UserID      pulid.ID
}

// GenerateExport collects every approved week in a period that has not gone to
// payroll yet, records the run, and locks the sheets to it in one statement.
//
// The sheets are stamped rather than merely listed: a period that could be
// exported twice is a period somebody gets paid twice for, and nothing in a
// payroll system reads worse in an audit.
func (s *Service) GenerateExport(
	ctx context.Context,
	req *GenerateExportRequest,
) (*worker.PayrollExport, error) {
	sheets, err := s.repo.ListTimesheets(ctx, &repositories.ListTimesheetsRequest{
		TenantInfo:     req.TenantInfo,
		Statuses:       []worker.TimesheetStatus{worker.TimesheetApproved},
		From:           req.PeriodStart,
		To:             req.PeriodEnd,
		UnexportedOnly: true,
	})
	if err != nil {
		return nil, err
	}
	if len(sheets) == 0 {
		return nil, errortypes.NewValidationError(
			"periodStart",
			errortypes.ErrInvalidOperation,
			"There are no approved timesheets waiting in that period",
		)
	}

	now := timeutils.NowUnix()
	export := &worker.PayrollExport{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Status:         worker.PayrollExportGenerated,
		PeriodStart:    req.PeriodStart,
		PeriodEnd:      req.PeriodEnd,
		GeneratedAt:    &now,
		GeneratedByID:  req.UserID,
		Note:           req.Note,
	}

	ids := make([]pulid.ID, 0, len(sheets))
	for _, sheet := range sheets {
		ids = append(ids, sheet.ID)
		export.RegularMinutes += sheet.RegularMinutes
		export.OvertimeMinutes += sheet.OvertimeMinutes
		export.PaidLeaveMinutes += sheet.PaidLeaveMinutes
	}
	export.TimesheetCount = int32(len(sheets)) //nolint:gosec // a period of weeks

	multiErr := errortypes.NewMultiError()
	export.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.repo.CreateExport(ctx, export)
	if err != nil {
		return nil, err
	}

	stamped, err := s.repo.StampExport(ctx, &repositories.StampExportRequest{
		TenantInfo:   req.TenantInfo,
		ExportID:     created.ID,
		TimesheetIDs: ids,
		Status:       worker.TimesheetLocked,
	})
	if err != nil {
		return nil, err
	}
	// A sheet somebody reopened between the listing and the stamp is skipped
	// by the statement rather than locked out from under them, so the run's
	// totals are corrected to what it actually carried.
	if stamped != len(sheets) {
		if created, err = s.reconcileExport(ctx, req.TenantInfo, created); err != nil {
			return nil, err
		}
	}

	s.audit(auditParams{
		resource:   permission.ResourceTimesheet,
		resourceID: created.ID.String(),
		operation:  permission.OpExport,
		userID:     req.UserID,
		tenantInfo: req.TenantInfo,
		current:    created,
		comment:    fmt.Sprintf("Payroll export of %d timesheet(s)", created.TimesheetCount),
	})

	return created, nil
}

// reconcileExport re-totals a run from the sheets it actually locked.
func (s *Service) reconcileExport(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	export *worker.PayrollExport,
) (*worker.PayrollExport, error) {
	sheets, err := s.repo.ListTimesheets(ctx, &repositories.ListTimesheetsRequest{
		TenantInfo: tenantInfo,
		ExportID:   export.ID,
	})
	if err != nil {
		return nil, err
	}

	export.RegularMinutes = 0
	export.OvertimeMinutes = 0
	export.PaidLeaveMinutes = 0
	for _, sheet := range sheets {
		export.RegularMinutes += sheet.RegularMinutes
		export.OvertimeMinutes += sheet.OvertimeMinutes
		export.PaidLeaveMinutes += sheet.PaidLeaveMinutes
	}
	export.TimesheetCount = int32(len(sheets)) //nolint:gosec // a period of weeks

	return s.repo.UpdateExport(ctx, export)
}

// ExportRows is the file's contents: one line per week the run carried.
func (s *Service) ExportRows(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	exportID pulid.ID,
) ([]*PayrollExportRow, error) {
	sheets, err := s.repo.ListTimesheets(ctx, &repositories.ListTimesheetsRequest{
		TenantInfo:    tenantInfo,
		ExportID:      exportID,
		IncludeWorker: true,
	})
	if err != nil {
		return nil, err
	}

	rows := make([]*PayrollExportRow, 0, len(sheets))
	for _, sheet := range sheets {
		row := &PayrollExportRow{
			WorkerID:         sheet.WorkerID,
			PeriodStart:      sheet.PeriodStart,
			PeriodEnd:        sheet.PeriodEnd,
			RegularMinutes:   sheet.RegularMinutes,
			OvertimeMinutes:  sheet.OvertimeMinutes,
			PaidLeaveMinutes: sheet.PaidLeaveMinutes,
		}
		if sheet.Worker != nil {
			row.WorkerName = sheet.Worker.FirstName + " " + sheet.Worker.LastName
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func (s *Service) ListExports(
	ctx context.Context,
	req *repositories.ListPayrollExportsRequest,
) ([]*worker.PayrollExport, error) {
	return s.repo.ListExports(ctx, req)
}

func (s *Service) GetExport(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.PayrollExport, error) {
	return s.repo.GetExportByID(ctx, &repositories.GetPayrollExportByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
}

// VoidExportRequest takes a payroll run back.
type VoidExportRequest struct {
	ID         pulid.ID
	Reason     string
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
}

// VoidExport takes a run back and puts every week in it back to Approved, so
// the period can be exported again once whatever was wrong is fixed. The run
// itself is kept rather than deleted: a payroll file that went out and was
// pulled back is a fact somebody will need to explain.
func (s *Service) VoidExport(
	ctx context.Context,
	req *VoidExportRequest,
) (*worker.PayrollExport, error) {
	export, err := s.repo.GetExportByID(ctx, &repositories.GetPayrollExportByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if export.Status == worker.PayrollExportVoided {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"That run has already been voided",
		)
	}

	previous := *export
	now := timeutils.NowUnix()
	export.Status = worker.PayrollExportVoided
	export.VoidedAt = &now
	export.VoidReason = req.Reason

	multiErr := errortypes.NewMultiError()
	export.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if _, err = s.repo.ClearExport(ctx, req.TenantInfo, export.ID); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateExport(ctx, export)
	if err != nil {
		return nil, err
	}

	s.audit(auditParams{
		resource:   permission.ResourceTimesheet,
		resourceID: updated.ID.String(),
		operation:  permission.OpCancel,
		userID:     req.UserID,
		tenantInfo: req.TenantInfo,
		current:    updated,
		previous:   &previous,
		comment:    req.Reason,
	})

	return updated, nil
}
