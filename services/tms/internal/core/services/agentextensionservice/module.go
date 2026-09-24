package agentextensionservice

import (
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

var Module = fx.Module("agent-extension-service",
	fx.Provide(
		New,
		func(svc *Service) serviceports.AgentExtensionGate { return svc },
		func(svc *Service) serviceports.WebResearcher { return svc },
	),
)
