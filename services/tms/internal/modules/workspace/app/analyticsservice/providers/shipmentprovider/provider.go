//nolint:gocritic // existing legacy workflow/API shape is intentionally kept stable
package shipmentprovider

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/app/costingservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/pkg/lanequery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/jackc/pgx/v5"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	_ kernelports.AnalyticsPageProvider = (*Provider)(nil)

	errUnexpectedRows = errors.New("shipment analytics query returned an unexpected row count")
)

const laneHeatmapInclude = "laneHeatmap"
const tomorrowsPickupsInclude = "tomorrowsPickups"
const defaultLaneHeatmapWindowDays = 7
const customerMixWindowDays = 30
const defaultTomorrowsPickupsLimit = 20

type ProviderParams struct {
	fx.In

	DB             *postgres.Connection
	Logger         *zap.Logger
	DispatchRepo   dispatchcontrolports.DispatchControlRepository
	CostingService *costingservice.Service
	Views          permissionports.ReferenceViews
	FieldAccess    permissionports.FieldAccess
}

type Provider struct {
	l              *zap.Logger
	db             *postgres.Connection
	dispatchRepo   dispatchcontrolports.DispatchControlRepository
	costingService *costingservice.Service
	views          permissionports.ReferenceViews
	fieldAccess    permissionports.FieldAccess
}

type laneStateRow struct {
	OriginState      string `db:"origin_state"`
	DestinationState string `db:"destination_state"`
	Count            int    `db:"count"`
}

type customerMixRow struct {
	CustomerID      string  `db:"customer_id"`
	Name            string  `db:"name"`
	Revenue         float64 `db:"revenue"`
	Loads           int     `db:"loads"`
	PreviousRevenue float64 `db:"previous_revenue"`
	TotalRevenue    float64 `db:"total_revenue"`
}

type tomorrowPickupRow struct {
	ShipmentID        string          `db:"shipment_id"`
	ProNumber         string          `db:"pro_number"`
	PickupWindowStart int64           `db:"pickup_window_start"`
	Customer          string          `db:"customer"`
	Origin            string          `db:"origin"`
	Destination       string          `db:"destination"`
	Driver            string          `db:"driver"`
	ShipmentStatus    shipment.Status `db:"shipment_status"`
	HasPrimaryWorker  bool            `db:"has_primary_worker"`
}

type hourlyMetricRow struct {
	Hour  int     `db:"hr"`
	Value float64 `db:"value"`
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
		l:              p.Logger.Named("analyticsprovider.shipment"),
		db:             p.DB,
		dispatchRepo:   p.DispatchRepo,
		costingService: p.CostingService,
		views:          p.Views,
		fieldAccess:    p.FieldAccess,
	}
}

func (p *Provider) GetPage() kernelports.AnalyticsPage {
	return kernelports.ShipmentAnalyticsPage
}

// GetAnalyticsData answers the page for one reader. The page is checked for
// shipment read; every card value about a customer, a location or a driver is
// then decided for this reader through the reference view rule, and the
// average margin through the margins field group.
func (p *Provider) GetAnalyticsData(
	ctx context.Context,
	opts *kernelports.AnalyticsRequestOptions,
) (kernelports.AnalyticsData, error) {
	data, err := p.analyticsData(ctx, opts)
	if err != nil {
		return nil, err
	}
	p.accessFor(ctx, opts).gate(ctx, data)

	return data, nil
}

func (p *Provider) analyticsData(
	ctx context.Context,
	opts *kernelports.AnalyticsRequestOptions,
) (kernelports.AnalyticsData, error) {
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

		return kernelports.AnalyticsData{
			"tomorrowsPickups": tomorrowsPickups,
		}, nil
	}

	laneHeatmap, err := p.getLaneHeatmap(ctx, opts.OrgID, opts.BuID, opts.WindowDays)
	if err != nil {
		log.Error("failed to get lane heatmap", zap.Error(err))
		return nil, err
	}

	if opts.Include == laneHeatmapInclude {
		return kernelports.AnalyticsData{
			"laneHeatmap": laneHeatmap,
		}, nil
	}

	return p.getFullAnalyticsData(ctx, opts, tz, laneHeatmap, log)
}

