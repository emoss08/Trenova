package reporting

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/cronutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type SaveScheduleRequest struct {
	Request

	ScheduleID      pulid.ID
	DefinitionID    pulid.ID
	CronExpression  string
	Timezone        string
	Formats         []string
	EmailRecipients []string
	Enabled         bool
	Version         int64
}

func (s *Service) validateScheduleRequest(
	ctx context.Context,
	req *SaveScheduleRequest,
) error {
	if _, err := s.GetDefinition(ctx, &GetDefinitionRequest{
		Request:      req.Request,
		DefinitionID: req.DefinitionID,
	}); err != nil {
		return err
	}

	timezone := req.Timezone
	if timezone == "" {
		timezone = s.orgTimezone(ctx, req.TenantInfo)
		req.Timezone = timezone
	}

	if _, err := cronutils.NextRun(req.CronExpression, timezone, timeutils.NowUnix()); err != nil {
		return errortypes.NewValidationError(
			"cronExpression", errortypes.ErrInvalid, err.Error(),
		)
	}

	return nil
}

func (s *Service) CreateSchedule(
	ctx context.Context,
	req *SaveScheduleRequest,
) (*report.ReportSchedule, error) {
	if err := s.validateScheduleRequest(ctx, req); err != nil {
		return nil, err
	}

	nextRun, err := cronutils.NextRun(req.CronExpression, req.Timezone, timeutils.NowUnix())
	if err != nil {
		return nil, err
	}

	entity := &report.ReportSchedule{
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		DefinitionID:   req.DefinitionID,
		CronExpression: req.CronExpression,
		Timezone:       req.Timezone,
		Formats:        req.Formats,
		Delivery:       &report.ScheduleDelivery{EmailRecipients: req.EmailRecipients},
		Enabled:        req.Enabled,
		RunAsID:        req.TenantInfo.UserID,
		NextRunAt:      nextRun,
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.scheduleRepo.Create(ctx, entity)
	if err != nil {
		s.l.Error("failed to create report schedule", zap.Error(err))
		return nil, err
	}

	return created, nil
}

func (s *Service) UpdateSchedule(
	ctx context.Context,
	req *SaveScheduleRequest,
) (*report.ReportSchedule, error) {
	existing, err := s.scheduleRepo.GetByID(ctx, &repositories.GetReportScheduleRequest{
		TenantInfo: req.TenantInfo,
		ScheduleID: req.ScheduleID,
	})
	if err != nil {
		return nil, err
	}
	if existing.RunAsID != req.TenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError(
			"Only the schedule owner can modify this schedule",
		)
	}

	if err = s.validateScheduleRequest(ctx, req); err != nil {
		return nil, err
	}

	nextRun, err := cronutils.NextRun(req.CronExpression, req.Timezone, timeutils.NowUnix())
	if err != nil {
		return nil, err
	}

	existing.DefinitionID = req.DefinitionID
	existing.CronExpression = req.CronExpression
	existing.Timezone = req.Timezone
	existing.Formats = req.Formats
	existing.Delivery = &report.ScheduleDelivery{EmailRecipients: req.EmailRecipients}
	existing.Enabled = req.Enabled
	existing.NextRunAt = nextRun
	if req.Enabled {
		existing.ConsecutiveFailures = 0
	}
	existing.Version = req.Version

	multiErr := errortypes.NewMultiError()
	existing.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.scheduleRepo.Update(ctx, existing)
}

type GetScheduleRequest struct {
	Request

	ScheduleID pulid.ID
}

func (s *Service) GetSchedule(
	ctx context.Context,
	req *GetScheduleRequest,
) (*report.ReportSchedule, error) {
	return s.scheduleRepo.GetByID(ctx, &repositories.GetReportScheduleRequest{
		TenantInfo: req.TenantInfo,
		ScheduleID: req.ScheduleID,
	})
}

type ListSchedulesRequest struct {
	Request

	DefinitionID pulid.ID
	EnabledOnly  bool
	Limit        int
	Offset       int
}

func (s *Service) ListSchedules(
	ctx context.Context,
	req *ListSchedulesRequest,
) ([]*report.ReportSchedule, error) {
	return s.scheduleRepo.List(ctx, &repositories.ListReportSchedulesRequest{
		TenantInfo:   req.TenantInfo,
		DefinitionID: req.DefinitionID,
		EnabledOnly:  req.EnabledOnly,
		Limit:        req.Limit,
		Offset:       req.Offset,
	})
}

func (s *Service) DeleteSchedule(ctx context.Context, req *GetScheduleRequest) error {
	existing, err := s.GetSchedule(ctx, req)
	if err != nil {
		return err
	}
	if existing.RunAsID != req.TenantInfo.UserID {
		return errortypes.NewAuthorizationError(
			"Only the schedule owner can delete this schedule",
		)
	}

	return s.scheduleRepo.Delete(ctx, &repositories.GetReportScheduleRequest{
		TenantInfo: req.TenantInfo,
		ScheduleID: req.ScheduleID,
	})
}
