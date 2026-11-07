package auditjobs

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
			ID:            "audit-partition-check",
			Description:   "Report how many months of audit partitions remain; deploys create them",
			Spec:          schedule.Cron("17 1 * * *"),
			Workflow:      AuditPartitionCheckWorkflow,
			TaskQueue:     temporaltype.AuditTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		},
		{
			ID:            "audit-retention-purge",
			Description:   "Delete audit entries past each organization's retention, outside legal holds",
			Spec:          schedule.Cron("43 2 * * *"),
			Workflow:      AuditRetentionWorkflow,
			TaskQueue:     temporaltype.AuditTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		},
		{
			ID:            "audit-seal",
			Description:   "Seal the last hour of each organization's audit entries",
			Spec:          schedule.Cron("12 * * * *"),
			Workflow:      AuditSealWorkflow,
			TaskQueue:     temporaltype.AuditTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		},
		{
			ID:            "audit-verify",
			Description:   "Verify each organization's audit seals against its entries",
			Spec:          schedule.Cron("27 4 * * *"),
			Workflow:      AuditVerifyWorkflow,
			TaskQueue:     temporaltype.AuditTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		},
		{
			ID:            "audit-export-cleanup",
			Description:   "Delete audit export files past their download window",
			Spec:          schedule.Every(time.Hour),
			Workflow:      AuditExportCleanupWorkflow,
			TaskQueue:     temporaltype.AuditTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		},
	}
}
