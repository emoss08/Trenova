package retrievaljobs

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
			ID:            "retrieval-index-sweep",
			Description:   "Mark retrieval sources changed since they were indexed, per organization",
			Spec:          schedule.Cron("17 * * * *"),
			Workflow:      RetrievalIndexSweepWorkflow,
			Args:          []any{&SweepInput{}},
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "retrieval-index-sweep",
			},
		},
	}
}
