// Package completionrouter resolves which configured provider serves a given
// task and calls it through the adapter for its protocol, falling through the
// remaining candidates when one cannot produce a usable answer.
package completionrouter

import (
	"context"
	"errors"
	"fmt"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/shopspring/decimal"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger     *zap.Logger
	Config     *config.Config
	Repo       repositories.AIProviderRepository
	Encryption *encryptionservice.Service
	// Usage records every attempt. Optional so a router built without a
	// database still answers; without it nothing is counted.
	Usage repositories.AIUsageRepository `optional:"true"`
}

type Service struct {
	logger *zap.Logger
	// ai gates every model-backed feature. cfg keeps the document-intelligence
	// timeouts and retry budget the router still reads; the two were one field
	// until the switch named after document intelligence was found to be
	// turning off the assistant.
	ai         *config.AIConfig
	cfg        *config.DocumentIntelligenceConfig
	repo       repositories.AIProviderRepository
	encryption *encryptionservice.Service
	adapters   *modeladapter.Registry
	usage      repositories.AIUsageRepository
	// health rests a provider that keeps failing, so a turn does not pay for
	// attempts on a provider that is down before reaching one that is up.
	health *providerHealth
	// pause waits out a backoff. It is a field so tests can count the waits
	// instead of sitting through them.
	pause func(ctx context.Context, wait time.Duration) error

	// clients are cached per egress policy. Building one per call would discard
	// connection reuse, and the policy must stay bound to the client so a
	// provider allowed onto a private network cannot lend that reach to another.
	clientsMu sync.Mutex
	clients   map[bool]*http.Client
	// streamClients are the same, minus the whole-request deadline. Kept apart
	// rather than shared because a blocking call wants that deadline and a
	// stream is killed by it.
	streamClients map[bool]*http.Client
}

func New(p Params) serviceports.CompletionService {
	return &Service{
		logger:        p.Logger.Named("service.completion-router"),
		ai:            p.Config.GetAIConfig(),
		cfg:           p.Config.GetDocumentIntelligenceConfig(),
		repo:          p.Repo,
		encryption:    p.Encryption,
		usage:         p.Usage,
		adapters:      modeladapter.NewRegistry(),
		health:        newProviderHealth(nil),
		pause:         pauseFor,
		clients:       make(map[bool]*http.Client, 2),
		streamClients: make(map[bool]*http.Client, 2),
	}
}

func (s *Service) CompleteStructured(
	ctx context.Context,
	req *serviceports.StructuredCompletionRequest,
) (*serviceports.StructuredCompletionResult, error) {
	if !s.ai.AIEnabled() {
		return nil, errortypes.NewBusinessError(aiDisabledMessage)
	}

	task := req.Task
	if task == "" {
		task = aiprovider.TaskGeneral
	}

	outcome, err := s.run(ctx, &runRequest{
		TenantInfo:          req.TenantInfo,
		Task:                task,
		System:              req.System,
		UserContent:         modeladapter.BuildContextText(req.Context),
		Schema:              req.OutputSchema,
		SchemaName:          req.SchemaName,
		MaxTokens:           req.MaxTokens,
		PreferredProviderID: req.PreferredProviderID,
		Attribution:         req.Attribution,
	})
	if err != nil {
		return nil, err
	}

	return &serviceports.StructuredCompletionResult{
		Text:            outcome.Text,
		ModelIdentifier: outcome.Model,
		InputTokens:     outcome.InputTokens,
		OutputTokens:    outcome.OutputTokens,
		LatencyMs:       outcome.LatencyMs,
		CostUSD:         outcome.CostUSD,
		ProviderID:      outcome.ProviderID,
		ProviderKind:    outcome.ProviderKind,
	}, nil
}

type runRequest struct {
	TenantInfo  pagination.TenantInfo
	Task        aiprovider.Task
	System      string
	UserContent string
	Schema      map[string]any
	SchemaName  string
	MaxTokens   int
	// PreferredProviderID asks for one configured provider first, subject to the
	// same task and trust checks as any other candidate.
	PreferredProviderID pulid.ID
	Attribution         serviceports.AIUsageAttribution
}

type runOutcome struct {
	Text            string
	Model           string
	InputTokens     int
	OutputTokens    int
	ReasoningTokens int
	ProviderID      pulid.ID
	ProviderKind    aiprovider.Kind
	LatencyMs       int64
	CostUSD         *decimal.Decimal
}

