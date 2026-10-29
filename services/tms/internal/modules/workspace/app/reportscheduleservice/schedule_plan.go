package reportscheduleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/cronutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

type ScheduleChange struct {
	Before     *report.ReportSchedule
	After      *report.ReportSchedule
	Definition *report.ReportDefinition
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