func (p *Provider) getFullAnalyticsData(
	ctx context.Context,
	opts *kernelports.AnalyticsRequestOptions,
	tz string,
	laneHeatmap *LaneHeatmapCard,
	log *zap.Logger,
) (kernelports.AnalyticsData, error) {
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

	profitability, err := p.getProfitability(ctx, opts.OrgID, opts.BuID)
	if err != nil {
		log.Error("failed to get profitability summary", zap.Error(err))
		return nil, err
	}

	data := kernelports.AnalyticsData{
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
		"profitability":      profitability,
	}

	return data, nil
}

func (p *Provider) getProfitability(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*ProfitabilityCard, error) {
	now := timeutils.NowUnix()
	summary, err := p.costingService.FleetSummary(
		ctx,
		pagination.TenantInfo{OrgID: orgID, BuID: buID},
		now-costingcontrol.DefaultFleetSummaryWindowSeconds,
		now,
	)
	if err != nil {
		return nil, err
	}

	card := &ProfitabilityCard{
		UnprofitableCount: &summary.UnprofitableCount,
		ShipmentCount:     summary.ShipmentCount,
		TotalMiles:        summary.TotalMiles,
	}
	card.AvgCPM, _ = summary.AvgCPM.Float64()
	if summary.AvgMarginPercent.Valid {
		card.HasMargin = true
		margin, _ := summary.AvgMarginPercent.Decimal.Float64()
		card.AvgMarginPct = &margin
	}

	return card, nil
}

var activeStatuses = []shipment.Status{
	shipment.StatusNew,
	shipment.StatusPartiallyAssigned,
	shipment.StatusAssigned,
	shipment.StatusInTransit,
	shipment.StatusDelayed,
	shipment.StatusPartiallyCompleted,
}

func queryOne[T any](ctx context.Context, db *keel.DB, sql string, args ...any) (T, error) {
	var zero T
	rows, err := keel.QueryAll(ctx, db, pgx.RowToStructByNameLax[T], sql, args...)
	if err != nil {
		return zero, err
	}
	if len(rows) != 1 {
		return zero, fmt.Errorf("%w: expected one row, got %d", errUnexpectedRows, len(rows))
	}

	return rows[0], nil
}

func queryAll[T any](ctx context.Context, db *keel.DB, sql string, args ...any) ([]T, error) {
	rows, err := keel.QueryAll(ctx, db, pgx.RowToStructByNameLax[T], sql, args...)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []T{}
	}

	return rows, nil
}

func statusList[S ~string](values []S) []string {
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = string(value)
	}

	return out
}

const activeShipmentsSQL = `SELECT
	COUNT(*) FILTER (WHERE sp.status = ANY($3))::int AS total_active,
	COUNT(*) FILTER (WHERE sp.created_at >= $4 AND sp.created_at <= $5)::int AS created_today,
	COUNT(*) FILTER (WHERE sp.created_at >= $6 AND sp.created_at <= $7)::int AS created_yesterday,
	COUNT(*) FILTER (WHERE sp.status = $8)::int AS in_transit,
	COUNT(*) FILTER (WHERE sp.status = $9)::int AS at_risk,
	COUNT(*) FILTER (WHERE sp.status = $10)::int AS loading,
	COUNT(*) FILTER (WHERE sp.status = ANY($11))::int AS done
FROM shipments sp
WHERE sp.organization_id = $1 AND sp.business_unit_id = $2`

const hourlyShipmentsSQL = `SELECT
	EXTRACT(HOUR FROM TO_TIMESTAMP(sp.created_at) AT TIME ZONE $3)::int AS hr,
	COUNT(*)::float8 AS value
FROM shipments sp
WHERE sp.organization_id = $1
	AND sp.business_unit_id = $2
	AND sp.created_at >= $4
	AND sp.created_at <= $5
GROUP BY hr
ORDER BY hr ASC`

