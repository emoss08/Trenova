package agentjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/worker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// The agent's work is split across queues so one class cannot starve another.
// A registry binds one set of workflows to one queue, and each queue's worker
// registers the whole activity set: a run's model and tool calls run on the
// queue the run is on, and a heavy tool on the heavy queue, whichever queue
// called it.
var (
	BackgroundDomainConfig = registry.DomainConfig{
		Name:      "agent-background-worker",
		TaskQueue: temporaltype.TaskQueueAgentBackground.String(),
		// A run now schedules an activity per model call and per tool call,
		// most of them waiting on a model or a database rather than a CPU.
		WorkerConfig: registry.WorkerConfig{
			MaxConcurrentActivityExecutionSize:     50,
			MaxConcurrentWorkflowTaskExecutionSize: 20,
			MaxConcurrentWorkflowTaskPollers:       2,
			MaxConcurrentActivityTaskPollers:       4,
			WorkerStopTimeout:                      registry.DefaultWorkerConfig().WorkerStopTimeout,
		},
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

type RegistryParams struct {
	fx.In

	Activities *Activities
	Flow       *agentflow.Activities
	Workflows  *Workflows
	Logger     *zap.Logger
}

func newRegistry(
	p RegistryParams,
	config *registry.DomainConfig,
	workflows []registry.WorkflowDefinition,
) *registry.ComposedRegistry {
	return registry.NewComposedRegistry(registry.ComposedParams{
		Config:     config,
		Activities: []any{p.Activities, p.Flow},
		Register: func(w worker.ActivityRegistry) {
			agentflow.RegisterDynamic(w, p.Flow)
		},
		Workflows: workflows,
		Logger:    p.Logger,
	})
}

func NewBackgroundRegistry(p RegistryParams) registry.WorkerRegistry {
	return newRegistry(p, &BackgroundDomainConfig, p.Workflows.background())
}

func NewHeavyRegistry(p RegistryParams) registry.WorkerRegistry {
	return newRegistry(p, &HeavyDomainConfig, p.Workflows.heavy())
}

func NewDrainRegistry(p RegistryParams) registry.WorkerRegistry {
	return newRegistry(p, &DrainDomainConfig, p.Workflows.drain())
}
