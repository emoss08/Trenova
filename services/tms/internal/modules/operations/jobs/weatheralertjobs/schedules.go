package weatheralertjobs

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
			ID:            "weather-alert-poll",
			Description:   "Poll active NWS weather alerts and persist them for all tenants",
			Spec:          schedule.Every(5 * time.Minute),
			Workflow:      PollNWSAlertsWorkflow,
			TaskQueue:     temporaltype.TaskQueueWeatherAlert.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "weather-alert-poll",
				"source":  "api.weather.gov/alerts/active",
			},
		},
	}
}
