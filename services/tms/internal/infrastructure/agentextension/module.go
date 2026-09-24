package agentextension

import (
	"github.com/emoss08/trenova/internal/infrastructure/agentextension/exaconnector"
	"go.uber.org/fx"
)

var Module = fx.Module("agent-extension-infrastructure",
	fx.Provide(exaconnector.New),
)
