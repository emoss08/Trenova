// Package completionengine resolves which configured provider serves a given
// task and calls it through the adapter for its protocol, falling through the
// remaining candidates when one cannot produce a usable answer.
package completionengine

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/adapterregistry"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aimetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aipricing"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger     *zap.Logger
	AIConfig   *aiconfig.Config
	Repo       aiproviderports.AIProviderRepository
	Encryption *encryptionservice.Service
	// Usage records every attempt. Optional so a router built without a
	// database still answers; without it nothing is counted.
	Usage    aiusageports.AIUsageRepository     `optional:"true"`
	Metrics  *metrics.Registry                  `optional:"true"`
	Breakers llmports.ProviderBreakerRepository `optional:"true"`
	Quota    platformplanports.QuotaGuard       `optional:"true"`
	// Spend, Slots and Keys enforce each provider's monthly cap and calls at
	// once, and record when its key was last used.
	Spend aiproviderports.AIProviderSpendService  `optional:"true"`
	Slots llmports.ProviderSlotRepository         `optional:"true"`
	Keys  aiproviderports.AIProviderKeyRepository `optional:"true"`
	// Advisors records the API's refusal of a provider's advisor on the
	// provider. Optional: without it the refusal is only logged.
	Advisors aiproviderports.AIProviderAdvisorRecorder `optional:"true"`
	// Files reads the files a person attached, for a model that reads files
	// itself. Optional: without it every file is read as its text.
	Files conversationports.MessageFileReader `optional:"true"`
}

type Engine struct {
	Logger *zap.Logger
	// ai is the whole AI section: the master switch the router gates on,
	// and the timeouts and retry budget it calls with.
	AI         *aiconfig.Config
	Repo       aiproviderports.AIProviderRepository
	Encryption *encryptionservice.Service
	Adapters   *adapterregistry.Registry
	Usage      aiusageports.AIUsageRepository
	Quota      platformplanports.QuotaGuard
	GenAI      *aimetrics.GenAI
	// health rests a provider that keeps failing, so a turn does not pay for
	// attempts on a provider that is down before reaching one that is up.
	Health *ProviderHealth
	// limits holds each provider to its cap, its calls at once and its
	// timeout.
	Limits *providerLimits
	// files reads attached files for a model that reads files itself.
	Files conversationports.MessageFileReader
	// advisors records the API's refusal of a provider's advisor.
	Advisors aiproviderports.AIProviderAdvisorRecorder
	// pause waits out a backoff. It is a field so tests can count the waits
	// instead of sitting through them.
	Pause func(ctx context.Context, wait time.Duration) error

	// clients are cached per egress policy. Building one per call would discard
	// connection reuse, and the policy must stay bound to the client so a
	// provider allowed onto a private network cannot lend that reach to another.
	clientsMu sync.Mutex
	Clients   map[bool]*http.Client
	// streamClients are the same, minus the whole-request deadline. Kept apart
	// rather than shared because a blocking call wants that deadline and a
	// stream is killed by it.
	StreamClients map[bool]*http.Client
}

func NewEngine(p *Params) *Engine {
	logger := p.Logger.Named("service.completion-router")

	return &Engine{
		Logger:        logger,
		AI:            p.AIConfig,
		Repo:          p.Repo,
		Encryption:    p.Encryption,
		Usage:         p.Usage,
		Quota:         quotaservice.OrUnlimited(p.Quota),
		GenAI:         p.Metrics.GenAI(),
		Adapters:      adapterregistry.NewRegistry(),
		Health:        NewProviderHealth(nil).Share(p.Breakers, logger),
		Limits:        NewProviderLimits(p),
		Files:         p.Files,
		Advisors:      p.Advisors,
		Pause:         pauseFor,
		Clients:       make(map[bool]*http.Client, 2),
		StreamClients: make(map[bool]*http.Client, 2),
	}
}

func (s *Engine) AssertWithinSpend(ctx context.Context, tenantInfo pagination.TenantInfo) error {
	if tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() {
		return nil
	}

	return quotaservice.Preflight(ctx, s.Quota, &platformplanports.QuotaRequest{
		TenantInfo: tenantInfo,
		Meter:      platformcatalog.MeterAISpendCents,
		Quantity:   1,
	})
}

