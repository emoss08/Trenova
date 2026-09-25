package reporting

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/cronutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

// SchedulePreview is a schedule as creating it would store it, with the
// report it runs.
type SchedulePreview struct {
	Schedule   *report.ReportSchedule
	Definition *report.ReportDefinition
}

// PreviewSchedule is CreateSchedule without the save.
func (s *Service) PreviewSchedule(
	ctx context.Context,
	req *SaveScheduleRequest,
) (*SchedulePreview, error) {
	request := *req

	return s.planSchedule(ctx, &request)
}

// planSchedule validates a new schedule, with the report it runs, and builds
// the schedule with its first run.
func (s *Service) planSchedule(
	ctx context.Context,
	req *SaveScheduleRequest,
) (*SchedulePreview, error) {
	definition, err := s.validateScheduleRequest(ctx, req)
	if err != nil {
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
		Delivery:       req.delivery(),
		Alert:          req.Alert,
		Enabled:        req.Enabled,
		RunAsID:        req.TenantInfo.UserID,
		NextRunAt:      nextRun,
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return &SchedulePreview{Schedule: entity, Definition: definition}, nil
}
