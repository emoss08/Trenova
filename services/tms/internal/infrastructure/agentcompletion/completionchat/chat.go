package completionchat

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/completionengine"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type Chat struct {
	*completionengine.Engine
}

const (
	failureStatusFailed      = "Failed"
	failureStatusTimedOut    = "Timed out"
	failureStatusOverloaded  = "Overloaded"
	failureStatusUnavailable = "Unavailable"
)

// CompleteChat runs one conversational turn with tools.
//
// It deliberately does not loop. Deciding whether a requested tool may run — and
// whether it runs at all or becomes a proposal for a person — depends on the
// agent's configuration, which this layer knows nothing about. Looping here
// would mean executing tools without that check.
func (s *Chat) CompleteChat(
	ctx context.Context,
	req *llmports.ChatCompletionRequest,
) (*llmports.ChatCompletionResult, error) {
	return s.runChat(ctx, req, nil)
}

// StreamChat is CompleteChat with the text handed to sink as it arrives.
//
// A reply whose context ends partway returns an error wrapping the context's
// own, so a person who pressed Stop is recorded as having stopped the turn
// rather than as having read a finished one. The result is still returned
// alongside when text had arrived, marked Truncated, holding what the reader
// was shown.
func (s *Chat) StreamChat(
	ctx context.Context,
	req *llmports.ChatCompletionRequest,
	sink llmports.ChatStreamSink,
) (*llmports.ChatCompletionResult, error) {
	return s.runChat(ctx, req, sink)
}

func (s *Chat) runChat(
	ctx context.Context,
	req *llmports.ChatCompletionRequest,
	sink llmports.ChatStreamSink,
) (*llmports.ChatCompletionResult, error) {
	usable, err := s.chatProviders(ctx, req)
	if err != nil {
		return nil, err
	}

	run := &chatRun{
		req:   req,
		sink:  sink,
		queue: append(make([]*aiprovider.Provider, 0, len(usable)+MaxMidReplyRetries), usable...),
	}
	for idx := 0; idx < len(run.queue); idx++ {
		// A person who pressed Stop is not answered by the next provider
		// either. Every further attempt would fail at once on the dead
		// context, and each would be a failure charged to a provider that
		// did nothing wrong.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}

		provider := run.queue[idx]
		attempt := s.attempt(ctx, run, idx)
		if attempt.err == nil {
			return attempt.result, nil
		}

		// A stopped reply is an error, never a finished one: the turn is
		// recorded as Stopped only when the cancellation reaches it. The
		// words that had arrived still travel with it, for a caller that
		// keeps what the reader was shown.
		if ctx.Err() != nil {
			return partialReply(provider, attempt.streamed.text, attempt.err), attempt.err
		}

		if errors.Is(attempt.err, completionengine.ErrRefused) {
			return nil, errortypes.NewBusinessError("The model declined this request")
		}

		if emitted := attempt.streamed.text; emitted != "" {
			if s.restartReply(ctx, run, idx, attempt) {
				continue
			}

			// Nothing left to try. What arrived is kept rather than thrown
			// away: the reader watched it appear, and the half that arrived
			// is usually the half that answered. The result says it was cut
			// off, and everything downstream treats it as a finished turn
			// with a truncated answer.
			s.Logger.Warn("chat provider stopped partway through a reply",
				zap.String("provider", provider.Name),
				zap.Int("characters", len(emitted)),
				zap.Error(attempt.err),
			)

			return partialReply(provider, emitted, attempt.err), nil
		}

		run.fail(provider, attempt)
		s.Logger.Warn("chat provider attempt failed, falling through",
			zap.String("provider", provider.Name),
			zap.Error(attempt.err),
		)
	}

	return nil, &llmports.ChatProvidersFailedError{
		Failures: append(distinctFailures(run.failures), s.notSetUp(ctx, req, usable)...),
		Err:      run.lastErr,
	}
}

