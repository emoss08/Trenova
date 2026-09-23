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
			TaskQueue:     temporaltype.TaskQueueAgentBackground.String(),
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
			TaskQueue:     temporaltype.TaskQueueAgentBackground.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": ExpireStaleProposalsScheduleID,
			},
		},
		{
			ID:            DeleteStaleAskThreadsScheduleID,
			Description:   "Remove quick questions nobody kept once they have gone quiet for a month",
			Spec:          schedule.Cron("30 4 * * *"),
			Workflow:      DeleteStaleAskThreadsWorkflow,
			TaskQueue:     temporaltype.TaskQueueAgentBackground.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": DeleteStaleAskThreadsScheduleID,
			},
		},
	}
}