type RunRequest struct {
	TenantInfo  pagination.TenantInfo
	Task        aitask.Task
	System      string
	UserContent string
	Schema      map[string]any
	SchemaName  string
	MaxTokens   int
	// PreferredProviderID asks for one configured provider first, subject to the
	// same task and trust checks as any other candidate.
	PreferredProviderID pulid.ID
	RequireProvider     bool
	// HedgeAfter asks the next provider too when the one asked last has not
	// answered by then; zero tries them one after another.
	HedgeAfter time.Duration
	// Reasoning overrides each provider's configured effort for this call.
	Reasoning   aireasoning.ReasoningEffort
	Attribution aiusageports.AIUsageAttribution
}

type RunOutcome struct {
	Text             string
	Model            string
	InputTokens      int
	OutputTokens     int
	ReasoningTokens  int
	CacheReadTokens  int
	CacheWriteTokens int
	FinishReason     string
	Truncated        bool
	ThinkingDropped  int
	CacheDiagnosis   *modeladapter.CacheDiagnosis
	ProviderID       pulid.ID
	ProviderKind     aiprovider.Kind
	LatencyMs        int64
	CostUSD          *decimal.Decimal
	Advisor          *aipricing.AdvisorUsage
	Batched          bool
}

func (o *RunOutcome) Cost(provider *aiprovider.Provider, task aitask.Task) *decimal.Decimal {
	if o.Batched {
		return aipricing.BatchCostForTask(provider, task, o.TokenUsage())
	}

	return aipricing.AdvisedCostForTask(provider, task, o.TokenUsage(), o.Advisor)
}

// tokenUsage is what the provider reported, in the shape a price is read
// against, so every cost the router records is priced the same way.
func (o *RunOutcome) TokenUsage() aipricing.TokenUsage {
	return aipricing.TokenUsage{
		Input:      o.InputTokens,
		Output:     o.OutputTokens,
		CacheRead:  o.CacheReadTokens,
		CacheWrite: o.CacheWriteTokens,
	}
}

