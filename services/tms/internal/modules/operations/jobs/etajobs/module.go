package etajobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/etajobs/etacontract"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	SweepScheduleID    = "shipment-eta-sweep"
	sweepInterval      = 5 * time.Minute
	refreshDescription = "Project a shipment's arrival at each open stop after it changes"
	sweepDescription   = "Project arrivals for every active shipment and mark stale inputs"
)

var DomainConfig = registry.DomainConfig{
	Name:         "shipment-eta-worker",
	TaskQueue:    temporaltype.TaskQueueDispatch.String(),
	WorkerConfig: registry.DefaultWorkerConfig(),
}

var Workflows = []registry.WorkflowDefinition{
	{
		Name:        etacontract.RefreshWorkflowName,
		Fn:          RefreshShipmentEtaWorkflow,
		Description: refreshDescription,
	},
	{
		Name:        etacontract.SweepWorkflowName,
		Fn:          ShipmentEtaSweepWorkflow,
		Description: sweepDescription,
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
			ID:            SweepScheduleID,
			Description:   sweepDescription,
			Spec:          schedule.Every(sweepInterval),
			Workflow:      ShipmentEtaSweepWorkflow,
			TaskQueue:     temporaltype.TaskQueueDispatch.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": SweepScheduleID,
			},
		},
	}
}

var Module = fx.Module("shipment-eta-jobs",
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
