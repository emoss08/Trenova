package timesheetservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type ExportChange = services.RecordChange[worker.PayrollExport]

// ExportPlan is the payroll run GenerateExport would make and the approved
// weeks it would lock into it.
type ExportPlan struct {
	Export       *worker.PayrollExport
	TimesheetIDs []pulid.ID
}

// PlanGenerateExport totals the approved weeks in the period that have not
// gone to payroll, as GenerateExport would, locking nothing.
func (s *Service) PlanGenerateExport(
	ctx context.Context,
	req *GenerateExportRequest,
) (*ExportPlan, error) {
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
	return &ExportPlan{Export: export, TimesheetIDs: ids}, nil
}

// PlanVoidExport is what VoidExport would leave the run as; a run already
// voided is refused.
func (s *Service) PlanVoidExport(ctx context.Context, req *VoidExportRequest) (*ExportChange, error) {
	original, err := s.repo.GetExportByID(ctx, &repositories.GetPayrollExportByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if original.Status == worker.PayrollExportVoided {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"That run has already been voided",
		)
	}

	export := *original
	now := timeutils.NowUnix()
	export.Status = worker.PayrollExportVoided
	export.VoidedAt = &now
	export.VoidReason = req.Reason

	multiErr := errortypes.NewMultiError()
	export.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return &ExportChange{Before: original, After: &export}, nil
}
