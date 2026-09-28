package capturejobs

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
			ID:          "capture-maintenance",
			Description: "Expire stale capture requests, restart lost batch runs and enforce intake retention",
			// Every five minutes keeps an unanswered scan request from sitting
			// in the web app long after its two minutes ran out, and restarts
			// a lost batch before anybody has noticed it was lost.
			Spec:          schedule.Cron("*/5 * * * *"),
			Workflow:      CaptureMaintenanceWorkflow,
			TaskQueue:     temporaltype.CaptureTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "capture-maintenance",
			},
		},
	}
}
