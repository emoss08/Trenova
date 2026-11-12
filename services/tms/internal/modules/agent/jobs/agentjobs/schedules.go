package agentjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/evalcasejobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

// reconcileEvery is how often every agent's schedule is checked against the
// agent. A save syncs its own schedule at once; this repairs one a failed save
// left behind, and catches an agent changed by anything other than a save.
const reconcileEvery = time.Hour

// proposalExpiryEvery is how often expired proposals are marked and the runs
// they leave settled are closed. A proposal past its expiry is refused at
// decision time whatever the sweep has done; the sweep only moves the
// status, so the run's record catches up within this.
const proposalExpiryEvery = 15 * time.Minute

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return append([]*schedule.Schedule{
		{
			ID:            ReconcileSchedulesScheduleID,
			Description:   "Keep one schedule behind every scheduled or continuous agent",
			Spec:          schedule.Every(reconcileEvery),
			Workflow:      ReconcileDefinitionSchedulesWorkflow,
			TaskQueue:     temporaltype.TaskQueueAgentBackground.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": ReconcileSchedulesScheduleID,
			},
		},
		{
			ID:            ExpireStaleProposalsScheduleID,
			Description:   "Expire pending agent proposals past their decision window and close the runs they leave settled",
			Spec:          schedule.Every(proposalExpiryEvery),
			Workflow:      ExpireStaleProposalsWorkflow,
			TaskQueue:     temporaltype.TaskQueueAgentBackground.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": ExpireStaleProposalsScheduleID,
			},
		},
		{
			ID:            AssistantRetentionScheduleID,
			Description:   "Remove unsaved formula conversations and old palette question records",
			Spec:          schedule.Cron("30 4 * * *"),
			Workflow:      AssistantRetentionWorkflow,
			TaskQueue:     temporaltype.TaskQueueAgentBackground.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": AssistantRetentionScheduleID,
			},
		},
	}, evalcasejobs.Schedules()...)
}
