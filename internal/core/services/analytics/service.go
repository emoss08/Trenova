// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package analytics

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// ServiceParams holds the dependencies for the analytics service
type ServiceParams struct {
	fx.In

	Logger   *logger.Logger
	Registry services.AnalyticsRegistry
}

// Service implements the analytics service interface
type Service struct {
	l        *zerolog.Logger
	registry services.AnalyticsRegistry
}

// NewService creates a new analytics service
func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "analytics").
		Logger()

	return &Service{
		l:        &log,
		registry: p.Registry,
	}
}

// GetRegistry returns the analytics registry
func (s *Service) GetRegistry() services.AnalyticsRegistry {
	return s.registry
}

// GetAnalytics returns analytics data for a specific page
func (s *Service) GetAnalytics(
	ctx context.Context,
	opts *services.AnalyticsRequestOptions,
) (services.AnalyticsData, error) {
	log := s.l.With().
		Str("operation", "GetAnalytics").
		Str("page", string(opts.Page)).
		Logger()

	// Get the provider for the requested page
	provider, exists := s.registry.GetProvider(opts.Page)
	if !exists {
		return nil, errors.NewValidationError("page", "invalid_page",
			fmt.Sprintf("No analytics provider found for page: %s", opts.Page))
	}

	// Get analytics data from the provider
	data, err := provider.GetAnalyticsData(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get analytics data from provider")
		return nil, err
	}

	// Add metadata to the response
	data["page"] = string(opts.Page)

	return data, nil
}
