package dispatchjobs

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
			ID:          "horizon-plan-sweep",
			Description: "Re-plan the dispatch horizon for organizations running horizon planning",
			Spec:        schedule.Cron("*/30 * * * *"),
			Workflow:    HorizonPlanSweepWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			// A plan that is still running has nothing to hand over to the next one,
			// and two concurrent passes would race on the same proposals.
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "horizon-plan-sweep",
			},
		},
	}
}