func (p *Provider) getActiveShipments(
	ctx context.Context,
	orgID, buID pulid.ID,
	tz string,
) (*ActiveShipmentsCard, error) {
	return dbtx.Read(ctx, p.db, func(ctx context.Context) (*ActiveShipmentsCard, error) {
		window, err := shipmentAnalyticsWindow(tz)
		if err != nil {
			return nil, fmt.Errorf("load timezone %q: %w", tz, err)
		}

		db := p.db.Keel(ctx)
		result, err := queryOne[struct {
			TotalActive      int `db:"total_active"`
			CreatedToday     int `db:"created_today"`
			CreatedYesterday int `db:"created_yesterday"`
			InTransit        int `db:"in_transit"`
			AtRisk           int `db:"at_risk"`
			Loading          int `db:"loading"`
			Done             int `db:"done"`
		}](
			ctx,
			db,
			activeShipmentsSQL,
			orgID.String(),
			buID.String(),
			statusList(activeStatuses),
			window.TodayStart,
			window.Now,
			window.YesterdayStart,
			window.YesterdayEnd,
			string(shipment.StatusInTransit),
			string(shipment.StatusDelayed),
			string(shipment.StatusAssigned),
			statusList([]shipment.Status{
				shipment.StatusCompleted,
				shipment.StatusInvoiced,
				shipment.StatusReadyToInvoice,
			}),
		)
		if err != nil {
			return nil, err
		}

		hourlyRows, err := queryAll[hourlyMetricRow](
			ctx,
			db,
			hourlyShipmentsSQL,
			orgID.String(),
			buID.String(),
			tz,
			window.TodayStart,
			window.Now,
		)
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
	})
}

const onTimeSQL = `SELECT
	COUNT(*) FILTER (WHERE stp.actual_arrival >= $1 AND stp.actual_arrival <= $2)::int AS total,
	COUNT(*) FILTER (WHERE stp.actual_arrival >= $1 AND stp.actual_arrival <= $2
		AND stp.actual_arrival <= COALESCE(stp.scheduled_window_end, stp.scheduled_window_start))::int AS on_time,
	COUNT(*) FILTER (WHERE stp.actual_arrival >= $3 AND stp.actual_arrival <= $4)::int AS yesterday_total,
	COUNT(*) FILTER (WHERE stp.actual_arrival >= $3 AND stp.actual_arrival <= $4
		AND stp.actual_arrival <= COALESCE(stp.scheduled_window_end, stp.scheduled_window_start))::int AS yesterday_on_time,
	COUNT(*) FILTER (WHERE stp.actual_arrival >= $5 AND stp.actual_arrival < $1)::int AS seven_day_total,
	COUNT(*) FILTER (WHERE stp.actual_arrival >= $5 AND stp.actual_arrival < $1
		AND stp.actual_arrival <= COALESCE(stp.scheduled_window_end, stp.scheduled_window_start))::int AS seven_day_on_time
FROM stops stp
WHERE stp.organization_id = $6
	AND stp.business_unit_id = $7
	AND stp.status = $8
	AND stp.type = ANY($9)
	AND stp.actual_arrival IS NOT NULL
	AND stp.actual_arrival > 0
	AND stp.scheduled_window_start > 0`

func (p *Provider) getOnTimePercent(
	ctx context.Context,
	orgID, buID pulid.ID,
	tz string,
) (*OnTimeCard, error) {
	return dbtx.Write(ctx, p.db, func(ctx context.Context) (*OnTimeCard, error) {
		window, err := shipmentAnalyticsWindow(tz)
		if err != nil {
			return nil, fmt.Errorf("load timezone %q: %w", tz, err)
		}

		result, err := queryOne[struct {
			Total           int `db:"total"`
			OnTime          int `db:"on_time"`
			YesterdayTotal  int `db:"yesterday_total"`
			YesterdayOnTime int `db:"yesterday_on_time"`
			SevenDayTotal   int `db:"seven_day_total"`
			SevenDayOnTime  int `db:"seven_day_on_time"`
		}](
			ctx,
			p.db.Keel(ctx),
			onTimeSQL,
			window.TodayStart,
			window.Now,
			window.YesterdayStart,
			window.YesterdayEnd,
			window.SevenDayStart,
			orgID.String(),
			buID.String(),
			string(shipment.StopStatusCompleted),
			statusList([]shipment.StopType{
				shipment.StopTypeDelivery,
				shipment.StopTypeSplitDelivery,
			}),
		)
		if err != nil {
			return nil, err
		}

		var target *float64
		if p.dispatchRepo != nil {
			control, controlErr := p.dispatchRepo.GetByOrgID(
				ctx,
				dispatchcontrolports.GetDispatchControlRequest{
					TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
				},
			)
			if controlErr != nil {
				return nil, controlErr
			}
			target = control.ServiceFailureTarget
		}

		return &OnTimeCard{
			Percent:     percent(result.OnTime, result.Total),
			OnTimeCount: result.OnTime,
			TotalCount:  result.Total,
			Target:      target,
			DeltaPp: roundTenth(
				percent(result.OnTime, result.Total) -
					percent(result.YesterdayOnTime, result.YesterdayTotal),
			),
			SevenDayPercent: percent(result.SevenDayOnTime, result.SevenDayTotal),
		}, nil
	})
}

