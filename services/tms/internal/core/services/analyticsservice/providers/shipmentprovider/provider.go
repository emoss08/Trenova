package shipmentprovider

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ services.AnalyticsPageProvider = (*Provider)(nil)

type ProviderParams struct {
	fx.In

	DB          *postgres.Connection
	Logger      *zap.Logger
	ControlRepo repositories.ShipmentControlRepository
}

type Provider struct {
	l           *zap.Logger
	db          *postgres.Connection
	controlRepo repositories.ShipmentControlRepository
}

func NewProvider(p ProviderParams) *Provider {
	return &Provider{
		l:           p.Logger.Named("analyticsprovider.shipment"),
		db:          p.DB,
		controlRepo: p.ControlRepo,
	}
}

func (p *Provider) GetPage() services.AnalyticsPage {
	return services.ShipmentAnalyticsPage
}

func (p *Provider) GetAnalyticsData(
	ctx context.Context,
	opts *services.AnalyticsRequestOptions,
) (services.AnalyticsData, error) {
	log := p.l.With(zap.String("operation", "GetAnalyticsData"), zap.Any("opts", opts))

	tz := timeutils.NormalizeTimezone(opts.Timezone)

	activeShipments, err := p.getActiveShipments(ctx, opts.OrgID, opts.BuID, tz)
	if err != nil {
		log.Error("failed to get active shipments", zap.Error(err))
		return nil, err
	}

	onTime, err := p.getOnTimePercent(ctx, opts.OrgID, opts.BuID)
	if err != nil {
		log.Error("failed to get on-time percent", zap.Error(err))
		return nil, err
	}

	revenue, err := p.getRevenueToday(ctx, opts.OrgID, opts.BuID, tz)
	if err != nil {
		log.Error("failed to get revenue today", zap.Error(err))
		return nil, err
	}

	emptyMile, err := p.getEmptyMilePercent(ctx, opts.OrgID, opts.BuID)
	if err != nil {
		log.Error("failed to get empty mile percent", zap.Error(err))
		return nil, err
	}

	readyToDispatch, err := p.getReadyToDispatch(ctx, opts.OrgID, opts.BuID)
	if err != nil {
		log.Error("failed to get ready to dispatch", zap.Error(err))
		return nil, err
	}

	detentionAlerts, err := p.getDetentionAlerts(ctx, opts.OrgID, opts.BuID)
	if err != nil {
		log.Error("failed to get detention alerts", zap.Error(err))
		return nil, err
	}

	data := services.AnalyticsData{
		"activeShipments":  activeShipments,
		"onTimePercent":    onTime,
		"revenueToday":     revenue,
		"emptyMilePercent": emptyMile,
		"readyToDispatch":  readyToDispatch,
		"detentionAlerts":  detentionAlerts,
	}

	return data, nil
}

var activeStatuses = []shipment.Status{
	shipment.StatusNew,
	shipment.StatusPartiallyAssigned,
	shipment.StatusAssigned,
	shipment.StatusInTransit,
	shipment.StatusDelayed,
	shipment.StatusPartiallyCompleted,
}

func (p *Provider) getActiveShipments(
	ctx context.Context,
	orgID, buID pulid.ID,
	tz string,
) (*ActiveShipmentsCard, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", tz, err)
	}

	totalActive, err := p.db.DB().NewSelect().
		Model((*shipment.Shipment)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("sp.organization_id = ?", orgID).
				Where("sp.business_unit_id = ?", buID).
				Where("sp.status IN (?)", bun.In(activeStatuses))
		}).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now().In(loc)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).Unix()
	todayEnd := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc).Unix()

	yesterday := now.AddDate(0, 0, -1)
	yesterdayStart := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, loc).
		Unix()
	yesterdayEnd := todayStart

	createdToday, err := p.db.DB().NewSelect().
		Model((*shipment.Shipment)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("sp.organization_id = ?", orgID).
				Where("sp.business_unit_id = ?", buID).
				Where("sp.created_at >= ?", todayStart).
				Where("sp.created_at < ?", todayEnd)
		}).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	createdYesterday, err := p.db.DB().NewSelect().
		Model((*shipment.Shipment)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("sp.organization_id = ?", orgID).
				Where("sp.business_unit_id = ?", buID).
				Where("sp.created_at >= ?", yesterdayStart).
				Where("sp.created_at < ?", yesterdayEnd)
		}).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	return &ActiveShipmentsCard{
		Count:               totalActive,
		ChangeFromYesterday: createdToday - createdYesterday,
	}, nil
}

