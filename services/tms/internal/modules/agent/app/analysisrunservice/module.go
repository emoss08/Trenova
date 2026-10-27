package analysisrunservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/analysisports"
	"go.uber.org/fx"
)

var Module = fx.Module("agent-analysis-runs",
	fx.Provide(
		New,
		func(svc *Service) analysisports.RunRecorder { return svc },
		func(svc *Service) analysisports.Runs { return svc },
	),
)