const revenueTodaySQL = `WITH revenue AS (
	SELECT
		COALESCE(SUM(sp.total_charge_amount) FILTER (
			WHERE sp.actual_delivery_date >= $3 AND sp.actual_delivery_date <= $4
		), 0)::float8 AS amount,
		COALESCE(SUM(sp.total_charge_amount) FILTER (
			WHERE sp.actual_delivery_date >= $5 AND sp.actual_delivery_date <= $6
		), 0)::float8 AS yesterday_amount
	FROM shipments sp
	WHERE sp.organization_id = $1
		AND sp.business_unit_id = $2
		AND sp.actual_delivery_date >= $5
		AND sp.actual_delivery_date <= $4
),
mileage AS (
	SELECT COALESCE(SUM(sm.distance), 0)::float8 AS miles
	FROM shipment_moves sm
	INNER JOIN shipments sp
		ON sp.id = sm.shipment_id
		AND sp.organization_id = sm.organization_id
		AND sp.business_unit_id = sm.business_unit_id
	WHERE sp.organization_id = $1
		AND sp.business_unit_id = $2
		AND sp.actual_delivery_date >= $3
		AND sp.actual_delivery_date <= $4
		AND sm.distance IS NOT NULL
		AND sm.distance > 0
)
SELECT revenue.amount, revenue.yesterday_amount, mileage.miles
FROM revenue CROSS JOIN mileage`

const hourlyRevenueSQL = `SELECT
	EXTRACT(HOUR FROM TO_TIMESTAMP(sp.actual_delivery_date) AT TIME ZONE $3)::int AS hr,
	COALESCE(SUM(sp.total_charge_amount), 0)::float8 AS value
FROM shipments sp
WHERE sp.organization_id = $1
	AND sp.business_unit_id = $2
	AND sp.actual_delivery_date >= $4
	AND sp.actual_delivery_date <= $5
GROUP BY hr
ORDER BY hr ASC`

func (p *Provider) getRevenueToday(
	ctx context.Context,
	orgID, buID pulid.ID,
	tz string,
) (*RevenueTodayCard, error) {
	return dbtx.Write(ctx, p.db, func(ctx context.Context) (*RevenueTodayCard, error) {
		window, err := shipmentAnalyticsWindow(tz)
		if err != nil {
			return nil, fmt.Errorf("load timezone %q: %w", tz, err)
		}

		db := p.db.Keel(ctx)
		result, err := queryOne[struct {
			Amount          float64 `db:"amount"`
			YesterdayAmount float64 `db:"yesterday_amount"`
			Miles           float64 `db:"miles"`
		}](
			ctx,
			db,
			revenueTodaySQL,
			orgID.String(),
			buID.String(),
			window.TodayStart,
			window.Now,
			window.YesterdayStart,
			window.YesterdayEnd,
		)
		if err != nil {
			return nil, err
		}

		hourlyRows, err := queryAll[hourlyMetricRow](
			ctx,
			db,
			hourlyRevenueSQL,
			orgID.String(),
			buID.String(),
			tz,
			window.TodayStart,
			window.Now,
		)
		if err != nil {
			return nil, err
		}

		return &RevenueTodayCard{
			Total:     roundCents(result.Amount),
			Sparkline: zeroFilledSparkline(hourlyRows, true),
			DeltaPct:  percentChange(result.Amount, result.YesterdayAmount),
			RPM:       rpm(result.Amount, result.Miles),
		}, nil
	})
}

const emptyMileSQL = `SELECT
	COALESCE(SUM(sm.distance) FILTER (WHERE sm.created_at >= $3 AND sm.created_at <= $4), 0)::float8 AS total_miles,
	COALESCE(SUM(sm.distance) FILTER (WHERE sm.created_at >= $3 AND sm.created_at <= $4 AND sm.loaded = false), 0)::float8 AS empty_miles,
	COALESCE(SUM(sm.distance) FILTER (WHERE sm.created_at >= $5 AND sm.created_at <= $6), 0)::float8 AS yesterday_total_miles,
	COALESCE(SUM(sm.distance) FILTER (WHERE sm.created_at >= $5 AND sm.created_at <= $6 AND sm.loaded = false), 0)::float8 AS yesterday_empty_miles
FROM shipment_moves sm
WHERE sm.organization_id = $1
	AND sm.business_unit_id = $2
	AND sm.created_at >= $5
	AND sm.created_at <= $4
	AND sm.distance IS NOT NULL
	AND sm.distance > 0`

