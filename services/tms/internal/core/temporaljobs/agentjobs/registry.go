package agentjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// The agent's work is split across queues so one class cannot starve another.
// A registry binds one set of workflows to one queue; the worker manager shares
// a worker between registries that name the same queue, and registering an
// activity on a queue it is never called from costs nothing, so all three
// registries hand it the same activity set.
var (
	BackgroundDomainConfig = registry.DomainConfig{
		Name:         "agent-background-worker",
		TaskQueue:    temporaltype.TaskQueueAgentBackground.String(),
		WorkerConfig: registry.DefaultWorkerConfig(),
	}

	HeavyDomainConfig = registry.DomainConfig{
		Name:         "agent-heavy-worker",
		TaskQueue:    temporaltype.TaskQueueAgentHeavy.String(),
		WorkerConfig: registry.DefaultWorkerConfig(),
	}

	// DrainDomainConfig polls the queue everything used before the split, so
	// runs already in flight when it deployed still find a worker.
	DrainDomainConfig = registry.DomainConfig{
		Name:         "agent-drain-worker",
		TaskQueue:    temporaltype.TaskQueueAgent.String(),
		WorkerConfig: registry.DefaultWorkerConfig(),
	}
)

var (
	BackgroundWorkflows = convertWorkflows(RegisterBackgroundWorkflows())
	HeavyWorkflows      = convertWorkflows(RegisterHeavyWorkflows())
	DrainWorkflows      = convertWorkflows(RegisterDrainWorkflows())
)

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

func NewBackgroundRegistry(p RegistryParams) registry.WorkerRegistry {
	return registry.NewDomainRegistry(
		&BackgroundDomainConfig,
		p.Activities,
		BackgroundWorkflows,
		p.Logger,
	)
}

func NewHeavyRegistry(p RegistryParams) registry.WorkerRegistry {
	return registry.NewDomainRegistry(
		&HeavyDomainConfig,
		p.Activities,
		HeavyWorkflows,
		p.Logger,
	)
}

func NewDrainRegistry(p RegistryParams) registry.WorkerRegistry {
	return registry.NewDomainRegistry(
		&DrainDomainConfig,
		p.Activities,
		DrainWorkflows,
		p.Logger,
	)
}
