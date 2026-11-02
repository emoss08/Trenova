package completionrouter

import (
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/adapterregistry"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"go.uber.org/fx"
)

var Module = fx.Module("completion-router",
	fx.Provide(
		adapterregistry.NewRegistry,
		newService,
		asCompletionService,
		asBatchCompletionService,
		asEmbeddingService,
		asProviderAvailability,
		asScriptProviders,
	),
)

func asCompletionService(s *Service) llmports.CompletionService { return s }

func asBatchCompletionService(s *Service) llmports.BatchCompletionService { return s }

func asEmbeddingService(s *Service) aiusageports.EmbeddingService { return s }

func asProviderAvailability(s *Service) llmports.ProviderAvailability { return s }

func asScriptProviders(s *Service) llmports.ScriptProviders { return s }
