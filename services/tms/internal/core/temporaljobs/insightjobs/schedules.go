package insightjobs

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
			ID:          "insight-refresh",
			Description: "Recompute home-screen operational insights for every organization",
			// Every six hours, offset off the hour so it does not contend with the
			// nightly reporting jobs. The numbers describe a 30-day window, so
			// running more often would burn model calls to move a figure by a
			// fraction of a point; running less often lets a card go stale within
			// a working day.
			Spec:      schedule.Cron("20 */6 * * *"),
			Workflow:  InsightRefreshWorkflow,
			Args:      []any{&InsightRefreshInput{}},
			TaskQueue: temporaltype.TaskQueueSystem.String(),
			// A refresh that is still running when the next one fires means the
			// previous window is still being computed. Starting a second would
			// have both trying to supersede the same rows.
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "insight-refresh",
			},
		},
	}
}
