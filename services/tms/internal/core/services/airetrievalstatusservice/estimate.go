package airetrievalstatusservice

import (
	"context"
	"fmt"
	"math"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/retrievalservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/llmtokens"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
)

const (
	EstimateTextSample = 500
	MemoryHeaderTokens = 8
	ChunkHeaderTokens  = 24
	pricePlaces        = 6
)

type EstimateInputs struct {
	SourceType     airetrieval.SourceType
	Sources        int
	Skipped        int
	AverageChars   float64
	MeasuredChunks repositories.IndexChunkAverage
	InputPrice     *decimal.Decimal
	HasModel       bool
}

type Estimate struct {
	Sources               int
	AverageChunks         float64
	ChunksMeasured        bool
	AverageTokensPerChunk float64
	EstimatedTokens       int
	CostUSD               *decimal.Decimal
}

func EstimateReindex(in EstimateInputs) Estimate {
	out := Estimate{Sources: max(0, in.Sources-in.Skipped)}

	tokensPerSource := in.AverageChars / llmtokens.RunesPerToken
	chunkCap := float64(maxChunks(in.SourceType))

	switch {
	case in.MeasuredChunks.Entries > 0 && in.MeasuredChunks.AverageChunks > 0:
		out.AverageChunks = min(in.MeasuredChunks.AverageChunks, chunkCap)
		out.ChunksMeasured = true
	case in.SourceType == airetrieval.SourceTypeMemory:
		out.AverageChunks = 1
	default:
		stride := float64(retrievalservice.WindowTokens) *
			(1 - float64(retrievalservice.WindowOverlapPct)/100)
		out.AverageChunks = min(max(1, math.Ceil(tokensPerSource/stride)), chunkCap)
	}

	if in.SourceType == airetrieval.SourceTypeMemory {
		out.AverageTokensPerChunk = tokensPerSource/out.AverageChunks + MemoryHeaderTokens
	} else {
		withOverlap := tokensPerSource * (1 + float64(retrievalservice.WindowOverlapPct)/100)
		body := min(withOverlap/out.AverageChunks, float64(retrievalservice.WindowTokens))
		out.AverageTokensPerChunk = body + ChunkHeaderTokens
	}
	out.AverageTokensPerChunk = math.Round(out.AverageTokensPerChunk*10) / 10
	out.AverageChunks = math.Round(out.AverageChunks*100) / 100

	out.EstimatedTokens = int(math.Ceil(
		float64(out.Sources) * out.AverageChunks * out.AverageTokensPerChunk,
	))

	if in.HasModel && in.InputPrice != nil {
		cost := in.InputPrice.
			Mul(decimal.NewFromInt(int64(out.EstimatedTokens))).
			Div(decimal.NewFromInt(1_000_000))
		out.CostUSD = &cost
	}

	return out
}

func maxChunks(sourceType airetrieval.SourceType) int {
	switch sourceType {
	case airetrieval.SourceTypeDocument:
		return retrievalservice.MaxDocumentChunks
	case airetrieval.SourceTypeInboundMessage:
		return retrievalservice.MaxEmailChunks
	case airetrieval.SourceTypeMemory:
	}

	return 1
}

