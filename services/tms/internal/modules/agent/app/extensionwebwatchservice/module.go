package extensionwebwatchservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"go.uber.org/fx"
)

var Module = fx.Module("extension-web-watch-service",
	fx.Provide(
		New,
		func(svc *Service) agentextensionports.WebWatcher { return svc },
		func(svc *Service) agentextensionports.WebWatchRunner { return svc },
		func(svc *Service) agentextensionports.WebWatchDeliveries { return svc },
	),
)