func (s *Chat) chatProviders(
	ctx context.Context,
	req *llmports.ChatCompletionRequest,
) ([]*aiprovider.Provider, error) {
	if !s.AI.AIEnabled() {
		return nil, errortypes.NewBusinessError(completionengine.AIDisabledMessage)
	}
	if err := s.AssertWithinSpend(ctx, req.TenantInfo); err != nil {
		return nil, err
	}

	usable, err := s.UsableFor(ctx, aitask.TaskAssistantChat, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	usable = completionengine.PreferFirst(usable, req.PreferredProviderID)
	if req.PinPreferred {
		usable = pinPreferred(usable, req.PreferredProviderID)
	}
	// A pinned provider that is resting is not tried anyway: the person
	// chose it, but a choice of a provider that is down is a wait, not a
	// reply, and the error names the wait.
	usable, err = s.Awake(ctx, usable)
	if err != nil {
		return nil, err
	}

	return s.UnderCap(ctx, req.TenantInfo, usable)
}

type chatRun struct {
	req             *llmports.ChatCompletionRequest
	sink            llmports.ChatStreamSink
	queue           []*aiprovider.Provider
	failures        []llmports.ChatProviderFailure
	lastErr         error
	midReplyRetries int
}

func (r *chatRun) fail(provider *aiprovider.Provider, attempt *chatAttempt) {
	r.lastErr = attempt.err
	r.failures = append(
		r.failures,
		providerFailure(provider, attempt.err, attempt.elapsed, attempt.streamed.retries+1),
	)
}

type chatAttempt struct {
	result   *llmports.ChatCompletionResult
	streamed chatStream
	err      error
	elapsed  time.Duration
}

func (s *Chat) attempt(ctx context.Context, run *chatRun, idx int) *chatAttempt {
	req := run.req
	provider := run.queue[idx]
	failover := provider.ID != run.queue[0].ID
	attemptCtx, span := s.StartAttempt(ctx, &completionengine.AttemptSpec{
		Operation:   aitrace.OperationChat,
		Provider:    provider,
		Attempt:     idx + 1,
		Failover:    failover,
		MaxTokens:   req.MaxTokens,
		Attribution: req.Attribution,
	})
	started := time.Now()
	attemptStart := time.Now()
	result, streamed, attemptErr := s.attemptChat(
		completionengine.LastCandidate(attemptCtx, idx, len(run.queue)), provider, req, run.sink,
	)
	elapsed := time.Since(attemptStart)
	latency := time.Since(started)
	attemptErr = completionengine.Stopped(ctx, attemptErr)
	s.Observe(ctx, provider, attemptErr)
	outcome := chatOutcome(provider, result, &streamed, attemptErr)
	s.SettleAttempt(attemptCtx, span, &completionengine.UsageAttempt{
		Provider:    provider,
		Task:        aitask.TaskAssistantChat,
		Surface:     completionengine.SurfaceFor(aiusage.SurfaceChat, &req.Attribution),
		Attribution: req.Attribution,
		Tenant:      req.TenantInfo,
		Latency:     latency,
		FirstToken:  streamed.FirstToken(started),
		Streamed:    run.sink != nil,
		Outcome:     outcome,
		Err:         attemptErr,
		Operation:   aitrace.OperationChat,
		Attempt:     idx + 1,
		Failover:    failover,
	})
	if attemptErr == nil {
		s.TouchKey(ctx, provider, req.TenantInfo)
		result.LatencyMs = latency.Milliseconds()
		if first := firstOtherThan(run.failures, provider.ID); first != nil {
			result.FallbackFrom = first
		}
		result.CostUSD = outcome.Cost(provider, aitask.TaskAssistantChat)
	}

	return &chatAttempt{result: result, streamed: streamed, err: attemptErr, elapsed: elapsed}
}

// restartReply replaces a provider that died partway through a reply rather
// than splicing it: the reader is told the reply is starting over and the
// words that arrived are discarded, because half an answer followed by a
// second model's whole one reads as neither. The retries are bounded.
func (s *Chat) restartReply(
	ctx context.Context,
	run *chatRun,
	idx int,
	attempt *chatAttempt,
) bool {
	if run.midReplyRetries >= MaxMidReplyRetries {
		return false
	}
	run.midReplyRetries++
	aitrace.TallyFrom(ctx).Restarted()
	provider := run.queue[idx]
	if idx == len(run.queue)-1 && modeladapter.IsRetryable(attempt.err) {
		run.queue = append(run.queue, provider)
	}
	if idx >= len(run.queue)-1 {
		return false
	}
	s.Logger.Warn("chat provider stopped partway through a reply; starting over",
		zap.String("provider", provider.Name),
		zap.Int("characters", len(attempt.streamed.text)),
		zap.Int("retry", run.midReplyRetries),
		zap.Error(attempt.err),
	)
	if run.req.RetrySink != nil {
		run.req.RetrySink(llmports.ChatRetryNotice{
			Attempt:     run.midReplyRetries,
			Provider:    run.queue[idx+1].Name,
			Reason:      attempt.err.Error(),
			Kind:        llmports.RetryKindRestart,
			MaxAttempts: MaxMidReplyRetries + 1,
		})
	}
	run.fail(provider, attempt)

	return true
}

// providerFailure names what went wrong with one provider in words a reader
// can act on: a provider that is busy is worth asking again later, one that
// timed out may be slow today, and anything else is a fault to look at. The
// provider's own message is for the administrator and stays in the logs and
// the usage record; the reader is told who answered with what.
func providerFailure(
	provider *aiprovider.Provider,
	err error,
	elapsed time.Duration,
	asked int,
) llmports.ChatProviderFailure {
	code := 0
	var answered llmports.ProviderFailure
	if errors.As(err, &answered) {
		code = answered.ProviderStatus()
	}
	// Overloaded is what a provider says when it is busy: a rate limit, a
	// 503 or Anthropic's 529. Any other fault on its side, or a connection
	// that never reached it, is Unavailable.
	status := failureStatusFailed
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		status = failureStatusTimedOut
	case code == 429 || code == 503 || code == 529:
		status = failureStatusOverloaded
	case code >= 500 || completionengine.Unavailability(err) || modeladapter.IsRetryable(err):
		status = failureStatusUnavailable
	}

	return llmports.ChatProviderFailure{
		ProviderID: provider.ID,
		Name:       provider.Name,
		Model:      provider.Model,
		Vendor:     provider.Vendor(),
		Status:     status,
		HTTPStatus: code,
		Attempts:   asked,
		Detail:     failureDetail(provider, status, code, asked, elapsed),
	}
}

