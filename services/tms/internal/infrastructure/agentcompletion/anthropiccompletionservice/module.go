package anthropiccompletionservice

import "go.uber.org/fx"

var Module = fx.Module("anthropic-completion-service",
	fx.Provide(New),
)
