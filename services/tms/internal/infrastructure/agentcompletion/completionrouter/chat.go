package completionrouter

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/zap"
)

// CompleteChat runs one conversational turn with tools.
//
// It deliberately does not loop. Deciding whether a requested tool may run — and
// whether it runs at all or becomes a proposal for a person — depends on the
// agent's configuration, which this layer knows nothing about. Looping here
// would mean executing tools without that check.
func (s *Service) CompleteChat(
	ctx context.Context,
	req *serviceports.ChatCompletionRequest,
) (*serviceports.ChatCompletionResult, error) {
	if !s.cfg.AIEnabled() {
		return nil, errortypes.NewBusinessError("AI features are disabled")
	}

	candidates, err := s.repo.ListForTask(ctx, repositories.ListAIProvidersForTaskRequest{
		Task:       aiprovider.TaskAssistantChat,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	usable := make([]*aiprovider.Provider, 0, len(candidates))
	for _, provider := range candidates {
		if ok, reason := provider.CanServeTask(aiprovider.TaskAssistantChat); ok {
			usable = append(usable, provider)
		} else {
			s.logger.Debug("skipping provider for chat",
				zap.String("provider", provider.Name),
				zap.String("reason", reason),
			)
		}
	}

	if len(usable) == 0 {
		return nil, errortypes.NewBusinessError(
			"No AI provider is configured for {0}", string(aiprovider.TaskAssistantChat),
		).WithInternal(serviceports.ErrNoProviderConfigured)
	}

	var lastErr error
	for _, provider := range usable {
		result, attemptErr := s.attemptChat(ctx, provider, req)
		if attemptErr == nil {
			return result, nil
		}

		if errors.Is(attemptErr, errRefused) {
			return nil, errortypes.NewBusinessError("The model declined this request")
		}

		lastErr = attemptErr
		s.logger.Warn("chat provider attempt failed, falling through",
			zap.String("provider", provider.Name),
			zap.Error(attemptErr),
		)
	}

	return nil, fmt.Errorf("every configured chat provider failed: %w", lastErr)
}

func (s *Service) attemptChat(
	ctx context.Context,
	provider *aiprovider.Provider,
	req *serviceports.ChatCompletionRequest,
) (*serviceports.ChatCompletionResult, error) {
	adapter, err := s.adapters.Get(provider.Kind)
	if err != nil {
		return nil, err
	}

	apiKey, err := s.resolveAPIKey(provider)
	if err != nil {
		return nil, err
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = provider.ResolvedMaxTokens()
	}

	resp, err := s.executeWithRetry(ctx, adapter, &modeladapter.Call{
		Provider: provider,
		APIKey:   apiKey,
		Client:   s.clientFor(provider),
		Request: &modeladapter.Request{
			System:    req.System,
			Messages:  req.Messages,
			Tools:     req.Tools,
			MaxTokens: maxTokens,
		},
	})
	if err != nil {
		return nil, err
	}

	if resp.Refused {
		return nil, errRefused
	}

	// A turn with neither text nor a tool call is a dead end rather than an
	// answer, so it counts as a failure and the next provider gets a try.
	if resp.Text == "" && len(resp.ToolCalls) == 0 {
		return nil, errors.New("provider returned neither content nor a tool call")
	}

	return &serviceports.ChatCompletionResult{
		Text:            resp.Text,
		ToolCalls:       resp.ToolCalls,
		ModelIdentifier: resp.ModelIdentifier,
		InputTokens:     resp.InputTokens,
		OutputTokens:    resp.OutputTokens,
		ProviderID:      provider.ID,
		ProviderKind:    provider.Kind,
	}, nil
}