// failureDetail is the right-hand line of a failed model's row.
func failureDetail(
	provider *aiprovider.Provider,
	status string,
	code, attempts int,
	elapsed time.Duration,
) string {
	who := vendorName(provider)
	switch {
	case status == failureStatusTimedOut:
		return fmt.Sprintf(
			"No response after %ds",
			max(1, int(elapsed.Round(time.Second).Seconds())),
		)
	case code > 0:
		return who + " returned " + strconv.Itoa(code) + times(attempts)
	case status == failureStatusOverloaded || status == failureStatusUnavailable:
		return who + " could not be reached" + times(attempts)
	default:
		return who + " could not answer" + times(attempts)
	}
}

func times(n int) string {
	switch {
	case n <= 1:
		return ""
	case n == 2:
		return " twice"
	default:
		return " " + strconv.Itoa(n) + " times"
	}
}

// vendorNames is what a vendor is called where a person reads it.
var vendorNames = map[string]string{
	"anthropic":   "Anthropic",
	"openai":      "OpenAI",
	"gemini":      "Google",
	"groq":        "Groq",
	"mistral":     "Mistral",
	"openrouter":  "OpenRouter",
	"together":    "Together",
	"fireworks":   "Fireworks",
	"deepseek":    "DeepSeek",
	"bedrock":     "AWS Bedrock",
	"huggingface": "Hugging Face",
}

func vendorName(provider *aiprovider.Provider) string {
	if name, ok := vendorNames[provider.Vendor()]; ok {
		return name
	}

	return provider.Name
}

