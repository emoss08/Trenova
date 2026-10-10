package agenttoolruleservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"go.uber.org/fx"
)

var LoaderModule = fx.Module("agent-tool-rule-loader",
	fx.Provide(
		NewLoader,
		func(l *Loader) agentports.ToolRuleOverrides { return l },
	),
)
