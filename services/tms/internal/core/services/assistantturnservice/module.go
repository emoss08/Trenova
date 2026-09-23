package assistantturnservice

import (
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

var Module = fx.Module("assistantturnservice",
	fx.Provide(New),
	fx.Provide(func(s *Service) serviceports.AssistantTurnStopper { return s }),
)
