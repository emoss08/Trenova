package rollupjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

// refreshEvery is how far a rollup may trail its source.
const refreshEvery = time.Minute

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider { return &ScheduleProvider{} }

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{{
		ID:            RefreshScheduleID,
		Description:   "Recompute the rollup days writes marked dirty",
		Spec:          schedule.Every(refreshEvery),
		Workflow:      RefreshRollupsWorkflow,
		TaskQueue:     temporaltype.TaskQueueSystem.String(),
		OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		Memo:          map[string]any{"purpose": RefreshScheduleID},
	}}
}
