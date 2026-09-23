package aifeedbackjobs

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
			ID:            "ai-feedback-maintenance",
			Description:   "Purge expired AI feedback and suggest agent memories from recent ratings",
			Spec:          schedule.Cron("25 2 * * *"),
			Workflow:      AIFeedbackMaintenanceWorkflow,
			Args:          []any{&AIFeedbackMaintenanceInput{}},
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "ai-feedback-maintenance",
			},
		},
	}
}
