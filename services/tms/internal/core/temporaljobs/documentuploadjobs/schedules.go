package documentuploadjobs

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
			ID:            "document-upload-reconciliation",
			Description:   "Reconcile stale document uploads and pending previews",
			Spec:          schedule.Every(5 * time.Minute),
			Workflow:      ReconcileDocumentUploadsWorkflow,
			TaskQueue:     temporaltype.UploadTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "document-upload-reconciliation",
			},
		},
	}
}
