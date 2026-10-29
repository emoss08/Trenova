package reportscheduleservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportingtest"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errScheduleWrote = errors.New("a plan wrote")

type planSchedules struct {
	reportports.ReportScheduleRepository

	schedule *report.ReportSchedule
}

func (r *planSchedules) GetByID(
	context.Context,
	*reportports.GetReportScheduleRequest,
) (*report.ReportSchedule, error) {
	copied := *r.schedule

	return &copied, nil
}

func (r *planSchedules) Update(
	context.Context,
	*report.ReportSchedule,
) (*report.ReportSchedule, error) {
	return nil, errScheduleWrote
}

func TestPlanUpdateAndDeleteSchedule_OnlyTheOwnerAndNothingSaved(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("usr_")
	definition := &report.ReportDefinition{ID: pulid.MustNew("rd_"), Name: "Aging", OwnerID: owner}
	schedule := &report.ReportSchedule{
		ID:             pulid.MustNew("rsch_"),
		DefinitionID:   definition.ID,
		CronExpression: "0 7 * * 1",
		Timezone:       "America/Chicago",
		Formats:        []string{"xlsx"},
		Delivery:       &report.ScheduleDelivery{EmailRecipients: []string{"ops@example.com"}},
		Enabled:        true,
		RunAsID:        owner,
		Version:        4,
	}
	svc := &Service{
		l:            zap.NewNop(),
		reporting:    scheduleCore(t, &previewDefinitions{definition: definition}),
		scheduleRepo: &planSchedules{schedule: schedule},
	}

	change, err := svc.PlanUpdateSchedule(t.Context(), &SaveScheduleRequest{
		Request:         reporting.Request{TenantInfo: reportingtest.Tenant(owner)},
		ScheduleID:      schedule.ID,
		DefinitionID:    definition.ID,
		CronExpression:  "0 6 * * 1-5",
		Timezone:        "America/Chicago",
		Formats:         []string{"csv"},
		EmailRecipients: []string{"ops@example.com"},
		Enabled:         true,
		Version:         schedule.Version,
	})
	require.NoError(t, err)
	assert.Equal(t, "0 7 * * 1", change.Before.CronExpression)
	assert.Equal(t, "0 6 * * 1-5", change.After.CronExpression)
	assert.Equal(t, []string{"csv"}, change.After.Formats)
	assert.Equal(t, definition.ID, change.Definition.ID)

	_, err = svc.PlanUpdateSchedule(t.Context(), &SaveScheduleRequest{
		Request:        reporting.Request{TenantInfo: reportingtest.Tenant(owner)},
		ScheduleID:     schedule.ID,
		DefinitionID:   definition.ID,
		CronExpression: "every monday",
		Timezone:       "America/Chicago",
		Formats:        []string{"csv"},
		Enabled:        true,
		Version:        schedule.Version,
	})
	require.Error(t, err)

	planned, err := svc.PlanDeleteSchedule(t.Context(), &GetScheduleRequest{
		Request: reporting.Request{
			TenantInfo: reportingtest.Tenant(owner),
		}, ScheduleID: schedule.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, schedule.ID, planned.ID)

	_, err = svc.PlanDeleteSchedule(t.Context(), &GetScheduleRequest{
		Request: reporting.Request{
			TenantInfo: reportingtest.Tenant(pulid.MustNew("usr_")),
		}, ScheduleID: schedule.ID,
	})
	var denied *errortypes.AuthorizationError
	require.ErrorAs(t, err, &denied)
}
