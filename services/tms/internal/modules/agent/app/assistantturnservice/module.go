package assistantturnservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantturnports"
	"go.uber.org/fx"
)

var Module = fx.Module("assistantturnservice",
	fx.Provide(New),
	fx.Provide(func(s *Service) assistantturnports.AssistantTurnStopper { return s }),
)
