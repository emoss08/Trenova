package accountingsyncjobs

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
			ID:            "accounting-connection-check",
			Description:   "Refresh expiring accounting authorizations and check every connection answers",
			Spec:          schedule.Cron("*/5 * * * *"),
			Workflow:      CheckAccountingConnectionsWorkflow,
			TaskQueue:     temporaltype.IntegrationTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "accounting-connection-check",
			},
		},
		{
			ID:            "accounting-reference-refresh",
			Description:   "Pull every active accounting connection's reference data and re-score open mappings",
			Spec:          schedule.Cron("17 7 * * *"),
			Workflow:      RefreshAllAccountingReferenceWorkflow,
			Args:          []any{&ReferenceSweepPayload{}},
			TaskQueue:     temporaltype.IntegrationTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "accounting-reference-refresh",
			},
		},
		{
			ID:            "accounting-sync-dispatch",
			Description:   "Wake the sender for every accounting connection with documents due",
			Spec:          schedule.Cron("* * * * *"),
			Workflow:      KickDueAccountingSyncWorkflow,
			TaskQueue:     temporaltype.IntegrationTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "accounting-sync-dispatch",
			},
		},
		{
			ID:            "accounting-sync-safety-net",
			Description:   "Queue posted documents that never reached the accounting outbox",
			Spec:          schedule.Cron("23 * * * *"),
			Workflow:      AccountingSafetyNetWorkflow,
			TaskQueue:     temporaltype.IntegrationTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "accounting-sync-safety-net",
			},
		},
		{
			ID:            "accounting-sync-retention",
			Description:   "Clear accounting sync payloads and attempts older than 90 days",
			Spec:          schedule.Cron("41 4 * * *"),
			Workflow:      PurgeAccountingSyncWorkflow,
			TaskQueue:     temporaltype.IntegrationTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "accounting-sync-retention",
			},
		},
	}
}
