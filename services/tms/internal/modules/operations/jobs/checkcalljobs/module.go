package checkcalljobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const purgeDescription = "Purge carrier tracking links that are no longer usable"

var DomainConfig = registry.DomainConfig{
	Name:         "check-call-worker",
	TaskQueue:    temporaltype.TaskQueueDispatch.String(),
	WorkerConfig: registry.DefaultWorkerConfig(),
}

var Workflows = []registry.WorkflowDefinition{
	{
		Name:        PurgeTrackingLinksWorkflowName,
		Fn:          PurgeTrackingLinksWorkflow,
		Description: purgeDescription,
	},
}

type RegistryParams struct {
	fx.In

	Activities *Activities
	Logger     *zap.Logger
}

func NewRegistry(p RegistryParams) registry.WorkerRegistry {
	return registry.NewDomainRegistry(&DomainConfig, p.Activities, Workflows, p.Logger)
}

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{
		{
			ID:            "carrier-tracking-link-retention",
			Description:   purgeDescription,
			Spec:          schedule.Every(24 * time.Hour),
			Workflow:      PurgeTrackingLinksWorkflow,
			TaskQueue:     temporaltype.TaskQueueDispatch.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "carrier-tracking-link-retention",
			},
		},
	}
}

var Module = fx.Module("check-call-jobs",
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
