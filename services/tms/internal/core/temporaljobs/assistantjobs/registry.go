package assistantjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/worker"
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
		MaxConcurrentActivityExecutionSize:     100,
		MaxConcurrentWorkflowTaskExecutionSize: 50,
		MaxConcurrentWorkflowTaskPollers:       4,
		MaxConcurrentActivityTaskPollers:       8,
		WorkerStopTimeout:                      registry.DefaultWorkerConfig().WorkerStopTimeout,
	},
}

type RegistryParams struct {
	fx.In

	Activities *Activities
	Flow       *agentflow.Activities
	Workflows  *Workflows
	Logger     *zap.Logger
}

// NewRegistry puts a turn's workflow and every activity it runs on the chat
// queue: its own first and last steps, the model call, the tool search, the
// publish, and every tool as the worker's dynamic activity.
func NewRegistry(p RegistryParams) *registry.ComposedRegistry {
	return registry.NewComposedRegistry(registry.ComposedParams{
		Config:     &DomainConfig,
		Activities: []any{p.Activities, p.Flow},
		Register: func(w worker.ActivityRegistry) {
			agentflow.RegisterDynamic(w, p.Flow)
		},
		Workflows: []registry.WorkflowDefinition{{
			Name:        AssistantTurnWorkflowName,
			Fn:          p.Workflows.AssistantTurnWorkflow,
			Description: "Answer one question in a conversation, publishing the reply as it is written",
		}},
		Logger: p.Logger,
	})
}
