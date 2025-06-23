package analytics

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type DedicatedLaneAnalyticsParams struct {
	fx.In

	Logger         *logger.Logger
	SuggestionRepo repositories.DedicatedLaneSuggestionRepository
}

type DedicatedLaneAnalyticsProvider struct {
	l        *zerolog.Logger
	suggRepo repositories.DedicatedLaneSuggestionRepository
}

func NewDedicatedLaneAnalyticsProvider(
	p DedicatedLaneAnalyticsParams,
) *DedicatedLaneAnalyticsProvider {
	log := p.Logger.With().
		Str("analytics_provider", "dedicated_lane").
		Logger()

	return &DedicatedLaneAnalyticsProvider{
		l:        &log,
		suggRepo: p.SuggestionRepo,
	}
}

func (dlap *DedicatedLaneAnalyticsProvider) GetPage() services.AnalyticsPage {
	return services.DedicatedLaneSuggestionsPage
}

func (dlap *DedicatedLaneAnalyticsProvider) GetAnalyticsData(
	ctx context.Context,
	opts *services.AnalyticsRequestOptions,
) (services.AnalyticsData, error) {
	log := dlap.l.With().
		Str("operation", "GetAnalyticsData").
		Str("orgId", opts.OrgID.String()).
		Logger()

	log.Info().Msg("fetching dedicated lane suggestion analytics")

	data := make(services.AnalyticsData)

	// Get pending suggestions count
	pendingCount, err := dlap.getPendingSuggestionsCount(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get pending suggestions count")
		return nil, err
	}
	data["pendingSuggestionsCount"] = pendingCount

	// Get processed suggestions count
	processedCount, err := dlap.getProcessedSuggestionsCount(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get processed suggestions count")
		return nil, err
	}
	data["processedSuggestionsCount"] = processedCount

	// Get acceptance rate
	acceptanceRate, err := dlap.getAcceptanceRate(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get acceptance rate")
		return nil, err
	}
	data["acceptanceRate"] = acceptanceRate

	// Get recent suggestions
	recentSuggestions, err := dlap.getRecentSuggestions(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get recent suggestions")
		return nil, err
	}
	data["recentSuggestions"] = recentSuggestions

	// Get top customers by suggestion count
	topCustomers, err := dlap.getTopCustomersBySuggestions(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get top customers")
		return nil, err
	}
	data["topCustomersBySuggestions"] = topCustomers

	// Get suggestion status breakdown
	statusBreakdown, err := dlap.getSuggestionStatusBreakdown(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get status breakdown")
		return nil, err
	}
	data["suggestionStatusBreakdown"] = statusBreakdown

	// Get expired suggestions count
	expiredCount, err := dlap.getExpiredSuggestionsCount(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get expired suggestions count")
		return nil, err
	}
	data["expiredSuggestionsCount"] = expiredCount

	log.Info().Msg("dedicated lane suggestion analytics fetched successfully")

	return data, nil
}

func (dlap *DedicatedLaneAnalyticsProvider) getPendingSuggestionsCount(
	ctx context.Context,
	opts *services.AnalyticsRequestOptions,
) (int64, error) {
	status := dedicatedlane.SuggestionStatusPending
	req := &repositories.ListDedicatedLaneSuggestionRequest{
		Filter: &ports.LimitOffsetQueryOptions{
			TenantOpts: &ports.TenantOptions{
				OrgID:  opts.OrgID,
				BuID:   opts.BuID,
				UserID: opts.UserID,
			},
			Limit:  1,
			Offset: 0,
		},
		Status:           &status,
		IncludeExpired:   false,
		IncludeProcessed: false,
	}

	result, err := dlap.suggRepo.List(ctx, req)
	if err != nil {
		return 0, err
	}

	return int64(result.Total), nil
}

func (dlap *DedicatedLaneAnalyticsProvider) getProcessedSuggestionsCount(
	ctx context.Context,
	opts *services.AnalyticsRequestOptions,
) (int64, error) {
	req := &repositories.ListDedicatedLaneSuggestionRequest{
		Filter: &ports.LimitOffsetQueryOptions{
			TenantOpts: &ports.TenantOptions{
				OrgID:  opts.OrgID,
				BuID:   opts.BuID,
				UserID: opts.UserID,
			},
			Limit:  1,
			Offset: 0,
		},
		IncludeExpired:   true,
		IncludeProcessed: true,
	}

	result, err := dlap.suggRepo.List(ctx, req)
	if err != nil {
		return 0, err
	}

	processedCount := int64(0)
	for _, suggestion := range result.Items {
		if suggestion.IsProcessed() {
			processedCount++
		}
	}

	return processedCount, nil
}

