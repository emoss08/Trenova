package assistantjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// DomainConfig puts interactive turns on their own queue.
//
// It is the reason the queue exists: a person watching a reply arrive must not
// wait behind a scheduled run that takes ten minutes. The worker is shaped for
// that — many turns at once, each one long and mostly idle, waiting on a model
// rather than on a CPU.
var DomainConfig = registry.DomainConfig{
	Name:      "assistant-worker",
	TaskQueue: temporaltype.TaskQueueAgentChat.String(),
	WorkerConfig: registry.WorkerConfig{
		MaxConcurrentActivityExecutionSize:     50,
		MaxConcurrentWorkflowTaskExecutionSize: 50,
		MaxConcurrentWorkflowTaskPollers:       2,
		MaxConcurrentActivityTaskPollers:       4,
		WorkerStopTimeout:                      registry.DefaultWorkerConfig().WorkerStopTimeout,
	},
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
