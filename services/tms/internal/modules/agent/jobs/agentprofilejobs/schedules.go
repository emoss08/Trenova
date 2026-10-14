package agentprofilejobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
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
			ID:          "agent-profiles",
			Description: "Write the agent profiles of every organization and the people who talk to its agents",
			// Once a day, in the small hours across the US, off the hour so it
			// does not contend with the jobs that start on it. A profile
			// describes a 90-day or 30-day window; running more often would
			// spend model calls on facts that have not moved.
			Spec:      schedule.Cron("17 9 * * *"),
			Workflow:  SweepWorkflow,
			Args:      []any{&SweepInput{}},
			TaskQueue: temporaltype.TaskQueueSystem.String(),
			// A sweep still running when the next fires is still writing; a
			// second would only race it to the same profiles.
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "agent-profiles",
			},
		},
	}
}