// notSetUp are the organization's other enabled models, which were never
// asked because they have not been given the task. A reader told that every
// model failed is told about these too, since giving one the task is the fix.
func (s *Chat) notSetUp(
	ctx context.Context,
	req *llmports.ChatCompletionRequest,
	asked []*aiprovider.Provider,
) []llmports.ChatProviderFailure {
	all, err := s.Repo.List(ctx, &aiproviderports.ListAIProviderRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: req.TenantInfo,
			Pagination: pagination.Info{Limit: maxNotSetUp * 4},
		},
	})
	if err != nil || all == nil {
		return nil
	}
	seen := make(map[pulid.ID]bool, len(asked))
	for _, provider := range asked {
		seen[provider.ID] = true
	}

	out := make([]llmports.ChatProviderFailure, 0, maxNotSetUp)
	for _, provider := range all.Items {
		if len(out) == maxNotSetUp {
			break
		}
		if seen[provider.ID] || !provider.Enabled {
			continue
		}
		out = append(out, llmports.ChatProviderFailure{
			ProviderID: provider.ID,
			Name:       provider.Name,
			Model:      provider.Model,
			Vendor:     provider.Vendor(),
			Status:     "Not set up",
			Detail:     "Not given the assistant task",
		})
	}

	return out
}

// maxNotSetUp keeps the list of unasked models to what fits under a card.
const maxNotSetUp = 3

// retold is a failure's line again once it is known how often it happened.
func retold(failure *llmports.ChatProviderFailure) string {
	who := vendorNames[failure.Vendor]
	if who == "" {
		who = failure.Name
	}
	switch {
	case failure.HTTPStatus > 0:
		return who + " returned " + strconv.Itoa(failure.HTTPStatus) + times(failure.Attempts)
	case failure.Status == failureStatusOverloaded || failure.Status == failureStatusUnavailable:
		return who + " could not be reached" + times(failure.Attempts)
	default:
		return who + " could not answer" + times(failure.Attempts)
	}
}

// distinctFailures keeps the last word from each provider: one asked twice
// is listed once, as it ended, and says how many times it was asked.
func distinctFailures(failures []llmports.ChatProviderFailure) []llmports.ChatProviderFailure {
	out := make([]llmports.ChatProviderFailure, 0, len(failures))
	seen := make(map[pulid.ID]int, len(failures))
	for _, failure := range failures {
		if at, ok := seen[failure.ProviderID]; ok {
			failure.Attempts += out[at].Attempts
			if failure.Status != failureStatusTimedOut {
				failure.Detail = retold(&failure)
			}
			out[at] = failure
			continue
		}
		seen[failure.ProviderID] = len(out)
		out = append(out, failure)
	}

	return out
}

// firstOtherThan is the first provider that failed before the one that
// answered, which is the one the reader expected to hear from.
func firstOtherThan(
	failures []llmports.ChatProviderFailure,
	answered pulid.ID,
) *llmports.ChatProviderFailure {
	for idx := range failures {
		if failures[idx].ProviderID != answered {
			failure := failures[idx]
			return &failure
		}
	}

	return nil
}

// chatStream is what one attempt put in front of the reader before it ended.
// A failed attempt has no result, but it may well have had these, and they
// decide both whether the failure may fall through and what it cost.
type chatStream struct {
	// text is the reply that reached the sink.
	text string
	// retries is how many times a busy provider was asked again.
	retries int
	// reasoningRunes is how much thinking the provider streamed.
	reasoningRunes   int
	cacheReadTokens  int
	cacheWriteTokens int
	finishReason     string
	thinkingDropped  int
	cacheDiagnosis   *modeladapter.CacheDiagnosis
	// firstAt is when the first text or thinking reached the reader: how
	// long a person watched nothing. Zero when none did.
	firstAt time.Time
}

func (c *chatStream) markFirst() {
	if c.firstAt.IsZero() {
		c.firstAt = time.Now()
	}
}

