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
	}
}