func (p *Provider) getEmptyMilePercent(
	ctx context.Context,
	orgID, buID pulid.ID,
	tz string,
) (*EmptyMileCard, error) {
	return dbtx.Read(ctx, p.db, func(ctx context.Context) (*EmptyMileCard, error) {
		window, err := shipmentAnalyticsWindow(tz)
		if err != nil {
			return nil, fmt.Errorf("load timezone %q: %w", tz, err)
		}

		result, err := queryOne[struct {
			TotalMiles          float64 `db:"total_miles"`
			EmptyMiles          float64 `db:"empty_miles"`
			YesterdayTotalMiles float64 `db:"yesterday_total_miles"`
			YesterdayEmptyMiles float64 `db:"yesterday_empty_miles"`
		}](
			ctx,
			p.db.Keel(ctx),
			emptyMileSQL,
			orgID.String(),
			buID.String(),
			window.TodayStart,
			window.Now,
			window.YesterdayStart,
			window.YesterdayEnd,
		)
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
	})
}

const atRiskSQL = `WITH risky_shipments AS (
	SELECT DISTINCT sp.id, sp.created_at, sp.temperature_min, sp.temperature_max,
		EXISTS (
			SELECT 1
			FROM stop_etas seta
			WHERE seta.shipment_id = sp.id
				AND seta.organization_id = sp.organization_id
				AND seta.business_unit_id = sp.business_unit_id
				AND seta.verdict = ANY($3)
				AND NOT seta.arrived
		) AS eta_slipping
	FROM shipments sp
	LEFT JOIN shipment_moves sm
		ON sm.shipment_id = sp.id
		AND sm.organization_id = sp.organization_id
		AND sm.business_unit_id = sp.business_unit_id
	LEFT JOIN stops stp
		ON stp.shipment_move_id = sm.id
		AND stp.organization_id = sm.organization_id
		AND stp.business_unit_id = sm.business_unit_id
	WHERE sp.organization_id = $1
		AND sp.business_unit_id = $2
		AND sp.status = ANY($4)
		AND (
			sp.status = $5
			OR (
				stp.status != $6
				AND stp.scheduled_window_start > 0
				AND stp.scheduled_window_start < $7
			)
			OR EXISTS (
				SELECT 1
				FROM stop_etas seta
				WHERE seta.shipment_id = sp.id
					AND seta.organization_id = sp.organization_id
					AND seta.business_unit_id = sp.business_unit_id
					AND seta.verdict = ANY($3)
					AND NOT seta.arrived
			)
		)
),
active_weather AS (
	SELECT COUNT(*)::int AS weather
	FROM weather_alerts wa
	WHERE wa.organization_id = $1
		AND wa.business_unit_id = $2
		AND wa.expired_at IS NULL
		AND (wa.expires IS NULL OR wa.expires >= $7)
)
SELECT
	COUNT(rs.id)::int AS count,
	COUNT(rs.id) FILTER (WHERE rs.created_at >= $8 AND rs.created_at <= $7)::int AS created_today,
	COUNT(rs.id) FILTER (WHERE rs.created_at >= $9 AND rs.created_at <= $10)::int AS created_yesterday,
	COUNT(rs.id) FILTER (WHERE rs.eta_slipping)::int AS eta_slip,
	active_weather.weather,
	COUNT(rs.id) FILTER (WHERE rs.temperature_min IS NOT NULL OR rs.temperature_max IS NOT NULL)::int AS reefer
FROM active_weather
LEFT JOIN risky_shipments rs ON true
GROUP BY active_weather.weather`

func (p *Provider) getAtRisk(
	ctx context.Context,
	orgID, buID pulid.ID,
	tz string,
) (*AtRiskCard, error) {
	return dbtx.Write(ctx, p.db, func(ctx context.Context) (*AtRiskCard, error) {
		window, err := shipmentAnalyticsWindow(tz)
		if err != nil {
			return nil, fmt.Errorf("load timezone %q: %w", tz, err)
		}

		result, err := queryOne[struct {
			Count            int `db:"count"`
			CreatedToday     int `db:"created_today"`
			CreatedYesterday int `db:"created_yesterday"`
			ETASlip          int `db:"eta_slip"`
			Weather          int `db:"weather"`
			Reefer           int `db:"reefer"`
		}](
			ctx,
			p.db.Keel(ctx),
			atRiskSQL,
			orgID.String(),
			buID.String(),
			statusList(etaAttentionVerdicts),
			statusList(activeStatuses),
			string(shipment.StatusDelayed),
			string(shipment.StopStatusCompleted),
			window.Now,
			window.TodayStart,
			window.YesterdayStart,
			window.YesterdayEnd,
		)
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
	})
}

