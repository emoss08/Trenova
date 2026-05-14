package exchangeratejobs

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
			ID:            "exchange-rate-refresh",
			Description:   "Refresh cached exchange rates from ExchangeRate-API daily for all enabled tenants",
			Spec:          schedule.Cron("0 6 * * *"),
			Workflow:      RefreshExchangeRatesWorkflow,
			TaskQueue:     temporaltype.IntegrationTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "exchange-rate-refresh",
				"source":  "v6.exchangerate-api.com",
			},
		},
	}
}
