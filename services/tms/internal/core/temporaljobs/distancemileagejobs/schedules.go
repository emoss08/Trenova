package distancemileagejobs

import (
	"time"

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
			ID:            "stored-mileage-buffer-flush",
			Description:   "Flush stored mileage candidates from Redis",
			Spec:          schedule.Every(5 * time.Minute),
			Workflow:      ScheduledStoredMileageFlushWorkflow,
			TaskQueue:     temporaltype.DistanceMileageTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "stored-mileage-flush",
				"target":  "stored_mileage_redis_buffer",
			},
		},
	}
}
