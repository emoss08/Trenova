package shipmentprovider

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
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
func (p *Provider) GetAnalyticsData(ctx context.Context, opts *services.AnalyticsRequestOptions) (services.AnalyticsData, error) {
	log := p.l.With().
		Str("operation", "GetAnalyticsData").
		Str("orgID", opts.OrgID.String()).
		Str("buID", opts.BuID.String()).
		Str("userID", opts.UserID.String()).
		Logger()

	shpCount, err := p.getShipmentCount(ctx, opts.OrgID, opts.BuID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get shipment count")
		return nil, eris.Wrap(err, "failed to get shipment count")
	}

	countByStatus, err := p.getCountByShipmentStatus(ctx, opts.OrgID, opts.BuID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get count by shipment status")
		return nil, eris.Wrap(err, "failed to get count by shipment status")
	}

	log.Info().Msgf("shipment count: %d", shpCount.Count)
	log.Info().Msgf("count by shipment status: %v", countByStatus)
	// Build the analytics response
	data := services.AnalyticsData{
		"shipmentCountCard":     shpCount,
		"countByShipmentStatus": countByStatus,
	}

	return data, nil
}

func (p *Provider) getShipmentCount(ctx context.Context, orgID, buID pulid.ID) (*ShipmentCountCard, error) {
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
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("sp.organization_id = ?", orgID).
				Where("sp.business_unit_id = ?", buID).
				Where("DATE_TRUNC('month', TO_TIMESTAMP(sp.created_at)::timestamp) = DATE_TRUNC('month', CURRENT_TIMESTAMP)")
		}).
		Count(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get current month shipment count")
		return nil, eris.Wrap(err, "failed to get current month shipment count")
	}

	// Get previous month's count
	previousMonthCount, err := dba.NewSelect().
		Model((*shipment.Shipment)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("sp.organization_id = ?", orgID).
				Where("sp.business_unit_id = ?", buID).
				Where("DATE_TRUNC('month', TO_TIMESTAMP(sp.created_at)::timestamp) = DATE_TRUNC('month', CURRENT_TIMESTAMP - INTERVAL '1 month')")
		}).
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

func (p *Provider) getCountByShipmentStatus(ctx context.Context, orgID, buID pulid.ID) ([]*CountByShipmentStatus, error) {
	log := p.l.With().
		Str("query", "getCountByShipmentStatus").
		Logger()

	dba, err := p.db.DB(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get database connection")
		return nil, eris.Wrap(err, "failed to get database connection")
	}

	countByStatus := make([]*CountByShipmentStatus, 0)
	err = dba.NewSelect().
		Model((*shipment.Shipment)(nil)).
		ColumnExpr("sp.status").
		ColumnExpr("COUNT(*)").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("sp.organization_id = ?", orgID).
				Where("sp.business_unit_id = ?", buID)
		}).
		GroupExpr("sp.status").
		Scan(ctx, &countByStatus)
	if err != nil {
		log.Error().Err(err).Msg("failed to get count by shipment status")
		return nil, eris.Wrap(err, "failed to get count by shipment status")
	}

	return countByStatus, nil
}