func (dlap *DedicatedLaneAnalyticsProvider) getAcceptanceRate(
	ctx context.Context,
	opts *services.AnalyticsRequestOptions,
) (float64, error) {
	req := &repositories.ListDedicatedLaneSuggestionRequest{
		Filter: &ports.LimitOffsetQueryOptions{
			TenantOpts: &ports.TenantOptions{
				OrgID:  opts.OrgID,
				BuID:   opts.BuID,
				UserID: opts.UserID,
			},
			Limit:  1000, // Get enough for calculation
			Offset: 0,
		},
		IncludeExpired:   true,
		IncludeProcessed: true,
	}

	result, err := dlap.suggRepo.List(ctx, req)
	if err != nil {
		return 0, err
	}

	totalProcessed := int64(0)
	totalAccepted := int64(0)

	for _, suggestion := range result.Items {
		if suggestion.IsProcessed() {
			totalProcessed++
			if suggestion.Status == dedicatedlane.SuggestionStatusAccepted {
				totalAccepted++
			}
		}
	}

	if totalProcessed == 0 {
		return 0, nil
	}

	return float64(totalAccepted) / float64(totalProcessed), nil
}

func (dlap *DedicatedLaneAnalyticsProvider) getRecentSuggestions(
	ctx context.Context,
	opts *services.AnalyticsRequestOptions,
) ([]*dedicatedlane.DedicatedLaneSuggestion, error) {
	limit := 10
	if opts.Limit > 0 && opts.Limit < 50 {
		limit = opts.Limit
	}

	req := &repositories.ListDedicatedLaneSuggestionRequest{
		Filter: &ports.LimitOffsetQueryOptions{
			TenantOpts: &ports.TenantOptions{
				OrgID:  opts.OrgID,
				BuID:   opts.BuID,
				UserID: opts.UserID,
			},
			Limit:  limit,
			Offset: 0,
		},
		IncludeExpired:   true,
		IncludeProcessed: true,
	}

	result, err := dlap.suggRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	return result.Items, nil
}

func (dlap *DedicatedLaneAnalyticsProvider) getTopCustomersBySuggestions(
	ctx context.Context,
	opts *services.AnalyticsRequestOptions,
) (map[string]int64, error) {
	req := &repositories.ListDedicatedLaneSuggestionRequest{
		Filter: &ports.LimitOffsetQueryOptions{
			TenantOpts: &ports.TenantOptions{
				OrgID:  opts.OrgID,
				BuID:   opts.BuID,
				UserID: opts.UserID,
			},
			Limit:  1000,
			Offset: 0,
		},
		IncludeExpired:   true,
		IncludeProcessed: true,
	}

	result, err := dlap.suggRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	customerCounts := make(map[string]int64)
	for _, suggestion := range result.Items {
		customerCounts[suggestion.CustomerID.String()]++
	}

	return customerCounts, nil
}

func (dlap *DedicatedLaneAnalyticsProvider) getSuggestionStatusBreakdown(
	ctx context.Context,
	opts *services.AnalyticsRequestOptions,
) (map[string]int64, error) {
	req := &repositories.ListDedicatedLaneSuggestionRequest{
		Filter: &ports.LimitOffsetQueryOptions{
			TenantOpts: &ports.TenantOptions{
				OrgID:  opts.OrgID,
				BuID:   opts.BuID,
				UserID: opts.UserID,
			},
			Limit:  1000,
			Offset: 0,
		},
		IncludeExpired:   true,
		IncludeProcessed: true,
	}

	result, err := dlap.suggRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	breakdown := map[string]int64{
		string(dedicatedlane.SuggestionStatusPending):  0,
		string(dedicatedlane.SuggestionStatusAccepted): 0,
		string(dedicatedlane.SuggestionStatusRejected): 0,
		string(dedicatedlane.SuggestionStatusExpired):  0,
	}

	now := timeutils.NowUnix()
	for _, suggestion := range result.Items {
		if suggestion.ExpiresAt < now &&
			suggestion.Status == dedicatedlane.SuggestionStatusPending {
			breakdown[string(dedicatedlane.SuggestionStatusExpired)]++
		} else {
			breakdown[string(suggestion.Status)]++
		}
	}

	return breakdown, nil
}

func (dlap *DedicatedLaneAnalyticsProvider) getExpiredSuggestionsCount(
	ctx context.Context,
	opts *services.AnalyticsRequestOptions,
) (int64, error) {
	req := &repositories.ListDedicatedLaneSuggestionRequest{
		Filter: &ports.LimitOffsetQueryOptions{
			TenantOpts: &ports.TenantOptions{
				OrgID:  opts.OrgID,
				BuID:   opts.BuID,
				UserID: opts.UserID,
			},
			Limit:  1000,
			Offset: 0,
		},
		IncludeExpired:   true,
		IncludeProcessed: false,
	}

	result, err := dlap.suggRepo.List(ctx, req)
	if err != nil {
		return 0, err
	}

	expiredCount := int64(0)
	for _, suggestion := range result.Items {
		if suggestion.IsExpired() {
			expiredCount++
		}
	}

	return expiredCount, nil
}
