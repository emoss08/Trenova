package formulatemplatejobs

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
			ID:            "formula-template-expire-stale-submissions",
			Description:   "Return formula templates waiting in review for more than 14 days to draft",
			Spec:          schedule.Cron("0 6 * * *"),
			Workflow:      ExpireStaleSubmissionsWorkflow,
			TaskQueue:     string(temporaltype.TaskQueueSystem),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "formula-template-review-expiry",
			},
		},
	}
}
