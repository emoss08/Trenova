package accessjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

const (
	SweepScheduleID   = "orphaned-organization-access-sweep"
	SweepWorkflowName = "SweepOrphanedOrganizationAccessWorkflow"
	sweepEvery        = time.Hour
)

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider { return &ScheduleProvider{} }

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{{
		ID: SweepScheduleID,
		Description: "Revoke the roles and sessions people still hold in organizations " +
			"they no longer belong to",
		Spec:          schedule.Every(sweepEvery),
		Workflow:      SweepOrphanedOrganizationAccessWorkflow,
		TaskQueue:     temporaltype.TaskQueueSystem.String(),
		OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		Memo:          map[string]any{"purpose": SweepScheduleID},
	}}
}
