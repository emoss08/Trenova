package extractionshadowjobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentextractionjobs"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionshadowports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func workflows() []registry.WorkflowDefinition {
	defs := RegisterWorkflows()
	out := make([]registry.WorkflowDefinition, len(defs))
	for i, def := range defs {
		out[i] = registry.WorkflowDefinition{
			Name:        def.Name,
			Fn:          def.Fn,
			Description: def.Description,
		}
	}

	return out
}

type RegistryParams struct {
	fx.In

	Activities *Activities
	AIConfig   *aiconfig.Config
	Logger     *zap.Logger
}

func NewDrainingRegistry(p RegistryParams) registry.WorkerRegistry {
	return documentextractionjobs.DrainingRegistry(
		"extraction-shadow-worker",
		p.Activities,
		workflows(),
		p.AIConfig,
		p.Logger,
	)
}

func NewRegistry(p RegistryParams) registry.WorkerRegistry {
	return registry.NewDomainRegistry(
		&registry.DomainConfig{
			Name:         "extraction-shadow-worker",
			TaskQueue:    temporaltype.DocumentExtractionTaskQueue,
			WorkerConfig: documentextractionjobs.QueueWorkerConfig(p.AIConfig),
		},
		p.Activities,
		workflows(),
		p.Logger,
	)
}

var Module = fx.Module("extraction-shadow-jobs",
	fx.Provide(
		NewActivities,
		NewStarter,
		func(s *Starter) extractionshadowports.ExtractionShadowStarter { return s },
	),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
		fx.Annotate(
			NewDrainingRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
)
