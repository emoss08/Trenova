package auditjobs

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
			ID:            "audit-buffer-flush",
			Description:   "Flush audit buffer from Redis for batch processing",
			Spec:          schedule.Every(10 * time.Second),
			Workflow:      ScheduledAuditFlushWorkflow,
			TaskQueue:     temporaltype.AuditTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "batch-processing",
				"target":  "audit_redis_buffer",
			},
		},
		{
			ID:            "audit-dlq-retry",
			Description:   "Retry failed audit entries from dead-letter queue",
			Spec:          schedule.Every(5 * time.Minute),
			Workflow:      DLQRetryWorkflow,
			TaskQueue:     temporaltype.AuditTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "dlq-retry",
				"target":  "audit_dlq",
			},
		},
	}
}
