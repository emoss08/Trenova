package watchtowerjobs

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
			ID:          "watchtower-reconcile",
			Description: "Correct every organization's watchtower against its sources",
			// Nightly, before the briefing is written, so the morning's feed
			// is the corrected one. A live projection lost to a failed write
			// costs one night, not a permanent ghost on someone's tower.
			Spec:      schedule.Cron("40 3 * * *"),
			Workflow:  WatchtowerReconcileWorkflow,
			TaskQueue: temporaltype.TaskQueueSystem.String(),
			// A reconcile still running when the next fires means the
			// previous sweep has not finished reading the sources. Two at
			// once would resolve against half-read snapshots.
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "watchtower-reconcile",
			},
		},
		{
			ID:          "watchtower-retention",
			Description: "Remove watchtower items resolved more than a month ago",
			// Weekly is often enough for a month-long window, and it keeps
			// the delete off the nightly hour the reconcile already uses.
			Spec:          schedule.Cron("10 4 * * 0"),
			Workflow:      WatchtowerRetentionWorkflow,
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "watchtower-retention",
			},
		},
	}
}
