package fuelrollupjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

const recentEvery = 5 * time.Minute

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider { return &ScheduleProvider{} }

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{
		{
			ID:            RecentScheduleID,
			Feature:       platformcatalog.FeatureAdministration,
			Description:   "Rebuild today's and yesterday's fuel and miles by tractor",
			Spec:          schedule.Every(recentEvery),
			Workflow:      RefreshFuelTractorDaysWorkflow,
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			Args:          []any{&RefreshInput{Days: fuelpurchaseports.TractorDaysRecent}},
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo:          map[string]any{"purpose": RecentScheduleID},
		},
		{
			ID:            TrailingScheduleID,
			Feature:       platformcatalog.FeatureAdministration,
			Description:   "Rebuild the trailing two months of fuel and miles by tractor",
			Spec:          schedule.Cron("30 3 * * *"),
			Workflow:      RefreshFuelTractorDaysWorkflow,
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			Args:          []any{&RefreshInput{Days: fuelpurchaseports.TractorDaysTrailing}},
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo:          map[string]any{"purpose": TrailingScheduleID},
		},
	}
}