var etaAttentionVerdicts = []stopeta.Verdict{stopeta.VerdictAtRisk, stopeta.VerdictLate}

const unassignedSQL = `WITH unassigned_shipments AS (
	SELECT DISTINCT sp.id, sp.created_at, sp.total_charge_amount
	FROM shipments sp
	WHERE sp.organization_id = $1
		AND sp.business_unit_id = $2
		AND sp.status = ANY($3)
		AND NOT EXISTS (
			SELECT 1
			FROM shipment_moves sm
			INNER JOIN assignments a
				ON a.shipment_move_id = sm.id
				AND a.organization_id = sm.organization_id
				AND a.business_unit_id = sm.business_unit_id
				AND a.archived_at IS NULL
				AND a.status != $4
			WHERE sm.shipment_id = sp.id
				AND sm.organization_id = sp.organization_id
				AND sm.business_unit_id = sp.business_unit_id
				AND sm.status != $5
		)
)
SELECT
	COUNT(*)::int AS count,
	COUNT(*) FILTER (WHERE created_at >= $6 AND created_at <= $7)::int AS created_today,
	COUNT(*) FILTER (WHERE created_at >= $8 AND created_at <= $9)::int AS created_yesterday,
	COALESCE(SUM(total_charge_amount), 0)::float8 AS revenue_waiting
FROM unassigned_shipments`

func (p *Provider) getUnassigned(
	ctx context.Context,
	orgID, buID pulid.ID,
	tz string,
) (*UnassignedCard, error) {
	return dbtx.Write(ctx, p.db, func(ctx context.Context) (*UnassignedCard, error) {
		window, err := shipmentAnalyticsWindow(tz)
		if err != nil {
			return nil, fmt.Errorf("load timezone %q: %w", tz, err)
		}

		result, err := queryOne[struct {
			Count            int     `db:"count"`
			CreatedToday     int     `db:"created_today"`
			CreatedYesterday int     `db:"created_yesterday"`
			RevenueWaiting   float64 `db:"revenue_waiting"`
		}](
			ctx,
			p.db.Keel(ctx),
			unassignedSQL,
			orgID.String(),
			buID.String(),
			statusList(activeStatuses),
			string(shipment.AssignmentStatusCanceled),
			string(shipment.MoveStatusCanceled),
			window.TodayStart,
			window.Now,
			window.YesterdayStart,
			window.YesterdayEnd,
		)
		if err != nil {
			return nil, err
		}

		return &UnassignedCard{
			Count:          result.Count,
			Delta:          result.CreatedToday - result.CreatedYesterday,
			RevenueWaiting: roundCents(result.RevenueWaiting),
		}, nil
	})
}

const readyToDispatchSQL = `WITH active_shipments AS (
	SELECT sp.id, sp.status, sp.created_at
	FROM shipments sp
	WHERE sp.organization_id = $1
		AND sp.business_unit_id = $2
		AND sp.status = ANY($3)
),
shipment_assignments AS (
	SELECT DISTINCT sp.id, a.primary_worker_id
	FROM active_shipments sp
	INNER JOIN shipment_moves sm
		ON sm.shipment_id = sp.id
		AND sm.organization_id = $1
		AND sm.business_unit_id = $2
		AND sm.status != $4
	INNER JOIN assignments a
		ON a.shipment_move_id = sm.id
		AND a.organization_id = sm.organization_id
		AND a.business_unit_id = sm.business_unit_id
		AND a.archived_at IS NULL
		AND a.status != $5
)
SELECT
	COUNT(*) FILTER (WHERE ash.status = $6 AND sa.id IS NOT NULL)::int AS count,
	COUNT(*) FILTER (WHERE ash.status = $6 AND sa.id IS NOT NULL
		AND ash.created_at >= $7 AND ash.created_at <= $8)::int AS created_today,
	COUNT(*) FILTER (WHERE ash.status = $6 AND sa.id IS NOT NULL
		AND ash.created_at >= $9 AND ash.created_at < $10)::int AS created_yesterday,
	COUNT(*) FILTER (WHERE sa.id IS NULL)::int AS unassigned,
	COUNT(*) FILTER (WHERE sa.primary_worker_id IS NOT NULL)::int AS driver_ready
FROM active_shipments ash
LEFT JOIN shipment_assignments sa ON sa.id = ash.id`

