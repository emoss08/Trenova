package bulkeditjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var DomainConfig = registry.DomainConfig{
	Name:         "bulk-edit-worker",
	TaskQueue:    temporaltype.TaskQueueSystem.String(),
	WorkerConfig: registry.DefaultWorkerConfig(),
}

var Workflows = convertWorkflows(RegisterWorkflows())

func convertWorkflows(wfs []temporaltype.WorkflowDefinition) []registry.WorkflowDefinition {
	result := make([]registry.WorkflowDefinition, len(wfs))
	for i, wf := range wfs {
		result[i] = registry.WorkflowDefinition{
			Name:        wf.Name,
			Fn:          wf.Fn,
			Description: wf.Description,
		}
	}
	return result
}

type RegistryParams struct {
	fx.In

	Activities *Activities
	Logger     *zap.Logger
}

func NewRegistry(p RegistryParams) registry.WorkerRegistry {
	return registry.NewDomainRegistry(&DomainConfig, p.Activities, Workflows, p.Logger)
}

var Module = fx.Module("bulk-edit-jobs",
	fx.Provide(NewActivities),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
)
