package completionrouter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/llmtokens"
	"github.com/emoss08/trenova/shared/piiscrub"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	EmbeddingBatchMaxInputs       = 96
	EmbeddingBatchMaxTokens       = 60_000
	GeminiEmbeddingBatchMaxInputs = 100
	MaxEmbeddingInputs            = 10_000

	geminiEmbeddingHost = "generativelanguage.googleapis.com"
)

type embeddingBatchLimits struct {
	inputs int
	tokens int
}

func embeddingLimitsFor(provider *aiprovider.Provider) embeddingBatchLimits {
	limits := embeddingBatchLimits{
		inputs: EmbeddingBatchMaxInputs,
		tokens: EmbeddingBatchMaxTokens,
	}
	if provider.EmbeddingHost() == geminiEmbeddingHost {
		limits.inputs = GeminiEmbeddingBatchMaxInputs
	}

	return limits
}

type embeddingBatch struct {
	inputs []string
	tokens int
}

func batchEmbeddingInputs(inputs []string, limits embeddingBatchLimits) []embeddingBatch {
	batches := make([]embeddingBatch, 0, len(inputs)/limits.inputs+1)
	start := 0
	tokens := 0
	for idx, input := range inputs {
		estimate := llmtokens.Estimate(input)
		count := idx - start
		if count > 0 && (count >= limits.inputs || tokens+estimate > limits.tokens) {
			batches = append(batches, embeddingBatch{inputs: inputs[start:idx], tokens: tokens})
			start = idx
			tokens = 0
		}
		tokens += estimate
	}
	if start < len(inputs) {
		batches = append(batches, embeddingBatch{inputs: inputs[start:], tokens: tokens})
	}

	return batches
}

func (s *Service) Embed(
	ctx context.Context,
	req *serviceports.EmbedRequest,
) (serviceports.EmbedResult, error) {
	if !s.ai.AIEnabled() {
		return serviceports.EmbedResult{}, errortypes.NewBusinessError(aiDisabledMessage)
	}
	if req == nil {
		return serviceports.EmbedResult{}, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"An embedding request is required",
		)
	}
	if err := validateEmbedRequest(req); err != nil {
		return serviceports.EmbedResult{}, err
	}

	if req.Purpose == serviceports.EmbeddingPurposeQuery {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.ResolvedQueryTimeout())
		defer cancel()
	}

	candidates, modelKey, err := s.embeddingCandidates(ctx, req.TenantInfo, req.ModelKey)
	if err != nil {
		return serviceports.EmbedResult{}, err
	}

	inputs := req.Inputs
	if req.Purpose == serviceports.EmbeddingPurposeDocument {
		inputs = piiscrub.ScrubAll(req.Inputs)
	}

	started := time.Now()
	result := serviceports.EmbedResult{
		Vectors:    make([][]float32, 0, len(inputs)),
		ModelKey:   modelKey,
		Dimensions: candidates[0].ResolvedEmbeddingDimensions(),
	}
	cost := decimal.Zero
	priced := true

	for _, batch := range batchEmbeddingInputs(inputs, embeddingLimitsFor(candidates[0])) {
		served, batchErr := s.embedBatch(ctx, candidates, req, batch)
		if batchErr != nil {
			return serviceports.EmbedResult{}, batchErr
		}

		result.Vectors = append(result.Vectors, served.vectors...)
		result.InputTokens += served.tokens
		result.ProviderID = served.provider.ID
		result.ProviderKind = served.provider.Kind
		result.Model = served.model
		if served.cost == nil {
			priced = false
		} else {
			cost = cost.Add(*served.cost)
		}
	}

	if priced {
		result.CostUSD = &cost
	}
	result.LatencyMs = time.Since(started).Milliseconds()

	return result, nil
}

