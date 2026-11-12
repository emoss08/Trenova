package triagewakejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/triagewakejobs/triagewakecontract"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

// wakeEvery is how late a snoozed or waiting message may come back: the sweep
// is the only timer, so a message returns within this of its time.
const wakeEvery = time.Minute

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider { return &ScheduleProvider{} }

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{{
		ID:            triagewakecontract.WakeDueScheduleID,
		Description:   "Wake the snoozed and waiting inbound messages whose time has come",
		Spec:          schedule.Every(wakeEvery),
		Workflow:      WakeDueTriageWorkflow,
		TaskQueue:     temporaltype.TaskQueueSystem.String(),
		OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		Memo:          map[string]any{"purpose": triagewakecontract.WakeDueScheduleID},
	}}
}
