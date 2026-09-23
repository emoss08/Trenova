package agentjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

// reconcileEvery is how often every agent's schedule is checked against the
// agent. A save syncs its own schedule at once; this repairs one a failed save
// left behind, and catches an agent changed by anything other than a save.
const reconcileEvery = 15 * time.Minute

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{
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
		{
			ID:            CaptureEvalCaseCandidatesScheduleID,
			Description:   "Capture decided proposals as candidate evaluation cases",
			Spec:          schedule.Every(evalCaseCaptureEvery),
			Workflow:      CaptureEvalCaseCandidatesWorkflow,
			TaskQueue:     temporaltype.TaskQueueAgentBackground.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": CaptureEvalCaseCandidatesScheduleID,
			},
		},
		{
			ID:            PurgeEvalCasesScheduleID,
			Description:   "Purge expired evaluation cases and those of deleted conversations",
			Spec:          schedule.Cron("45 4 * * *"),
			Workflow:      PurgeEvalCasesWorkflow,
			TaskQueue:     temporaltype.TaskQueueAgentBackground.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": PurgeEvalCasesScheduleID,
			},
		},
	}
}
