package ai

import (
	"go.uber.org/fx"
)

var Module = fx.Module("ai", fx.Provide(
	NewOpenAIClient,
))