// firstToken is how long after started the first text or thinking arrived,
// or zero when none did.
func (c *chatStream) FirstToken(started time.Time) time.Duration {
	if c.firstAt.IsZero() {
		return 0
	}

	return c.firstAt.Sub(started)
}

// attemptChat runs the turn on one provider. The returned stream says what
// reached the reader, which decides whether a failure may fall through.
func (s *Chat) attemptChat(
	ctx context.Context,
	provider *aiprovider.Provider,
	req *llmports.ChatCompletionRequest,
	sink llmports.ChatStreamSink,
) (*llmports.ChatCompletionResult, chatStream, error) {
	var streamed chatStream

	adapter, err := s.Adapters.Get(provider.Kind)
	if err != nil {
		return nil, streamed, err
	}

	timeout := provider.ResolvedTimeout()
	release, err := s.ClaimSlot(ctx, provider, timeout)
	if err != nil {
		return nil, streamed, err
	}
	defer release()

	// The timeout bounds the wait for the first word. A streamed reply that
	// has started is held to the idle guard instead, so a long answer is not
	// cut off for being long.
	limited := s.WithDeadline(ctx, timeout)
	ctx = limited.Ctx

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = provider.ResolvedMaxTokens()
	}

	advisor, advisorFromSetting := advisorFor(provider, req.Advisor)
	call := s.chatCall(provider, req, advisor, maxTokens)

	// The thinking is counted whether or not anyone is shown it: a stream
	// cut off before its usage frame reports no tokens, and thinking is
	// billed as output all the same.
	call.Reasoning = func(delta string) {
		limited.Disarm()
		streamed.markFirst()
		streamed.reasoningRunes += utf8.RuneCountInString(delta)
		if req.ReasoningSink != nil {
			req.ReasoningSink(delta)
		}
	}

	execution := &chatExecution{
		adapter:  adapter,
		provider: provider,
		call:     call,
		sink:     sink,
		disarm:   limited.Disarm,
		streamed: &streamed,
		busy:     s.busyNotice(provider, req, &streamed),
	}
	resp, err := s.executeChat(ctx, execution)
	if refusedToolChoice(provider, call.Request, err) {
		s.Logger.Warn("the provider refused tool_choice none; asking once more with the tools "+
			"and no tool_choice",
			zap.String("provider", provider.ID.String()),
			zap.String("model", provider.Model),
			zap.Error(err),
		)
		call.Request.ToolChoiceNone = false
		resp, err = s.executeChat(ctx, execution)
	}
	if message, refused := refusedAdvisor(call.Request, err); refused && advisorFromSetting {
		s.RecordAdvisorRefusal(ctx, provider, req.TenantInfo, call.Request.Advisor.Model, message)
		call.Request.Advisor = nil
		resp, err = s.executeChat(ctx, execution)
	}
	err = limited.Finish(err)
	if err != nil {
		return nil, streamed, err
	}

	s.NoteFiles(provider, resp.Files)
	s.NoteCacheDiagnosis(provider, req, resp)
	if resp.Refused {
		return nil, streamed, completionengine.ErrRefused
	}

	// A turn with neither text nor a tool call is a dead end rather than an
	// answer, so it counts as a failure and the next provider gets a try.
	if resp.Text == "" && len(resp.ToolCalls) == 0 && !resp.Truncated {
		return nil, streamed, errors.New("provider returned neither content nor a tool call")
	}

	streamed.record(resp)

	return chatResult(provider, resp, maxTokens), streamed, nil
}

