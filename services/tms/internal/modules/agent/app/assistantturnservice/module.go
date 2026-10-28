package assistantturnservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"go.uber.org/fx"
)

var Module = fx.Module("assistantturnservice",
	fx.Provide(New),
	fx.Provide(func(s *Service) assistantports.AssistantTurnStopper { return s }),
)