func (s *Service) ConfiguredModelKey(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (string, error) {
	if !s.ai.AIEnabled() {
		return "", errortypes.NewBusinessError(aiDisabledMessage)
	}

	usable, err := s.usableFor(ctx, aiprovider.TaskEmbedding, tenant)
	if err != nil {
		return "", err
	}

	return usable[0].EmbeddingModelKey(), nil
}

func validateEmbedRequest(req *serviceports.EmbedRequest) error {
	multiErr := errortypes.NewMultiError()

	if !req.Purpose.IsValid() {
		multiErr.Add(
			"purpose",
			errortypes.ErrInvalid,
			"Embedding purpose must be Document or Query",
		)
	}

	if !req.ResolvedSurface().IsEmbedding() {
		multiErr.Add(
			"surface",
			errortypes.ErrInvalid,
			"Embedding usage must be recorded as Indexing or Retrieval",
		)
	}

	switch {
	case len(req.Inputs) == 0:
		multiErr.Add("inputs", errortypes.ErrRequired, "At least one text is required to embed")
	case len(req.Inputs) > MaxEmbeddingInputs:
		multiErr.Add(
			"inputs",
			errortypes.ErrInvalid,
			"At most {0} texts can be embedded in one request",
			MaxEmbeddingInputs,
		)
	}

	for idx, input := range req.Inputs {
		if strings.TrimSpace(input) == "" {
			multiErr.Add(
				fmt.Sprintf("inputs[%d]", idx),
				errortypes.ErrRequired,
				"A text to embed cannot be blank",
			)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) embeddingCandidates(
	ctx context.Context,
	tenant pagination.TenantInfo,
	pinned string,
) ([]*aiprovider.Provider, string, error) {
	usable, err := s.usableFor(ctx, aiprovider.TaskEmbedding, tenant)
	if err != nil {
		return nil, "", err
	}

	modelKey := pinned
	if modelKey == "" {
		modelKey = usable[0].EmbeddingModelKey()
	}

	sameModel := make([]*aiprovider.Provider, 0, len(usable))
	for _, provider := range usable {
		if provider.EmbeddingModelKey() == modelKey {
			sameModel = append(sameModel, provider)
		}
	}
	if len(sameModel) == 0 {
		return nil, "", errortypes.NewBusinessError(
			"No enabled embedding provider serves {0}. Re-enable it, or re-index with the model now configured",
			modelKey,
		).WithInternal(serviceports.ErrNoProviderConfigured)
	}

	ready, err := s.awake(sameModel)
	if err != nil {
		return nil, "", err
	}

	return ready, modelKey, nil
}

type servedBatch struct {
	vectors  [][]float32
	tokens   int
	cost     *decimal.Decimal
	model    string
	provider *aiprovider.Provider
}

func (s *Service) embedBatch(
	ctx context.Context,
	candidates []*aiprovider.Provider,
	req *serviceports.EmbedRequest,
	batch embeddingBatch,
) (*servedBatch, error) {
	var lastErr error
	for _, provider := range candidates {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		started := time.Now()
		served, attemptErr := s.attemptEmbed(ctx, provider, req.Purpose, batch)
		latency := time.Since(started)
		attemptErr = stopped(ctx, attemptErr)
		s.observe(ctx, provider, attemptErr)

		var outcome *runOutcome
		if served != nil {
			outcome = &runOutcome{Model: served.model, InputTokens: served.tokens}
		}
		s.record(ctx, usageAttempt{
			provider:    provider,
			task:        aiprovider.TaskEmbedding,
			surface:     surfaceFor(req.ResolvedSurface(), req.Attribution),
			attribution: req.Attribution,
			tenant:      req.TenantInfo,
			latency:     latency,
			outcome:     outcome,
			err:         attemptErr,
		})

		if attemptErr == nil {
			served.cost = provider.InputCostFor(served.tokens)

			return served, nil
		}
		if ctx.Err() != nil {
			return nil, attemptErr
		}

		lastErr = attemptErr
		s.logger.Warn("embedding provider attempt failed, falling through",
			zap.String("provider", provider.Name),
			zap.String("kind", string(provider.Kind)),
			zap.Int("inputs", len(batch.inputs)),
			zap.Error(attemptErr),
		)
	}

	return nil, fmt.Errorf(
		"every embedding provider for %s failed: %w",
		candidates[0].EmbeddingModelKey(),
		lastErr,
	)
}

func (s *Service) attemptEmbed(
	ctx context.Context,
	provider *aiprovider.Provider,
	purpose serviceports.EmbeddingPurpose,
	batch embeddingBatch,
) (*servedBatch, error) {
	embedder, err := s.adapters.Embedder(provider.Kind)
	if err != nil {
		return nil, err
	}

	apiKey, err := s.resolveAPIKey(provider)
	if err != nil {
		return nil, err
	}

	call := &modeladapter.EmbedCall{
		Provider: provider,
		APIKey:   apiKey,
		Client:   s.clientFor(provider),
		Purpose:  purpose,
		Inputs:   batch.inputs,
	}

	var resp *modeladapter.EmbedResponse
	if err = s.retrying(ctx, func() error {
		var callErr error
		resp, callErr = embedder.Embed(ctx, call)

		return callErr
	}, nil); err != nil {
		return nil, err
	}

	tokens := resp.InputTokens
	if tokens <= 0 {
		tokens = batch.tokens
	}

	return &servedBatch{
		vectors:  resp.Vectors,
		tokens:   tokens,
		model:    resp.ModelIdentifier,
		provider: provider,
	}, nil
}
