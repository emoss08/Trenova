package tooldispatch

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
)

// CodeModeGate decides from the chat providers' script settings, and is
// forced open on request only in a development or test deployment, where the
// desk bench runs.
func CodeModeGate(cfg *appconfig.Config) codemode.Gate {
	return codemode.Gate{
		Forced: cfg != nil && (cfg.IsDevelopment() || cfg.IsTest()),
	}
}

func ShadowFrom(resolver *agentshadow.Resolver) ShadowVerdicts {
	if resolver == nil {
		return nil
	}

	return resolver
}