func (s *Service) ReindexEstimate(
	ctx context.Context,
	tenant pagination.TenantInfo,
	sourceType airetrieval.SourceType,
) (*serviceports.AIRetrievalReindexEstimate, error) {
	if err := validateTenant(tenant); err != nil {
		return nil, err
	}
	if err := validateSourceType(sourceType); err != nil {
		return nil, err
	}

	settings, err := s.repo.GetSettings(ctx, tenant)
	if err != nil {
		return nil, fmt.Errorf("read retrieval settings: %w", err)
	}
	configured, err := s.configuredModelKey(ctx, tenant)
	if err != nil {
		return nil, err
	}
	modelKey := configured
	if modelKey == "" {
		modelKey = settings.ActiveModelKey
	}

	in := EstimateInputs{SourceType: sourceType, HasModel: modelKey != ""}
	var spent *repositories.AIUsageCost

	group, gctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		count, gerr := s.sources.CountSources(gctx, repositories.CountRetrievalSourcesRequest{
			TenantInfo: tenant,
			SourceType: sourceType,
		})
		in.Sources = count
		return gerr
	})
	group.Go(func() error {
		chars, gerr := s.sources.AverageSourceChars(
			gctx,
			repositories.AverageRetrievalSourceCharsRequest{
				TenantInfo: tenant,
				SourceType: sourceType,
				Sample:     EstimateTextSample,
			},
		)
		in.AverageChars = chars.AverageChars
		return gerr
	})
	group.Go(func() error {
		var gerr error
		spent, gerr = s.surfaceCost(gctx, tenant, aiusage.SurfaceIndexing,
			timeutils.MonthStartUTC(s.now()))
		return gerr
	})
	if settings.HasActiveModel() {
		group.Go(func() error {
			return s.measureIndexed(gctx, tenant, sourceType, settings.ActiveModelKey, &in)
		})
	}
	if modelKey != "" {
		group.Go(func() error {
			price, priced, gerr := s.inputPrice(gctx, tenant, modelKey)
			if priced {
				in.InputPrice = &price
			}
			return gerr
		})
	}
	if err = group.Wait(); err != nil {
		return nil, err
	}

	estimate := EstimateReindex(in)
	remaining := decimal.Max(
		decimal.Zero,
		settings.MonthlyIndexingBudgetUSD.Sub(costOf(spent)),
	)

	out := &serviceports.AIRetrievalReindexEstimate{
		SourceType:            sourceType,
		ModelKey:              stringutils.Ptr(modelKey),
		Sources:               estimate.Sources,
		AverageChunks:         estimate.AverageChunks,
		ChunksMeasured:        estimate.ChunksMeasured,
		AverageTokensPerChunk: estimate.AverageTokensPerChunk,
		EstimatedTokens:       estimate.EstimatedTokens,
		RemainingBudgetUSD:    remaining.StringFixed(moneyPlaces),
	}
	if in.InputPrice != nil {
		out.InputCostPerMillionUSD = stringutils.Ptr(in.InputPrice.String())
	}
	if estimate.CostUSD != nil {
		out.EstimatedCostUSD = stringutils.Ptr(estimate.CostUSD.StringFixed(pricePlaces))
	}

	return out, nil
}

func (s *Service) measureIndexed(
	ctx context.Context,
	tenant pagination.TenantInfo,
	sourceType airetrieval.SourceType,
	modelKey string,
	in *EstimateInputs,
) error {
	average, err := s.repo.AverageIndexChunks(ctx, &repositories.AverageIndexChunksRequest{
		TenantInfo: tenant,
		SourceType: sourceType,
		ModelKey:   modelKey,
	})
	if err != nil {
		return fmt.Errorf("measure indexed %s chunks: %w", sourceType, err)
	}
	in.MeasuredChunks = average

	counts, err := s.countEntries(ctx, tenant, modelKey)
	if err != nil {
		return err
	}
	for _, count := range counts {
		if count.SourceType == sourceType && count.Status == airetrieval.IndexStatusSkipped {
			in.Skipped += count.Count
		}
	}

	return nil
}

func (s *Service) inputPrice(
	ctx context.Context,
	tenant pagination.TenantInfo,
	modelKey string,
) (decimal.Decimal, bool, error) {
	providers, err := s.providers.ListForTask(ctx, repositories.ListAIProvidersForTaskRequest{
		Task:       aiprovider.TaskEmbedding,
		TenantInfo: tenant,
	})
	if err != nil {
		return decimal.Zero, false, fmt.Errorf("list embedding providers: %w", err)
	}

	for _, provider := range providers {
		if ok, _ := provider.CanServeTask(aiprovider.TaskEmbedding); !ok {
			continue
		}
		if provider.EmbeddingModelKey() != modelKey || provider.InputCostPerMillion == nil {
			continue
		}
		return *provider.InputCostPerMillion, true, nil
	}

	return decimal.Zero, false, nil
}
