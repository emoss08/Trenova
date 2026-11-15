package accessjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

const (
	SweepScheduleID   = "orphaned-organization-access-sweep"
	SweepWorkflowName = "SweepOrphanedOrganizationAccessWorkflow"
	sweepEvery        = time.Hour

	SupportExpiryScheduleID   = "support-access-expiry-sweep"
	SupportExpiryWorkflowName = "RecordSupportAccessExpiriesWorkflow"
	supportExpiryEvery        = 5 * time.Minute
)

type ScheduleProvider struct {
	managed bool
}

// NewScheduleProvider schedules the support access expiry record only on a managed instance:
// self-hosted installs have no support access.
func NewScheduleProvider(cfg *managedconfig.Config) *ScheduleProvider {
	return &ScheduleProvider{managed: cfg.Enabled}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	schedules := []*schedule.Schedule{{
		ID:      SweepScheduleID,
		Feature: platformcatalog.FeatureAdministration,
		Description: "Revoke the roles and sessions people still hold in organizations " +
			"they no longer belong to",
		Spec:          schedule.Every(sweepEvery),
		Workflow:      SweepOrphanedOrganizationAccessWorkflow,
		TaskQueue:     temporaltype.TaskQueueSystem.String(),
		OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		Memo:          map[string]any{"purpose": SweepScheduleID},
	}}
	if !p.managed {
		return schedules
	}
	return append(schedules, &schedule.Schedule{
		ID:            SupportExpiryScheduleID,
		Feature:       platformcatalog.FeatureAdministration,
		Description:   "Record the Trenova support access grants that ended on their own",
		Spec:          schedule.Every(supportExpiryEvery),
		Workflow:      RecordSupportAccessExpiriesWorkflow,
		TaskQueue:     temporaltype.TaskQueueSystem.String(),
		OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		Memo:          map[string]any{"purpose": SupportExpiryScheduleID},
	})
}
