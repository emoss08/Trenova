package detentionjobs

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
			ID:            "detention-notice-sweep",
			Description:   "Send due detention notices for auto-send policies before the contractual deadline lapses",
			Spec:          schedule.Cron("*/5 * * * *"),
			Workflow:      SweepDetentionNoticesWorkflow,
			TaskQueue:     temporaltype.TaskQueueBilling.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "detention-notice-sweep",
			},
		},
	}
}
