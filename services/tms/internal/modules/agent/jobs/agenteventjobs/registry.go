package agenteventjobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentjobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// DomainConfig shares the agent background queue, so its worker options are
// that queue's: the manager runs one worker per queue.
var DomainConfig = registry.DomainConfig{
	Name:         "agent-event-worker",
	TaskQueue:    temporaltype.TaskQueueAgentBackground.String(),
	WorkerConfig: agentjobs.BackgroundDomainConfig.WorkerConfig,
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
