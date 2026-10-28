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
		asBatchCompletionService,
		asEmbeddingService,
		asProviderAvailability,
		asScriptProviders,
		NewPromptRenderer,
	),
)

func asCompletionService(s *Service) agentports.CompletionService { return s }

func asBatchCompletionService(s *Service) agentports.BatchCompletionService { return s }

func asEmbeddingService(s *Service) aiusageports.EmbeddingService { return s }

func asProviderAvailability(s *Service) agentports.ProviderAvailability { return s }

func asScriptProviders(s *Service) agentports.ScriptProviders { return s }