// candidatesFor resolves the providers allowed to serve a task, in the order
// they should be tried.
func (s *Engine) CandidatesFor(
	ctx context.Context,
	req *RunRequest,
) ([]*aiprovider.Provider, error) {
	usable, err := s.UsableFor(ctx, req.Task, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if req.RequireProvider {
		if usable, err = requiredProvider(usable, req.PreferredProviderID, req.Task); err != nil {
			return nil, err
		}
	}

	ready, err := s.Awake(ctx, usable)
	if err != nil {
		return nil, err
	}
	if ready, err = s.UnderCap(ctx, req.TenantInfo, ready); err != nil {
		return nil, err
	}

	return PreferFirst(ready, req.PreferredProviderID), nil
}

func requiredProvider(
	providers []*aiprovider.Provider,
	required pulid.ID,
	task aitask.Task,
) ([]*aiprovider.Provider, error) {
	for _, provider := range providers {
		if provider.ID == required {
			return []*aiprovider.Provider{provider}, nil
		}
	}

	return nil, errortypes.NewBusinessError(
		"The chosen AI provider is not enabled for {0}", string(task),
	).WithInternal(llmports.ErrRequiredProviderUnavailable)
}

func (s *Engine) UsableFor(
	ctx context.Context,
	task aitask.Task,
	tenant pagination.TenantInfo,
) ([]*aiprovider.Provider, error) {
	usable, err := s.servingTask(ctx, task, tenant, nil)
	if err != nil {
		return nil, err
	}
	if borrowed := task.BorrowsFrom(); borrowed != "" {
		if usable, err = s.servingTask(ctx, borrowed, tenant, usable); err != nil {
			return nil, err
		}
	}

	if len(usable) == 0 {
		return nil, errortypes.NewBusinessError(
			"No AI provider is configured for {0}", string(task),
		).WithInternal(llmports.ErrNoProviderConfigured)
	}

	return usable, nil
}

// servingTask appends to usable the providers that may serve task and are not
// already in it, in the repository's priority order.
func (s *Engine) servingTask(
	ctx context.Context,
	task aitask.Task,
	tenant pagination.TenantInfo,
	usable []*aiprovider.Provider,
) ([]*aiprovider.Provider, error) {
	candidates, err := s.Repo.ListForTask(ctx, aiproviderports.ListAIProvidersForTaskRequest{
		Task:       task,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}

	for _, provider := range candidates {
		if slices.ContainsFunc(usable, func(held *aiprovider.Provider) bool {
			return held.ID == provider.ID
		}) {
			continue
		}
		if ok, reason := provider.CanServeTask(task); ok {
			usable = append(usable, provider)
		} else {
			s.Logger.Debug("skipping provider for task",
				zap.String("provider", provider.Name),
				zap.String("task", string(task)),
				zap.String("reason", reason),
			)
		}
	}

	return usable, nil
}

// awake drops the providers resting after repeated failures. When every one
// of them is resting the caller is told so, with when the first is due back,
// rather than sent through a list that will fail at each step.
func (s *Engine) Awake(
	ctx context.Context,
	usable []*aiprovider.Provider,
) ([]*aiprovider.Provider, error) {
	ready, resting, until := s.Health.Rested(ctx, usable)
	for _, provider := range resting {
		s.Logger.Debug("skipping provider resting after repeated failures",
			zap.String("provider", provider.Name),
			zap.Time("until", until),
		)
	}
	if len(ready) == 0 {
		return nil, restingError(resting, until.Sub(s.Health.now()))
	}

	return ready, nil
}

// restingError says which providers are resting and how long until the
// first is back.
func restingError(resting []*aiprovider.Provider, wait time.Duration) error {
	seconds := max(1, int(wait.Round(time.Second).Seconds()))
	if len(resting) == 1 {
		return errortypes.NewBusinessError(
			"{0} is paused for {1} seconds after repeated failures. Try again then, or pick another model",
			resting[0].Name,
			seconds,
		).WithInternal(llmports.ErrProvidersResting)
	}

	return errortypes.NewBusinessError(
		"Every AI provider is paused after repeated failures; the first is back in {0} seconds",
		seconds,
	).WithInternal(llmports.ErrProvidersResting)
}

func (s *Engine) Run(ctx context.Context, req *RunRequest) (*RunOutcome, error) {
	usable, err := s.CandidatesFor(ctx, req)
	if err != nil {
		return nil, err
	}

	return s.RunAmong(ctx, usable, req)
}

func (s *Engine) RunAmong(
	ctx context.Context,
	usable []*aiprovider.Provider,
	req *RunRequest,
) (*RunOutcome, error) {
	if req.HedgeAfter > 0 && len(usable) > 1 {
		return s.runHedged(ctx, usable, req)
	}

	var lastErr error
	for idx, provider := range usable {
		// A caller that has gone is not answered by the next provider: the
		// attempt would fail at once and be charged to that provider.
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		outcome, attemptErr := s.tryProvider(ctx, &candidateAttempt{
			Provider: provider,
			idx:      idx,
			count:    len(usable),
		}, req)
		if attemptErr == nil {
			return outcome, nil
		}

		if ctx.Err() != nil {
			return nil, attemptErr
		}
		if errors.Is(attemptErr, ErrRefused) {
			return nil, errDeclined()
		}

		lastErr = attemptErr
		s.fellThrough(provider, req, attemptErr)
	}

	return nil, fmt.Errorf("every configured provider for %s failed: %w", req.Task, lastErr)
}

type candidateAttempt struct {
	Provider *aiprovider.Provider
	idx      int
	count    int
}

// tryProvider asks one candidate, settling the attempt's usage, trace and
// health. ctx is the context the attempt runs under: the caller's, or for a
// hedged call one that is canceled when another provider answers first, which
// observe and stopped then read as a cancellation rather than a fault.
func (s *Engine) tryProvider(
	ctx context.Context,
	candidate *candidateAttempt,
	req *RunRequest,
) (*RunOutcome, error) {
	provider, idx := candidate.Provider, candidate.idx
	attemptCtx, span := s.StartAttempt(ctx, &AttemptSpec{
		Operation:   aitrace.OperationChat,
		Provider:    provider,
		Attempt:     idx + 1,
		Failover:    idx > 0,
		MaxTokens:   req.MaxTokens,
		Attribution: req.Attribution,
	})
	started := time.Now()
	outcome, attemptErr := s.Attempt(LastCandidate(attemptCtx, idx, candidate.count), provider, req)
	latency := time.Since(started)
	attemptErr = Stopped(ctx, attemptErr)
	s.Observe(ctx, provider, attemptErr)
	s.SettleAttempt(attemptCtx, span, &UsageAttempt{
		Provider:    provider,
		Task:        req.Task,
		Surface:     SurfaceFor(aiusage.SurfaceStructured, &req.Attribution),
		Attribution: req.Attribution,
		Tenant:      req.TenantInfo,
		Latency:     latency,
		Outcome:     outcome,
		Err:         attemptErr,
		Operation:   aitrace.OperationChat,
		Attempt:     idx + 1,
		Failover:    idx > 0,
	})
	if attemptErr != nil {
		return nil, attemptErr
	}

	s.TouchKey(ctx, provider, req.TenantInfo)
	outcome.LatencyMs = latency.Milliseconds()
	outcome.CostUSD = aipricing.CostFor(provider, outcome.TokenUsage())

	return outcome, nil
}

// errDeclined ends the chain on a refusal. A refusal is the model's decision,
// not a fault in the endpoint, and asking the next provider the same question
// invites it to answer something the first declined.
func errDeclined() error {
	return errortypes.NewBusinessError("The model declined this request")
}

func (s *Engine) fellThrough(provider *aiprovider.Provider, req *RunRequest, err error) {
	s.Logger.Warn("provider attempt failed, falling through",
		zap.String("provider", provider.Name),
		zap.String("kind", string(provider.Kind)),
		zap.String("task", string(req.Task)),
		zap.Error(err),
	)
}

var ErrRefused = errors.New("model declined the request")

// stopped makes the failure of an attempt whose caller has gone read as that
// caller's own cancellation. A severed stream surfaces as whatever the
// connection happened to report — a reset, an EOF, a closed body — and a Stop
// that reads as any of those is recorded as a failed turn and charged to the
// provider. Wrapping the context's error is what lets everything downstream
// see it as a Stop. An error while the caller is still there is returned as is.
func Stopped(ctx context.Context, err error) error {
	ctxErr := ctx.Err()
	if err == nil || ctxErr == nil || errors.Is(err, ctxErr) {
		return err
	}

	return fmt.Errorf("%w: %w", ctxErr, err)
}

func (s *Engine) Attempt(
	ctx context.Context,
	provider *aiprovider.Provider,
	req *RunRequest,
) (*RunOutcome, error) {
	adapter, err := s.Adapters.Get(provider.Kind)
	if err != nil {
		return nil, err
	}

	timeout := provider.ResolvedTimeout()
	release, err := s.ClaimSlot(ctx, provider, timeout)
	if err != nil {
		return nil, err
	}
	defer release()

	limited := s.WithDeadline(ctx, timeout)
	var resp *modeladapter.Response
	err = limited.Finish(s.WithKeyFallback(provider, func(apiKey string) error {
		var callErr error
		resp, callErr = s.executeWithRetry(limited.Ctx, adapter, s.CallFor(provider, apiKey, req))

		return callErr
	}))
	if err != nil {
		return nil, err
	}

	if resp.Refused {
		return nil, ErrRefused
	}

	text := strings.TrimSpace(resp.Text)
	if text == "" {
		return nil, errors.New("provider returned no content")
	}

	if err = validateStructuredOutput(req.Schema, text); err != nil {
		return nil, err
	}

	return &RunOutcome{
		Text:             resp.Text,
		Model:            resp.ModelIdentifier,
		InputTokens:      resp.InputTokens,
		OutputTokens:     resp.OutputTokens,
		ReasoningTokens:  resp.ReasoningTokens,
		CacheReadTokens:  resp.CacheReadTokens,
		CacheWriteTokens: resp.CacheWriteTokens,
		FinishReason:     FinishReason(resp),
		Truncated:        resp.Truncated,
		ThinkingDropped:  resp.ThinkingDropped,
		ProviderID:       provider.ID,
		ProviderKind:     provider.Kind,
	}, nil
}

// callFor builds the provider-shaped call for a structured request. It is shared
// with the background path so a deferred call carries exactly the same prompt,
// schema and token budget as the synchronous one.
func (s *Engine) CallFor(
	provider *aiprovider.Provider,
	apiKey string,
	req *RunRequest,
) *modeladapter.Call {
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = provider.ResolvedMaxTokens()
	}

	return &modeladapter.Call{
		Provider:                 provider,
		APIKey:                   apiKey,
		Client:                   s.ClientFor(provider),
		ServerCredentialsAllowed: s.AI.ServerCloudCredentialsAllowed(),
		Request: &modeladapter.Request{
			// A provider that cannot enforce the schema on the wire is given it in
			// the prompt instead, so every candidate sees the same requirements.
			System:       StructuredSystem(req, provider.StructuredOutputMode),
			Messages:     modeladapter.UserMessage(req.UserContent),
			OutputSchema: req.Schema,
			SchemaName:   req.SchemaName,
			MaxTokens:    maxTokens,
			Sampling:     modeladapter.SamplingForTask(req.Task),
			Reasoning:    req.Reasoning,
		},
	}
}

// validateStructuredOutput rejects a reply that will not decode against the
// requested schema. Checking here rather than at the call site is what makes a
// provider that cannot hold the format fall through to the next candidate; a
// caller that received the prose instead would have no way to ask again.
//
// The check is structural, not a full JSON Schema validation: every schema this
// router sends declares a top-level object, so text that yields one has honoured
// the shape, and text that does not is a refusal or a ramble either way.
func validateStructuredOutput(schema map[string]any, text string) error {
	if len(schema) == 0 {
		return nil
	}

	return ObjectReply(text)
}

func ObjectReply(text string) error {
	decoded := make(map[string]any)
	if err := modeladapter.ExtractJSON(text, &decoded); err != nil {
		return fmt.Errorf("%w: %w", llmports.ErrModelSchemaValidation, err)
	}

	return nil
}

func (s *Engine) executeWithRetry(
	ctx context.Context,
	adapter modeladapter.Adapter,
	call *modeladapter.Call,
) (*modeladapter.Response, error) {
	return s.ExecuteWithRetryNoticed(ctx, adapter, call, nil)
}

const (
	// maxBusyAttempts is how many times a provider answering 429 or 5xx is
	// asked in one call. A busy provider usually answers within a few
	// seconds, and a person who picked that model has nowhere else to go.
	MaxBusyAttempts = 4
	// busyWaitBudget bounds the total wait on one provider in one call, and
	// maxRetryWait bounds any single wait, whatever the provider asked for.
	busyWaitBudget = 20 * time.Second
	MaxRetryWait   = 15 * time.Second
)

// busyAttempts is how many times a busy provider is asked in all.
func (s *Engine) BusyAttempts() int {
	return max(max(1, s.AI.GetMaxRetries()), MaxBusyAttempts)
}

// retryWait decides whether one more attempt on the same provider is worth
// it after err, and how long to wait first.
//
// A request the provider refused is never retried. Any other retryable
// failure gets the configured attempts with the usual backoff. A provider
// that is busy — a 429, a 5xx, a timeout — gets more attempts, since asking
// again in a few seconds is what such an answer means, and the wait honours
// what the provider asked for, within a budget that keeps a person from
// watching a spinner for a minute.
func (s *Engine) RetryWait(err error, attempt int, waited time.Duration) (time.Duration, bool) {
	if !modeladapter.IsRetryable(err) {
		return 0, false
	}

	limit := max(1, s.AI.GetMaxRetries())
	busy := Unavailability(err)
	if busy {
		limit = max(limit, MaxBusyAttempts)
	}
	if attempt+1 >= limit {
		return 0, false
	}

	wait := RetryDelay(attempt)
	if busy {
		wait = max(wait, retryAfterOf(err))
		wait = min(wait, MaxRetryWait)
		if waited+wait > busyWaitBudget {
			return 0, false
		}
	}

	return wait, true
}

// retryAfterOf is how long the provider asked to be left alone, if it said.
func retryAfterOf(err error) time.Duration {
	var transport *modeladapter.TransportError
	if errors.As(err, &transport) {
		return transport.RetryAfter
	}

	return 0
}

// wait sits out a backoff through the pause hook.
func (s *Engine) Wait(ctx context.Context, wait time.Duration) error {
	if s.Pause == nil {
		return pauseFor(ctx, wait)
	}

	return s.Pause(ctx, wait)
}

func CredentialAAD(provider *aiprovider.Provider) encryptionservice.AAD {
	return encryptionservice.AIProviderCredentialAAD(
		provider.OrganizationID,
		provider.BusinessUnitID,
		provider.ID,
	)
}

func (s *Engine) ResolveAPIKey(provider *aiprovider.Provider) (string, error) {
	stored := strings.TrimSpace(provider.APIKey)
	if stored == "" {
		if provider.RequiresCredential() {
			return "", errortypes.NewBusinessError(
				"AI provider {0} has no API key configured", provider.Name,
			)
		}

		return "", nil
	}

	decrypted, _, err := s.Encryption.DecryptStringWithLegacy(stored, CredentialAAD(provider))
	if err != nil {
		return "", errortypes.NewBusinessError(
			"failed to decrypt the credential for AI provider {0}", provider.Name,
		).WithInternal(err)
	}

	return decrypted, nil
}

// retryDelay is how long to wait before attempt n+1. It doubles from half a
// second and stops at five, because the errors worth retrying — a 429, a 5xx
// — are the ones an immediate retry makes worse. Retrying a rate limit at once
// just spends the next request on the same limit.
func RetryDelay(attempt int) time.Duration {
	const (
		base     = 500 * time.Millisecond
		maxDelay = 5 * time.Second
	)

	delay := base << attempt
	if delay > maxDelay || delay <= 0 {
		return maxDelay
	}

	return delay
}

// pauseFor sleeps out the wait, or returns the context's error the moment
// it is cancelled: a person who stopped a reply is not kept waiting for a
// backoff to elapse.
func pauseFor(ctx context.Context, wait time.Duration) error {
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// clientFor returns the shared client matching the provider's egress policy.
//
// Its deadline is the completion budget, not the probe budget: this client
// carries generations, and a model writing a long answer routinely outlasts the
// time a reachability check is allowed.
func (s *Engine) ClientFor(provider *aiprovider.Provider) *http.Client {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	allowPrivate := provider.AllowPrivateNetwork && s.AI.PrivateNetworkProvidersAllowed()
	if client, ok := s.Clients[allowPrivate]; ok {
		return client
	}
	if s.Clients == nil {
		s.Clients = make(map[bool]*http.Client, 2)
	}

	// A blocking call sends nothing until the model has finished, so its
	// time-to-headers is the same budget as the call itself. Bounding the
	// headers by the probe timeout instead made every request to a queued
	// free tier — NVIDIA's NIM endpoints, a shared router, a cold
	// self-hosted model — fail at twenty seconds with a transport error,
	// which reads as an outage rather than as a busy endpoint.
	policy := s.egressPolicy(allowPrivate)
	policy.ResponseHeaderTimeout = s.AI.GetCompletionTimeout()
	client := httpsafe.NewClientWithPolicy(s.AI.GetCompletionTimeout(), policy)
	s.Clients[allowPrivate] = client

	return client
}

// streamClientFor returns the client streaming calls use.
//
// It differs from clientFor in one way that matters: no whole-request timeout.
// http.Client.Timeout spans reading the response body, so on a streamed reply
// it is a wall-clock budget for the entire answer — the connection is severed
// mid-sentence however healthy it is and however fast the tokens are arriving.
// That is what made long answers stop partway through and be reported as cut
// off. Liveness is enforced between bytes instead, by the idle guard the
// adapters wrap the body in, so a slow answer is allowed and a silent one is
// not.
func (s *Engine) StreamClientFor(provider *aiprovider.Provider) *http.Client {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	allowPrivate := provider.AllowPrivateNetwork && s.AI.PrivateNetworkProvidersAllowed()
	if client, ok := s.StreamClients[allowPrivate]; ok {
		return client
	}
	if s.StreamClients == nil {
		s.StreamClients = make(map[bool]*http.Client, 2)
	}

	// A stream's time-to-headers is bounded by the same silence budget as its
	// body, not by the probe timeout: an endpoint that holds the headers until
	// it has something to say is slow, not down.
	policy := s.egressPolicy(allowPrivate)
	policy.ResponseHeaderTimeout = s.AI.GetStreamIdleTimeout()
	client := httpsafe.NewStreamingClientWithPolicy(policy)
	s.StreamClients[allowPrivate] = client

	return client
}

// egressPolicy is the network guard both clients share. Each caller sets
// its own ResponseHeaderTimeout on top, because what counts as too long
// before the first byte is entirely different for a blocking call and a
// streamed one.
func (s *Engine) egressPolicy(allowPrivate bool) httpsafe.Policy {
	return httpsafe.Policy{
		AllowPrivateNetworks: allowPrivate,
		// A self-hosted model holds the connection open while it generates and
		// sends nothing until the first token, which on a loaded GPU outlasts the
		// transport default.
		ResponseHeaderTimeout: s.AI.GetCompletionTimeout(),
	}
}

// aiDisabledMessage names the key rather than the symptom.
//
// The old text was "AI features are disabled" from a gate on
// documentIntelligence.enableAI, which ships false. Somebody who had just
// configured a provider had no way to know which switch was refusing them, or
// that it was named after a feature they were not using.
var AIDisabledMessage = "AI features are disabled. Set " + aiconfig.EnabledKey +
	" to true in the server configuration."
