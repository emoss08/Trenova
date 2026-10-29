package completionrouter

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/completionbackground"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/completionbatch"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/completionchat"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/completionembed"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/completionengine"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type Service struct {
	*completionengine.Engine
	*completionchat.Chat
	*completionbatch.Batch
	*completionembed.Embedder
	*completionbackground.Background
}

func New(
	p completionengine.Params, //nolint:gocritic // fx params are passed by value
) llmports.CompletionService {
	return newService(&p)
}

func newService(p *completionengine.Params) *Service {
	return compose(completionengine.NewEngine(p))
}

func compose(engine *completionengine.Engine) *Service {
	return &Service{
		Engine:     engine,
		Chat:       &completionchat.Chat{Engine: engine},
		Batch:      &completionbatch.Batch{Engine: engine},
		Embedder:   &completionembed.Embedder{Engine: engine},
		Background: &completionbackground.Background{Engine: engine},
	}
}

func (s *Service) CompleteStructured(
	ctx context.Context,
	req *llmports.StructuredCompletionRequest,
) (*llmports.StructuredCompletionResult, error) {
	if !s.AI.AIEnabled() {
		return nil, errortypes.NewBusinessError(completionengine.AIDisabledMessage)
	}

	if err := s.AssertWithinSpend(ctx, req.TenantInfo); err != nil {
		return nil, err
	}

	outcome, err := s.Run(ctx, completionengine.StructuredRun(req))
	if err != nil {
		return nil, err
	}

	return &llmports.StructuredCompletionResult{
		Text:            outcome.Text,
		ModelIdentifier: outcome.Model,
		InputTokens:     outcome.InputTokens,
		OutputTokens:    outcome.OutputTokens,
		LatencyMs:       outcome.LatencyMs,
		CostUSD:         outcome.CostUSD,
		ProviderID:      outcome.ProviderID,
		ProviderKind:    outcome.ProviderKind,

		CacheReadTokens:  outcome.CacheReadTokens,
		CacheWriteTokens: outcome.CacheWriteTokens,
	}, nil
}
