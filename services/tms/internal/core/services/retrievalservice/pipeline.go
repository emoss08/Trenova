package retrievalservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	DefaultBatchSize  = 25
	MaxBatchSize      = 200
	claimLease        = 10 * time.Minute
	sweepPageSize     = 500
	maxSweepPages     = 10
	reindexPageSize   = 500
	purgeBatchSize    = 1000
	maxPurgeBatches   = 20
	maxAttempts       = 8
	firstRetryDelay   = time.Minute
	maxRetryDelay     = 6 * time.Hour
	maxOutcomeMessage = 500
)

func unavailable(reason airetrieval.UnavailableReason) serviceports.RetrievalIndexPlan {
	return serviceports.RetrievalIndexPlan{Reason: reason}
}

func (s *Service) Plan(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (serviceports.RetrievalIndexPlan, error) {
	availability, err := s.repo.VectorAvailability(ctx)
	if err != nil {
		return serviceports.RetrievalIndexPlan{}, fmt.Errorf("check vector storage: %w", err)
	}
	if !availability.Available {
		return unavailable(availability.Reason), nil
	}

	settings, err := s.repo.GetSettings(ctx, tenant)
	if err != nil {
		return serviceports.RetrievalIndexPlan{}, fmt.Errorf("read retrieval settings: %w", err)
	}

	sourceTypes := settings.EnabledSourceTypes()
	if len(sourceTypes) == 0 || (settings.Paused && !settings.PausedByBudget()) {
		return unavailable(airetrieval.UnavailableReasonDisabled), nil
	}

	settings, reason, err := s.adoptConfiguredModel(ctx, tenant, settings)
	if err != nil || reason != "" {
		return unavailable(reason), err
	}

	settings, within, err := s.applyBudget(ctx, tenant, settings)
	if err != nil {
		return serviceports.RetrievalIndexPlan{}, err
	}
	if !within {
		return unavailable(airetrieval.UnavailableReasonBudgetPaused), nil
	}

	retired, err := s.retiredKeys(ctx, tenant, settings)
	if err != nil {
		return serviceports.RetrievalIndexPlan{}, err
	}

	return serviceports.RetrievalIndexPlan{
		Active:          true,
		ActiveModelKey:  settings.ActiveModelKey,
		PendingModelKey: settings.PendingModelKey,
		SourceTypes:     sourceTypes,
		RetiredKeys:     retired,
	}, nil
}

func (s *Service) adoptConfiguredModel(
	ctx context.Context,
	tenant pagination.TenantInfo,
	settings *airetrieval.Settings,
) (*airetrieval.Settings, airetrieval.UnavailableReason, error) {
	if s.embeddings == nil {
		return settings, airetrieval.UnavailableReasonNoProvider, nil
	}

	configured, err := s.embeddings.ConfiguredModelKey(ctx, tenant)
	if err != nil {
		if isNoProvider(err) {
			return settings, airetrieval.UnavailableReasonNoProvider, nil
		}
		return settings, "", fmt.Errorf("resolve the embedding model: %w", err)
	}

	dimensions, ok := airetrieval.ModelKeyDimensions(configured)
	if !ok {
		s.l.Warn("the configured embedding model key names no allowed dimension",
			zap.String("organizationId", tenant.OrgID.String()),
			zap.String("modelKey", configured),
		)
		return settings, airetrieval.UnavailableReasonNoProvider, nil
	}

	next, changed := ResolveModelChange(settings, configured, dimensions)
	if !changed {
		return settings, "", nil
	}

	updated, err := s.repo.UpdateSettings(ctx, next)
	if err != nil {
		return settings, "", fmt.Errorf("record the embedding model: %w", err)
	}

	s.l.Info("retrieval embedding model recorded",
		zap.String("organizationId", tenant.OrgID.String()),
		zap.String("activeModelKey", updated.ActiveModelKey),
		zap.String("pendingModelKey", updated.PendingModelKey),
	)

	return updated, "", nil
}

func ResolveModelChange(
	settings *airetrieval.Settings,
	configured string,
	dimensions int,
) (*airetrieval.Settings, bool) {
	next := *settings

	switch {
	case !settings.HasActiveModel():
		next.ActiveModelKey = configured
		next.Dimensions = dimensions
		next.PendingModelKey = ""
		next.PendingDimensions = 0
	case configured == settings.ActiveModelKey:
		if !settings.HasPendingModel() {
			return settings, false
		}
		next.PendingModelKey = ""
		next.PendingDimensions = 0
	case configured == settings.PendingModelKey:
		return settings, false
	default:
		next.PendingModelKey = configured
		next.PendingDimensions = dimensions
	}

	return &next, true
}

func (s *Service) spentThisMonth(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (decimal.Decimal, error) {
	cost, err := s.usage.SurfaceCost(ctx, repositories.AIUsageSurfaceCostRequest{
		TenantInfo: tenant,
		Surface:    aiusage.SurfaceIndexing,
		Since:      timeutils.MonthStartUTC(s.now()),
	})
	if err != nil {
		return decimal.Zero, fmt.Errorf("sum this month's indexing cost: %w", err)
	}

	return cost.CostUSD, nil
}

func (s *Service) applyBudget(
	ctx context.Context,
	tenant pagination.TenantInfo,
	settings *airetrieval.Settings,
) (*airetrieval.Settings, bool, error) {
	spent, err := s.spentThisMonth(ctx, tenant)
	if err != nil {
		return settings, false, err
	}

	within := spent.LessThan(settings.MonthlyIndexingBudgetUSD)
	switch {
	case !within && !settings.PausedByBudget():
		return s.setBudgetPause(ctx, tenant, settings, true)
	case within && settings.PausedByBudget():
		return s.setBudgetPause(ctx, tenant, settings, false)
	default:
		return settings, within, nil
	}
}

func (s *Service) setBudgetPause(
	ctx context.Context,
	tenant pagination.TenantInfo,
	settings *airetrieval.Settings,
	paused bool,
) (*airetrieval.Settings, bool, error) {
	req := repositories.SetAIRetrievalPausedRequest{
		TenantInfo: tenant,
		Paused:     paused,
		Now:        s.now(),
	}
	if paused {
		req.Reason = airetrieval.PauseReasonBudget
	}

	updated, err := s.repo.SetPaused(ctx, req)
	if err != nil {
		return settings, false, fmt.Errorf("record the indexing budget pause: %w", err)
	}

	s.l.Info("retrieval indexing budget pause changed",
		zap.String("organizationId", tenant.OrgID.String()),
		zap.Bool("paused", paused),
	)

	return updated, !paused, nil
}

func (s *Service) retiredKeys(
	ctx context.Context,
	tenant pagination.TenantInfo,
	settings *airetrieval.Settings,
) ([]string, error) {
	keys, err := s.sources.ListModelKeys(ctx, repositories.ListRetrievalModelKeysRequest{
		TenantInfo:        tenant,
		IncludeEmbeddings: true,
	})
	if err != nil {
		return nil, fmt.Errorf("list indexed model keys: %w", err)
	}

	indexed := settings.IndexedModelKeys()

	return slices.DeleteFunc(keys, func(key string) bool {
		return slices.Contains(indexed, key)
	}), nil
}

func (s *Service) IndexBatch(
	ctx context.Context,
	req *serviceports.RetrievalIndexBatchRequest,
) (serviceports.RetrievalIndexBatchResult, error) {
	result := serviceports.RetrievalIndexBatchResult{CostUSD: decimal.Zero}

	if req == nil {
		return result, fmt.Errorf(
			"%w: an index batch needs a request", airetrieval.ErrInvalidStorageRequest,
		)
	}

	dimensions, ok := airetrieval.ModelKeyDimensions(req.ModelKey)
	if !ok {
		return result, fmt.Errorf("%w: model key %q names no allowed dimension",
			airetrieval.ErrInvalidStorageRequest, req.ModelKey)
	}
	if s.embeddings == nil {
		return result, nil
	}

	settings, err := s.repo.GetSettings(ctx, req.TenantInfo)
	if err != nil {
		return result, fmt.Errorf("read retrieval settings: %w", err)
	}
	if !slices.Contains(settings.IndexedModelKeys(), req.ModelKey) {
		return result, nil
	}
	if settings.Paused {
		result.BudgetReached = settings.PausedByBudget()
		return result, nil
	}

	sourceTypes := enabledOf(settings, req.SourceTypes)
	if len(sourceTypes) == 0 {
		return result, nil
	}

	spent, err := s.spentThisMonth(ctx, req.TenantInfo)
	if err != nil {
		return result, err
	}
	if !spent.LessThan(settings.MonthlyIndexingBudgetUSD) {
		_, _, err = s.setBudgetPause(ctx, req.TenantInfo, settings, true)
		result.BudgetReached = err == nil
		return result, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = DefaultBatchSize
	}

	entries, err := s.repo.ClaimIndexEntries(ctx, repositories.ClaimIndexEntriesRequest{
		TenantInfo:  req.TenantInfo,
		ModelKey:    req.ModelKey,
		SourceTypes: sourceTypes,
		Limit:       min(limit, MaxBatchSize),
		Lease:       claimLease,
		Now:         s.now(),
	})
	if err != nil {
		return result, fmt.Errorf("claim index entries: %w", err)
	}
	result.Claimed = len(entries)
	if len(entries) == 0 {
		return result, nil
	}

	batch := &indexBatch{
		service:    s,
		tenant:     req.TenantInfo,
		modelKey:   req.ModelKey,
		dimensions: dimensions,
		now:        s.now(),
	}
	if err = batch.run(ctx, entries); err != nil {
		return result, err
	}

	batch.fill(&result)
	if !spent.Add(result.CostUSD).LessThan(settings.MonthlyIndexingBudgetUSD) {
		if _, _, err = s.setBudgetPause(ctx, req.TenantInfo, settings, true); err != nil {
			return result, err
		}
		result.BudgetReached = true
	}

	return result, nil
}

func enabledOf(
	settings *airetrieval.Settings,
	requested []airetrieval.SourceType,
) []airetrieval.SourceType {
	enabled := settings.EnabledSourceTypes()
	if len(requested) == 0 {
		return enabled
	}

	return slices.DeleteFunc(slices.Clone(requested), func(sourceType airetrieval.SourceType) bool {
		return !slices.Contains(enabled, sourceType)
	})
}

func RetryAt(now int64, attempts int) int64 {
	if attempts >= maxAttempts {
		return 0
	}

	delay := firstRetryDelay << max(attempts-1, 0)
	if delay <= 0 || delay > maxRetryDelay {
		delay = maxRetryDelay
	}

	return now + int64(delay/time.Second)
}

func (s *Service) CompleteModelChange(
	ctx context.Context,
	tenant pagination.TenantInfo,
	pendingModelKey string,
) (serviceports.RetrievalModelChangeResult, error) {
	var result serviceports.RetrievalModelChangeResult

	settings, err := s.repo.GetSettings(ctx, tenant)
	if err != nil {
		return result, fmt.Errorf("read retrieval settings: %w", err)
	}
	if pendingModelKey == "" || settings.PendingModelKey != pendingModelKey {
		return result, nil
	}

	complete, err := s.pendingModelComplete(ctx, tenant, settings)
	if err != nil || !complete {
		return result, err
	}

	swapped, err := s.repo.SwapModel(ctx, repositories.SwapAIRetrievalModelRequest{
		TenantInfo:      tenant,
		PendingModelKey: pendingModelKey,
	})
	if err != nil {
		if errors.Is(err, airetrieval.ErrNoPendingModel) ||
			errors.Is(err, airetrieval.ErrPendingModelMoved) {
			return result, nil
		}
		return result, fmt.Errorf("swap the embedding model: %w", err)
	}

	s.l.Info("retrieval embedding model swapped",
		zap.String("organizationId", tenant.OrgID.String()),
		zap.String("activeModelKey", swapped.Settings.ActiveModelKey),
		zap.String("retiredModelKey", swapped.RetiredModelKey),
	)

	return serviceports.RetrievalModelChangeResult{
		Swapped:         true,
		RetiredModelKey: swapped.RetiredModelKey,
	}, nil
}

func (s *Service) pendingModelComplete(
	ctx context.Context,
	tenant pagination.TenantInfo,
	settings *airetrieval.Settings,
) (bool, error) {
	for _, sourceType := range settings.EnabledSourceTypes() {
		stale, err := s.repo.FindStaleSources(ctx, repositories.FindStaleAIRetrievalSourcesRequest{
			TenantInfo: tenant,
			SourceType: sourceType,
			ModelKey:   settings.PendingModelKey,
			Limit:      1,
		})
		if err != nil {
			return false, fmt.Errorf(
				"check %s sources under the pending model: %w",
				sourceType,
				err,
			)
		}
		if len(stale) > 0 {
			return false, nil
		}
	}

	counts, err := s.repo.CountIndexEntries(ctx, repositories.CountIndexEntriesRequest{
		TenantInfo: tenant,
		ModelKey:   settings.PendingModelKey,
	})
	if err != nil {
		return false, fmt.Errorf("count pending model entries: %w", err)
	}

	return !slices.ContainsFunc(counts, func(count repositories.IndexEntryCount) bool {
		return count.Status == airetrieval.IndexStatusPending && count.Count > 0
	}), nil
}

func (s *Service) PurgeRetiredModel(
	ctx context.Context,
	tenant pagination.TenantInfo,
	modelKey string,
) (serviceports.RetrievalPurgeResult, error) {
	var result serviceports.RetrievalPurgeResult

	for range maxPurgeBatches {
		purged, err := s.repo.PurgeModel(ctx, repositories.PurgeAIRetrievalModelRequest{
			TenantInfo: tenant,
			ModelKey:   modelKey,
			BatchSize:  purgeBatchSize,
		})
		if errors.Is(err, airetrieval.ErrModelInUse) {
			result.Done = true
			return result, nil
		}
		if err != nil {
			return result, fmt.Errorf("purge retired model %s: %w", modelKey, err)
		}

		result.Embeddings += purged.Embeddings
		result.IndexEntries += purged.IndexEntries
		if purged.Done() {
			result.Done = true
			return result, nil
		}
		if err = ctx.Err(); err != nil {
			return result, err
		}
	}

	return result, nil
}

func (s *Service) Sweep(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (serviceports.RetrievalSweepResult, error) {
	var result serviceports.RetrievalSweepResult

	settings, err := s.repo.GetSettings(ctx, tenant)
	if err != nil {
		return result, fmt.Errorf("read retrieval settings: %w", err)
	}
	if !settings.HasActiveModel() {
		return result, nil
	}

	keys := settings.IndexedModelKeys()
	for _, sourceType := range settings.EnabledSourceTypes() {
		for _, modelKey := range keys {
			marked, sweepErr := s.sweepSource(ctx, tenant, sourceType, modelKey)
			result.Marked += marked
			if sweepErr != nil {
				return result, sweepErr
			}
		}
	}

	for _, modelKey := range keys {
		counts, countErr := s.repo.CountIndexEntries(ctx, repositories.CountIndexEntriesRequest{
			TenantInfo: tenant,
			ModelKey:   modelKey,
		})
		if countErr != nil {
			return result, fmt.Errorf("count index entries: %w", countErr)
		}
		for _, count := range counts {
			if count.Status == airetrieval.IndexStatusPending {
				result.Waiting += count.Count
			}
		}
	}

	return result, nil
}

func (s *Service) sweepSource(
	ctx context.Context,
	tenant pagination.TenantInfo,
	sourceType airetrieval.SourceType,
	modelKey string,
) (int, error) {
	marked := 0
	after := pulid.Nil
	for range maxSweepPages {
		ids, err := s.repo.FindStaleSources(ctx, repositories.FindStaleAIRetrievalSourcesRequest{
			TenantInfo: tenant,
			SourceType: sourceType,
			ModelKey:   modelKey,
			AfterID:    after,
			Limit:      sweepPageSize,
		})
		if err != nil {
			return marked, fmt.Errorf("find stale %s sources: %w", sourceType, err)
		}
		if len(ids) == 0 {
			return marked, nil
		}

		count, err := s.markStale(ctx, tenant, sourceType, []string{modelKey}, ids)
		marked += count
		if err != nil || len(ids) < sweepPageSize {
			return marked, err
		}
		after = ids[len(ids)-1]
	}

	return marked, nil
}

func (s *Service) ReindexPage(
	ctx context.Context,
	req *serviceports.RetrievalReindexPageRequest,
) (serviceports.RetrievalReindexPage, error) {
	var page serviceports.RetrievalReindexPage

	if req == nil {
		return page, fmt.Errorf(
			"%w: a re-index page needs a request", airetrieval.ErrInvalidStorageRequest,
		)
	}
	if err := validateSourceType(req.SourceType); err != nil {
		return page, err
	}

	settings, err := s.repo.GetSettings(ctx, req.TenantInfo)
	if err != nil {
		return page, fmt.Errorf("read retrieval settings: %w", err)
	}
	if !settings.SourceEnabled(req.SourceType) || !settings.HasActiveModel() {
		page.Done = true
		return page, nil
	}

	limit := req.Limit
	if limit <= 0 {
		limit = reindexPageSize
	}

	ids, err := s.sources.ListSourceIDs(ctx, &repositories.ListRetrievalSourceIDsRequest{
		TenantInfo: req.TenantInfo,
		SourceType: req.SourceType,
		AfterID:    req.AfterID,
		Limit:      limit,
	})
	if err != nil {
		return page, fmt.Errorf("list %s sources: %w", req.SourceType, err)
	}

	page.Done = len(ids) < limit
	if len(ids) == 0 {
		return page, nil
	}
	page.Next = ids[len(ids)-1]

	page.Marked, err = s.markStale(
		ctx,
		req.TenantInfo,
		req.SourceType,
		settings.IndexedModelKeys(),
		ids,
	)
	if err != nil {
		return page, err
	}
	if page.Marked > 0 && !settings.Paused {
		s.wake(ctx, req.TenantInfo, IndexSignal{SourceType: req.SourceType, Count: page.Marked})
	}

	return page, nil
}
