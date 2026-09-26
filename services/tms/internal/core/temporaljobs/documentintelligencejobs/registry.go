package documentintelligencejobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

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
	Config     *config.Config
	Logger     *zap.Logger
}

func QueueWorkerConfig(cfg *config.Config) registry.WorkerConfig {
	workerConfig := registry.DefaultWorkerConfig()
	workerConfig.MaxConcurrentActivityExecutionSize = cfg.GetAIConfig().
		GetMaxConcurrentActivities()
	workerConfig.MaxConcurrentWorkflowTaskExecutionSize = max(
		2, workerConfig.MaxConcurrentActivityExecutionSize,
	)
	workerConfig.MaxConcurrentActivityTaskPollers = 2
	workerConfig.MaxConcurrentWorkflowTaskPollers = 2

	return workerConfig
}

func NewRegistry(p RegistryParams) registry.WorkerRegistry {
	return registry.NewDomainRegistry(
		&registry.DomainConfig{
			Name:         "document-intelligence-worker",
			TaskQueue:    temporaltype.DocumentIntelligenceTaskQueue,
			WorkerConfig: QueueWorkerConfig(p.Config),
		},
		p.Activities,
		Workflows,
		p.Logger,
	)
}
