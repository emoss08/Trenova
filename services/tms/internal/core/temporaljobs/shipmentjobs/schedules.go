package shipmentjobs

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
			ID:            "shipment-auto-delay",
			Description:   "Automatically mark eligible shipments delayed across tenants",
			Spec:          schedule.Every(5 * time.Minute),
			Workflow:      AutoDelayShipmentsWorkflow,
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "shipment-auto-delay",
			},
		},
		{
			ID:            "shipment-auto-cancel",
			Description:   "Automatically cancel eligible shipments across tenants",
			Spec:          schedule.Cron("0 0 * * *"),
			Workflow:      AutoCancelShipmentsWorkflow,
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "shipment-auto-cancel",
			},
		},
	}
}
