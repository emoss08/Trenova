package triagewakejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/triagewakejobs/triagewakecontract"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

const reconcileEvery = time.Hour

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider { return &ScheduleProvider{} }

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{{
		ID:            triagewakecontract.ReconcileScheduleID,
		Description:   "Wake the snoozed and waiting inbound messages whose own timer was lost",
		Spec:          schedule.Every(reconcileEvery),
		Workflow:      ReconcileTriageWakesWorkflow,
		TaskQueue:     temporaltype.TaskQueueSystem.String(),
		OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		Memo:          map[string]any{"purpose": triagewakecontract.ReconcileScheduleID},
	}}
}