// candidatesFor resolves the providers allowed to serve a task, in the order
// they should be tried.
func (s *Service) candidatesFor(
	ctx context.Context,
	req *runRequest,
) ([]*aiprovider.Provider, error) {
	candidates, err := s.repo.ListForTask(ctx, repositories.ListAIProvidersForTaskRequest{
		Task:       req.Task,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	usable := make([]*aiprovider.Provider, 0, len(candidates))
	for _, provider := range candidates {
		if ok, reason := provider.CanServeTask(req.Task); ok {
			usable = append(usable, provider)
		} else {
			s.logger.Debug("skipping provider for task",
				zap.String("provider", provider.Name),
				zap.String("task", string(req.Task)),
				zap.String("reason", reason),
			)
		}
	}

	if len(usable) == 0 {
		return nil, errortypes.NewBusinessError(
			"No AI provider is configured for {0}", string(req.Task),
		).WithInternal(serviceports.ErrNoProviderConfigured)
	}

	ready, err := s.awake(usable)
	if err != nil {
		return nil, err
	}

	return preferFirst(ready, req.PreferredProviderID), nil
}

// awake drops the providers resting after repeated failures. When every one
// of them is resting the caller is told so, with when the first is due back,
// rather than sent through a list that will fail at each step.
func (s *Service) awake(usable []*aiprovider.Provider) ([]*aiprovider.Provider, error) {
	ready, resting, until := s.health.rested(usable)
	for _, provider := range resting {
		s.logger.Debug("skipping provider resting after repeated failures",
			zap.String("provider", provider.Name),
			zap.Time("until", until),
		)
	}
	if len(ready) == 0 {
		return nil, restingError(resting, until.Sub(s.health.now()))
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
			resting[0].Name, seconds,
		).WithInternal(serviceports.ErrProvidersResting)
	}

	return errortypes.NewBusinessError(
		"Every AI provider is paused after repeated failures; the first is back in {0} seconds",
		seconds,
	).WithInternal(serviceports.ErrProvidersResting)
}

func (s *Service) run(ctx context.Context, req *runRequest) (*runOutcome, error) {
	usable, err := s.candidatesFor(ctx, req)
	if err != nil {
		return nil, err
	}

	return s.runAmong(ctx, usable, req)
}

func (s *Service) runAmong(
	ctx context.Context,
	usable []*aiprovider.Provider,
	req *runRequest,
) (*runOutcome, error) {
	var lastErr error
	for _, provider := range usable {
		started := time.Now()
		outcome, attemptErr := s.attempt(ctx, provider, req)
		latency := time.Since(started)
		s.health.Observe(provider.ID, attemptErr)
		s.record(ctx, usageAttempt{
			provider:    provider,
			task:        req.Task,
			surface:     aiusage.SurfaceStructured,
			attribution: req.Attribution,
			tenant:      req.TenantInfo,
			latency:     latency,
			outcome:     outcome,
			err:         attemptErr,
		})
		if attemptErr == nil {
			outcome.LatencyMs = latency.Milliseconds()
			outcome.CostUSD = provider.CostFor(outcome.InputTokens, outcome.OutputTokens)

			return outcome, nil
		}

		// A refusal is the model's decision, not a fault in the endpoint. Asking
		// the next provider the same question invites it to answer something the
		// first declined, so the chain stops here.
		if errors.Is(attemptErr, errRefused) {
			return nil, errortypes.NewBusinessError("The model declined this request")
		}

		lastErr = attemptErr
		s.logger.Warn("provider attempt failed, falling through",
			zap.String("provider", provider.Name),
			zap.String("kind", string(provider.Kind)),
			zap.String("task", string(req.Task)),
			zap.Error(attemptErr),
		)
	}

	return nil, fmt.Errorf("every configured provider for %s failed: %w", req.Task, lastErr)
}

var errRefused = errors.New("model declined the request")

func (s *Service) attempt(
	ctx context.Context,
	provider *aiprovider.Provider,
	req *runRequest,
) (*runOutcome, error) {
	adapter, err := s.adapters.Get(provider.Kind)
	if err != nil {
		return nil, err
	}

	apiKey, err := s.resolveAPIKey(provider)
	if err != nil {
		return nil, err
	}

	resp, err := s.executeWithRetry(ctx, adapter, s.callFor(provider, apiKey, req))
	if err != nil {
		return nil, err
	}

	if resp.Refused {
		return nil, errRefused
	}

	text := strings.TrimSpace(resp.Text)
	if text == "" {
		return nil, errors.New("provider returned no content")
	}

	if err = validateStructuredOutput(req.Schema, text); err != nil {
		return nil, err
	}

	return &runOutcome{
		Text:            resp.Text,
		Model:           resp.ModelIdentifier,
		InputTokens:     resp.InputTokens,
		OutputTokens:    resp.OutputTokens,
		ReasoningTokens: resp.ReasoningTokens,
		ProviderID:      provider.ID,
		ProviderKind:    provider.Kind,
	}, nil
}

// callFor builds the provider-shaped call for a structured request. It is shared
// with the background path so a deferred call carries exactly the same prompt,
// schema and token budget as the synchronous one.
func (s *Service) callFor(
	provider *aiprovider.Provider,
	apiKey string,
	req *runRequest,
) *modeladapter.Call {
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = provider.ResolvedMaxTokens()
	}

	return &modeladapter.Call{
		Provider: provider,
		APIKey:   apiKey,
		Client:   s.clientFor(provider),
		Request: &modeladapter.Request{
			// A provider that cannot enforce the schema on the wire is given it in
			// the prompt instead, so every candidate sees the same requirements.
			System: modeladapter.WithSchemaInstruction(
				req.System,
				req.Schema,
				provider.StructuredOutputMode,
			),
			Messages:     modeladapter.UserMessage(req.UserContent),
			OutputSchema: req.Schema,
			SchemaName:   req.SchemaName,
			MaxTokens:    maxTokens,
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

	decoded := make(map[string]any, len(schema))
	if err := modeladapter.ExtractJSON(text, &decoded); err != nil {
		return fmt.Errorf("%w: %w", serviceports.ErrModelSchemaValidation, err)
	}

	return nil
}

func (s *Service) executeWithRetry(
	ctx context.Context,
	adapter modeladapter.Adapter,
	call *modeladapter.Call,
) (*modeladapter.Response, error) {
	return s.executeWithRetryNoticed(ctx, adapter, call, nil)
}

const (
	// maxBusyAttempts is how many times a provider answering 429 or 5xx is
	// asked in one call. A busy provider usually answers within a few
	// seconds, and a person who picked that model has nowhere else to go.
	maxBusyAttempts = 4
	// busyWaitBudget bounds the total wait on one provider in one call, and
	// maxRetryWait bounds any single wait, whatever the provider asked for.
	busyWaitBudget = 20 * time.Second
	maxRetryWait   = 15 * time.Second
)

// retryWait decides whether one more attempt on the same provider is worth
// it after err, and how long to wait first.
//
// A request the provider refused is never retried. Any other retryable
// failure gets the configured attempts with the usual backoff. A provider
// that is busy — a 429, a 5xx, a timeout — gets more attempts, since asking
// again in a few seconds is what such an answer means, and the wait honours
// what the provider asked for, within a budget that keeps a person from
// watching a spinner for a minute.
func (s *Service) retryWait(err error, attempt int, waited time.Duration) (time.Duration, bool) {
	if !modeladapter.IsRetryable(err) {
		return 0, false
	}

	limit := max(1, s.cfg.GetAIMaxRetries())
	busy := unavailability(err)
	if busy {
		limit = max(limit, maxBusyAttempts)
	}
	if attempt+1 >= limit {
		return 0, false
	}

	wait := retryDelay(attempt)
	if busy {
		wait = max(wait, retryAfterOf(err))
		wait = min(wait, maxRetryWait)
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
func (s *Service) wait(ctx context.Context, wait time.Duration) error {
	if s.pause == nil {
		return pauseFor(ctx, wait)
	}

	return s.pause(ctx, wait)
}

func (s *Service) resolveAPIKey(provider *aiprovider.Provider) (string, error) {
	stored := strings.TrimSpace(provider.APIKey)
	if stored == "" {
		if provider.Kind.RequiresAPIKey() {
			return "", errortypes.NewBusinessError(
				"AI provider {0} has no API key configured", provider.Name,
			)
		}

		return "", nil
	}

	decrypted, err := s.encryption.DecryptString(stored)
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
func retryDelay(attempt int) time.Duration {
	const (
		base = 500 * time.Millisecond
		cap  = 5 * time.Second
	)

	delay := base << attempt
	if delay > cap || delay <= 0 {
		return cap
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
func (s *Service) clientFor(provider *aiprovider.Provider) *http.Client {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	allowPrivate := provider.AllowPrivateNetwork
	if client, ok := s.clients[allowPrivate]; ok {
		return client
	}
	if s.clients == nil {
		s.clients = make(map[bool]*http.Client, 2)
	}

	client := httpsafe.NewClientWithPolicy(
		s.cfg.GetAICompletionTimeout(),
		s.egressPolicy(allowPrivate),
	)
	s.clients[allowPrivate] = client

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
func (s *Service) streamClientFor(provider *aiprovider.Provider) *http.Client {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	allowPrivate := provider.AllowPrivateNetwork
	if client, ok := s.streamClients[allowPrivate]; ok {
		return client
	}
	if s.streamClients == nil {
		s.streamClients = make(map[bool]*http.Client, 2)
	}

	// A stream's time-to-headers is bounded by the same silence budget as its
	// body, not by the probe timeout: an endpoint that holds the headers until
	// it has something to say is slow, not down.
	policy := s.egressPolicy(allowPrivate)
	policy.ResponseHeaderTimeout = s.cfg.GetAIStreamIdleTimeout()
	client := httpsafe.NewStreamingClientWithPolicy(policy)
	s.streamClients[allowPrivate] = client

	return client
}

// egressPolicy is the network guard both clients share.
func (s *Service) egressPolicy(allowPrivate bool) httpsafe.Policy {
	return httpsafe.Policy{
		AllowPrivateNetworks: allowPrivate,
		// A self-hosted model holds the connection open while it generates and
		// sends nothing until the first token, which on a loaded GPU outlasts the
		// transport default.
		ResponseHeaderTimeout: s.cfg.GetAITimeout(),
	}
}

// aiDisabledMessage names the key rather than the symptom.
//
// The old text was "AI features are disabled" from a gate on
// documentIntelligence.enableAI, which ships false. Somebody who had just
// configured a provider had no way to know which switch was refusing them, or
// that it was named after a feature they were not using.
var aiDisabledMessage = "AI features are disabled. Set " + config.AIEnabledKey +
	" to true in the server configuration."
