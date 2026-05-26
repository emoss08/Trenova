//nolint:gocritic // existing legacy workflow/API shape is intentionally kept stable
package shipmentprovider

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
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

const laneHeatmapInclude = "laneHeatmap"
const tomorrowsPickupsInclude = "tomorrowsPickups"
const savedViewCountsInclude = "savedViewCounts"
const defaultLaneHeatmapWindowDays = 7
const customerMixWindowDays = 30
const defaultTomorrowsPickupsLimit = 20

type ProviderParams struct {
	fx.In

	DB           *postgres.Connection
	Logger       *zap.Logger
	DispatchRepo repositories.DispatchControlRepository
}

type Provider struct {
	l            *zap.Logger
	db           *postgres.Connection
	dispatchRepo repositories.DispatchControlRepository
}

type laneStateRow struct {
	OriginState      string `bun:"origin_state"`
	DestinationState string `bun:"destination_state"`
	Count            int    `bun:"count"`
}

type customerMixRow struct {
	CustomerID      string  `bun:"customer_id"`
	Name            string  `bun:"name"`
	Revenue         float64 `bun:"revenue"`
	Loads           int     `bun:"loads"`
	PreviousRevenue float64 `bun:"previous_revenue"`
	TotalRevenue    float64 `bun:"total_revenue"`
}

type tomorrowPickupRow struct {
	ShipmentID        string          `bun:"shipment_id"`
	ProNumber         string          `bun:"pro_number"`
	PickupWindowStart int64           `bun:"pickup_window_start"`
	Customer          string          `bun:"customer"`
	Origin            string          `bun:"origin"`
	Destination       string          `bun:"destination"`
	Driver            string          `bun:"driver"`
	ShipmentStatus    shipment.Status `bun:"shipment_status"`
	HasPrimaryWorker  bool            `bun:"has_primary_worker"`
}

type hourlyMetricRow struct {
	Hour  int     `bun:"hr"`
	Value float64 `bun:"value"`
}

type analyticsWindow struct {
	TodayStart     int64
	TomorrowStart  int64
	Now            int64
	YesterdayStart int64
	YesterdayEnd   int64
	SevenDayStart  int64
}

type tomorrowsPickupsRequest struct {
	orgID  pulid.ID
	buID   pulid.ID
	tz     string
	limit  int
	offset int
}

