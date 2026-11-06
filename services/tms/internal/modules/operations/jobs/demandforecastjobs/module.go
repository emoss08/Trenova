package demandforecastjobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/demandforecastjobs/demandforecastcontract"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const fitDescription = "Fit each organization's freight forecast from its shipment history and test it on its last weeks"

var DomainConfig = registry.DomainConfig{
	Name:         "demand-forecast-worker",
	TaskQueue:    temporaltype.TaskQueueDispatch.String(),
	WorkerConfig: registry.DefaultWorkerConfig(),
}

var Workflows = []registry.WorkflowDefinition{
	{
		Name:        demandforecastcontract.FitAllWorkflowName,
		Fn:          FitDemandForecastsWorkflow,
		Description: fitDescription,
	},
	{
		Name:        demandforecastcontract.FitTenantWorkflowName,
		Fn:          FitDemandForecastWorkflow,
		Description: "Fit one organization's freight forecast now",
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
			ID:            "demand-forecast-fit",
			Description:   fitDescription,
			Spec:          schedule.Cron("15 3 * * *"),
			Workflow:      FitDemandForecastsWorkflow,
			Args:          []any{&FanOutInput{}},
			TaskQueue:     temporaltype.TaskQueueDispatch.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "demand-forecast-fit",
			},
		},
	}
}

var Module = fx.Module("demand-forecast-jobs",
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
