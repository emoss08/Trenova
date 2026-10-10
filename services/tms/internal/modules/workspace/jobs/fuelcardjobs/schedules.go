package fuelcardjobs

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
			ID:          "fuel-card-feed-sync",
			Description: "Read fuel card transactions for all connected tenants and post the ones that resolve",
			// Hourly matches how the networks publish: a fleet card export lands
			// on a billing cycle, not in real time, so polling harder would only
			// re-read the same files.
			Spec:          schedule.Every(time.Hour),
			Workflow:      FuelCardSyncWorkflow,
			TaskQueue:     temporaltype.IntegrationTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				memoPurposeKey: "fuel-card-feed-sync",
			},
		},
	}
}
