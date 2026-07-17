package reportjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

const memoPurposeKey = "purpose"

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{
		{
			ID:            "report-schedule-dispatch",
			Description:   "Dispatch report runs for due user schedules",
			Spec:          schedule.Every(time.Minute),
			Workflow:      DispatchDueReportSchedulesWorkflow,
			TaskQueue:     temporaltype.ReportTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				memoPurposeKey: "report-schedule-dispatch",
			},
		},
		{
			ID:            "report-artifact-cleanup",
			Description:   "Delete expired report artifacts and mark runs expired",
			Spec:          schedule.Every(15 * time.Minute),
			Workflow:      CleanupExpiredReportRunsWorkflow,
			TaskQueue:     temporaltype.ReportTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				memoPurposeKey: "report-artifact-cleanup",
			},
		},
		{
			ID:            "report-zombie-reconciliation",
			Description:   "Fail report runs whose workflows are no longer running",
			Spec:          schedule.Every(30 * time.Minute),
			Workflow:      ReconcileZombieReportRunsWorkflow,
			TaskQueue:     temporaltype.ReportTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				memoPurposeKey: "report-zombie-reconciliation",
			},
		},
	}
}
