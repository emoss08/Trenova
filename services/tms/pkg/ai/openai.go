package ai

import (
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"

	"go.uber.org/fx"
)

type ClientParams struct {
	fx.In

	Config *config.Config
}

func NewOpenAIClient(p ClientParams) openai.Client {
	return openai.NewClient(option.WithAPIKey(p.Config.AI.OpenAIAPIKey))
}
