package factoringjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/factoringjobs/factoringcontract"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	SweepScheduleID     = "factoring-readiness-sweep"
	sweepCron           = "17 3 * * *"
	sweepJitter         = 20 * time.Minute
	evaluateDescription = "Work out which of an organization's invoices are ready to factor after they change"
	sweepDescription    = "Re-check every factored invoice nightly, catching expired documents and notices"
)

var DomainConfig = registry.DomainConfig{
	Name:         "factoring-worker",
	TaskQueue:    temporaltype.TaskQueueBilling.String(),
	WorkerConfig: registry.DefaultWorkerConfig(),
}

var Workflows = []registry.WorkflowDefinition{
	{
		Name:        factoringcontract.EvaluateWorkflowName,
		Fn:          EvaluateFactoringReadinessWorkflow,
		Description: evaluateDescription,
	},
	{
		Name:        factoringcontract.SweepWorkflowName,
		Fn:          FactoringReadinessSweepWorkflow,
		Description: sweepDescription,
	},
}

type RegistryParams struct {
	fx.In

	Activities *Activities
	Logger     *zap.Logger
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
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
			ID:            SweepScheduleID,
			Description:   sweepDescription,
			Spec:          schedule.Cron(sweepCron).WithJitter(sweepJitter),
			Workflow:      FactoringReadinessSweepWorkflow,
			TaskQueue:     temporaltype.TaskQueueBilling.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo:          map[string]any{"purpose": SweepScheduleID},
		},
	}
}

var Module = fx.Module("factoring-jobs",
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
