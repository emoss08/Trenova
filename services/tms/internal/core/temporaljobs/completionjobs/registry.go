package completionjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/assistantjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// DomainConfig puts one-shot calls on the chat queue, beside the turns: in
// both a person is waiting. The two registries share the queue's one worker,
// which is why the worker options are the chat queue's own.
var DomainConfig = registry.DomainConfig{
	Name:         "one-shot-worker",
	TaskQueue:    assistantjobs.DomainConfig.TaskQueue,
	WorkerConfig: assistantjobs.DomainConfig.WorkerConfig,
}

type RegistryParams struct {
	fx.In

	Activities *Activities
	Logger     *zap.Logger
}

func NewRegistry(p RegistryParams) *registry.ComposedRegistry {
	return registry.NewComposedRegistry(registry.ComposedParams{
		Config:     &DomainConfig,
		Activities: []any{p.Activities},
		Workflows: []registry.WorkflowDefinition{
			{
				Name:        StructuredCompletionWorkflowName,
				Fn:          StructuredCompletionWorkflow,
				Description: "Ask the model one structured question for a person waiting on the answer",
			},
			{
				Name:        TestAIProviderWorkflowName,
				Fn:          TestAIProviderWorkflow,
				Description: "Probe an AI provider once and record whether it answered",
			},
			{
				Name:        WriteBriefingWorkflowName,
				Fn:          WriteBriefingWorkflow,
				Description: "Write a day's briefing again for a person who asked",
			},
		},
		Logger: p.Logger,
	})
}