func NewProvider(p ProviderParams) *Provider {
	return &Provider{
		l:            p.Logger.Named("analyticsprovider.shipment"),
		db:           p.DB,
		dispatchRepo: p.DispatchRepo,
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

	if opts.Include == tomorrowsPickupsInclude {
		tomorrowsPickups, err := p.getTomorrowsPickups(ctx, tomorrowsPickupsRequest{
			orgID:  opts.OrgID,
			buID:   opts.BuID,
			tz:     tz,
			limit:  opts.Limit,
			offset: opts.Offset,
		})
		if err != nil {
			log.Error("failed to get tomorrow's pickups", zap.Error(err))
			return nil, err
		}

		return services.AnalyticsData{
			"tomorrowsPickups": tomorrowsPickups,
		}, nil
	}

	if opts.Include == savedViewCountsInclude {
		counts, err := p.getSavedViewCounts(ctx, opts.OrgID, opts.BuID, tz)
		if err != nil {
			log.Error("failed to get saved view counts", zap.Error(err))
			return nil, err
		}

		return services.AnalyticsData{
			"savedViewCounts": counts,
			"page":            string(services.ShipmentAnalyticsPage),
		}, nil
	}

	laneHeatmap, err := p.getLaneHeatmap(ctx, opts.OrgID, opts.BuID, opts.WindowDays)
	if err != nil {
		log.Error("failed to get lane heatmap", zap.Error(err))
		return nil, err
	}

	if opts.Include == laneHeatmapInclude {
		return services.AnalyticsData{
			"laneHeatmap": laneHeatmap,
		}, nil
	}

	return p.getFullAnalyticsData(ctx, opts, tz, laneHeatmap, log)
}

func (p *Provider) getFullAnalyticsData(
	ctx context.Context,
	opts *services.AnalyticsRequestOptions,
	tz string,
	laneHeatmap *LaneHeatmapCard,
	log *zap.Logger,
) (services.AnalyticsData, error) {
	activeShipments, err := p.getActiveShipments(ctx, opts.OrgID, opts.BuID, tz)
	if err != nil {
		log.Error("failed to get active shipments", zap.Error(err))
		return nil, err
	}

	revenue, err := p.getRevenueToday(ctx, opts.OrgID, opts.BuID, tz)
	if err != nil {
		log.Error("failed to get revenue today", zap.Error(err))
		return nil, err
	}

	onTime, err := p.getOnTimePercent(ctx, opts.OrgID, opts.BuID, tz)
	if err != nil {
		log.Error("failed to get on-time percent", zap.Error(err))
		return nil, err
	}

	emptyMile, err := p.getEmptyMilePercent(ctx, opts.OrgID, opts.BuID, tz)
	if err != nil {
		log.Error("failed to get empty mile percent", zap.Error(err))
		return nil, err
	}

	atRisk, err := p.getAtRisk(ctx, opts.OrgID, opts.BuID, tz)
	if err != nil {
		log.Error("failed to get at-risk shipments", zap.Error(err))
		return nil, err
	}

	unassigned, err := p.getUnassigned(ctx, opts.OrgID, opts.BuID, tz)
	if err != nil {
		log.Error("failed to get unassigned shipments", zap.Error(err))
		return nil, err
	}

	readyToDispatch, err := p.getReadyToDispatch(ctx, opts.OrgID, opts.BuID, tz)
	if err != nil {
		log.Error("failed to get ready to dispatch", zap.Error(err))
		return nil, err
	}

	detentionWatchlist, err := p.getDetentionWatchlist(ctx, opts.OrgID, opts.BuID)
	if err != nil {
		log.Error("failed to get detention watchlist", zap.Error(err))
		return nil, err
	}

	customerMix, err := p.getCustomerMix(ctx, opts.OrgID, opts.BuID)
	if err != nil {
		log.Error("failed to get customer mix", zap.Error(err))
		return nil, err
	}

	tomorrowsPickups, err := p.getTomorrowsPickups(ctx, tomorrowsPickupsRequest{
		orgID: opts.OrgID,
		buID:  opts.BuID,
		tz:    tz,
		limit: defaultTomorrowsPickupsLimit,
	})
	if err != nil {
		log.Error("failed to get tomorrow's pickups", zap.Error(err))
		return nil, err
	}

	data := services.AnalyticsData{
		"activeShipments":    activeShipments,
		"onTimePercent":      onTime,
		"revenueToday":       revenue,
		"emptyMilePercent":   emptyMile,
		"atRisk":             atRisk,
		"unassigned":         unassigned,
		"readyToDispatch":    readyToDispatch,
		"detentionWatchlist": detentionWatchlist,
		"customerMix":        customerMix,
		"tomorrowsPickups":   tomorrowsPickups,
		"laneHeatmap":        laneHeatmap,
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
	window, err := shipmentAnalyticsWindow(tz)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", tz, err)
	}

	var result struct {
		TotalActive      int `bun:"total_active"`
		CreatedToday     int `bun:"created_today"`
		CreatedYesterday int `bun:"created_yesterday"`
		InTransit        int `bun:"in_transit"`
		AtRisk           int `bun:"at_risk"`
		Loading          int `bun:"loading"`
		Done             int `bun:"done"`
	}

	err = p.db.DB().NewSelect().
		TableExpr("shipments sp").
		ColumnExpr("COUNT(*) FILTER (WHERE sp.status IN (?))::int AS total_active", bun.List(activeStatuses)).
		ColumnExpr("COUNT(*) FILTER (WHERE sp.created_at >= ? AND sp.created_at <= ?)::int AS created_today", window.TodayStart, window.Now).
		ColumnExpr("COUNT(*) FILTER (WHERE sp.created_at >= ? AND sp.created_at <= ?)::int AS created_yesterday", window.YesterdayStart, window.YesterdayEnd).
		ColumnExpr("COUNT(*) FILTER (WHERE sp.status = ?)::int AS in_transit", shipment.StatusInTransit).
		ColumnExpr("COUNT(*) FILTER (WHERE sp.status = ?)::int AS at_risk", shipment.StatusDelayed).
		ColumnExpr("COUNT(*) FILTER (WHERE sp.status = ?)::int AS loading", shipment.StatusAssigned).
		ColumnExpr("COUNT(*) FILTER (WHERE sp.status IN (?))::int AS done", bun.List([]shipment.Status{
			shipment.StatusCompleted,
			shipment.StatusInvoiced,
			shipment.StatusReadyToInvoice,
		})).
		Where("sp.organization_id = ?", orgID).
		Where("sp.business_unit_id = ?", buID).
		Scan(ctx, &result)
	if err != nil {
		return nil, err
	}

	hourlyRows := make([]hourlyMetricRow, 0, 24)
	err = p.db.DB().NewSelect().
		TableExpr("shipments sp").
		ColumnExpr("EXTRACT(HOUR FROM TO_TIMESTAMP(sp.created_at) AT TIME ZONE ?)::int AS hr", tz).
		ColumnExpr("COUNT(*)::float8 AS value").
		Where("sp.organization_id = ?", orgID).
		Where("sp.business_unit_id = ?", buID).
		Where("sp.created_at >= ?", window.TodayStart).
		Where("sp.created_at <= ?", window.Now).
		GroupExpr("hr").
		OrderExpr("hr ASC").
		Scan(ctx, &hourlyRows)
	if err != nil {
		return nil, err
	}

	return &ActiveShipmentsCard{
		Count:               result.TotalActive,
		ChangeFromYesterday: result.CreatedToday - result.CreatedYesterday,
		Sparkline:           zeroFilledSparkline(hourlyRows, false),
		Breakdown: &ActiveShipmentsBreakdown{
			InTransit: result.InTransit,
			AtRisk:    result.AtRisk,
			Loading:   result.Loading,
			Done:      result.Done,
		},
	}, nil
}

//nolint:govet // existing scoped variable reuse is local and behavior-preserving
func (p *Provider) getOnTimePercent(
	ctx context.Context,
	orgID, buID pulid.ID,
	tz string,
) (*OnTimeCard, error) {
	window, err := shipmentAnalyticsWindow(tz)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", tz, err)
	}

	var result struct {
		Total           int `bun:"total"`
		OnTime          int `bun:"on_time"`
		YesterdayTotal  int `bun:"yesterday_total"`
		YesterdayOnTime int `bun:"yesterday_on_time"`
		SevenDayTotal   int `bun:"seven_day_total"`
		SevenDayOnTime  int `bun:"seven_day_on_time"`
	}

	err = p.db.DB().NewSelect().
		TableExpr("stops stp").
		ColumnExpr("COUNT(*) FILTER (WHERE stp.actual_arrival >= ? AND stp.actual_arrival <= ?)::int AS total", window.TodayStart, window.Now).
		ColumnExpr("COUNT(*) FILTER (WHERE stp.actual_arrival >= ? AND stp.actual_arrival <= ? AND stp.actual_arrival <= COALESCE(stp.scheduled_window_end, stp.scheduled_window_start))::int AS on_time", window.TodayStart, window.Now).
		ColumnExpr("COUNT(*) FILTER (WHERE stp.actual_arrival >= ? AND stp.actual_arrival <= ?)::int AS yesterday_total", window.YesterdayStart, window.YesterdayEnd).
		ColumnExpr("COUNT(*) FILTER (WHERE stp.actual_arrival >= ? AND stp.actual_arrival <= ? AND stp.actual_arrival <= COALESCE(stp.scheduled_window_end, stp.scheduled_window_start))::int AS yesterday_on_time", window.YesterdayStart, window.YesterdayEnd).
		ColumnExpr("COUNT(*) FILTER (WHERE stp.actual_arrival >= ? AND stp.actual_arrival < ?)::int AS seven_day_total", window.SevenDayStart, window.TodayStart).
		ColumnExpr("COUNT(*) FILTER (WHERE stp.actual_arrival >= ? AND stp.actual_arrival < ? AND stp.actual_arrival <= COALESCE(stp.scheduled_window_end, stp.scheduled_window_start))::int AS seven_day_on_time", window.SevenDayStart, window.TodayStart).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("stp.organization_id = ?", orgID).
				Where("stp.business_unit_id = ?", buID).
				Where("stp.status = ?", shipment.StopStatusCompleted).
				Where("stp.type IN (?)", bun.List([]shipment.StopType{
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

	var target *float64
	if p.dispatchRepo != nil {
		control, err := p.dispatchRepo.GetByOrgID(ctx, repositories.GetDispatchControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		})
		if err != nil {
			return nil, err
		}
		target = control.ServiceFailureTarget
	}

	return &OnTimeCard{
		Percent:     percent(result.OnTime, result.Total),
		OnTimeCount: result.OnTime,
		TotalCount:  result.Total,
		Target:      target,
		DeltaPp: roundTenth(
			percent(
				result.OnTime,
				result.Total,
			) - percent(
				result.YesterdayOnTime,
				result.YesterdayTotal,
			),
		),
		SevenDayPercent: percent(result.SevenDayOnTime, result.SevenDayTotal),
	}, nil
}

func (p *Provider) getRevenueToday(
	ctx context.Context,
	orgID, buID pulid.ID,
	tz string,
) (*RevenueTodayCard, error) {
	window, err := shipmentAnalyticsWindow(tz)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", tz, err)
	}

	var result struct {
		Amount          float64 `bun:"amount"`
		YesterdayAmount float64 `bun:"yesterday_amount"`
		Miles           float64 `bun:"miles"`
	}

	err = p.db.DB().NewRaw(
		`WITH revenue AS (
			SELECT
				COALESCE(SUM(sp.total_charge_amount) FILTER (
					WHERE sp.actual_delivery_date >= ? AND sp.actual_delivery_date <= ?
				), 0)::float8 AS amount,
				COALESCE(SUM(sp.total_charge_amount) FILTER (
					WHERE sp.actual_delivery_date >= ? AND sp.actual_delivery_date <= ?
				), 0)::float8 AS yesterday_amount
			FROM shipments sp
			WHERE sp.organization_id = ?
				AND sp.business_unit_id = ?
				AND sp.actual_delivery_date >= ?
				AND sp.actual_delivery_date <= ?
		),
		mileage AS (
			SELECT COALESCE(SUM(sm.distance), 0)::float8 AS miles
			FROM shipment_moves sm
			INNER JOIN shipments sp
				ON sp.id = sm.shipment_id
				AND sp.organization_id = sm.organization_id
				AND sp.business_unit_id = sm.business_unit_id
			WHERE sp.organization_id = ?
				AND sp.business_unit_id = ?
				AND sp.actual_delivery_date >= ?
				AND sp.actual_delivery_date <= ?
				AND sm.distance IS NOT NULL
				AND sm.distance > 0
		)
		SELECT revenue.amount, revenue.yesterday_amount, mileage.miles
		FROM revenue CROSS JOIN mileage`,
		window.TodayStart,
		window.Now,
		window.YesterdayStart,
		window.YesterdayEnd,
		orgID,
		buID,
		window.YesterdayStart,
		window.Now,
		orgID,
		buID,
		window.TodayStart,
		window.Now,
	).Scan(ctx, &result)
	if err != nil {
		return nil, err
	}

	hourlyRows := make([]hourlyMetricRow, 0, 24)
	err = p.db.DB().NewSelect().
		TableExpr("shipments sp").
		ColumnExpr("EXTRACT(HOUR FROM TO_TIMESTAMP(sp.actual_delivery_date) AT TIME ZONE ?)::int AS hr", tz).
		ColumnExpr("COALESCE(SUM(sp.total_charge_amount), 0)::float8 AS value").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("sp.organization_id = ?", orgID).
				Where("sp.business_unit_id = ?", buID).
				Where("sp.actual_delivery_date >= ?", window.TodayStart).
				Where("sp.actual_delivery_date <= ?", window.Now)
		}).
		GroupExpr("hr").
		OrderExpr("hr ASC").
		Scan(ctx, &hourlyRows)
	if err != nil {
		return nil, err
	}

	return &RevenueTodayCard{
		Total:     roundCents(result.Amount),
		Sparkline: zeroFilledSparkline(hourlyRows, true),
		DeltaPct:  percentChange(result.Amount, result.YesterdayAmount),
		RPM:       rpm(result.Amount, result.Miles),
	}, nil
}

