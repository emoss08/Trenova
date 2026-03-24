package analyticsservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ServiceParams holds the dependencies for the analytics service
type Params struct {
	fx.In

	Logger   *zap.Logger
	Registry services.AnalyticsRegistry
}

type Service struct {
	l        *zap.Logger
	registry services.AnalyticsRegistry
}

func NewService(p Params) services.AnalyticsService {
	return &Service{
		l:        p.Logger.Named("service.anayltics"),
		registry: p.Registry,
	}
}

func (s *Service) GetRegistry() services.AnalyticsRegistry {
	return s.registry
}

func (s *Service) GetAnalytics(
	ctx context.Context,
	opts *services.AnalyticsRequestOptions,
) (services.AnalyticsData, error) {
	log := s.l.With(zap.String("operation", "GetAnalytics"), zap.Any("req", opts))

	provider, exists := s.registry.GetProvider(opts.Page)
	if !exists {
		return nil, errortypes.NewValidationError("page", "invalid_page",
			fmt.Sprintf("No analytics provider found for page: %s", opts.Page))
	}

	data, err := provider.GetAnalyticsData(ctx, opts)
	if err != nil {
		log.Error("failed to get analytics data from provider", zap.Error(err))
		return nil, err
	}

	data["page"] = string(opts.Page)

	return data, nil
}
