package documentextractionjobs

import (
	"time"

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
			ID:            "document-extraction-reconciliation",
			Description:   "Reconcile stale document extraction jobs",
			Spec:          schedule.Every(2 * time.Hour),
			Workflow:      ReconcileDocumentExtractionWorkflow,
			TaskQueue:     temporaltype.DocumentExtractionTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		},
	}
}
