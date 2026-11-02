package agentqualityjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

const (
	reconcileEvery       = time.Hour
	reconcileDescription = "Keep one nightly quality sweep schedule behind every organization " +
		"that wants one"
)

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{
		{
			ID:            ReconcileQualitySchedulesScheduleID,
			Description:   reconcileDescription,
			Spec:          schedule.Every(reconcileEvery),
			Workflow:      ReconcileQualitySchedulesWorkflow,
			TaskQueue:     temporaltype.TaskQueueAgentHeavy.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": ReconcileQualitySchedulesScheduleID,
			},
		},
	}
}
