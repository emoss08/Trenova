package billingqueuejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.uber.org/fx"
)

const (
	RecheckSweepScheduleID        = "billing-queue-recheck-sweep"
	recheckSweepInterval          = 5 * time.Minute
	NovaGenerationSweepScheduleID = "billing-queue-nova-generation-sweep"
	novaGenerationSweepInterval   = 30 * time.Minute
)

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{
		{
			ID:            RecheckSweepScheduleID,
			Description:   "Re-run the billing checks on queue items whose checks are missing or stale",
			Spec:          schedule.Every(recheckSweepInterval),
			Workflow:      BillingQueueRecheckSweepWorkflow,
			TaskQueue:     temporaltype.TaskQueueBilling.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": RecheckSweepScheduleID,
			},
		},
		{
			ID:            NovaGenerationSweepScheduleID,
			Description:   "Write, in bulk, the reviews and briefs billing queues are missing",
			Spec:          schedule.Every(novaGenerationSweepInterval),
			Workflow:      BillingQueueNovaGenerationSweepWorkflow,
			TaskQueue:     temporaltype.TaskQueueBilling.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": NovaGenerationSweepScheduleID,
			},
		},
	}
}

var Module = fx.Module("billing-queue-jobs",
	fx.Provide(NewActivities),
	fx.Provide(schedule.AsProvider(NewScheduleProvider)),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
)
