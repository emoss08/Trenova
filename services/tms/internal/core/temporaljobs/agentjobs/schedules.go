package agentjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"time"
)

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{
		{
			ID:            SweepScheduleID,
			Description:   "Start scheduled and continuous agents whose slot has come",
			Spec:          schedule.Cron("* * * * *"),
			Workflow:      AgentSweepWorkflow,
			TaskQueue:     temporaltype.TaskQueueAgent.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": SweepScheduleID,
			},
		},
		{
			ID:            ExpireStaleProposalsScheduleID,
			Description:   "Mark pending agent proposals whose decision window has closed as expired",
			Spec:          schedule.Every(15 * time.Minute),
			Workflow:      ExpireStaleProposalsWorkflow,
			TaskQueue:     temporaltype.TaskQueueAgent.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": ExpireStaleProposalsScheduleID,
			},
		},
	}
}
