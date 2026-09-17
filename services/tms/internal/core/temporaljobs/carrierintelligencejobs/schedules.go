package carrierintelligencejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

const memoPurposeKey = "purpose"

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{
		{
			ID:            "carrier-intelligence-sweep",
			Description:   "Recompute carrier findings, sync monitoring watchlists, poll change feeds and refresh due snapshots",
			Spec:          schedule.Every(15 * time.Minute),
			Workflow:      CarrierIntelSweepWorkflow,
			TaskQueue:     temporaltype.IntegrationTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo:          map[string]any{memoPurposeKey: "carrier-intelligence-sweep"},
		},
		{
			ID:            "carrier-intelligence-reconcile",
			Description:   "Reconcile carrier monitoring enrollment with each organization's policy",
			Spec:          schedule.Cron("0 6 * * *"),
			Workflow:      CarrierIntelReconcileWorkflow,
			TaskQueue:     temporaltype.IntegrationTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo:          map[string]any{memoPurposeKey: "carrier-intelligence-reconcile"},
		},
		{
			ID:            "carrier-intelligence-digest",
			Description:   "Daily digest of carrier intelligence changes awaiting review",
			Spec:          schedule.Cron("0 12 * * *"),
			Workflow:      CarrierIntelDigestWorkflow,
			TaskQueue:     temporaltype.IntegrationTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo:          map[string]any{memoPurposeKey: "carrier-intelligence-digest"},
		},
		{
			ID:            "carrier-intelligence-maintenance",
			Description:   "Roll up provider usage, purge expired payloads and prune snapshot history",
			Spec:          schedule.Cron("30 3 * * *"),
			Workflow:      CarrierIntelMaintenanceWorkflow,
			TaskQueue:     temporaltype.IntegrationTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo:          map[string]any{memoPurposeKey: "carrier-intelligence-maintenance"},
		},
	}
}
