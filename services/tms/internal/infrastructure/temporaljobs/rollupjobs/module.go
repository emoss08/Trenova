package rollupjobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var DomainConfig = registry.DomainConfig{
	Name:         "rollup-worker",
	TaskQueue:    temporaltype.TaskQueueSystem.String(),
	WorkerConfig: registry.DefaultWorkerConfig(),
}

func workflows() []registry.WorkflowDefinition {
	defs := RegisterWorkflows()
	out := make([]registry.WorkflowDefinition, len(defs))
	for i, wf := range defs {
		out[i] = registry.WorkflowDefinition{Name: wf.Name, Fn: wf.Fn, Description: wf.Description}
	}

	return out
}

type RegistryParams struct {
	fx.In

	Activities *Activities
	Logger     *zap.Logger
}

func NewRegistry(p RegistryParams) registry.WorkerRegistry {
	return registry.NewDomainRegistry(&DomainConfig, p.Activities, workflows(), p.Logger)
}

// Module refreshes the rollups the worker's bootstrap supplies as Rollups.
var Module = fx.Module("rollup-jobs",
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
