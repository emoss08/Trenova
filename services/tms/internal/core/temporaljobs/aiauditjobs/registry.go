package aiauditjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/core/temporaljobs/reportjobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// DomainConfig shares the audit queue's worker. Its options match the audit
// jobs' so both register on one worker.
var DomainConfig = registry.DomainConfig{
	Name:         "ai-audit-worker",
	TaskQueue:    temporaltype.AuditTaskQueue,
	WorkerConfig: registry.DefaultWorkerConfig(),
}

// ExportDomainConfig shares the report queue's worker, whose options it
// takes from the report jobs.
var ExportDomainConfig = registry.DomainConfig{
	Name:         "ai-audit-export-worker",
	TaskQueue:    temporaltype.ReportTaskQueue,
	WorkerConfig: reportjobs.DomainConfig.WorkerConfig,
}

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
	return registry.NewComposedRegistry(registry.ComposedParams{
		Config:     &DomainConfig,
		Activities: []any{p.Activities},
		Workflows:  convertWorkflows(RegisterWorkflows()),
		Logger:     p.Logger,
	})
}

type ExportRegistryParams struct {
	fx.In

	Activities *ExportActivities
	Logger     *zap.Logger
}

func NewExportRegistry(p ExportRegistryParams) registry.WorkerRegistry {
	return registry.NewComposedRegistry(registry.ComposedParams{
		Config:     &ExportDomainConfig,
		Activities: []any{p.Activities},
		Workflows:  convertWorkflows(RegisterExportWorkflows()),
		Logger:     p.Logger,
	})
}