func (p *Provider) getEmptyMilePercent(
	ctx context.Context,
	orgID, buID pulid.ID,
	tz string,
) (*EmptyMileCard, error) {
	window, err := shipmentAnalyticsWindow(tz)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", tz, err)
	}

	var result struct {
		TotalMiles          float64 `bun:"total_miles"`
		EmptyMiles          float64 `bun:"empty_miles"`
		YesterdayTotalMiles float64 `bun:"yesterday_total_miles"`
		YesterdayEmptyMiles float64 `bun:"yesterday_empty_miles"`
	}

	err = p.db.DB().NewSelect().
		TableExpr("shipment_moves sm").
		ColumnExpr("COALESCE(SUM(sm.distance) FILTER (WHERE sm.created_at >= ? AND sm.created_at <= ?), 0)::float8 AS total_miles", window.TodayStart, window.Now).
		ColumnExpr("COALESCE(SUM(sm.distance) FILTER (WHERE sm.created_at >= ? AND sm.created_at <= ? AND sm.loaded = false), 0)::float8 AS empty_miles", window.TodayStart, window.Now).
		ColumnExpr("COALESCE(SUM(sm.distance) FILTER (WHERE sm.created_at >= ? AND sm.created_at <= ?), 0)::float8 AS yesterday_total_miles", window.YesterdayStart, window.YesterdayEnd).
		ColumnExpr("COALESCE(SUM(sm.distance) FILTER (WHERE sm.created_at >= ? AND sm.created_at <= ? AND sm.loaded = false), 0)::float8 AS yesterday_empty_miles", window.YesterdayStart, window.YesterdayEnd).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("sm.organization_id = ?", orgID).
				Where("sm.business_unit_id = ?", buID).
				Where("sm.created_at >= ?", window.YesterdayStart).
				Where("sm.created_at <= ?", window.Now).
				Where("sm.distance IS NOT NULL").
				Where("sm.distance > 0")
		}).
		Scan(ctx, &result)
	if err != nil {
		return nil, err
	}

	pct := ratioPercent(result.EmptyMiles, result.TotalMiles)
	yesterdayPct := ratioPercent(result.YesterdayEmptyMiles, result.YesterdayTotalMiles)

	return &EmptyMileCard{
		Percent:    pct,
		EmptyMiles: roundCents(result.EmptyMiles),
		TotalMiles: roundCents(result.TotalMiles),
		DeltaPp:    roundTenth(pct - yesterdayPct),
	}, nil
}

