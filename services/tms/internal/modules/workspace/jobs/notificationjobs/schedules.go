package notificationjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
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
			ID:            "notification-daily-digest",
			Description:   "Email each person who bundles notifications their daily digest at 8:00 AM in their own time zone",
			Spec:          schedule.Every(time.Hour),
			Workflow:      NotificationDigestWorkflow,
			TaskQueue:     temporaltype.EmailTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				memoPurposeKey: "notification-daily-digest",
			},
		},
	}
}
