package importassistantjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/assistantjobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// DomainConfig puts the import assistant's turns on the chat queue, beside the
// assistant's: in both a person is watching the reply. The registries share
// the queue's one worker, so the worker options are the chat queue's own.
var DomainConfig = registry.DomainConfig{
	Name:         "import-assistant-worker",
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
		Workflows: []registry.WorkflowDefinition{{
			Name:        ImportAssistantTurnWorkflowName,
			Fn:          ImportAssistantTurnWorkflow,
			Description: "Answer one message to the shipment import assistant, streaming the reply",
		}},
		Logger: p.Logger,
	})
}