func (s *Chat) chatCall(
	provider *aiprovider.Provider,
	req *llmports.ChatCompletionRequest,
	advisor *llmports.AdvisorSpec,
	maxTokens int,
) *modeladapter.Call {
	return &modeladapter.Call{
		Provider:                 provider,
		Client:                   s.ClientFor(provider),
		StreamClient:             s.StreamClientFor(provider),
		StreamIdle:               s.AI.GetStreamIdleTimeout(),
		ServerCredentialsAllowed: s.AI.ServerCloudCredentialsAllowed(),
		Files:                    s.FileSource(req.TenantInfo),
		CacheDiagnostics:         s.AI.CacheDiagnosticsEnabled(),
		Request: &modeladapter.Request{
			System:         req.System,
			SystemStable:   req.SystemStable,
			CacheKey:       PromptCacheKey(req),
			Messages:       req.Messages,
			Tools:          req.Tools,
			ToolChoiceNone: req.ToolChoiceNone,
			Reasoning:      req.Reasoning,
			Advisor:        advisor,
			MaxTokens:      maxTokens,
			// A chat turn drives tools, so it is sampled for exactness: the
			// model has to name a tool that exists and fill its arguments
			// with JSON that parses, and invention there is only ever a bug.
			Sampling: modeladapter.SamplingForTask(aitask.TaskAssistantChat),
		},
	}
}

// busyNotice tells the reader a busy provider is being asked again; they are
// otherwise watching nothing happen for the length of the wait.
func (s *Chat) busyNotice(
	provider *aiprovider.Provider,
	req *llmports.ChatCompletionRequest,
	streamed *chatStream,
) completionengine.BusyNotice {
	return func(attempt int, wait time.Duration, cause error) {
		streamed.retries++
		if req.RetrySink == nil {
			return
		}
		req.RetrySink(llmports.ChatRetryNotice{
			Attempt:     attempt,
			Provider:    provider.Name,
			Reason:      cause.Error(),
			Kind:        llmports.RetryKindBusy,
			WaitSeconds: int(wait.Round(time.Second).Seconds()),
			MaxAttempts: s.BusyAttempts(),
		})
	}
}

type chatExecution struct {
	adapter  modeladapter.Adapter
	provider *aiprovider.Provider
	call     *modeladapter.Call
	sink     llmports.ChatStreamSink
	disarm   func()
	streamed *chatStream
	busy     completionengine.BusyNotice
}

func (s *Chat) executeChat(
	ctx context.Context,
	e *chatExecution,
) (*modeladapter.Response, error) {
	var resp *modeladapter.Response
	if streamer, ok := e.adapter.(modeladapter.Streamer); ok && e.sink != nil {
		first := func(delta string) {
			e.disarm()
			e.streamed.markFirst()
			e.sink(delta)
		}
		err := s.WithKeyFallback(e.provider, func(apiKey string) error {
			e.call.APIKey = apiKey
			var streamErr error
			resp, e.streamed.text, streamErr = s.executeStreamWithRetry(
				ctx,
				streamer,
				e.call,
				first,
				e.busy,
			)

			return streamErr
		})

		return resp, err
	}

	err := s.WithKeyFallback(e.provider, func(apiKey string) error {
		e.call.APIKey = apiKey
		var callErr error
		resp, callErr = s.ExecuteWithRetryNoticed(ctx, e.adapter, e.call, e.busy)

		return callErr
	})
	if err == nil && e.sink != nil && resp.Text != "" {
		e.sink(resp.Text)
		e.streamed.text = resp.Text
	}

	return resp, err
}

func (c *chatStream) record(resp *modeladapter.Response) {
	c.cacheReadTokens = resp.CacheReadTokens
	c.cacheWriteTokens = resp.CacheWriteTokens
	c.finishReason = completionengine.FinishReason(resp)
	c.thinkingDropped = resp.ThinkingDropped
	c.cacheDiagnosis = resp.CacheDiagnosis
}

func chatResult(
	provider *aiprovider.Provider,
	resp *modeladapter.Response,
	maxTokens int,
) *llmports.ChatCompletionResult {
	return &llmports.ChatCompletionResult{
		Text:              resp.Text,
		ToolCalls:         resp.ToolCalls,
		ModelIdentifier:   resp.ModelIdentifier,
		InputTokens:       resp.InputTokens,
		OutputTokens:      resp.OutputTokens,
		CacheReadTokens:   resp.CacheReadTokens,
		CacheWriteTokens:  resp.CacheWriteTokens,
		ProviderID:        provider.ID,
		ProviderKind:      provider.Kind,
		ProviderMessageID: resp.MessageID,
		Truncated:         resp.Truncated,
		Reasoning:         resp.Reasoning,
		ReasoningTokens:   resp.ReasoningTokens,
		OutputLimit:       cmp.Or(resp.OutputLimit, maxTokens),
		CutOffCall:        resp.CutOffCall,
		ContextWindow:     provider.ConfiguredContextWindow(),
		Advisor:           resp.Advisor,
	}
}

