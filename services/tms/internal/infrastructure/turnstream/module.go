package turnstream

import (
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

// Module provides the reader of a turn's stream to the API, which relays it.
var Module = fx.Module("turnstream",
	fx.Provide(fx.Annotate(New, fx.As(new(serviceports.TurnStreamReader)))),
)