func (p *Provider) getReadyToDispatch(
	ctx context.Context,
	orgID, buID pulid.ID,
	tz string,
) (*ReadyToDispatchCard, error) {
	return dbtx.Write(ctx, p.db, func(ctx context.Context) (*ReadyToDispatchCard, error) {
		window, err := shipmentAnalyticsWindow(tz)
		if err != nil {
			return nil, fmt.Errorf("load timezone %q: %w", tz, err)
		}

		result, err := queryOne[struct {
			Count            int `db:"count"`
			CreatedToday     int `db:"created_today"`
			CreatedYesterday int `db:"created_yesterday"`
			Unassigned       int `db:"unassigned"`
			DriverReady      int `db:"driver_ready"`
		}](
			ctx,
			p.db.Keel(ctx),
			readyToDispatchSQL,
			orgID.String(),
			buID.String(),
			statusList(activeStatuses),
			string(shipment.MoveStatusCanceled),
			string(shipment.AssignmentStatusCanceled),
			string(shipment.StatusAssigned),
			window.TodayStart,
			window.Now,
			window.YesterdayStart,
			window.YesterdayEnd,
		)
		if err != nil {
			return nil, err
		}

		return &ReadyToDispatchCard{
			Count:       result.Count,
			Delta:       result.CreatedToday - result.CreatedYesterday,
			Unassigned:  result.Unassigned,
			DriverReady: result.DriverReady,
		}, nil
	})
}

const detentionWatchlistSQL = `SELECT
	sp.pro_number AS shipment_id,
	cus.name AS customer,
	($1 - stp.actual_arrival)::bigint AS dwell_seconds
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
WHERE stp.organization_id = $2
	AND stp.business_unit_id = $3
	AND stp.status != $4
	AND stp.actual_arrival IS NOT NULL
	AND stp.actual_arrival > 0
	AND (stp.actual_departure IS NULL OR stp.actual_departure = 0)
	AND ($1 - stp.actual_arrival) > $5
ORDER BY dwell_seconds DESC, sp.pro_number ASC
LIMIT 10`

type detentionRow struct {
	ShipmentID   string  `db:"shipment_id"`
	Customer     *string `db:"customer"`
	DwellSeconds int64   `db:"dwell_seconds"`
}

func (p *Provider) getDetentionWatchlist(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*DetentionWatchlistCard, error) {
	return dbtx.Write(ctx, p.db, func(ctx context.Context) (*DetentionWatchlistCard, error) {
		rows, err := queryAll[detentionRow](
			ctx,
			p.db.Keel(ctx),
			detentionWatchlistSQL,
			timeutils.NowUnix(),
			orgID.String(),
			buID.String(),
			string(shipment.StopStatusCanceled),
			int64(2*60*60),
		)
		if err != nil {
			return nil, err
		}

		items := make([]*DetentionWatchlistItem, 0, len(rows))
		for _, row := range rows {
			items = append(items, &DetentionWatchlistItem{
				ShipmentID:   row.ShipmentID,
				Customer:     row.Customer,
				DwellSeconds: row.DwellSeconds,
				DwellLabel:   formatDwell(row.DwellSeconds),
				Tone:         detentionTone(row.DwellSeconds),
			})
		}

		return &DetentionWatchlistCard{Items: items}, nil
	})
}

const customerMixSQL = `WITH customer_revenue AS (
	SELECT
		sp.customer_id,
		COALESCE(cus.name, '') AS name,
		COALESCE(SUM(sp.total_charge_amount) FILTER (WHERE sp.created_at >= $1), 0)::float8 AS revenue,
		COUNT(*) FILTER (WHERE sp.created_at >= $1)::int AS loads,
		COALESCE(SUM(sp.total_charge_amount) FILTER (WHERE sp.created_at >= $2 AND sp.created_at < $1), 0)::float8 AS previous_revenue
	FROM shipments sp
	INNER JOIN customers cus
		ON cus.id = sp.customer_id
		AND cus.organization_id = sp.organization_id
		AND cus.business_unit_id = sp.business_unit_id
	WHERE sp.organization_id = $3
		AND sp.business_unit_id = $4
		AND sp.created_at >= $2
		AND sp.created_at <= $5
		AND sp.status != $6
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
LIMIT 5`

