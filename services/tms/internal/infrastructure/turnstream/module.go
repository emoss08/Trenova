package turnstream

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"go.uber.org/fx"
)

// Module provides the reader of a turn's stream to the API, which relays it.
var Module = fx.Module("turnstream",
	fx.Provide(fx.Annotate(New, fx.As(new(agentports.TurnStreamReader)))),
)
