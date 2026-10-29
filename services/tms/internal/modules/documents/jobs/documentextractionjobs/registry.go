package documentextractionjobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	Workflows           = convertWorkflows(RegisterExtractionWorkflows())
	DrainingWorkflows   = convertWorkflows(RegisterWorkflows())
	drainingQueueSuffix = "-draining"
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
	AIConfig   *aiconfig.Config
	Logger     *zap.Logger
}

func QueueWorkerConfig(cfg *aiconfig.Config) registry.WorkerConfig {
	workerConfig := registry.DefaultWorkerConfig()
	workerConfig.MaxConcurrentActivityExecutionSize = cfg.GetMaxConcurrentActivities()
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
			Name:         "document-extraction-worker",
			TaskQueue:    temporaltype.DocumentExtractionTaskQueue,
			WorkerConfig: QueueWorkerConfig(p.AIConfig),
		},
		p.Activities,
		Workflows,
		p.Logger,
	)
}

func NewDrainingRegistry(p RegistryParams) registry.WorkerRegistry {
	return DrainingRegistry(
		"document-intelligence-worker",
		p.Activities,
		DrainingWorkflows,
		p.AIConfig,
		p.Logger,
	)
}

func DrainingRegistry(
	name string,
	activities any,
	workflows []registry.WorkflowDefinition,
	cfg *aiconfig.Config,
	logger *zap.Logger,
) registry.WorkerRegistry {
	return registry.NewDomainRegistry(
		&registry.DomainConfig{
			Name:         name + drainingQueueSuffix,
			TaskQueue:    temporaltype.DocumentIntelligenceTaskQueue,
			WorkerConfig: QueueWorkerConfig(cfg),
		},
		activities,
		workflows,
		logger,
	)
}
