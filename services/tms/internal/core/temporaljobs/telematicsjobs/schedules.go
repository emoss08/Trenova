package telematicsjobs

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
			ID:            "samsara-telematics-poll",
			Description:   "Poll Samsara vehicle positions and HOS clocks for all enabled tenants",
			Spec:          schedule.Every(time.Minute),
			Workflow:      TelematicsPollWorkflow,
			TaskQueue:     temporaltype.IntegrationTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				memoPurposeKey: "samsara-telematics-poll",
			},
		},
		{
			ID:            "telematics-retention-sweep",
			Description:   "Prune telematics events older than 90 days and HOS violations older than a year",
			Spec:          schedule.Cron("0 9 * * *"),
			Workflow:      TelematicsRetentionWorkflow,
			TaskQueue:     temporaltype.IntegrationTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				memoPurposeKey: "telematics-retention-sweep",
			},
		},
		{
			ID:            "samsara-telematics-sweep",
			Description:   "Sync Samsara vehicle mappings and ingest HOS violations for all enabled tenants",
			Spec:          schedule.Every(15 * time.Minute),
			Workflow:      TelematicsSweepWorkflow,
			TaskQueue:     temporaltype.IntegrationTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				memoPurposeKey: "samsara-telematics-sweep",
			},
		},
	}
}
