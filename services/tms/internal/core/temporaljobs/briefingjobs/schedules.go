package briefingjobs

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
			ID:          "daily-briefing",
			Description: "Write the morning briefing for every organization whose hour has come",
			// Every hour on the hour, because the hour that matters is the
			// organization's own and a half-hour offset timezone still
			// lands inside the hour it belongs to. A tenant is written in
			// the one firing whose local hour matches its setting.
			Spec:      schedule.Cron("0 * * * *"),
			Workflow:  DailyBriefingWorkflow,
			TaskQueue: temporaltype.TaskQueueSystem.String(),
			// A sweep still running when the next hour fires would write
			// two hours' tenants at once and could write a tenant twice.
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "daily-briefing",
			},
		},
		{
			ID:            "briefing-retention",
			Description:   "Remove briefings older than the window a reader can page back through",
			Spec:          schedule.Cron("30 4 * * 0"),
			Workflow:      BriefingRetentionWorkflow,
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "briefing-retention",
			},
		},
	}
}
