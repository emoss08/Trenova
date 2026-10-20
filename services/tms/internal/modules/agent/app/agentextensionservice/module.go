package agentextensionservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"go.uber.org/fx"
)

var Module = fx.Module("agent-extension-service",
	fx.Provide(
		New,
		func(svc *Service) agentextensionports.AgentExtensionGate { return svc },
		func(svc *Service) agentextensionports.WebResearcher { return svc },
		func(svc *Service) agentextensionports.PortalAutomator { return svc },
		func(svc *Service) agentextensionports.PortalSites { return svc },
		func(svc *Service) agentextensionports.PortalRunFinisher { return svc },
		func(svc *Service) agentextensionports.DataAnalyzer { return svc },
		func(svc *Service) agentextensionports.ScriptRunner { return svc },
		func(svc *Service) agentextensionports.AnalysisRuns { return svc },
		func(svc *Service) agentextensionports.WebWatcher { return svc },
		func(svc *Service) agentextensionports.WebWatchRunner { return svc },
		func(svc *Service) agentextensionports.WebWatchDeliveries { return svc },
	),
)
