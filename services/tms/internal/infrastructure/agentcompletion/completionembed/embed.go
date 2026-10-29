package completionembed

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/completionengine"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aipricing"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
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

type Embedder struct {
	*completionengine.Engine
}

func (s *Embedder) Embed(
	ctx context.Context,
	req *aiusageports.EmbedRequest,
) (aiusageports.EmbedResult, error) {
	if !s.AI.AIEnabled() {
		return aiusageports.EmbedResult{}, errortypes.NewBusinessError(
			completionengine.AIDisabledMessage,
		)
	}
	if req == nil {
		return aiusageports.EmbedResult{}, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"An embedding request is required",
		)
	}
	if err := validateEmbedRequest(req); err != nil {
		return aiusageports.EmbedResult{}, err
	}

	if req.Purpose == aiusageports.EmbeddingPurposeQuery {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.ResolvedQueryTimeout())
		defer cancel()
	}

	candidates, modelKey, err := s.embeddingCandidates(ctx, req.TenantInfo, req.ModelKey)
	if err != nil {
		return aiusageports.EmbedResult{}, err
	}

	inputs := req.Inputs
	if req.Purpose == aiusageports.EmbeddingPurposeDocument {
		inputs = piiscrub.ScrubAll(req.Inputs)
	}

	started := time.Now()
	result := aiusageports.EmbedResult{
		Vectors:    make([][]float32, 0, len(inputs)),
		ModelKey:   modelKey,
		Dimensions: candidates[0].ResolvedEmbeddingDimensions(),
	}
	cost := decimal.Zero
	priced := true

	for _, batch := range batchEmbeddingInputs(inputs, embeddingLimitsFor(candidates[0])) {
		served, batchErr := s.embedBatch(ctx, candidates, req, batch)
		if batchErr != nil {
			return aiusageports.EmbedResult{}, batchErr
		}

		result.Vectors = append(result.Vectors, served.vectors...)
		result.InputTokens += served.tokens
		result.ProviderID = served.Provider.ID
		result.ProviderKind = served.Provider.Kind
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

func (s *Embedder) ConfiguredModelKey(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (string, error) {
	if !s.AI.AIEnabled() {
		return "", errortypes.NewBusinessError(completionengine.AIDisabledMessage)
	}

	usable, err := s.UsableFor(ctx, aitask.TaskEmbedding, tenant)
	if err != nil {
		return "", err
	}

	return usable[0].EmbeddingModelKey(), nil
}

func validateEmbedRequest(req *aiusageports.EmbedRequest) error {
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

func (s *Embedder) embeddingCandidates(
	ctx context.Context,
	tenant pagination.TenantInfo,
	pinned string,
) ([]*aiprovider.Provider, string, error) {
	usable, err := s.UsableFor(ctx, aitask.TaskEmbedding, tenant)
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
		).WithInternal(llmports.ErrNoProviderConfigured)
	}

	ready, err := s.Awake(ctx, sameModel)
	if err != nil {
		return nil, "", err
	}
	if ready, err = s.UnderCap(ctx, tenant, ready); err != nil {
		return nil, "", err
	}

	return ready, modelKey, nil
}

type servedBatch struct {
	vectors  [][]float32
	tokens   int
	cost     *decimal.Decimal
	model    string
	Provider *aiprovider.Provider
}

func (s *Embedder) embedBatch(
	ctx context.Context,
	candidates []*aiprovider.Provider,
	req *aiusageports.EmbedRequest,
	batch embeddingBatch,
) (*servedBatch, error) {
	var lastErr error
	for idx, provider := range candidates {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		attemptCtx, span := s.StartAttempt(ctx, &completionengine.AttemptSpec{
			Operation:   aitrace.OperationEmbeddings,
			Provider:    provider,
			Attempt:     idx + 1,
			Failover:    idx > 0,
			Attribution: req.Attribution,
		})
		started := time.Now()
		served, attemptErr := s.attemptEmbed(
			completionengine.LastCandidate(
				attemptCtx,
				idx,
				len(candidates),
			),
			provider,
			req.Purpose,
			batch,
		)
		latency := time.Since(started)
		attemptErr = completionengine.Stopped(ctx, attemptErr)
		s.Observe(ctx, provider, attemptErr)

		var outcome *completionengine.RunOutcome
		if served != nil {
			outcome = &completionengine.RunOutcome{Model: served.model, InputTokens: served.tokens}
		}
		s.SettleAttempt(attemptCtx, span, &completionengine.UsageAttempt{
			Provider:    provider,
			Task:        aitask.TaskEmbedding,
			Surface:     completionengine.SurfaceFor(req.ResolvedSurface(), req.Attribution),
			Attribution: req.Attribution,
			Tenant:      req.TenantInfo,
			Latency:     latency,
			Outcome:     outcome,
			Err:         attemptErr,
			Operation:   aitrace.OperationEmbeddings,
			Attempt:     idx + 1,
			Failover:    idx > 0,
		})

		if attemptErr == nil {
			s.TouchKey(ctx, provider, req.TenantInfo)
			served.cost = aipricing.InputCostFor(
				provider,
				aipricing.TokenUsage{Input: served.tokens},
			)

			return served, nil
		}
		if ctx.Err() != nil {
			return nil, attemptErr
		}

		lastErr = attemptErr
		s.Logger.Warn("embedding provider attempt failed, falling through",
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

func (s *Embedder) attemptEmbed(
	ctx context.Context,
	provider *aiprovider.Provider,
	purpose aiusageports.EmbeddingPurpose,
	batch embeddingBatch,
) (*servedBatch, error) {
	embedder, err := s.Adapters.Embedder(provider.Kind)
	if err != nil {
		return nil, err
	}

	timeout := provider.ResolvedTimeout()
	release, err := s.ClaimSlot(ctx, provider, timeout)
	if err != nil {
		return nil, err
	}
	defer release()

	call := &modeladapter.EmbedCall{
		Provider: provider,
		Client:   s.ClientFor(provider),
		Purpose:  purpose,
		Inputs:   batch.inputs,
	}

	limited := s.WithDeadline(ctx, timeout)
	var resp *modeladapter.EmbedResponse
	if err = limited.Finish(s.WithKeyFallback(provider, func(apiKey string) error {
		call.APIKey = apiKey
		return s.Retrying(limited.Ctx, func() error {
			var callErr error
			resp, callErr = embedder.Embed(limited.Ctx, call)

			return callErr
		}, nil)
	})); err != nil {
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
		Provider: provider,
	}, nil
}
