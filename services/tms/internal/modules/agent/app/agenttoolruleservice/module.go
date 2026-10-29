package agenttoolruleservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"go.uber.org/fx"
)

var LoaderModule = fx.Module("agent-tool-rule-loader",
	fx.Provide(
		NewLoader,
		func(l *Loader) agenttoolports.ToolRuleOverrides { return l },
	),
)
