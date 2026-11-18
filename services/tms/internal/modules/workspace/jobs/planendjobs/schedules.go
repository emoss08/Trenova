package planendjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

const (
	ReminderScheduleID   = "managed-plan-end-reminder"
	ReminderWorkflowName = "RemindManagedPlanEndWorkflow"
	reminderEvery        = time.Hour
	reminderDescription  = "Remind the administrators three days before the plan ends"
)

type ScheduleProvider struct {
	managed bool
}

func NewScheduleProvider(cfg *managedconfig.Config) *ScheduleProvider {
	return &ScheduleProvider{managed: cfg.Enabled}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	if !p.managed {
		return []*schedule.Schedule{}
	}
	return []*schedule.Schedule{{
		ID:            ReminderScheduleID,
		Feature:       platformcatalog.FeatureAdministration,
		Description:   reminderDescription,
		Spec:          schedule.Every(reminderEvery),
		Workflow:      RemindManagedPlanEndWorkflow,
		TaskQueue:     temporaltype.TaskQueueSystem.String(),
		OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		Memo:          map[string]any{"purpose": ReminderScheduleID},
	}}
}
