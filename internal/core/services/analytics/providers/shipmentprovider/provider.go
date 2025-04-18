package shipmentprovider

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// ShipmentProviderParams contains the dependencies for ShipmentProvider
type ProviderParams struct {
	fx.In

	Logger *logger.Logger
	DB     db.Connection
}

// ShipmentProvider provides analytics data for the shipment management page
type Provider struct {
	l  *zerolog.Logger
	db db.Connection
}

// NewProvider creates a new shipment analytics provider
func NewProvider(p ProviderParams) *Provider {
	log := p.Logger.With().
		Str("provider", "shipment_analytics").
		Logger()

	return &Provider{
		l:  &log,
		db: p.DB,
	}
}

// GetPage returns the page identifier this provider handles
func (p *Provider) GetPage() services.AnalyticsPage {
	return services.ShipmentAnalyticsPage
}

// GetAnalyticsData returns the analytics data for the shipment management page
func (p *Provider) GetAnalyticsData(ctx context.Context, _ *services.AnalyticsRequestOptions) (services.AnalyticsData, error) {
	log := p.l.With().
		Str("operation", "GetAnalyticsData").
		Logger()

	shpCount, err := p.getShipmentCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get shipment count")
		return nil, eris.Wrap(err, "failed to get shipment count")
	}

	// Build the analytics response
	data := services.AnalyticsData{
		"shipmentCountCard": shpCount,
	}

	return data, nil
}

func (p *Provider) getShipmentCount(ctx context.Context) (*ShipmentCountCard, error) {
	log := p.l.With().
		Str("query", "getShipmentCount").
		Logger()

	dba, err := p.db.DB(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get database connection")
		return nil, eris.Wrap(err, "failed to get database connection")
	}

	// Get current month's count
	currentMonthCount, err := dba.NewSelect().
		Model((*shipment.Shipment)(nil)).
		Where("DATE_TRUNC('month', TO_TIMESTAMP(sp.created_at)::timestamp) = DATE_TRUNC('month', CURRENT_TIMESTAMP)").
		Count(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get current month shipment count")
		return nil, eris.Wrap(err, "failed to get current month shipment count")
	}

	// Get previous month's count
	previousMonthCount, err := dba.NewSelect().
		Model((*shipment.Shipment)(nil)).
		Where("DATE_TRUNC('month', TO_TIMESTAMP(sp.created_at)::timestamp) = DATE_TRUNC('month', CURRENT_TIMESTAMP - INTERVAL '1 month')").
		Count(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get previous month shipment count")
		return nil, eris.Wrap(err, "failed to get previous month shipment count")
	}

	// Calculate trend percentage
	var trendPercentage int
	if previousMonthCount > 0 {
		trendPercentage = int(((float64(currentMonthCount) - float64(previousMonthCount)) / float64(previousMonthCount)) * 100)
	}

	return &ShipmentCountCard{
		Count:           currentMonthCount,
		TrendPercentage: trendPercentage,
	}, nil
}
