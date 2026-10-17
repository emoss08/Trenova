package documentextractionjobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionshadowports"
	"go.uber.org/fx"
)

var Module = fx.Module("document-extraction-jobs",
	fx.Provide(NewActivities),
	fx.Provide(NewCaptureAnalyzer),
	fx.Provide(
		NewShadowPredictor,
		func(p *ShadowPredictor) extractionshadowports.ExtractionShadowPredictor { return p },
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
	fx.Provide(schedule.AsProvider(NewScheduleProvider)),
)
