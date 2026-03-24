package fiscaljobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{
		{
			ID:            "fiscal-auto-close-periods",
			Description:   "Automatically close fiscal periods whose end date has passed",
			Spec:          schedule.Cron("0 1 * * *"), // runs at 1:00 AM UTC
			Workflow:      AutoCloseFiscalPeriodsWorkflow,
			TaskQueue:     temporaltype.FiscalTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "fiscal-period-auto-close",
			},
		},
		{
			ID:            "fiscal-auto-create-next-year",
			Description:   "Automatically create next fiscal year when current one is within 60 days of ending",
			Spec:          schedule.Cron("0 2 * * *"), // runs at 2:00 AM UTC
			Workflow:      AutoCreateNextFiscalYearWorkflow,
			TaskQueue:     temporaltype.FiscalTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "fiscal-year-auto-create",
			},
		},
	}
}
