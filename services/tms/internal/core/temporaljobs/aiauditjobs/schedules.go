package aiauditjobs

import (
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

const (
	ProjectorScheduleID = "ai-audit-projector"
	VerifyScheduleID    = "ai-audit-verify"
	PruneScheduleID     = "ai-audit-retention-purge"
	CleanupScheduleID   = "ai-audit-export-cleanup"
	exportCleanupEvery  = time.Hour
)

type ScheduleProvider struct {
	projectorInterval time.Duration
}

func NewScheduleProvider(cfg *config.Config) *ScheduleProvider {
	return &ScheduleProvider{projectorInterval: cfg.AIAudit.Projector.GetInterval()}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{
		{
			ID:            ProjectorScheduleID,
			Description:   "Write new agent activity to the AI audit trail",
			Spec:          schedule.Every(p.projectorInterval),
			Workflow:      ProjectAIAuditWorkflow,
			TaskQueue:     temporaltype.AuditTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "ai-audit-projection",
				"target":  "ai_audit_events",
			},
		},
		{
			ID:            VerifyScheduleID,
			Description:   "Verify every tenant's AI audit trail hash chain",
			Spec:          schedule.Cron("37 3 * * *"),
			Workflow:      VerifyAIAuditChainWorkflow,
			TaskQueue:     temporaltype.AuditTaskQueue,
			Args:          []any{&serviceports.AIAuditVerifyPayload{}},
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "ai-audit-verification",
				"target":  "ai_audit_events",
			},
		},
		{
			ID:            PruneScheduleID,
			Description:   "Remove AI audit trail rows past each organization's retention period",
			Spec:          schedule.Cron("13 4 * * *"),
			Workflow:      PruneAIAuditWorkflow,
			TaskQueue:     temporaltype.AuditTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "ai-audit-retention-purge",
				"target":  "ai_audit_events",
			},
		},
		{
			ID:            CleanupScheduleID,
			Description:   "Delete AI audit export files past their download window",
			Spec:          schedule.Every(exportCleanupEvery),
			Workflow:      CleanupAIAuditExportsWorkflow,
			TaskQueue:     temporaltype.AuditTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "ai-audit-export-cleanup",
				"target":  "ai_audit_exports",
			},
		},
	}
}
