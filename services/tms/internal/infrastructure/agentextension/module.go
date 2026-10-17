package agentextension

import (
	"github.com/emoss08/trenova/internal/infrastructure/agentextension/browseruseconnector"
	"github.com/emoss08/trenova/internal/infrastructure/agentextension/connectorkit"
	"github.com/emoss08/trenova/internal/infrastructure/agentextension/exaconnector"
	"github.com/emoss08/trenova/internal/infrastructure/agentextension/parallelconnector"
	"go.uber.org/fx"
)

var Module = fx.Module("agent-extension-infrastructure",
	connectorkit.Provide(exaconnector.New),
	connectorkit.Provide(parallelconnector.New),
	connectorkit.Provide(browseruseconnector.New),
)
