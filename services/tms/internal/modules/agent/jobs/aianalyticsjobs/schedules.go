package aianalyticsjobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
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
			ID:            "ai-analytics-refresh",
			Description:   "Work out each organization's AI analytics from its newest runs, turns and calls",
			Spec:          schedule.Cron("*/5 * * * *"),
			Workflow:      RefreshAIAnalyticsWorkflow,
			Args:          []any{&RefreshAIAnalyticsInput{}},
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "ai-analytics-refresh",
			},
		},
	}
}
