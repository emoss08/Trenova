package shipmentprovider

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
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
const defaultLaneHeatmapWindowDays = 7
const customerMixWindowDays = 30
const defaultTomorrowsPickupsLimit = 20

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

type tomorrowsPickupsRequest struct {
	orgID  pulid.ID
	buID   pulid.ID
	tz     string
	limit  int
	offset int
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
		"activeShipments":  activeShipments,
		"onTimePercent":    onTime,
		"revenueToday":     revenue,
		"emptyMilePercent": emptyMile,
		"readyToDispatch":  readyToDispatch,
		"detentionAlerts":  detentionAlerts,
		"customerMix":      customerMix,
		"tomorrowsPickups": tomorrowsPickups,
		"laneHeatmap":      laneHeatmap,
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
				Where("sp.status IN (?)", bun.List(activeStatuses))
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

	nowUnix := timeutils.NowUnix()

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

func (p *Provider) getTomorrowsPickups(
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

func detentionThresholdSeconds(control *tenant.ShipmentControl) int64 {
	if control.DetentionThreshold == nil {
		return int64(shipmentstate.DefaultDelayThresholdMinutes) * 60
	}

	return int64(shipmentstate.ResolveDelayThresholdMinutes(*control.DetentionThreshold)) * 60
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
	default:
		return TomorrowPickupStatusScheduled
	}
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
