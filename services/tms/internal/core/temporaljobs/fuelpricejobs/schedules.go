package fuelpricejobs

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
			ID:            "eia-fuel-price-refresh",
			Description:   "Ingest weekly DOE/EIA diesel prices for all enabled tenants (Tue-Sat to cover holiday-delayed publications)",
			Spec:          schedule.Cron("0 16 * * 2-6"),
			Workflow:      RefreshFuelPricesWorkflow,
			TaskQueue:     temporaltype.IntegrationTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "eia-fuel-price-refresh",
				"source":  "api.eia.gov",
			},
		},
	}
}
