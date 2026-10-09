package agenttoolruleservice

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

var LoaderModule = fx.Module("agent-tool-rule-loader",
	fx.Provide(
		NewLoader,
		func(l *Loader) services.ToolRuleOverrides { return l },
	),
)
