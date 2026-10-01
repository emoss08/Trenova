package completionrouter

import (
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

var Module = fx.Module("completion-router",
	fx.Provide(
		newService,
		asCompletionService,
		asEmbeddingService,
		asChatContextWindowResolver,
		NewPromptRenderer,
	),
)

func asCompletionService(s *Service) serviceports.CompletionService { return s }

func asChatContextWindowResolver(s *Service) serviceports.ChatContextWindowResolver { return s }

func asEmbeddingService(s *Service) serviceports.EmbeddingService { return s }