func (p *Provider) getAtRisk(
	ctx context.Context,
	orgID, buID pulid.ID,
	tz string,
) (*AtRiskCard, error) {
	window, err := shipmentAnalyticsWindow(tz)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", tz, err)
	}

	var result struct {
		Count            int `bun:"count"`
		CreatedToday     int `bun:"created_today"`
		CreatedYesterday int `bun:"created_yesterday"`
		ETASlip          int `bun:"eta_slip"`
		Weather          int `bun:"weather"`
		Reefer           int `bun:"reefer"`
	}

	err = p.db.DB().NewRaw(
		`WITH risky_shipments AS (
			SELECT DISTINCT sp.id, sp.created_at, sp.temperature_min, sp.temperature_max
			FROM shipments sp
			LEFT JOIN shipment_moves sm
				ON sm.shipment_id = sp.id
				AND sm.organization_id = sp.organization_id
				AND sm.business_unit_id = sp.business_unit_id
			LEFT JOIN stops stp
				ON stp.shipment_move_id = sm.id
				AND stp.organization_id = sm.organization_id
				AND stp.business_unit_id = sm.business_unit_id
			WHERE sp.organization_id = ?
				AND sp.business_unit_id = ?
				AND sp.status IN (?)
				AND (
					sp.status = ?
					OR (
						stp.status != ?
						AND stp.scheduled_window_start > 0
						AND stp.scheduled_window_start < ?
					)
				)
		),
		active_weather AS (
			SELECT COUNT(*)::int AS weather
			FROM weather_alerts wa
			WHERE wa.organization_id = ?
				AND wa.business_unit_id = ?
				AND wa.expired_at IS NULL
				AND (wa.expires IS NULL OR wa.expires >= ?)
		)
		SELECT
			COUNT(*)::int AS count,
			COUNT(*) FILTER (WHERE rs.created_at >= ? AND rs.created_at <= ?)::int AS created_today,
			COUNT(*) FILTER (WHERE rs.created_at >= ? AND rs.created_at <= ?)::int AS created_yesterday,
			COUNT(*)::int AS eta_slip,
			active_weather.weather,
			COUNT(*) FILTER (WHERE rs.temperature_min IS NOT NULL OR rs.temperature_max IS NOT NULL)::int AS reefer
		FROM active_weather
		LEFT JOIN risky_shipments rs ON true
		GROUP BY active_weather.weather`,
		orgID,
		buID,
		bun.List(activeStatuses),
		shipment.StatusDelayed,
		shipment.StopStatusCompleted,
		window.Now,
		orgID,
		buID,
		window.Now,
		window.TodayStart,
		window.Now,
		window.YesterdayStart,
		window.YesterdayEnd,
	).Scan(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &AtRiskCard{
		Count:   result.Count,
		Delta:   result.CreatedToday - result.CreatedYesterday,
		ETASlip: result.ETASlip,
		Weather: result.Weather,
		Reefer:  result.Reefer,
	}, nil
}

func (p *Provider) getUnassigned(
	ctx context.Context,
	orgID, buID pulid.ID,
	tz string,
) (*UnassignedCard, error) {
	window, err := shipmentAnalyticsWindow(tz)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", tz, err)
	}

	var result struct {
		Count            int     `bun:"count"`
		CreatedToday     int     `bun:"created_today"`
		CreatedYesterday int     `bun:"created_yesterday"`
		RevenueWaiting   float64 `bun:"revenue_waiting"`
	}

	err = p.db.DB().NewRaw(
		`WITH unassigned_shipments AS (
			SELECT DISTINCT sp.id, sp.created_at, sp.total_charge_amount
			FROM shipments sp
			WHERE sp.organization_id = ?
				AND sp.business_unit_id = ?
				AND sp.status IN (?)
				AND NOT EXISTS (
					SELECT 1
					FROM shipment_moves sm
					INNER JOIN assignments a
						ON a.shipment_move_id = sm.id
						AND a.organization_id = sm.organization_id
						AND a.business_unit_id = sm.business_unit_id
						AND a.archived_at IS NULL
						AND a.status != ?
					WHERE sm.shipment_id = sp.id
						AND sm.organization_id = sp.organization_id
						AND sm.business_unit_id = sp.business_unit_id
						AND sm.status != ?
				)
		)
		SELECT
			COUNT(*)::int AS count,
			COUNT(*) FILTER (WHERE created_at >= ? AND created_at <= ?)::int AS created_today,
			COUNT(*) FILTER (WHERE created_at >= ? AND created_at <= ?)::int AS created_yesterday,
			COALESCE(SUM(total_charge_amount), 0)::float8 AS revenue_waiting
		FROM unassigned_shipments`,
		orgID,
		buID,
		bun.List(activeStatuses),
		shipment.AssignmentStatusCanceled,
		shipment.MoveStatusCanceled,
		window.TodayStart,
		window.Now,
		window.YesterdayStart,
		window.YesterdayEnd,
	).Scan(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &UnassignedCard{
		Count:          result.Count,
		Delta:          result.CreatedToday - result.CreatedYesterday,
		RevenueWaiting: roundCents(result.RevenueWaiting),
	}, nil
}

