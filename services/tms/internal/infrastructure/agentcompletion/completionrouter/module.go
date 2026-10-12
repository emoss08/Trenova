package completionrouter

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"go.uber.org/fx"
)

var Module = fx.Module("completion-router",
	fx.Provide(
		newService,
		asCompletionService,
		asEmbeddingService,
		NewPromptRenderer,
	),
)

func asCompletionService(s *Service) agentports.CompletionService { return s }

func asEmbeddingService(s *Service) aiusageports.EmbeddingService { return s }
