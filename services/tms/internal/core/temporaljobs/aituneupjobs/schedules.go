package aituneupjobs

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
			ID:            "ai-tune-ups",
			Description:   "Work out each organization's AI tune-ups from the last 30 days of runs",
			Spec:          schedule.Cron("40 3 * * *"),
			Workflow:      ComputeAITuneUpsWorkflow,
			Args:          []any{&ComputeAITuneUpsInput{}},
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "ai-tune-ups",
			},
		},
	}
}