func (p *Provider) getReadyToDispatch(
	ctx context.Context,
	orgID, buID pulid.ID,
	tz string,
) (*ReadyToDispatchCard, error) {
	window, err := shipmentAnalyticsWindow(tz)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", tz, err)
	}

	var result struct {
		Count            int `bun:"count"`
		CreatedToday     int `bun:"created_today"`
		CreatedYesterday int `bun:"created_yesterday"`
		Unassigned       int `bun:"unassigned"`
		DriverReady      int `bun:"driver_ready"`
	}

	err = p.db.DB().NewRaw(
		`WITH active_shipments AS (
			SELECT sp.id, sp.status, sp.created_at
			FROM shipments sp
			WHERE sp.organization_id = ?
				AND sp.business_unit_id = ?
				AND sp.status IN (?)
		),
		shipment_assignments AS (
			SELECT DISTINCT sp.id, a.primary_worker_id
			FROM active_shipments sp
			INNER JOIN shipment_moves sm
				ON sm.shipment_id = sp.id
				AND sm.organization_id = ?
				AND sm.business_unit_id = ?
				AND sm.status != ?
			INNER JOIN assignments a
				ON a.shipment_move_id = sm.id
				AND a.organization_id = sm.organization_id
				AND a.business_unit_id = sm.business_unit_id
				AND a.archived_at IS NULL
				AND a.status != ?
		)
		SELECT
			COUNT(*) FILTER (WHERE ash.status = ? AND sa.id IS NOT NULL)::int AS count,
			COUNT(*) FILTER (WHERE ash.status = ? AND sa.id IS NOT NULL AND ash.created_at >= ? AND ash.created_at <= ?)::int AS created_today,
			COUNT(*) FILTER (WHERE ash.status = ? AND sa.id IS NOT NULL AND ash.created_at >= ? AND ash.created_at < ?)::int AS created_yesterday,
			COUNT(*) FILTER (WHERE sa.id IS NULL)::int AS unassigned,
			COUNT(*) FILTER (WHERE sa.primary_worker_id IS NOT NULL)::int AS driver_ready
		FROM active_shipments ash
		LEFT JOIN shipment_assignments sa ON sa.id = ash.id`,
		orgID,
		buID,
		bun.List(activeStatuses),
		orgID,
		buID,
		shipment.MoveStatusCanceled,
		shipment.AssignmentStatusCanceled,
		shipment.StatusAssigned,
		shipment.StatusAssigned,
		window.TodayStart,
		window.Now,
		shipment.StatusAssigned,
		window.YesterdayStart,
		window.YesterdayEnd,
	).Scan(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &ReadyToDispatchCard{
		Count:       result.Count,
		Delta:       result.CreatedToday - result.CreatedYesterday,
		Unassigned:  result.Unassigned,
		DriverReady: result.DriverReady,
	}, nil
}

func (p *Provider) getDetentionWatchlist(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*DetentionWatchlistCard, error) {
	nowUnix := timeutils.NowUnix()

	rows := make([]*DetentionWatchlistItem, 0, 10)
	err := p.db.DB().NewRaw(
		`SELECT
			sp.pro_number AS shipment_id,
			cus.name AS customer,
			(? - stp.actual_arrival)::bigint AS dwell_seconds
		FROM stops stp
		INNER JOIN shipment_moves sm
			ON sm.id = stp.shipment_move_id
			AND sm.organization_id = stp.organization_id
			AND sm.business_unit_id = stp.business_unit_id
		INNER JOIN shipments sp
			ON sp.id = sm.shipment_id
			AND sp.organization_id = sm.organization_id
			AND sp.business_unit_id = sm.business_unit_id
		INNER JOIN customers cus
			ON cus.id = sp.customer_id
			AND cus.organization_id = sp.organization_id
			AND cus.business_unit_id = sp.business_unit_id
		WHERE stp.organization_id = ?
			AND stp.business_unit_id = ?
			AND stp.status != ?
			AND stp.actual_arrival IS NOT NULL
			AND stp.actual_arrival > 0
			AND (stp.actual_departure IS NULL OR stp.actual_departure = 0)
			AND (? - stp.actual_arrival) > ?
		ORDER BY dwell_seconds DESC, sp.pro_number ASC
		LIMIT 10`,
		nowUnix,
		orgID,
		buID,
		shipment.StopStatusCanceled,
		nowUnix,
		int64(2*60*60),
	).Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		row.DwellLabel = formatDwell(row.DwellSeconds)
		row.Tone = detentionTone(row.DwellSeconds)
	}

	return &DetentionWatchlistCard{Items: rows}, nil
}

func (p *Provider) getCustomerMix(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*CustomerMixCard, error) {
	now := timeutils.NowUnix()
	currentStart := now - int64(customerMixWindowDays)*24*60*60
	previousStart := currentStart - int64(customerMixWindowDays)*24*60*60

	rows := make([]customerMixRow, 0)
	err := p.db.DB().NewRaw(
		`WITH customer_revenue AS (
			SELECT
				sp.customer_id,
				cus.name,
				COALESCE(SUM(sp.total_charge_amount) FILTER (WHERE sp.created_at >= ?), 0)::float8 AS revenue,
				COUNT(*) FILTER (WHERE sp.created_at >= ?)::int AS loads,
				COALESCE(SUM(sp.total_charge_amount) FILTER (WHERE sp.created_at >= ? AND sp.created_at < ?), 0)::float8 AS previous_revenue
			FROM shipments sp
			INNER JOIN customers cus
				ON cus.id = sp.customer_id
				AND cus.organization_id = sp.organization_id
				AND cus.business_unit_id = sp.business_unit_id
			WHERE sp.organization_id = ?
				AND sp.business_unit_id = ?
				AND sp.created_at >= ?
				AND sp.created_at <= ?
				AND sp.status != ?
			GROUP BY sp.customer_id, cus.name
		),
		ranked_customers AS (
			SELECT
				customer_id,
				name,
				revenue,
				loads,
				previous_revenue,
				SUM(revenue) OVER ()::float8 AS total_revenue
			FROM customer_revenue
			WHERE revenue > 0
		)
		SELECT customer_id, name, revenue, loads, previous_revenue, total_revenue
		FROM ranked_customers
		ORDER BY revenue DESC
		LIMIT 5`,
		currentStart,
		currentStart,
		previousStart,
		currentStart,
		orgID,
		buID,
		previousStart,
		now,
		shipment.StatusCanceled,
	).Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}

	return buildCustomerMixCard(rows), nil
}

