package completionrouter

import (
	"context"
	"errors"
	"fmt"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"strings"
	"time"

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
	if req.PinPreferred {
		usable = pinPreferred(usable, req.PreferredProviderID)
	}

	var lastErr error
	queue := append(make([]*aiprovider.Provider, 0, len(usable)+maxMidReplyRetries), usable...)
	midReplyRetries := 0
	for idx := 0; idx < len(queue); idx++ {
		provider := queue[idx]
		started := time.Now()
		result, emitted, attemptErr := s.attemptChat(ctx, provider, req, sink)
		latency := time.Since(started)
		s.record(ctx, usageAttempt{
			provider:    provider,
			task:        aiprovider.TaskAssistantChat,
			surface:     aiusage.SurfaceChat,
			attribution: req.Attribution,
			tenant:      req.TenantInfo,
			latency:     latency,
			streamed:    sink != nil,
			outcome:     chatOutcome(result),
			err:         attemptErr,
		})
		if attemptErr == nil {
			result.LatencyMs = latency.Milliseconds()
			result.CostUSD = provider.CostFor(result.InputTokens, result.OutputTokens)
		}
		if attemptErr == nil {
			return result, nil
		}

		if errors.Is(attemptErr, errRefused) {
			return nil, errortypes.NewBusinessError("The model declined this request")
		}

		// A provider that died partway through a reply gets replaced, not
		// spliced: the reader is told the reply is starting over and the words
		// that arrived are discarded, because half an answer followed by a
		// second model's whole one reads as neither. The retries are bounded,
		// and a person who pressed Stop is not retried at all.
		if emitted != "" {
			if ctx.Err() == nil && midReplyRetries < maxMidReplyRetries {
				midReplyRetries++
				if idx == len(queue)-1 && modeladapter.IsRetryable(attemptErr) {
					queue = append(queue, provider)
				}
				if idx < len(queue)-1 {
					s.logger.Warn("chat provider stopped partway through a reply; starting over",
						zap.String("provider", provider.Name),
						zap.Int("characters", len(emitted)),
						zap.Int("retry", midReplyRetries),
						zap.Error(attemptErr),
					)
					if req.RetrySink != nil {
						req.RetrySink(serviceports.ChatRetryNotice{
							Attempt:  midReplyRetries,
							Provider: queue[idx+1].Name,
							Reason:   attemptErr.Error(),
						})
					}
					lastErr = attemptErr

					continue
				}
			}

			// Nothing left to try. What arrived is kept rather than thrown
			// away: the reader watched it appear, and the half that arrived
			// is usually the half that answered. The result says it was cut
			// off, and everything downstream treats it as a finished turn
			// with a truncated answer.
			s.logger.Warn("chat provider stopped partway through a reply",
				zap.String("provider", provider.Name),
				zap.Int("characters", len(emitted)),
				zap.Error(attemptErr),
			)

			return &serviceports.ChatCompletionResult{
				Text:            emitted,
				ModelIdentifier: modeladapter.ServedModel(attemptErr, provider.Model),
				ProviderID:      provider.ID,
				ProviderKind:    provider.Kind,
				Truncated:       true,
			}, nil
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
) (*serviceports.ChatCompletionResult, string, error) {
	adapter, err := s.adapters.Get(provider.Kind)
	if err != nil {
		return nil, "", err
	}

	apiKey, err := s.resolveAPIKey(provider)
	if err != nil {
		return nil, "", err
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = provider.ResolvedMaxTokens()
	}

	call := &modeladapter.Call{
		Provider:     provider,
		APIKey:       apiKey,
		Client:       s.clientFor(provider),
		StreamClient: s.streamClientFor(provider),
		StreamIdle:   s.cfg.GetAIStreamIdleTimeout(),
		Request: &modeladapter.Request{
			System:    req.System,
			Messages:  req.Messages,
			Tools:     req.Tools,
			MaxTokens: maxTokens,
		},
	}

	if req.ReasoningSink != nil {
		call.Reasoning = modeladapter.StreamSink(req.ReasoningSink)
	}

	var (
		resp    *modeladapter.Response
		emitted string
	)
	if streamer, ok := adapter.(modeladapter.Streamer); ok && sink != nil {
		resp, emitted, err = s.executeStreamWithRetry(ctx, streamer, call, sink)
	} else {
		resp, err = s.executeWithRetry(ctx, adapter, call)
		if err == nil && sink != nil && resp.Text != "" {
			sink(resp.Text)
			emitted = resp.Text
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
		Truncated:       resp.Truncated,
		Reasoning:       resp.Reasoning,
		ReasoningTokens: resp.ReasoningTokens,
	}, emitted, nil
}

// executeStreamWithRetry retries a stream only while nothing has reached the
// sink. A retry after the first delta would replay text the reader already has.
func (s *Service) executeStreamWithRetry(
	ctx context.Context,
	streamer modeladapter.Streamer,
	call *modeladapter.Call,
	sink serviceports.ChatStreamSink,
) (*modeladapter.Response, string, error) {
	attempts := s.cfg.GetAIMaxRetries()
	if attempts < 1 {
		attempts = 1
	}

	// The text is kept as well as the fact of it. A stream that dies partway
	// leaves the reader watching an answer that then vanishes, and the words
	// that did arrive are the ones we can still give them.
	var partial strings.Builder
	tracked := func(delta string) {
		partial.WriteString(delta)
		sink(delta)
	}
	emittedText := func() string { return partial.String() }

	var lastErr error
	for attempt := range attempts {
		resp, err := streamer.Stream(ctx, call, tracked)
		if err == nil {
			return resp, emittedText(), nil
		}

		lastErr = err
		if emittedText() != "" || !modeladapter.IsRetryable(err) || ctx.Err() != nil {
			return nil, emittedText(), err
		}

		s.logger.Debug("retrying provider stream",
			zap.String("provider", call.Provider.Name),
			zap.Int("attempt", attempt+1),
			zap.Error(err),
		)
		if waitErr := waitBeforeRetry(ctx, attempt); waitErr != nil {
			return nil, emittedText(), waitErr
		}
	}

	return nil, emittedText(), lastErr
}

// maxMidReplyRetries bounds how many times a turn starts over after a
// provider died with words already on the reader's screen. Two is enough to
// get past one bad provider and one bad moment; more is a reader watching
// the same question asked and abandoned again and again.
const maxMidReplyRetries = 2

// pinPreferred keeps only the preferred provider when it is among the
// usable ones. A preference that is not usable leaves the order alone, so a
// deleted or disabled choice degrades to the organization's order rather
// than to nothing.
func pinPreferred(providers []*aiprovider.Provider, preferred pulid.ID) []*aiprovider.Provider {
	if preferred.IsNil() {
		return providers
	}
	for _, provider := range providers {
		if provider.ID == preferred {
			return []*aiprovider.Provider{provider}
		}
	}

	return providers
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

// chatOutcome is what the usage record reads off a chat result: nil for a
// failed attempt, whose tokens the provider never reported.
func chatOutcome(result *serviceports.ChatCompletionResult) *runOutcome {
	if result == nil {
		return nil
	}

	return &runOutcome{
		Model:           result.ModelIdentifier,
		InputTokens:     result.InputTokens,
		OutputTokens:    result.OutputTokens,
		ReasoningTokens: result.ReasoningTokens,
	}
}
