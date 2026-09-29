package documentintelligencejobs

import (
	services "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"go.uber.org/fx"
)

var Module = fx.Module("document-intelligence-jobs",
	fx.Provide(NewActivities),
	fx.Provide(NewCaptureAnalyzer),
	fx.Provide(
		NewShadowPredictor,
		func(p *ShadowPredictor) services.ExtractionShadowPredictor { return p },
	),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
	fx.Provide(schedule.AsProvider(NewScheduleProvider)),
)