func (p *Provider) getTomorrowsPickups( //nolint:funlen // legacy workflow
	ctx context.Context,
	req tomorrowsPickupsRequest,
) (*TomorrowsPickupsCard, error) {
	loc, err := time.LoadLocation(req.tz)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", req.tz, err)
	}

	limit := req.limit
	if limit <= 0 {
		limit = defaultTomorrowsPickupsLimit
	}

	now := time.Now().In(loc)
	tomorrow := now.AddDate(0, 0, 1)
	tomorrowStart := time.Date(
		tomorrow.Year(),
		tomorrow.Month(),
		tomorrow.Day(),
		0,
		0,
		0,
		0,
		loc,
	)
	tomorrowEnd := tomorrowStart.AddDate(0, 0, 1)

	rows := make([]tomorrowPickupRow, 0)
	err = p.db.DB().NewRaw(
		`SELECT
			sp.id AS shipment_id,
			sp.pro_number,
			stp.scheduled_window_start AS pickup_window_start,
			cus.name AS customer,
			COALESCE(NULLIF(pickup_loc.code, ''), NULLIF(pickup_loc.city, ''), pickup_loc.name) AS origin,
			COALESCE(NULLIF(dest_loc.code, ''), NULLIF(dest_loc.city, ''), dest_loc.name, '') AS destination,
			COALESCE(NULLIF(CONCAT_WS(' ', NULLIF(wrk.first_name, ''), NULLIF(wrk.last_name, '')), ''), '') AS driver,
			sp.status AS shipment_status,
			(a.primary_worker_id IS NOT NULL) AS has_primary_worker
		FROM stops stp
		INNER JOIN shipment_moves sm
			ON sm.id = stp.shipment_move_id
			AND sm.organization_id = stp.organization_id
			AND sm.business_unit_id = stp.business_unit_id
		INNER JOIN shipments sp
			ON sp.id = sm.shipment_id
			AND sp.organization_id = sm.organization_id
			AND sp.business_unit_id = sm.business_unit_id
		INNER JOIN customers cus
			ON cus.id = sp.customer_id
			AND cus.organization_id = sp.organization_id
			AND cus.business_unit_id = sp.business_unit_id
		INNER JOIN locations pickup_loc
			ON pickup_loc.id = stp.location_id
			AND pickup_loc.organization_id = stp.organization_id
			AND pickup_loc.business_unit_id = stp.business_unit_id
		LEFT JOIN assignments a
			ON a.shipment_move_id = sm.id
			AND a.organization_id = sm.organization_id
			AND a.business_unit_id = sm.business_unit_id
			AND a.archived_at IS NULL
			AND a.status != ?
		LEFT JOIN workers wrk
			ON wrk.id = a.primary_worker_id
			AND wrk.organization_id = a.organization_id
			AND wrk.business_unit_id = a.business_unit_id
		LEFT JOIN LATERAL (
			SELECT loc_dest.code, loc_dest.city, loc_dest.name
			FROM shipment_moves sm_dest
			INNER JOIN stops stp_dest
				ON stp_dest.shipment_move_id = sm_dest.id
				AND stp_dest.organization_id = sm_dest.organization_id
				AND stp_dest.business_unit_id = sm_dest.business_unit_id
			INNER JOIN locations loc_dest
				ON loc_dest.id = stp_dest.location_id
				AND loc_dest.organization_id = stp_dest.organization_id
				AND loc_dest.business_unit_id = stp_dest.business_unit_id
			WHERE sm_dest.shipment_id = sp.id
				AND sm_dest.organization_id = sp.organization_id
				AND sm_dest.business_unit_id = sp.business_unit_id
				AND stp_dest.status != ?
				AND stp_dest.type IN (?, ?)
			ORDER BY sm_dest.sequence DESC, stp_dest.sequence DESC
			LIMIT 1
		) dest_loc ON true
		WHERE stp.organization_id = ?
			AND stp.business_unit_id = ?
			AND stp.type IN (?, ?)
			AND stp.status != ?
			AND sm.status != ?
			AND sp.status != ?
			AND stp.scheduled_window_start >= ?
			AND stp.scheduled_window_start < ?
		ORDER BY stp.scheduled_window_start ASC, sp.pro_number ASC
		LIMIT ?
		OFFSET ?`,
		shipment.AssignmentStatusCanceled,
		shipment.StopStatusCanceled,
		shipment.StopTypeDelivery,
		shipment.StopTypeSplitDelivery,
		req.orgID,
		req.buID,
		shipment.StopTypePickup,
		shipment.StopTypeSplitPickup,
		shipment.StopStatusCanceled,
		shipment.MoveStatusCanceled,
		shipment.StatusCanceled,
		tomorrowStart.Unix(),
		tomorrowEnd.Unix(),
		limit,
		req.offset,
	).Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}

	return buildTomorrowsPickupsCard(tomorrowStart, rows), nil
}