func (p *Provider) getCustomerMix(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*CustomerMixCard, error) {
	return dbtx.Write(ctx, p.db, func(ctx context.Context) (*CustomerMixCard, error) {
		now := timeutils.NowUnix()
		currentStart := now - int64(customerMixWindowDays)*24*60*60
		previousStart := currentStart - int64(customerMixWindowDays)*24*60*60

		rows, err := queryAll[customerMixRow](
			ctx,
			p.db.Keel(ctx),
			customerMixSQL,
			currentStart,
			previousStart,
			orgID.String(),
			buID.String(),
			now,
			string(shipment.StatusCanceled),
		)
		if err != nil {
			return nil, err
		}

		return buildCustomerMixCard(rows), nil
	})
}

const tomorrowsPickupsSQL = `SELECT
	sp.id AS shipment_id,
	sp.pro_number,
	stp.scheduled_window_start AS pickup_window_start,
	COALESCE(cus.name, '') AS customer,
	COALESCE(NULLIF(pickup_loc.code, ''), NULLIF(pickup_loc.city, ''), pickup_loc.name, '') AS origin,
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
	AND a.status != $1
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
		AND stp_dest.status != $2
		AND stp_dest.type = ANY($3)
	ORDER BY sm_dest.sequence DESC, stp_dest.sequence DESC
	LIMIT 1
) dest_loc ON true
WHERE stp.organization_id = $4
	AND stp.business_unit_id = $5
	AND stp.type = ANY($6)
	AND stp.status != $2
	AND sm.status != $7
	AND sp.status != $8
	AND stp.scheduled_window_start >= $9
	AND stp.scheduled_window_start < $10
ORDER BY stp.scheduled_window_start ASC, sp.pro_number ASC
LIMIT $11
OFFSET $12`

func (p *Provider) getTomorrowsPickups(
	ctx context.Context,
	req tomorrowsPickupsRequest,
) (*TomorrowsPickupsCard, error) {
	return dbtx.Write(ctx, p.db, func(ctx context.Context) (*TomorrowsPickupsCard, error) {
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

		rows, err := queryAll[tomorrowPickupRow](
			ctx,
			p.db.Keel(ctx),
			tomorrowsPickupsSQL,
			string(shipment.AssignmentStatusCanceled),
			string(shipment.StopStatusCanceled),
			statusList(
				[]shipment.StopType{shipment.StopTypeDelivery, shipment.StopTypeSplitDelivery},
			),
			req.orgID.String(),
			req.buID.String(),
			statusList([]shipment.StopType{shipment.StopTypePickup, shipment.StopTypeSplitPickup}),
			string(shipment.MoveStatusCanceled),
			string(shipment.StatusCanceled),
			tomorrowStart.Unix(),
			tomorrowEnd.Unix(),
			int64(limit),
			int64(req.offset),
		)
		if err != nil {
			return nil, err
		}

		return buildTomorrowsPickupsCard(tomorrowStart, rows), nil
	})
}

func (p *Provider) getLaneHeatmap(
	ctx context.Context,
	_, _ pulid.ID,
	windowDays int,
) (*LaneHeatmapCard, error) {
	return dbtx.Read(ctx, p.db, func(ctx context.Context) (*LaneHeatmapCard, error) {
		if windowDays == 0 {
			windowDays = defaultLaneHeatmapWindowDays
		}

		now := timeutils.NowUnix()
		windowStart := now - int64(windowDays)*24*60*60

		s := shipmentdb.Shipments
		rows, err := keel.Into[laneStateRow](
			ctx,
			p.db.Keel(ctx),
			lanequery.New(lanequery.Options{
				Shipments: []keel.Pred{
					s.CreatedAt.Gte(windowStart),
					s.CreatedAt.Lte(now),
					s.Status.Ne(shipment.StatusCanceled),
				},
			}),
		)
		if err != nil {
			return nil, err
		}

		return buildLaneHeatmapCard(windowDays, rows), nil
	})
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
			Customer:          new(row.Customer),
			Origin:            new(row.Origin),
			Destination:       new(row.Destination),
			Driver:            new(row.Driver),
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
