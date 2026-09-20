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
	"github.com/emoss08/trenova/shared/pulid"
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
	return s.runChat(ctx, req, nil)
}

// StreamChat is CompleteChat with the text handed to sink as it arrives.
func (s *Service) StreamChat(
	ctx context.Context,
	req *serviceports.ChatCompletionRequest,
	sink serviceports.ChatStreamSink,
) (*serviceports.ChatCompletionResult, error) {
	return s.runChat(ctx, req, sink)
}

func (s *Service) runChat(
	ctx context.Context,
	req *serviceports.ChatCompletionRequest,
	sink serviceports.ChatStreamSink,
) (*serviceports.ChatCompletionResult, error) {
	if !s.ai.AIEnabled() {
		return nil, errortypes.NewBusinessError(aiDisabledMessage)
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

	usable = preferFirst(usable, req.PreferredProviderID)

	var lastErr error
	for _, provider := range usable {
		result, emitted, attemptErr := s.attemptChat(ctx, provider, req, sink)
		if attemptErr == nil {
			return result, nil
		}

		if errors.Is(attemptErr, errRefused) {
			return nil, errortypes.NewBusinessError("The model declined this request")
		}

		// Once a provider has started answering, the reader has seen its words.
		// Handing the same question to the next provider would splice a second
		// answer onto the first, so the failure is reported instead.
		if emitted {
			return nil, fmt.Errorf(
				"chat provider %s failed after it started replying: %w",
				provider.Name, attemptErr,
			)
		}

		lastErr = attemptErr
		s.logger.Warn("chat provider attempt failed, falling through",
			zap.String("provider", provider.Name),
			zap.Error(attemptErr),
		)
	}

	return nil, fmt.Errorf("every configured chat provider failed: %w", lastErr)
}

// attemptChat runs the turn on one provider. The returned flag reports whether
// any text reached the sink, which decides whether a failure may fall through.
func (s *Service) attemptChat(
	ctx context.Context,
	provider *aiprovider.Provider,
	req *serviceports.ChatCompletionRequest,
	sink serviceports.ChatStreamSink,
) (*serviceports.ChatCompletionResult, bool, error) {
	adapter, err := s.adapters.Get(provider.Kind)
	if err != nil {
		return nil, false, err
	}

	apiKey, err := s.resolveAPIKey(provider)
	if err != nil {
		return nil, false, err
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = provider.ResolvedMaxTokens()
	}

	call := &modeladapter.Call{
		Provider: provider,
		APIKey:   apiKey,
		Client:   s.clientFor(provider),
		Request: &modeladapter.Request{
			System:    req.System,
			Messages:  req.Messages,
			Tools:     req.Tools,
			MaxTokens: maxTokens,
		},
	}

	var (
		resp    *modeladapter.Response
		emitted bool
	)
	if streamer, ok := adapter.(modeladapter.Streamer); ok && sink != nil {
		resp, emitted, err = s.executeStreamWithRetry(ctx, streamer, call, sink)
	} else {
		resp, err = s.executeWithRetry(ctx, adapter, call)
		if err == nil && sink != nil && resp.Text != "" {
			sink(resp.Text)
			emitted = true
		}
	}
	if err != nil {
		return nil, emitted, err
	}

	if resp.Refused {
		return nil, emitted, errRefused
	}

	// A turn with neither text nor a tool call is a dead end rather than an
	// answer, so it counts as a failure and the next provider gets a try.
	if resp.Text == "" && len(resp.ToolCalls) == 0 {
		return nil, emitted, errors.New("provider returned neither content nor a tool call")
	}

	return &serviceports.ChatCompletionResult{
		Text:            resp.Text,
		ToolCalls:       resp.ToolCalls,
		ModelIdentifier: resp.ModelIdentifier,
		InputTokens:     resp.InputTokens,
		OutputTokens:    resp.OutputTokens,
		ProviderID:      provider.ID,
		ProviderKind:    provider.Kind,
	}, emitted, nil
}

// executeStreamWithRetry retries a stream only while nothing has reached the
// sink. A retry after the first delta would replay text the reader already has.
func (s *Service) executeStreamWithRetry(
	ctx context.Context,
	streamer modeladapter.Streamer,
	call *modeladapter.Call,
	sink serviceports.ChatStreamSink,
) (*modeladapter.Response, bool, error) {
	attempts := s.cfg.GetAIMaxRetries()
	if attempts < 1 {
		attempts = 1
	}

	emitted := false
	tracked := func(delta string) {
		emitted = true
		sink(delta)
	}

	var lastErr error
	for attempt := range attempts {
		resp, err := streamer.Stream(ctx, call, tracked)
		if err == nil {
			return resp, emitted, nil
		}

		lastErr = err
		if emitted || !modeladapter.IsRetryable(err) || ctx.Err() != nil {
			return nil, emitted, err
		}

		s.logger.Debug("retrying provider stream",
			zap.String("provider", call.Provider.Name),
			zap.Int("attempt", attempt+1),
			zap.Error(err),
		)
	}

	return nil, emitted, lastErr
}

func preferFirst(providers []*aiprovider.Provider, preferred pulid.ID) []*aiprovider.Provider {
	if preferred.IsNil() {
		return providers
	}

	for idx, provider := range providers {
		if provider.ID != preferred {
			continue
		}
		if idx == 0 {
			return providers
		}

		ordered := make([]*aiprovider.Provider, 0, len(providers))
		ordered = append(ordered, provider)
		ordered = append(ordered, providers[:idx]...)
		ordered = append(ordered, providers[idx+1:]...)

		return ordered
	}

	return providers
}
