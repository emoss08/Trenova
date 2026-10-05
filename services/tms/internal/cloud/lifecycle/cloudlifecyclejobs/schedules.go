package cloudlifecyclejobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.uber.org/fx"
)

type ScheduleProviderParams struct {
	fx.In

	Config *config.Config
}

type ScheduleProvider struct {
	cloud bool
}

func NewScheduleProvider(p ScheduleProviderParams) *ScheduleProvider {
	return &ScheduleProvider{cloud: p.Config.Platform.IsCloud()}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	if !p.cloud {
		return []*schedule.Schedule{}
	}

	return []*schedule.Schedule{
		{
			ID:            sweepScheduleID,
			Description:   "Move cloud trials to read-only and expired, and purge expired organizations",
			Spec:          schedule.Cron("7 * * * *"),
			Workflow:      CloudSubscriptionSweepWorkflow,
			Args:          []any{&SweepInput{}},
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": sweepScheduleID,
			},
		},
	}
}
