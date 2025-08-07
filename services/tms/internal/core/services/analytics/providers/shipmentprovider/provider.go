/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package shipmentprovider

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

var _ services.AnalyticsPageProvider = (*Provider)(nil)

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
func (p *Provider) GetAnalyticsData(
	ctx context.Context,
	opts *services.AnalyticsRequestOptions,
) (services.AnalyticsData, error) {
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

	shipmentsByExpectedDeliverDate, err := p.getShipmentsByExpectedDeliveryDate(
		ctx,
		opts.OrgID,
		opts.BuID,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to get shipments by expected delivery date")
		return nil, eris.Wrap(err, "failed to get shipments by expected delivery date")
	}

	log.Info().Int("shipment_count", shpCount.Count).Msg("shipment count")
	log.Info().Interface("count_by_shipment_status", countByStatus).Msg("count by shipment status")
	// Build the analytics response
	data := services.AnalyticsData{
		"shipmentCountCard":                  shpCount,
		"countByShipmentStatus":              countByStatus,
		"shipmentsByExpectedDeliverDateCard": shipmentsByExpectedDeliverDate,
	}

	return data, nil
}

func (p *Provider) getShipmentCount(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*ShipmentCountCard, error) {
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
		trendPercentage = int(
			((float64(currentMonthCount) - float64(previousMonthCount)) / float64(previousMonthCount)) * 100,
		)
	}

	return &ShipmentCountCard{
		Count:           currentMonthCount,
		TrendPercentage: trendPercentage,
	}, nil
}

func (p *Provider) getCountByShipmentStatus(
	ctx context.Context,
	orgID, buID pulid.ID,
) ([]*CountByShipmentStatus, error) {
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

// * Get the number of shipments by expected delivery date (today)
func (p *Provider) getShipmentsByExpectedDeliveryDate(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*ShipmentsByExpectedDeliverDateCard, error) {
	log := p.l.With().
		Str("query", "getShipmentsByExpectedDeliveryDate").
		Logger()

	dba, err := p.db.DB(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get database connection")
		return nil, eris.Wrap(err, "failed to get database connection")
	}

	shipments := make([]*ShipmentSummary, 0)

	// Get current date in application timezone for comparison
	currentDate := timeutils.CurrentDateInTimezone("America/New_York")

	// Query shipments with delivery stops scheduled for today, selecting only needed fields
	err = dba.NewSelect().
		TableExpr("shipments sp").
		ColumnExpr("sp.id").
		ColumnExpr("sp.pro_number").
		ColumnExpr("sp.bol").
		ColumnExpr("sp.status").
		ColumnExpr("sp.created_at").
		ColumnExpr("c.id as customer_id").
		ColumnExpr("c.name as customer_name").
		ColumnExpr("stp.planned_arrival as expected_delivery").
		ColumnExpr("COALESCE(l.name, stp.address_line) as delivery_location").
		ColumnExpr("COALESCE(l.id, stp.location_id) as delivery_location_id").
		Join("JOIN customers c ON sp.customer_id = c.id").
		Join("JOIN shipment_moves sm ON sp.id = sm.shipment_id").
		Join("JOIN stops stp ON sm.id = stp.shipment_move_id").
		Join("LEFT JOIN locations l ON stp.location_id = l.id").
		Where("sp.organization_id = ?", orgID).
		Where("sp.business_unit_id = ?", buID).
		Where("sm.organization_id = ?", orgID).
		Where("sm.business_unit_id = ?", buID).
		Where("stp.organization_id = ?", orgID).
		Where("stp.business_unit_id = ?", buID).
		Where("sp.status NOT IN (?)", bun.In([]shipment.Status{
			shipment.StatusCanceled,
			shipment.StatusCompleted,
			shipment.StatusReadyToBill,
			shipment.StatusReviewRequired,
			shipment.StatusBilled,
		})).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("stp.type = ?", "Delivery").
				WhereOr("stp.type = ?", "SplitDelivery")
		}).
		Where("DATE(TO_TIMESTAMP(stp.planned_arrival) AT TIME ZONE 'America/New_York') = ?", currentDate).

		// Get the latest delivery stop for each shipment
		Where("(sm.shipment_id, sm.sequence) IN (?)",
			dba.NewSelect().
				TableExpr("shipment_moves sm2").
				Column("sm2.shipment_id").
				ColumnExpr("MAX(sm2.sequence)").
				Where("sm2.organization_id = ?", orgID).
				Where("sm2.business_unit_id = ?", buID).
				Group("sm2.shipment_id"),
		).
		Where("(stp.shipment_move_id, stp.sequence) IN (?)",
			dba.NewSelect().
				TableExpr("stops stp2").
				Column("stp2.shipment_move_id").
				ColumnExpr("MAX(stp2.sequence)").
				Where("stp2.organization_id = ?", orgID).
				Where("stp2.business_unit_id = ?", buID).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.Where("stp2.type = ?", "Delivery").
						WhereOr("stp2.type = ?", "SplitDelivery")
				}).
				Group("stp2.shipment_move_id"),
		).
		Scan(ctx, &shipments)
	if err != nil {
		log.Error().Err(err).Msg("failed to get shipments by expected delivery date")
		return nil, eris.Wrap(err, "failed to get shipments by expected delivery date")
	}

	// Get today's date as unix timestamp for the response
	now := timeutils.TimeZoneAwareNow("America/New_York")

	return &ShipmentsByExpectedDeliverDateCard{
		Count:     len(shipments),
		Date:      now,
		Shipments: shipments,
	}, nil
}
