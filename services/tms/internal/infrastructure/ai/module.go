package ai

import (
	"github.com/emoss08/trenova/pkg/ai"
	"go.uber.org/fx"
)

var Module = fx.Module("ai", fx.Provide(
	ai.NewOpenAIClient,
))
