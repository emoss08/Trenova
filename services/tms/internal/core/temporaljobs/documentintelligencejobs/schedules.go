package documentintelligencejobs

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
			ID:            "document-intelligence-reconciliation",
			Description:   "Reconcile stale document intelligence jobs",
			Spec:          schedule.Every(10 * time.Minute),
			Workflow:      ReconcileDocumentIntelligenceWorkflow,
			TaskQueue:     temporaltype.DocumentIntelligenceTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		},
	}
}
