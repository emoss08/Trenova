// Package completionrouter resolves which configured provider serves a given
// task and calls it through the adapter for its protocol, falling through the
// remaining candidates when one cannot produce a usable answer.
package completionrouter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

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
}

type Service struct {
	logger     *zap.Logger
	cfg        *config.DocumentIntelligenceConfig
	repo       repositories.AIProviderRepository
	encryption *encryptionservice.Service
	adapters   *modeladapter.Registry

	// clients are cached per egress policy. Building one per call would discard
	// connection reuse, and the policy must stay bound to the client so a
	// provider allowed onto a private network cannot lend that reach to another.
	clientsMu sync.Mutex
	clients   map[bool]*http.Client
}

func New(p Params) serviceports.CompletionService {
	return &Service{
		logger:     p.Logger.Named("service.completion-router"),
		cfg:        p.Config.GetDocumentIntelligenceConfig(),
		repo:       p.Repo,
		encryption: p.Encryption,
		adapters:   modeladapter.NewRegistry(),
		clients:    make(map[bool]*http.Client, 2),
	}
}

func (s *Service) CompleteStructured(
	ctx context.Context,
	req *serviceports.StructuredCompletionRequest,
) (*serviceports.StructuredCompletionResult, error) {
	if !s.cfg.AIEnabled() {
		return nil, errortypes.NewBusinessError("AI features are disabled")
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
	})
	if err != nil {
		return nil, err
	}

	return &serviceports.StructuredCompletionResult{
		Text:            outcome.Text,
		ModelIdentifier: outcome.Model,
		InputTokens:     outcome.InputTokens,
		OutputTokens:    outcome.OutputTokens,
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
}

type runOutcome struct {
	Text         string
	Model        string
	InputTokens  int
	OutputTokens int
	ProviderID   pulid.ID
	ProviderKind aiprovider.Kind
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

	return preferFirst(usable, req.PreferredProviderID), nil
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
		outcome, attemptErr := s.attempt(ctx, provider, req)
		if attemptErr == nil {
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
		Text:         resp.Text,
		Model:        resp.ModelIdentifier,
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
		ProviderID:   provider.ID,
		ProviderKind: provider.Kind,
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
	attempts := s.cfg.GetAIMaxRetries()
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := range attempts {
		resp, err := adapter.Complete(ctx, call)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		if !modeladapter.IsRetryable(err) || ctx.Err() != nil {
			return nil, err
		}

		s.logger.Debug("retrying provider request",
			zap.String("provider", call.Provider.Name),
			zap.Int("attempt", attempt+1),
			zap.Error(err),
		)
	}

	return nil, lastErr
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

// clientFor returns the shared client matching the provider's egress policy.
func (s *Service) clientFor(provider *aiprovider.Provider) *http.Client {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	allowPrivate := provider.AllowPrivateNetwork
	if client, ok := s.clients[allowPrivate]; ok {
		return client
	}

	policy := httpsafe.Policy{
		AllowPrivateNetworks: allowPrivate,
		// A self-hosted model holds the connection open while it generates and
		// sends nothing until the first token, which on a loaded GPU outlasts the
		// transport default.
		ResponseHeaderTimeout: s.cfg.GetAITimeout(),
	}
	client := httpsafe.NewClientWithPolicy(s.cfg.GetAITimeout(), policy)
	s.clients[allowPrivate] = client

	return client
}