// partialReply is the reply that arrived before a provider stopped writing,
// marked as cut off. It is nil when nothing arrived, since an empty reply
// marked as truncated would read as an answer that said nothing.
func partialReply(
	provider *aiprovider.Provider,
	text string,
	cause error,
) *llmports.ChatCompletionResult {
	if text == "" {
		return nil
	}

	return &llmports.ChatCompletionResult{
		Text:            text,
		ModelIdentifier: modeladapter.ServedModel(cause, provider.Model),
		ProviderID:      provider.ID,
		ProviderKind:    provider.Kind,
		Truncated:       true,
		ContextWindow:   provider.ConfiguredContextWindow(),
	}
}

// executeStreamWithRetry retries a stream only while nothing has reached the
// sink. A retry after the first delta would replay text the reader already has.
func (s *Chat) executeStreamWithRetry(
	ctx context.Context,
	streamer modeladapter.Streamer,
	call *modeladapter.Call,
	sink llmports.ChatStreamSink,
	busy completionengine.BusyNotice,
) (*modeladapter.Response, string, error) {
	// The text is kept as well as the fact of it. A stream that dies partway
	// leaves the reader watching an answer that then vanishes, and the words
	// that did arrive are the ones we can still give them.
	var partial strings.Builder
	tracked := func(delta string) {
		partial.WriteString(delta)
		sink(delta)
	}
	emittedText := partial.String

	var waited time.Duration
	for attempt := 0; ; attempt++ {
		resp, err := streamer.Stream(ctx, call, tracked)
		if err == nil {
			return resp, emittedText(), nil
		}
		if emittedText() != "" || ctx.Err() != nil {
			return nil, emittedText(), err
		}

		wait, again := s.RetryWait(err, attempt, waited)
		if !again {
			return nil, emittedText(), err
		}

		s.Logger.Debug("retrying provider stream",
			zap.String("provider", call.Provider.Name),
			zap.Int("attempt", attempt+1),
			zap.Duration("wait", wait),
			zap.Error(err),
		)
		if completionengine.Unavailability(err) {
			completionengine.RecordBusyWait(ctx, wait)
			if busy != nil {
				busy(attempt+1, wait, err)
			}
		}
		if waitErr := s.Wait(ctx, wait); waitErr != nil {
			return nil, emittedText(), waitErr
		}
		waited += wait
	}
}

// maxMidReplyRetries bounds how many times a turn starts over after a
// provider died with words already on the reader's screen. Two is enough to
// get past one bad provider and one bad moment; more is a reader watching
// the same question asked and abandoned again and again.
const MaxMidReplyRetries = 2

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

// chatOutcome is what the usage record reads off a chat attempt. A finished
// attempt carries the provider's own count; one that failed or was stopped
// carries what was counted from its stream.
func chatOutcome(
	provider *aiprovider.Provider,
	result *llmports.ChatCompletionResult,
	streamed *chatStream,
	cause error,
) *completionengine.RunOutcome {
	if result == nil {
		return streamedOutcome(provider, streamed, cause)
	}

	return &completionengine.RunOutcome{
		Model:            result.ModelIdentifier,
		InputTokens:      result.InputTokens,
		OutputTokens:     result.OutputTokens,
		ReasoningTokens:  result.ReasoningTokens,
		CacheReadTokens:  streamed.cacheReadTokens,
		CacheWriteTokens: streamed.cacheWriteTokens,
		FinishReason:     streamed.finishReason,
		Truncated:        result.Truncated,
		ThinkingDropped:  streamed.thinkingDropped,
		CacheDiagnosis:   streamed.cacheDiagnosis,
		Advisor:          result.Advisor,
	}
}
