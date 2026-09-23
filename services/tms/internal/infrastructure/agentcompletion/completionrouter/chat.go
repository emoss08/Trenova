package completionrouter

import (
	"context"
	"errors"
	"fmt"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"strings"
	"time"
	"unicode/utf8"

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
//
// A reply whose context ends partway returns an error wrapping the context's
// own, so a person who pressed Stop is recorded as having stopped the turn
// rather than as having read a finished one. The result is still returned
// alongside when text had arrived, marked Truncated, holding what the reader
// was shown.
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
	// A pinned provider that is resting is not tried anyway: the person
	// chose it, but a choice of a provider that is down is a wait, not a
	// reply, and the error names the wait.
	usable, err = s.awake(usable)
	if err != nil {
		return nil, err
	}

	var lastErr error
	queue := append(make([]*aiprovider.Provider, 0, len(usable)+maxMidReplyRetries), usable...)
	midReplyRetries := 0
	for idx := 0; idx < len(queue); idx++ {
		// A person who pressed Stop is not answered by the next provider
		// either. Every further attempt would fail at once on the dead
		// context, and each would be a failure charged to a provider that
		// did nothing wrong.
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		provider := queue[idx]
		started := time.Now()
		result, streamed, attemptErr := s.attemptChat(ctx, provider, req, sink)
		latency := time.Since(started)
		attemptErr = stopped(ctx, attemptErr)
		s.observe(ctx, provider, attemptErr)
		s.record(ctx, usageAttempt{
			provider:    provider,
			task:        aiprovider.TaskAssistantChat,
			surface:     aiusage.SurfaceChat,
			attribution: req.Attribution,
			tenant:      req.TenantInfo,
			latency:     latency,
			streamed:    sink != nil,
			outcome:     chatOutcome(provider, result, streamed, attemptErr),
			err:         attemptErr,
		})
		if attemptErr == nil {
			result.LatencyMs = latency.Milliseconds()
			result.CostUSD = provider.CostFor(result.InputTokens, result.OutputTokens)

			return result, nil
		}

		// A stopped reply is an error, never a finished one: the turn is
		// recorded as Stopped only when the cancellation reaches it. The
		// words that had arrived still travel with it, for a caller that
		// keeps what the reader was shown.
		if ctx.Err() != nil {
			return partialReply(provider, streamed.text, attemptErr), attemptErr
		}

		if errors.Is(attemptErr, errRefused) {
			return nil, errortypes.NewBusinessError("The model declined this request")
		}

		// A provider that died partway through a reply gets replaced, not
		// spliced: the reader is told the reply is starting over and the words
		// that arrived are discarded, because half an answer followed by a
		// second model's whole one reads as neither. The retries are bounded.
		if emitted := streamed.text; emitted != "" {
			if midReplyRetries < maxMidReplyRetries {
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
							Kind:     serviceports.RetryKindRestart,
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

			return partialReply(provider, emitted, attemptErr), nil
		}

		lastErr = attemptErr
		s.logger.Warn("chat provider attempt failed, falling through",
			zap.String("provider", provider.Name),
			zap.Error(attemptErr),
		)
	}

	return nil, fmt.Errorf("every configured chat provider failed: %w", lastErr)
}

// chatStream is what one attempt put in front of the reader before it ended.
// A failed attempt has no result, but it may well have had these, and they
// decide both whether the failure may fall through and what it cost.
type chatStream struct {
	// text is the reply that reached the sink.
	text string
	// reasoningRunes is how much thinking the provider streamed.
	reasoningRunes int
}

// attemptChat runs the turn on one provider. The returned stream says what
// reached the reader, which decides whether a failure may fall through.
func (s *Service) attemptChat(
	ctx context.Context,
	provider *aiprovider.Provider,
	req *serviceports.ChatCompletionRequest,
	sink serviceports.ChatStreamSink,
) (*serviceports.ChatCompletionResult, chatStream, error) {
	var streamed chatStream

	adapter, err := s.adapters.Get(provider.Kind)
	if err != nil {
		return nil, streamed, err
	}

	apiKey, err := s.resolveAPIKey(provider)
	if err != nil {
		return nil, streamed, err
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
		StreamIdle:   s.ai.GetStreamIdleTimeout(),
		Request: &modeladapter.Request{
			System:    req.System,
			Messages:  req.Messages,
			Tools:     req.Tools,
			MaxTokens: maxTokens,
			// A chat turn drives tools, so it is sampled for exactness: the
			// model has to name a tool that exists and fill its arguments
			// with JSON that parses, and invention there is only ever a bug.
			Sampling: modeladapter.SamplingForTask(aiprovider.TaskAssistantChat),
		},
	}

	// The thinking is counted whether or not anyone is shown it: a stream
	// cut off before its usage frame reports no tokens, and thinking is
	// billed as output all the same.
	call.Reasoning = func(delta string) {
		streamed.reasoningRunes += utf8.RuneCountInString(delta)
		if req.ReasoningSink != nil {
			req.ReasoningSink(delta)
		}
	}

	var resp *modeladapter.Response
	// A busy provider being asked again is told to the reader, who is
	// otherwise watching nothing happen for the length of the wait.
	busy := func(attempt int, wait time.Duration, cause error) {
		if req.RetrySink == nil {
			return
		}
		req.RetrySink(serviceports.ChatRetryNotice{
			Attempt:     attempt,
			Provider:    provider.Name,
			Reason:      cause.Error(),
			Kind:        serviceports.RetryKindBusy,
			WaitSeconds: int(wait.Round(time.Second).Seconds()),
		})
	}

	if streamer, ok := adapter.(modeladapter.Streamer); ok && sink != nil {
		resp, streamed.text, err = s.executeStreamWithRetry(ctx, streamer, call, sink, busy)
	} else {
		resp, err = s.executeWithRetryNoticed(ctx, adapter, call, busy)
		if err == nil && sink != nil && resp.Text != "" {
			sink(resp.Text)
			streamed.text = resp.Text
		}
	}
	if err != nil {
		return nil, streamed, err
	}

	if resp.Refused {
		return nil, streamed, errRefused
	}

	// A turn with neither text nor a tool call is a dead end rather than an
	// answer, so it counts as a failure and the next provider gets a try.
	if resp.Text == "" && len(resp.ToolCalls) == 0 {
		return nil, streamed, errors.New("provider returned neither content nor a tool call")
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
	}, streamed, nil
}

// partialReply is the reply that arrived before a provider stopped writing,
// marked as cut off. It is nil when nothing arrived, since an empty reply
// marked as truncated would read as an answer that said nothing.
func partialReply(
	provider *aiprovider.Provider,
	text string,
	cause error,
) *serviceports.ChatCompletionResult {
	if text == "" {
		return nil
	}

	return &serviceports.ChatCompletionResult{
		Text:            text,
		ModelIdentifier: modeladapter.ServedModel(cause, provider.Model),
		ProviderID:      provider.ID,
		ProviderKind:    provider.Kind,
		Truncated:       true,
	}
}

// executeStreamWithRetry retries a stream only while nothing has reached the
// sink. A retry after the first delta would replay text the reader already has.
func (s *Service) executeStreamWithRetry(
	ctx context.Context,
	streamer modeladapter.Streamer,
	call *modeladapter.Call,
	sink serviceports.ChatStreamSink,
	busy busyNotice,
) (*modeladapter.Response, string, error) {
	// The text is kept as well as the fact of it. A stream that dies partway
	// leaves the reader watching an answer that then vanishes, and the words
	// that did arrive are the ones we can still give them.
	var partial strings.Builder
	tracked := func(delta string) {
		partial.WriteString(delta)
		sink(delta)
	}
	emittedText := func() string { return partial.String() }

	var waited time.Duration
	for attempt := 0; ; attempt++ {
		resp, err := streamer.Stream(ctx, call, tracked)
		if err == nil {
			return resp, emittedText(), nil
		}
		if emittedText() != "" || ctx.Err() != nil {
			return nil, emittedText(), err
		}

		wait, again := s.retryWait(err, attempt, waited)
		if !again {
			return nil, emittedText(), err
		}

		s.logger.Debug("retrying provider stream",
			zap.String("provider", call.Provider.Name),
			zap.Int("attempt", attempt+1),
			zap.Duration("wait", wait),
			zap.Error(err),
		)
		if busy != nil && unavailability(err) {
			busy(attempt+1, wait, err)
		}
		if waitErr := s.wait(ctx, wait); waitErr != nil {
			return nil, emittedText(), waitErr
		}
		waited += wait
	}
}

// busyNotice tells the reader a busy provider is being asked again.
type busyNotice func(attempt int, wait time.Duration, cause error)

// executeWithRetryNoticed is executeWithRetry with the reader told about
// each wait on a busy provider.
func (s *Service) executeWithRetryNoticed(
	ctx context.Context,
	adapter modeladapter.Adapter,
	call *modeladapter.Call,
	busy busyNotice,
) (*modeladapter.Response, error) {
	var waited time.Duration
	for attempt := 0; ; attempt++ {
		resp, err := adapter.Complete(ctx, call)
		if err == nil {
			return resp, nil
		}
		if ctx.Err() != nil {
			return nil, err
		}

		wait, again := s.retryWait(err, attempt, waited)
		if !again {
			return nil, err
		}
		if busy != nil && unavailability(err) {
			busy(attempt+1, wait, err)
		}
		if waitErr := s.wait(ctx, wait); waitErr != nil {
			return nil, waitErr
		}
		waited += wait
	}
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

// chatOutcome is what the usage record reads off a chat attempt. A finished
// attempt carries the provider's own count; one that failed or was stopped
// carries what was counted from its stream.
func chatOutcome(
	provider *aiprovider.Provider,
	result *serviceports.ChatCompletionResult,
	streamed chatStream,
	cause error,
) *runOutcome {
	if result == nil {
		return streamedOutcome(provider, streamed, cause)
	}

	return &runOutcome{
		Model:           result.ModelIdentifier,
		InputTokens:     result.InputTokens,
		OutputTokens:    result.OutputTokens,
		ReasoningTokens: result.ReasoningTokens,
	}
}