func (p *Provider) getSavedViewCounts(
	ctx context.Context,
	orgID, buID pulid.ID,
	tz string,
) (*SavedViewCounts, error) {
	window, err := shipmentAnalyticsWindow(tz)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", tz, err)
	}

	counts := new(SavedViewCounts)
	err = p.db.DB().NewRaw(
		`SELECT
			COUNT(*) FILTER (
				WHERE sp.organization_id = ? AND sp.business_unit_id = ?
			)::int AS "all",
			COUNT(*) FILTER (
				WHERE sp.organization_id = ? AND sp.business_unit_id = ? AND sp.status = ?
			)::int AS transit,
			COUNT(*) FILTER (
				WHERE sp.organization_id = ? AND sp.business_unit_id = ? AND sp.status = ?
			)::int AS at_risk,
			COUNT(*) FILTER (
				WHERE sp.organization_id = ? AND sp.business_unit_id = ? AND sp.status IN (?)
			)::int AS unassigned,
			COUNT(*) FILTER (
				WHERE sp.organization_id = ?
					AND sp.business_unit_id = ?
					AND EXISTS (
						SELECT 1
						FROM shipment_moves sm
						INNER JOIN stops stp
							ON stp.shipment_move_id = sm.id
							AND stp.organization_id = sm.organization_id
							AND stp.business_unit_id = sm.business_unit_id
						WHERE sm.shipment_id = sp.id
							AND sm.organization_id = sp.organization_id
							AND sm.business_unit_id = sp.business_unit_id
							AND sm.organization_id = ?
							AND sm.business_unit_id = ?
							AND stp.organization_id = ?
							AND stp.business_unit_id = ?
							AND sm.sequence = (
								SELECT MAX(sm2.sequence)
								FROM shipment_moves sm2
								WHERE sm2.shipment_id = sp.id
									AND sm2.organization_id = sp.organization_id
									AND sm2.business_unit_id = sp.business_unit_id
							)
							AND stp.type IN (?)
							AND stp.schedule_type = ?
							AND stp.scheduled_window_start >= ?
							AND stp.scheduled_window_start < ?
					)
			)::int AS delivering_today
		FROM shipments sp
		WHERE sp.organization_id = ?
			AND sp.business_unit_id = ?`,
		orgID,
		buID,
		orgID,
		buID,
		shipment.StatusInTransit,
		orgID,
		buID,
		shipment.StatusDelayed,
		orgID,
		buID,
		bun.List([]shipment.Status{
			shipment.StatusNew,
			shipment.StatusPartiallyAssigned,
		}),
		orgID,
		buID,
		orgID,
		buID,
		orgID,
		buID,
		bun.List([]shipment.StopType{
			shipment.StopTypeDelivery,
			shipment.StopTypeSplitDelivery,
		}),
		shipment.StopScheduleTypeAppointment,
		window.TodayStart,
		window.TomorrowStart,
		orgID,
		buID,
	).Scan(ctx, counts)
	if err != nil {
		return nil, err
	}

	return counts, nil
}

func (p *Provider) getLaneHeatmap(
	ctx context.Context,
	orgID, buID pulid.ID,
	windowDays int,
) (*LaneHeatmapCard, error) {
	if windowDays == 0 {
		windowDays = defaultLaneHeatmapWindowDays
	}

	now := timeutils.NowUnix()
	windowStart := now - int64(windowDays)*24*60*60

	rows := make([]laneStateRow, 0)
	err := p.db.DB().NewRaw(
		`WITH shipment_lanes AS (
			SELECT
				sp.id,
				(
					SELECT ust_orig.abbreviation
					FROM shipment_moves sm_orig
					INNER JOIN stops stp_orig
						ON stp_orig.shipment_move_id = sm_orig.id
						AND stp_orig.organization_id = sm_orig.organization_id
						AND stp_orig.business_unit_id = sm_orig.business_unit_id
					INNER JOIN locations loc_orig
						ON loc_orig.id = stp_orig.location_id
						AND loc_orig.organization_id = stp_orig.organization_id
						AND loc_orig.business_unit_id = stp_orig.business_unit_id
					INNER JOIN us_states ust_orig
						ON ust_orig.id = loc_orig.state_id
					WHERE sm_orig.shipment_id = sp.id
						AND sm_orig.organization_id = sp.organization_id
						AND sm_orig.business_unit_id = sp.business_unit_id
						AND stp_orig.type IN (?, ?)
					ORDER BY sm_orig.sequence ASC, stp_orig.sequence ASC
					LIMIT 1
				) AS origin_state,
				(
					SELECT ust_dest.abbreviation
					FROM shipment_moves sm_dest
					INNER JOIN stops stp_dest
						ON stp_dest.shipment_move_id = sm_dest.id
						AND stp_dest.organization_id = sm_dest.organization_id
						AND stp_dest.business_unit_id = sm_dest.business_unit_id
					INNER JOIN locations loc_dest
						ON loc_dest.id = stp_dest.location_id
						AND loc_dest.organization_id = stp_dest.organization_id
						AND loc_dest.business_unit_id = stp_dest.business_unit_id
					INNER JOIN us_states ust_dest
						ON ust_dest.id = loc_dest.state_id
					WHERE sm_dest.shipment_id = sp.id
						AND sm_dest.organization_id = sp.organization_id
						AND sm_dest.business_unit_id = sp.business_unit_id
						AND stp_dest.type IN (?, ?)
					ORDER BY sm_dest.sequence DESC, stp_dest.sequence DESC
					LIMIT 1
				) AS destination_state
			FROM shipments sp
			WHERE sp.organization_id = ?
				AND sp.business_unit_id = ?
				AND sp.created_at >= ?
				AND sp.created_at <= ?
				AND sp.status != ?
		)
		SELECT origin_state, destination_state, COUNT(*)::int AS count
		FROM shipment_lanes
		WHERE origin_state IS NOT NULL
			AND destination_state IS NOT NULL
		GROUP BY origin_state, destination_state`,
		shipment.StopTypePickup,
		shipment.StopTypeSplitPickup,
		shipment.StopTypeDelivery,
		shipment.StopTypeSplitDelivery,
		orgID,
		buID,
		windowStart,
		now,
		shipment.StatusCanceled,
	).Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}

	return buildLaneHeatmapCard(windowDays, rows), nil
}

