package inboundjobs

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
			ID:          "inbound-message-retention",
			Description: "Remove inbound messages settled more than six months ago",
			// Weekly is plenty for a six-month window, and Sunday morning keeps
			// the deletes off the hours the watchtower sweeps already use.
			Spec:          schedule.Cron("40 4 * * 0"),
			Workflow:      InboundMessageRetentionWorkflow,
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "inbound-message-retention",
			},
		},
	}
}
