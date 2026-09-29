package reporting

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/cronutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

type DefinitionChange struct {
	Before *report.ReportDefinition
	After  *report.ReportDefinition
}

type RunChange struct {
	Before *report.ReportRun
	After  *report.ReportRun
}

type ScheduleChange struct {
	Before     *report.ReportSchedule
	After      *report.ReportSchedule
	Definition *report.ReportDefinition
}

func (s *Service) PlanDeleteDefinition(
	ctx context.Context,
	req *GetDefinitionRequest,
) (*report.ReportDefinition, error) {
	existing, err := s.defRepo.GetByID(ctx, &repositories.GetReportDefinitionRequest{
		TenantInfo:   req.TenantInfo,
		DefinitionID: req.DefinitionID,
	})
	if err != nil {
		return nil, err
	}
	if existing.OwnerID != req.TenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError("Only the report owner can delete this report")
	}

	return existing, nil
}

func (s *Service) PlanResetCannedFork(
	ctx context.Context,
	req *GetDefinitionRequest,
) (*DefinitionChange, error) {
	existing, err := s.defRepo.GetByID(ctx, &repositories.GetReportDefinitionRequest{
		TenantInfo:   req.TenantInfo,
		DefinitionID: req.DefinitionID,
	})
	if err != nil {
		return nil, err
	}
	if existing.OwnerID != req.TenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError(
			"Only the report owner can reset this report",
		)
	}
	if existing.Kind != report.DefinitionKindCannedFork || existing.CannedKey == "" {
		return nil, errortypes.NewBusinessError(
			"Only reports customized from a canned report can be reset",
		)
	}

	entry, err := s.GetCanned(existing.CannedKey)
	if err != nil {
		return nil, err
	}

	reset := *existing
	reset.Definition = entry.Definition
	reset.CannedVersion = entry.Version
	reset.CatalogVersion = reportcatalog.Version
	reset.Status = report.DefinitionStatusActive
	reset.Diagnostics = nil

	return &DefinitionChange{Before: existing, After: &reset}, nil
}

func (s *Service) PlanCancelRun(ctx context.Context, req *GetRunRequest) (*RunChange, error) {
	run, err := s.GetRun(ctx, req)
	if err != nil {
		return nil, err
	}

	if run.RequestedByID != req.TenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError(
			"Only the user who requested a report run can cancel it",
		)
	}
	if run.Status.IsTerminal() {
		return nil, errortypes.NewBusinessError("This report run has already finished")
	}

	canceled := *run
	canceled.Status = report.RunStatusCanceled
	canceled.Error = &report.RunError{Code: "CANCELED", Message: "The run was canceled"}

	return &RunChange{Before: run, After: &canceled}, nil
}

func (s *Service) PlanDeleteDashboard(
	ctx context.Context,
	req *GetDashboardRequest,
) (*report.Dashboard, error) {
	existing, err := s.dashboardRepo.GetByID(ctx, &repositories.GetReportDashboardRequest{
		TenantInfo:  req.TenantInfo,
		DashboardID: req.DashboardID,
	})
	if err != nil {
		return nil, err
	}
	if existing.OwnerID != req.TenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError(
			"Only the dashboard owner can delete this dashboard",
		)
	}

	return existing, nil
}

func (s *Service) ownedSchedule(
	ctx context.Context,
	req *GetScheduleRequest,
	refusal *errortypes.AuthorizationError,
) (*report.ReportSchedule, error) {
	existing, err := s.GetSchedule(ctx, req)
	if err != nil {
		return nil, err
	}
	if existing.RunAsID != req.TenantInfo.UserID {
		return nil, refusal
	}

	return existing, nil
}

func (s *Service) PlanUpdateSchedule(
	ctx context.Context,
	req *SaveScheduleRequest,
) (*ScheduleChange, error) {
	existing, err := s.ownedSchedule(
		ctx,
		&GetScheduleRequest{Request: req.Request, ScheduleID: req.ScheduleID},
		errortypes.NewAuthorizationError("Only the schedule owner can modify this schedule"),
	)
	if err != nil {
		return nil, err
	}

	definition, err := s.validateScheduleRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	nextRun, err := cronutils.NextRun(req.CronExpression, req.Timezone, timeutils.NowUnix())
	if err != nil {
		return nil, err
	}

	updated := *existing
	updated.DefinitionID = req.DefinitionID
	updated.CronExpression = req.CronExpression
	updated.Timezone = req.Timezone
	updated.Formats = req.Formats
	updated.Delivery = req.delivery()
	if !scheduleAlertsEqual(existing.Alert, req.Alert) {
		updated.AlertFiring = false
	}
	updated.Alert = req.Alert
	updated.Enabled = req.Enabled
	updated.NextRunAt = nextRun
	if req.Enabled {
		updated.ConsecutiveFailures = 0
	}
	updated.Version = req.Version

	multiErr := errortypes.NewMultiError()
	updated.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return &ScheduleChange{Before: existing, After: &updated, Definition: definition}, nil
}

func (s *Service) PlanDeleteSchedule(
	ctx context.Context,
	req *GetScheduleRequest,
) (*report.ReportSchedule, error) {
	return s.ownedSchedule(
		ctx,
		req,
		errortypes.NewAuthorizationError("Only the schedule owner can delete this schedule"),
	)
}
