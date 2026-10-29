package reportjobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/modules/workspace/jobs/reportcatalogjobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var DomainConfig = registry.DomainConfig{
	Name:      "report-worker",
	TaskQueue: temporaltype.ReportTaskQueue,
	WorkerConfig: registry.WorkerConfig{
		MaxConcurrentActivityExecutionSize:     4,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
		MaxConcurrentWorkflowTaskPollers:       2,
		MaxConcurrentActivityTaskPollers:       2,
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
	Catalog    *reportcatalogjobs.Activities
	Logger     *zap.Logger
}

func NewRegistry(p RegistryParams) registry.WorkerRegistry {
	return registry.NewComposedRegistry(registry.ComposedParams{
		Config:     &DomainConfig,
		Activities: []any{p.Activities, p.Catalog},
		Workflows:  Workflows,
		Logger:     p.Logger,
	})
}
