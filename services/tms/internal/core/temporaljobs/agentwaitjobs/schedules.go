package agentwaitjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

// reconcileEvery is how often the waits whose workflow vanished are looked
// for. A wait's own workflow closes it at its expiry; this catches the one it
// never will, such as after the workflow was terminated by hand.
const reconcileEvery = time.Hour

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider { return &ScheduleProvider{} }

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{{
		ID:            ReconcileScheduleID,
		Description:   "Close and pick up the agent waits whose workflow vanished",
		Spec:          schedule.Every(reconcileEvery),
		Workflow:      ReconcileAgentWaitsWorkflow,
		TaskQueue:     temporaltype.TaskQueueSystem.String(),
		OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		Memo:          map[string]any{"purpose": ReconcileScheduleID},
	}}
}
