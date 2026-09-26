package extractionevaljobs

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/documentintelligencejobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/config"
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
	Config     *config.Config
	Logger     *zap.Logger
}

func NewRegistry(p RegistryParams) registry.WorkerRegistry {
	return registry.NewDomainRegistry(
		&registry.DomainConfig{
			Name:         "extraction-eval-worker",
			TaskQueue:    temporaltype.DocumentIntelligenceTaskQueue,
			WorkerConfig: documentintelligencejobs.QueueWorkerConfig(p.Config),
		},
		p.Activities,
		workflows(),
		p.Logger,
	)
}

var Module = fx.Module("extraction-eval-jobs",
	fx.Provide(
		NewActivities,
		NewPredictor,
		NewRunStarter,
		func(p *Predictor) services.ExtractionPredictor { return p },
		func(s *RunStarter) services.ExtractionEvalRunStarter { return s },
	),
	fx.Provide(fx.Annotate(
		NewRegistry,
		fx.As(new(registry.WorkerRegistry)),
		fx.ResultTags(`group:"worker_registries"`),
	)),
)
