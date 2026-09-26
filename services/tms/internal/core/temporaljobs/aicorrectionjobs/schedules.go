package aicorrectionjobs

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
			ID:            "ai-correction-retention",
			Description:   "Purge AI corrections older than each organization's retention period",
			Spec:          schedule.Cron("40 2 * * *"),
			Workflow:      AICorrectionRetentionWorkflow,
			Args:          []any{&AICorrectionRetentionInput{}},
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "ai-correction-retention",
			},
		},
	}
}
