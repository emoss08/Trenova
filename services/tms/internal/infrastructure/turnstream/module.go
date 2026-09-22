package turnstream

import (
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

// Module provides the turn stream to both processes: the worker publishes to
// it and the API relays from it, so it belongs in the options they share.
var Module = fx.Module("turnstream",
	fx.Provide(
		New,
		func(s *Service) serviceports.TurnStreamPublisher { return s },
		func(s *Service) serviceports.TurnStreamReader { return s },
	),
)
