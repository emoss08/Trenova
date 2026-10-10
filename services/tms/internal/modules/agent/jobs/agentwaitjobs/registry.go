package agentwaitjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var DomainConfig = registry.DomainConfig{
	Name:         "agent-wait-worker",
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