func (p *Provider) getOnTimePercent(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*OnTimeCard, error) {
	var result struct {
		Total  int `bun:"total"`
		OnTime int `bun:"on_time"`
	}

	err := p.db.DB().NewSelect().
		TableExpr("stops stp").
		ColumnExpr("COUNT(*) AS total").
		ColumnExpr("COUNT(*) FILTER (WHERE stp.actual_arrival <= COALESCE(stp.scheduled_window_end, stp.scheduled_window_start)) AS on_time").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("stp.organization_id = ?", orgID).
				Where("stp.business_unit_id = ?", buID).
				Where("stp.status = ?", shipment.StopStatusCompleted).
				Where("stp.type IN (?)", bun.In([]shipment.StopType{
					shipment.StopTypeDelivery,
					shipment.StopTypeSplitDelivery,
				})).
				Where("stp.actual_arrival IS NOT NULL").
				Where("stp.actual_arrival > 0").
				Where("stp.scheduled_window_start > 0")
		}).
		Scan(ctx, &result)
	if err != nil {
		return nil, err
	}

	var pct float64
	if result.Total > 0 {
		pct = math.Round(float64(result.OnTime)/float64(result.Total)*1000) / 10
	}

	return &OnTimeCard{
		Percent:     pct,
		OnTimeCount: result.OnTime,
		TotalCount:  result.Total,
	}, nil
}

func (p *Provider) getRevenueToday(
	ctx context.Context,
	orgID, buID pulid.ID,
	tz string,
) (*RevenueTodayCard, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", tz, err)
	}

	now := time.Now().In(loc)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).Unix()
	todayEnd := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc).Unix()

	var total struct {
		Amount float64 `bun:"amount"`
	}

	err = p.db.DB().NewSelect().
		TableExpr("shipments sp").
		ColumnExpr("COALESCE(SUM(sp.total_charge_amount), 0) AS amount").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("sp.organization_id = ?", orgID).
				Where("sp.business_unit_id = ?", buID).
				Where("sp.actual_delivery_date >= ?", todayStart).
				Where("sp.actual_delivery_date < ?", todayEnd)
		}).
		Scan(ctx, &total)
	if err != nil {
		return nil, err
	}

	type hourlyRow struct {
		Hour   int     `bun:"hr"`
		Amount float64 `bun:"amount"`
	}

	hourlyRows := make([]hourlyRow, 0)
	err = p.db.DB().NewSelect().
		TableExpr("shipments sp").
		ColumnExpr("EXTRACT(HOUR FROM TO_TIMESTAMP(sp.actual_delivery_date) AT TIME ZONE ?)::int AS hr", tz).
		ColumnExpr("COALESCE(SUM(sp.total_charge_amount), 0) AS amount").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("sp.organization_id = ?", orgID).
				Where("sp.business_unit_id = ?", buID).
				Where("sp.actual_delivery_date >= ?", todayStart).
				Where("sp.actual_delivery_date < ?", todayEnd)
		}).
		GroupExpr("hr").
		OrderExpr("hr ASC").
		Scan(ctx, &hourlyRows)
	if err != nil {
		return nil, err
	}

	sparkline := make([]*RevenueSparklinePoint, 0, len(hourlyRows))
	var cumulative float64
	for _, row := range hourlyRows {
		cumulative += row.Amount
		sparkline = append(sparkline, &RevenueSparklinePoint{
			Hour:  formatHour(row.Hour),
			Value: math.Round(cumulative*100) / 100,
		})
	}

	return &RevenueTodayCard{
		Total:     math.Round(total.Amount*100) / 100,
		Sparkline: sparkline,
	}, nil
}