func buildCustomerMixCard(rows []customerMixRow) *CustomerMixCard {
	entries := make([]*CustomerMixEntry, 0, len(rows))
	for _, row := range rows {
		var share float64
		if row.TotalRevenue > 0 {
			share = math.Round(row.Revenue/row.TotalRevenue*1000) / 10
		}

		entries = append(entries, &CustomerMixEntry{
			CustomerID: row.CustomerID,
			Name:       row.Name,
			Revenue:    math.Round(row.Revenue*100) / 100,
			Share:      share,
			Loads:      row.Loads,
			Trend:      customerMixTrend(row.Revenue, row.PreviousRevenue),
		})
	}

	return &CustomerMixCard{
		WindowDays: customerMixWindowDays,
		Entries:    entries,
	}
}

func customerMixTrend(current, previous float64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100
		}

		return 0
	}

	return math.Round((current-previous)/previous*1000) / 10
}

func buildTomorrowsPickupsCard(
	tomorrowStart time.Time,
	rows []tomorrowPickupRow,
) *TomorrowsPickupsCard {
	pickups := make([]*TomorrowPickup, 0, len(rows))
	for _, row := range rows {
		pickups = append(pickups, &TomorrowPickup{
			ShipmentID:        row.ShipmentID,
			ProNumber:         row.ProNumber,
			PickupWindowStart: row.PickupWindowStart,
			Customer:          row.Customer,
			Origin:            row.Origin,
			Destination:       row.Destination,
			Driver:            row.Driver,
			Status:            tomorrowPickupStatus(row),
		})
	}

	return &TomorrowsPickupsCard{
		Date:    tomorrowStart.Format(time.DateOnly),
		Pickups: pickups,
	}
}

func tomorrowPickupStatus(row tomorrowPickupRow) TomorrowPickupStatus {
	if !row.HasPrimaryWorker {
		return TomorrowPickupStatusUnassigned
	}

	switch row.ShipmentStatus {
	case shipment.StatusNew, shipment.StatusPartiallyAssigned:
		return TomorrowPickupStatusTentative
	case shipment.StatusAssigned, shipment.StatusInTransit:
		return TomorrowPickupStatusConfirmed
	case shipment.StatusDelayed,
		shipment.StatusPartiallyCompleted,
		shipment.StatusReadyToInvoice,
		shipment.StatusCompleted,
		shipment.StatusInvoiced,
		shipment.StatusCanceled:
		return TomorrowPickupStatusScheduled
	}

	return TomorrowPickupStatusScheduled
}

func buildLaneHeatmapCard(windowDays int, rows []laneStateRow) *LaneHeatmapCard {
	type regionPair struct {
		origin      usstate.Region
		destination usstate.Region
	}

	counts := make(map[regionPair]int, len(rows))
	var total int
	for _, row := range rows {
		originRegion, ok := usstate.RegionForStateAbbreviation(row.OriginState)
		if !ok {
			continue
		}

		destinationRegion, ok := usstate.RegionForStateAbbreviation(row.DestinationState)
		if !ok {
			continue
		}

		pair := regionPair{origin: originRegion, destination: destinationRegion}
		counts[pair] += row.Count
		total += row.Count
	}

	regions := []usstate.Region{
		usstate.RegionWest,
		usstate.RegionMidwest,
		usstate.RegionSouth,
		usstate.RegionNortheast,
	}
	cells := make([]*LaneHeatmapCell, 0, len(regions)*len(regions))
	for _, origin := range regions {
		for _, destination := range regions {
			pair := regionPair{origin: origin, destination: destination}
			count := counts[pair]
			if count == 0 && origin != destination {
				continue
			}

			cells = append(cells, &LaneHeatmapCell{
				Origin:      string(origin),
				Destination: string(destination),
				Count:       count,
			})
		}
	}

	return &LaneHeatmapCard{
		WindowDays: windowDays,
		Cells:      cells,
		Total:      total,
	}
}

func shipmentAnalyticsWindow(tz string) (analyticsWindow, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return analyticsWindow{}, err
	}

	now := time.Now().In(loc)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	yesterdayStart := todayStart.AddDate(0, 0, -1)
	elapsed := now.Sub(todayStart)

	return analyticsWindow{
		TodayStart:     todayStart.Unix(),
		TomorrowStart:  todayStart.AddDate(0, 0, 1).Unix(),
		Now:            now.Unix(),
		YesterdayStart: yesterdayStart.Unix(),
		YesterdayEnd:   yesterdayStart.Add(elapsed).Unix(),
		SevenDayStart:  todayStart.AddDate(0, 0, -7).Unix(),
	}, nil
}

func zeroFilledSparkline(rows []hourlyMetricRow, cumulative bool) []*RevenueSparklinePoint {
	values := make([]float64, 24)
	for _, row := range rows {
		if row.Hour < 0 || row.Hour >= len(values) {
			continue
		}
		values[row.Hour] = row.Value
	}

	points := make([]*RevenueSparklinePoint, 0, len(values))
	var running float64
	for hour, value := range values {
		if cumulative {
			running += value
			value = running
		}

		points = append(points, &RevenueSparklinePoint{
			Hour:  formatClockHour(hour),
			Value: roundCents(value),
		})
	}

	return points
}

func percent(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}

	return roundTenth(float64(numerator) / float64(denominator) * 100)
}

func ratioPercent(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}

	return roundTenth(numerator / denominator * 100)
}

func percentChange(current, previous float64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100
		}

		return 0
	}

	return roundTenth((current - previous) / previous * 100)
}

func rpm(revenue, miles float64) float64 {
	if miles == 0 {
		return 0
	}

	return roundCents(revenue / miles)
}

func roundCents(value float64) float64 {
	return math.Round(value*100) / 100
}

func roundTenth(value float64) float64 {
	return math.Round(value*10) / 10
}

func formatClockHour(hour int) string {
	return fmt.Sprintf("%02d:00", hour)
}

func formatDwell(seconds int64) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	return fmt.Sprintf("%dh %02dm", hours, minutes)
}

func detentionTone(seconds int64) string {
	if seconds > 4*60*60 {
		return "danger"
	}

	return "warning"
}
