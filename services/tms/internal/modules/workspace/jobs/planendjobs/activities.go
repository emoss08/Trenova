package planendjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	Reminder managedfeaturesports.PlanEndReminder
}

type Activities struct {
	reminder managedfeaturesports.PlanEndReminder
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{reminder: p.Reminder}
}

func (a *Activities) RemindManagedPlanEndActivity(
	ctx context.Context,
) (*managedfeaturesports.PlanEndReminderResult, error) {
	return a.reminder.RemindPlanEnd(ctx)
}