func (p *Provider) getEmptyMilePercent(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*EmptyMileCard, error) {
	var result struct {
		TotalMiles float64 `bun:"total_miles"`
		EmptyMiles float64 `bun:"empty_miles"`
	}

	err := p.db.DB().NewSelect().
		TableExpr("shipment_moves sm").
		ColumnExpr("COALESCE(SUM(sm.distance), 0) AS total_miles").
		ColumnExpr("COALESCE(SUM(sm.distance) FILTER (WHERE sm.loaded = false), 0) AS empty_miles").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("sm.organization_id = ?", orgID).
				Where("sm.business_unit_id = ?", buID).
				Where("sm.distance IS NOT NULL").
				Where("sm.distance > 0")
		}).
		Scan(ctx, &result)
	if err != nil {
		return nil, err
	}

	var pct float64
	if result.TotalMiles > 0 {
		pct = math.Round(result.EmptyMiles/result.TotalMiles*1000) / 10
	}

	return &EmptyMileCard{
		Percent:    pct,
		EmptyMiles: math.Round(result.EmptyMiles*100) / 100,
		TotalMiles: math.Round(result.TotalMiles*100) / 100,
	}, nil
}

func (p *Provider) getReadyToDispatch(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*ReadyToDispatchCard, error) {
	count, err := p.db.DB().NewSelect().
		Model((*shipment.Shipment)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("sp.organization_id = ?", orgID).
				Where("sp.business_unit_id = ?", buID).
				Where("sp.status = ?", shipment.StatusAssigned)
		}).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	return &ReadyToDispatchCard{Count: count}, nil
}

func (p *Provider) getDetentionAlerts(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*DetentionAlertsCard, error) {
	control, err := p.controlRepo.Get(ctx, repositories.GetShipmentControlRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
	})
	if err != nil {
		return nil, err
	}

	if !control.TrackDetentionTime {
		return &DetentionAlertsCard{Count: 0}, nil
	}

	detentionThreshold := detentionThresholdSeconds(control)

	nowUnix := time.Now().Unix()

	count, err := p.db.DB().NewSelect().
		TableExpr("stops stp").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("stp.organization_id = ?", orgID).
				Where("stp.business_unit_id = ?", buID).
				Where("stp.status != ?", shipment.StopStatusCanceled).
				Where("stp.actual_arrival IS NOT NULL").
				Where("stp.actual_arrival > 0")
		}).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				WhereGroup(" OR ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.
						Where("stp.actual_departure IS NOT NULL").
						Where("stp.actual_departure > 0").
						Where("(stp.actual_departure - stp.actual_arrival) > ?", detentionThreshold)
				}).
				WhereGroup(" OR ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.
						Where("stp.actual_departure IS NULL").
						Where("(? - stp.actual_arrival) > ?", nowUnix, detentionThreshold)
				})
		}).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	return &DetentionAlertsCard{Count: count}, nil
}

func detentionThresholdSeconds(control *tenant.ShipmentControl) int64 {
	if control.DetentionThreshold == nil {
		return int64(shipmentstate.DefaultDelayThresholdMinutes) * 60
	}

	return int64(shipmentstate.ResolveDelayThresholdMinutes(*control.DetentionThreshold)) * 60
}

func formatHour(h int) string {
	switch {
	case h == 0:
		return "12am"
	case h < 12:
		return fmt.Sprintf("%dam", h)
	case h == 12:
		return "12pm"
	default:
		return fmt.Sprintf("%dpm", h-12)
	}
}
