package aitrainingjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

const RetrainingScheduleID = "extraction-retraining"

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{
		{
			ID:            RetrainingScheduleID,
			Description:   "Start a document extraction retraining when enough new corrections have been confirmed",
			Spec:          schedule.Cron("10 8 * * 1"),
			Workflow:      ExtractionRetrainingWorkflow,
			Args:          []any{&RetrainingScheduleInput{}},
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "extraction-retraining",
			},
		},
	}
}
