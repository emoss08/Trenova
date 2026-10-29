package extensionportalservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"go.uber.org/fx"
)

var Module = fx.Module("extension-portal-service",
	fx.Provide(
		New,
		func(svc *Service) agentextensionports.PortalAutomator { return svc },
		func(svc *Service) agentextensionports.PortalSites { return svc },
		func(svc *Service) agentextensionports.PortalRunFinisher { return svc },
	),
)
