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
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta/stopetadb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/weatheralert/weatheralertdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/pkg/lanequery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

	DB              *postgres.Connection
	Logger          *zap.Logger
	DispatchRepo    dispatchcontrolports.DispatchControlRepository
	DispatchConsole dispatchboardports.DispatchConsoleRepository
	CostingService  *costingservice.Service
	Views           permissionports.ReferenceViews
	FieldAccess     permissionports.FieldAccess
}

type Provider struct {
	l               *zap.Logger
	db              *postgres.Connection
	dispatchRepo    dispatchcontrolports.DispatchControlRepository
	dispatchConsole dispatchboardports.DispatchConsoleRepository
	costingService  *costingservice.Service
	views           permissionports.ReferenceViews
	fieldAccess     permissionports.FieldAccess
}

type laneStateRow struct {
	OriginState      string
	DestinationState string
	Count            int
}

type customerMixRow struct {
	CustomerID      string
	Name            string
	Revenue         float64
	Loads           int
	PreviousRevenue float64
	TotalRevenue    float64
}

type tomorrowPickupRow struct {
	ShipmentID        string
	ProNumber         string
	PickupWindowStart int64
	Customer          string
	Origin            string
	Destination       string
	Driver            string
	ShipmentStatus    shipmentvocab.Status
	HasPrimaryWorker  bool
}

type hourlyMetricRow struct {
	Hour  int
	Value float64
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
	tz     string
	limit  int
	offset int
}

func NewProvider(p ProviderParams) *Provider {
	return &Provider{
		l:               p.Logger.Named("analyticsprovider.shipment"),
		db:              p.DB,
		dispatchRepo:    p.DispatchRepo,
		dispatchConsole: p.DispatchConsole,
		costingService:  p.CostingService,
		views:           p.Views,
		fieldAccess:     p.FieldAccess,
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

	if opts.Include == kernelports.AnalyticsIncludeMetrics {
		return p.getMetricCards(ctx, opts, tz, log)
	}

	laneHeatmap, err := p.getLaneHeatmap(ctx, opts.WindowDays)
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
	data, err := p.getMetricCards(ctx, opts, tz, log)
	if err != nil {
		return nil, err
	}

	detentionWatchlist, err := p.getDetentionWatchlist(ctx)
	if err != nil {
		log.Error("failed to get detention watchlist", zap.Error(err))
		return nil, err
	}

	customerMix, err := p.getCustomerMix(ctx)
	if err != nil {
		log.Error("failed to get customer mix", zap.Error(err))
		return nil, err
	}

	tomorrowsPickups, err := p.getTomorrowsPickups(ctx, tomorrowsPickupsRequest{
		tz:    tz,
		limit: defaultTomorrowsPickupsLimit,
	})
	if err != nil {
		log.Error("failed to get tomorrow's pickups", zap.Error(err))
		return nil, err
	}

	data["detentionWatchlist"] = detentionWatchlist
	data["customerMix"] = customerMix
	data["tomorrowsPickups"] = tomorrowsPickups
	data["laneHeatmap"] = laneHeatmap

	return data, nil
}

func (p *Provider) getMetricCards(
	ctx context.Context,
	opts *kernelports.AnalyticsRequestOptions,
	tz string,
	log *zap.Logger,
) (kernelports.AnalyticsData, error) {
	activeShipments, err := p.getActiveShipments(ctx, tz)
	if err != nil {
		log.Error("failed to get active shipments", zap.Error(err))
		return nil, err
	}

	revenue, err := p.getRevenueToday(ctx, tz)
	if err != nil {
		log.Error("failed to get revenue today", zap.Error(err))
		return nil, err
	}

	onTime, err := p.getOnTimePercent(ctx, opts.OrgID, opts.BuID, tz)
	if err != nil {
		log.Error("failed to get on-time percent", zap.Error(err))
		return nil, err
	}

	emptyMile, err := p.getEmptyMilePercent(ctx, tz)
	if err != nil {
		log.Error("failed to get empty mile percent", zap.Error(err))
		return nil, err
	}

	atRisk, err := p.getAtRisk(ctx, tz)
	if err != nil {
		log.Error("failed to get at-risk shipments", zap.Error(err))
		return nil, err
	}

	unassigned, err := p.getUnassigned(ctx, tz)
	if err != nil {
		log.Error("failed to get unassigned shipments", zap.Error(err))
		return nil, err
	}

	readyToDispatch, err := p.getReadyToDispatch(ctx, tz)
	if err != nil {
		log.Error("failed to get ready to dispatch", zap.Error(err))
		return nil, err
	}

	profitability, err := p.getProfitability(ctx, opts.OrgID, opts.BuID)
	if err != nil {
		log.Error("failed to get profitability summary", zap.Error(err))
		return nil, err
	}

	dispatch, err := p.getDispatch(ctx, opts, tz)
	if err != nil {
		log.Error("failed to get dispatch counts", zap.Error(err))
		return nil, err
	}

	return kernelports.AnalyticsData{
		"activeShipments":  activeShipments,
		"onTimePercent":    onTime,
		"revenueToday":     revenue,
		"emptyMilePercent": emptyMile,
		"atRisk":           atRisk,
		"unassigned":       unassigned,
		"readyToDispatch":  readyToDispatch,
		"profitability":    profitability,
		"dispatch":         dispatch,
	}, nil
}

// dispatchWindowSeconds is the day ahead the dispatch counts cover, the
// window the dispatch board opens on.
const dispatchWindowSeconds = 24 * 60 * 60

// getDispatch counts the next day's moves from the dispatch board's own
// summary, in one query, so a KPI tile or a board figure and the dispatch
// board never disagree.
func (p *Provider) getDispatch(
	ctx context.Context,
	opts *kernelports.AnalyticsRequestOptions,
	tz string,
) (*DispatchCard, error) {
	now := timeutils.NowUnix()
	dayStart, err := timeutils.DayStartUnix(now, tz)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", tz, err)
	}

	summary, err := p.dispatchConsole.GetBoardSummary(ctx, &dispatchboardports.DispatchBoardFilter{
		TenantInfo:     pagination.TenantInfo{OrgID: opts.OrgID, BuID: opts.BuID},
		WindowStart:    now,
		WindowEnd:      now + dispatchWindowSeconds,
		DayStartUnix:   dayStart,
		IncludeCovered: true,
	})
	if err != nil {
		return nil, err
	}

	return &DispatchCard{
		Moves:     summary.UncoveredMoves + summary.CoveredMoves,
		Late:      summary.OverdueMoves,
		DueNow:    summary.DueNowMoves,
		DueToday:  summary.DueTodayMoves,
		Uncovered: summary.UncoveredMoves,
	}, nil
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

var activeStatuses = []shipmentvocab.Status{
	shipmentvocab.StatusNew,
	shipmentvocab.StatusPartiallyAssigned,
	shipmentvocab.StatusAssigned,
	shipmentvocab.StatusInTransit,
	shipmentvocab.StatusDelayed,
	shipmentvocab.StatusPartiallyCompleted,
}

var etaAttentionVerdicts = []stopeta.Verdict{stopeta.VerdictAtRisk, stopeta.VerdictLate}

var doneStatuses = []shipmentvocab.Status{
	shipmentvocab.StatusCompleted,
	shipmentvocab.StatusInvoiced,
	shipmentvocab.StatusReadyToInvoice,
}

const (
	detentionDwellFloorSeconds = int64(2 * 60 * 60)
	detentionWatchlistLimit    = 10
	customerMixLimit           = 5
	mileageShipmentAlias       = "mi_sp"
	readyAssignmentsCTE        = "shipment_assignments"
	readyAssignmentsShipment   = "id"
	readyAssignmentsWorker     = "primary_worker_id"
	readyShipmentAlias         = "ra_sp"
	pickupLocationAlias        = "pickup_loc"
	pickupWorkerAlias          = "wrk"
	destinationStopAlias       = "stp_dest"
	destinationMoveAlias       = "sm_dest"
	destinationLocationAlias   = "dest_loc"
	hourField                  = "hour"
)

type activeShipmentsRow struct {
	TotalActive      int
	CreatedToday     int
	CreatedYesterday int
	InTransit        int
	AtRisk           int
	Loading          int
	Done             int
}

type onTimeRow struct {
	Total           int
	OnTime          int
	YesterdayTotal  int
	YesterdayOnTime int
	SevenDayTotal   int
	SevenDayOnTime  int
}

type revenueTodayRow struct {
	Amount          float64
	YesterdayAmount float64
	Miles           float64
}

type emptyMileRow struct {
	TotalMiles          float64
	EmptyMiles          float64
	YesterdayTotalMiles float64
	YesterdayEmptyMiles float64
}

type atRiskRow struct {
	Count            int
	CreatedToday     int
	CreatedYesterday int
	ETASlip          int
	Weather          int
	Reefer           int
}

type unassignedRow struct {
	Count            int
	CreatedToday     int
	CreatedYesterday int
	RevenueWaiting   float64
}

type readyToDispatchRow struct {
	Count            int
	CreatedToday     int
	CreatedYesterday int
	Unassigned       int
	DriverReady      int
}

func queryOne[T any](ctx context.Context, db *keel.DB, p keel.Projection) (T, error) {
	var zero T
	rows, err := keel.Into[T](ctx, db, p)
	if err != nil {
		return zero, err
	}
	if len(rows) != 1 {
		return zero, fmt.Errorf("%w: expected one row, got %d", errUnexpectedRows, len(rows))
	}

	return rows[0], nil
}

func queryAll[T any](ctx context.Context, db *keel.DB, p keel.Projection) ([]T, error) {
	rows, err := keel.Into[T](ctx, db, p)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []T{}
	}

	return rows, nil
}

func within(c keel.Int64Col, from, to int64) keel.Pred {
	return keel.And(c.Gte(from), c.Lte(to))
}

func countWhere(ps ...keel.Pred) keel.Agg {
	return keel.Count().Filter(keel.And(ps...))
}

func floatTotal(sum keel.Agg) keel.Float64Expr {
	return keel.CastFloat64(keel.NumericOf(sum).Coalesce(0))
}

func localHour(ts keel.NumberValue, tz string) keel.Int64Expr {
	return keel.CastInt64(
		keel.DatePart(hourField, keel.ToTimestamp(ts).AtTimeZone(keel.Text(tz))),
	)
}

func activeShipmentsQuery(window analyticsWindow) *keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments

	return sp.Pick(
		countWhere(sp.Status.In(activeStatuses...)),
		countWhere(within(sp.CreatedAt, window.TodayStart, window.Now)),
		countWhere(within(sp.CreatedAt, window.YesterdayStart, window.YesterdayEnd)),
		countWhere(sp.Status.Eq(shipmentvocab.StatusInTransit)),
		countWhere(sp.Status.Eq(shipmentvocab.StatusDelayed)),
		countWhere(sp.Status.Eq(shipmentvocab.StatusAssigned)),
		countWhere(sp.Status.In(doneStatuses...)),
	)
}

func hourlyShipmentsQuery(window analyticsWindow, tz string) *keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments
	hour := localHour(sp.CreatedAt, tz)

	return sp.Pick(hour, keel.CastFloat64(keel.Int64Of(keel.Count()))).
		Where(within(sp.CreatedAt, window.TodayStart, window.Now)).
		Group(hour).
		Order(hour.Asc())
}

func (p *Provider) getActiveShipments(
	ctx context.Context,
	tz string,
) (*ActiveShipmentsCard, error) {
	return dbtx.Read(ctx, p.db, func(ctx context.Context) (*ActiveShipmentsCard, error) {
		window, err := shipmentAnalyticsWindow(tz)
		if err != nil {
			return nil, fmt.Errorf("load timezone %q: %w", tz, err)
		}

		db := p.db.Keel(ctx)
		result, err := queryOne[activeShipmentsRow](ctx, db, activeShipmentsQuery(window))
		if err != nil {
			return nil, err
		}

		hourlyRows, err := queryAll[hourlyMetricRow](ctx, db, hourlyShipmentsQuery(window, tz))
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

func onTimeQuery(window analyticsWindow) *keel.Query[*shipment.Stop] {
	stp := shipmentdb.Stops
	onTime := keel.Int64Of(stp.ActualArrival).
		LteExpr(stp.ScheduledWindowEnd.Coalesce(stp.ScheduledWindowStart))
	today := within(stp.ActualArrival, window.TodayStart, window.Now)
	yesterday := within(stp.ActualArrival, window.YesterdayStart, window.YesterdayEnd)
	sevenDay := keel.And(
		stp.ActualArrival.Gte(window.SevenDayStart),
		stp.ActualArrival.Lt(window.TodayStart),
	)

	return stp.Pick(
		countWhere(today),
		countWhere(today, onTime),
		countWhere(yesterday),
		countWhere(yesterday, onTime),
		countWhere(sevenDay),
		countWhere(sevenDay, onTime),
	).Where(
		stp.Status.Eq(shipmentvocab.StopStatusCompleted),
		stp.Type.In(shipmentvocab.DeliveryStopTypes()...),
		stp.ActualArrival.IsNotNull(),
		stp.ActualArrival.Gt(0),
		stp.ScheduledWindowStart.Gt(0),
	)
}

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

		result, err := queryOne[onTimeRow](ctx, p.db.Keel(ctx), onTimeQuery(window))
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

func revenueTodayQuery(window analyticsWindow) *keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments
	sm := shipmentdb.ShipmentMoves
	delivered := shipmentdb.Shipments.As(mileageShipmentAlias)

	miles := keel.Scalar(sm.
		Join(delivered, delivered.ID.EqCol(sm.ShipmentID)).
		Pick(floatTotal(sm.Distance.Sum())).
		Where(
			within(delivered.ActualDeliveryDate, window.TodayStart, window.Now),
			sm.Distance.IsNotNull(),
			sm.Distance.Gt(0),
		))

	return sp.Pick(
		floatTotal(sp.TotalChargeAmount.Sum().
			Filter(within(sp.ActualDeliveryDate, window.TodayStart, window.Now))),
		floatTotal(sp.TotalChargeAmount.Sum().
			Filter(within(sp.ActualDeliveryDate, window.YesterdayStart, window.YesterdayEnd))),
		miles,
	).Where(within(sp.ActualDeliveryDate, window.YesterdayStart, window.Now))
}

func hourlyRevenueQuery(window analyticsWindow, tz string) *keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments
	hour := localHour(sp.ActualDeliveryDate, tz)

	return sp.Pick(hour, floatTotal(sp.TotalChargeAmount.Sum())).
		Where(within(sp.ActualDeliveryDate, window.TodayStart, window.Now)).
		Group(hour).
		Order(hour.Asc())
}

func (p *Provider) getRevenueToday(
	ctx context.Context,
	tz string,
) (*RevenueTodayCard, error) {
	return dbtx.Write(ctx, p.db, func(ctx context.Context) (*RevenueTodayCard, error) {
		window, err := shipmentAnalyticsWindow(tz)
		if err != nil {
			return nil, fmt.Errorf("load timezone %q: %w", tz, err)
		}

		db := p.db.Keel(ctx)
		result, err := queryOne[revenueTodayRow](ctx, db, revenueTodayQuery(window))
		if err != nil {
			return nil, err
		}

		hourlyRows, err := queryAll[hourlyMetricRow](ctx, db, hourlyRevenueQuery(window, tz))
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

func emptyMileQuery(window analyticsWindow) *keel.Query[*shipment.ShipmentMove] {
	sm := shipmentdb.ShipmentMoves
	today := within(sm.CreatedAt, window.TodayStart, window.Now)
	yesterday := within(sm.CreatedAt, window.YesterdayStart, window.YesterdayEnd)
	empty := sm.Loaded.Eq(false)

	return sm.Pick(
		floatTotal(sm.Distance.Sum().Filter(today)),
		floatTotal(sm.Distance.Sum().Filter(keel.And(today, empty))),
		floatTotal(sm.Distance.Sum().Filter(yesterday)),
		floatTotal(sm.Distance.Sum().Filter(keel.And(yesterday, empty))),
	).Where(
		within(sm.CreatedAt, window.YesterdayStart, window.Now),
		sm.Distance.IsNotNull(),
		sm.Distance.Gt(0),
	)
}

func (p *Provider) getEmptyMilePercent(
	ctx context.Context,
	tz string,
) (*EmptyMileCard, error) {
	return dbtx.Read(ctx, p.db, func(ctx context.Context) (*EmptyMileCard, error) {
		window, err := shipmentAnalyticsWindow(tz)
		if err != nil {
			return nil, fmt.Errorf("load timezone %q: %w", tz, err)
		}

		result, err := queryOne[emptyMileRow](ctx, p.db.Keel(ctx), emptyMileQuery(window))
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

func atRiskQuery(window analyticsWindow) *keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments
	moves := sp.Moves()
	stops := moves.Stops()
	eta := stopetadb.StopETAs
	wa := weatheralertdb.WeatherAlerts

	etaSlipping := keel.Exists(eta.Where(
		eta.ShipmentID.EqCol(sp.ID),
		eta.Verdict.In(etaAttentionVerdicts...),
		eta.Arrived.IsFalse(),
	))
	weather := keel.Scalar(wa.Pick(keel.Count()).Where(
		wa.ExpiredAt.IsNull(),
		keel.Or(wa.Expires.IsNull(), wa.Expires.Gte(window.Now)),
	))

	return sp.Pick(
		keel.Count(),
		countWhere(within(sp.CreatedAt, window.TodayStart, window.Now)),
		countWhere(within(sp.CreatedAt, window.YesterdayStart, window.YesterdayEnd)),
		countWhere(etaSlipping),
		weather,
		countWhere(keel.Or(sp.TemperatureMin.IsNotNull(), sp.TemperatureMax.IsNotNull())),
	).Where(
		sp.Status.In(activeStatuses...),
		keel.Or(
			sp.Status.Eq(shipmentvocab.StatusDelayed),
			moves.Any(stops.Any(
				stops.Status.Ne(shipmentvocab.StopStatusCompleted),
				stops.ScheduledWindowStart.Gt(0),
				stops.ScheduledWindowStart.Lt(window.Now),
			)),
			etaSlipping,
		),
	)
}

func (p *Provider) getAtRisk(
	ctx context.Context,
	tz string,
) (*AtRiskCard, error) {
	return dbtx.Write(ctx, p.db, func(ctx context.Context) (*AtRiskCard, error) {
		window, err := shipmentAnalyticsWindow(tz)
		if err != nil {
			return nil, fmt.Errorf("load timezone %q: %w", tz, err)
		}

		result, err := queryOne[atRiskRow](ctx, p.db.Keel(ctx), atRiskQuery(window))
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

func unassignedQuery(window analyticsWindow) *keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments
	sm := shipmentdb.ShipmentMoves
	a := shipmentdb.Assignments

	return sp.Pick(
		keel.Count(),
		countWhere(within(sp.CreatedAt, window.TodayStart, window.Now)),
		countWhere(within(sp.CreatedAt, window.YesterdayStart, window.YesterdayEnd)),
		floatTotal(sp.TotalChargeAmount.Sum()),
	).Where(
		sp.Status.In(activeStatuses...),
		keel.NotExists(sm.
			Join(
				a,
				a.ShipmentMoveID.EqCol(sm.ID),
				a.ArchivedAt.IsNull(),
				a.Status.Ne(shipmentvocab.AssignmentStatusCanceled),
			).
			Where(sm.ShipmentID.EqCol(sp.ID), sm.Status.Ne(shipmentvocab.MoveStatusCanceled))),
	)
}

func (p *Provider) getUnassigned(
	ctx context.Context,
	tz string,
) (*UnassignedCard, error) {
	return dbtx.Write(ctx, p.db, func(ctx context.Context) (*UnassignedCard, error) {
		window, err := shipmentAnalyticsWindow(tz)
		if err != nil {
			return nil, fmt.Errorf("load timezone %q: %w", tz, err)
		}

		result, err := queryOne[unassignedRow](ctx, p.db.Keel(ctx), unassignedQuery(window))
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

func readyToDispatchQuery(window analyticsWindow) *keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments
	assigned := shipmentdb.Shipments.As(readyShipmentAlias)
	sm := shipmentdb.ShipmentMoves
	a := shipmentdb.Assignments

	sa := keel.CTE(
		readyAssignmentsCTE,
		[]string{readyAssignmentsShipment, readyAssignmentsWorker},
		assigned.
			Join(
				sm,
				sm.ShipmentID.EqCol(assigned.ID),
				sm.Status.Ne(shipmentvocab.MoveStatusCanceled),
			).
			Join(
				a,
				a.ShipmentMoveID.EqCol(sm.ID),
				a.ArchivedAt.IsNull(),
				a.Status.Ne(shipmentvocab.AssignmentStatusCanceled),
			).
			DistinctOn(assigned.ID, a.PrimaryWorkerID).
			Pick(assigned.ID, a.PrimaryWorkerID).
			Where(assigned.Status.In(activeStatuses...)).
			Order(assigned.ID.Asc(), a.PrimaryWorkerID.Asc()),
	)
	ready := keel.And(
		sp.Status.Eq(shipmentvocab.StatusAssigned),
		sa.Col(readyAssignmentsShipment).IsNotNull(),
	)

	return sp.With(sa).
		LeftJoin(sa, sa.Col(readyAssignmentsShipment).EqCol(sp.ID)).
		Pick(
			countWhere(ready),
			countWhere(ready, within(sp.CreatedAt, window.TodayStart, window.Now)),
			countWhere(
				ready,
				sp.CreatedAt.Gte(window.YesterdayStart),
				sp.CreatedAt.Lt(window.YesterdayEnd),
			),
			countWhere(sa.Col(readyAssignmentsShipment).IsNull()),
			countWhere(sa.Col(readyAssignmentsWorker).IsNotNull()),
		).
		Where(sp.Status.In(activeStatuses...))
}

func (p *Provider) getReadyToDispatch(
	ctx context.Context,
	tz string,
) (*ReadyToDispatchCard, error) {
	return dbtx.Write(ctx, p.db, func(ctx context.Context) (*ReadyToDispatchCard, error) {
		window, err := shipmentAnalyticsWindow(tz)
		if err != nil {
			return nil, fmt.Errorf("load timezone %q: %w", tz, err)
		}

		result, err := queryOne[readyToDispatchRow](
			ctx,
			p.db.Keel(ctx),
			readyToDispatchQuery(window),
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

type detentionRow struct {
	ShipmentID   string
	Customer     *string
	DwellSeconds int64
}

func detentionWatchlistQuery(now int64) *keel.Query[*shipment.Stop] {
	stp := shipmentdb.Stops
	sm := shipmentdb.ShipmentMoves
	sp := shipmentdb.Shipments
	cus := customerdb.Customers
	dwell := keel.Int64(now).Sub(stp.ActualArrival)

	return stp.
		Join(sm, sm.ID.EqCol(stp.ShipmentMoveID)).
		Join(sp, sp.ID.EqCol(sm.ShipmentID)).
		Join(cus, cus.ID.EqCol(sp.CustomerID)).
		Pick(sp.ProNumber, cus.Name, dwell).
		Where(
			stp.Status.Ne(shipmentvocab.StopStatusCanceled),
			stp.ActualArrival.IsNotNull(),
			stp.ActualArrival.Gt(0),
			keel.Or(stp.ActualDeparture.IsNull(), stp.ActualDeparture.Eq(0)),
			dwell.Gt(detentionDwellFloorSeconds),
		).
		Order(dwell.Desc(), sp.ProNumber.Asc()).
		Limit(detentionWatchlistLimit)
}

func (p *Provider) getDetentionWatchlist(ctx context.Context) (*DetentionWatchlistCard, error) {
	return dbtx.Write(ctx, p.db, func(ctx context.Context) (*DetentionWatchlistCard, error) {
		rows, err := queryAll[detentionRow](
			ctx,
			p.db.Keel(ctx),
			detentionWatchlistQuery(timeutils.NowUnix()),
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

func customerMixQuery(now, currentStart, previousStart int64) *keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments
	cus := customerdb.Customers
	revenue := floatTotal(sp.TotalChargeAmount.Sum().Filter(sp.CreatedAt.Gte(currentStart)))

	return sp.
		Join(cus, cus.ID.EqCol(sp.CustomerID)).
		Pick(
			sp.CustomerID,
			cus.Name.Coalesce(""),
			revenue,
			countWhere(sp.CreatedAt.Gte(currentStart)),
			floatTotal(sp.TotalChargeAmount.Sum().Filter(keel.And(
				sp.CreatedAt.Gte(previousStart),
				sp.CreatedAt.Lt(currentStart),
			))),
			keel.CastFloat64(keel.Float64Of(keel.Sum(revenue).OverWindow(keel.Window{}))),
		).
		Where(
			within(sp.CreatedAt, previousStart, now),
			sp.Status.Ne(shipmentvocab.StatusCanceled),
		).
		Group(sp.CustomerID, cus.Name).
		Having(revenue.Gt(0)).
		Order(revenue.Desc()).
		Limit(customerMixLimit)
}

func (p *Provider) getCustomerMix(ctx context.Context) (*CustomerMixCard, error) {
	return dbtx.Write(ctx, p.db, func(ctx context.Context) (*CustomerMixCard, error) {
		now := timeutils.NowUnix()
		currentStart := now - int64(customerMixWindowDays)*24*60*60
		previousStart := currentStart - int64(customerMixWindowDays)*24*60*60

		rows, err := queryAll[customerMixRow](
			ctx,
			p.db.Keel(ctx),
			customerMixQuery(now, currentStart, previousStart),
		)
		if err != nil {
			return nil, err
		}

		return buildCustomerMixCard(rows), nil
	})
}

func placeLabel(loc *locationdb.LocationsTable) keel.StringExpr {
	return loc.Code.NullIf("").Coalesce(loc.City.NullIf(""), loc.Name, "")
}

func tomorrowsPickupsQuery(from, to int64, limit, offset int) *keel.Query[*shipment.Stop] {
	stp := shipmentdb.Stops
	sm := shipmentdb.ShipmentMoves
	sp := shipmentdb.Shipments
	a := shipmentdb.Assignments
	cus := customerdb.Customers
	pickupLoc := locationdb.Locations.As(pickupLocationAlias)
	wrk := shipmentdb.Assignments.PrimaryWorker().As(pickupWorkerAlias)
	destLoc := locationdb.Locations.As(destinationLocationAlias)
	destStop := shipmentdb.Stops.As(destinationStopAlias)
	destMove := shipmentdb.ShipmentMoves.As(destinationMoveAlias)

	destination := destLoc.
		Join(destStop, destStop.LocationID.EqCol(destLoc.ID)).
		Join(destMove, destMove.ID.EqCol(destStop.ShipmentMoveID)).
		Pick(destLoc.Code, destLoc.City, destLoc.Name).
		Where(
			destMove.ShipmentID.EqCol(sp.ID),
			destStop.Status.Ne(shipmentvocab.StopStatusCanceled),
			destStop.Type.In(shipmentvocab.DeliveryStopTypes()...),
		).
		Order(destMove.Sequence.Desc(), destStop.Sequence.Desc()).
		Limit(1)
	driver := keel.ConcatWS(" ", wrk.FirstName.NullIf(""), wrk.LastName.NullIf("")).
		NullIf("").
		Coalesce("")

	return stp.
		Join(sm, sm.ID.EqCol(stp.ShipmentMoveID)).
		Join(sp, sp.ID.EqCol(sm.ShipmentID)).
		Join(cus, cus.ID.EqCol(sp.CustomerID)).
		Join(pickupLoc, pickupLoc.ID.EqCol(stp.LocationID)).
		LeftJoin(
			a,
			a.ShipmentMoveID.EqCol(sm.ID),
			a.ArchivedAt.IsNull(),
			a.Status.Ne(shipmentvocab.AssignmentStatusCanceled),
		).
		LeftJoin(wrk, wrk.ID.EqCol(a.PrimaryWorkerID)).
		LeftJoinLateral(destination).
		Pick(
			sp.ID,
			sp.ProNumber,
			stp.ScheduledWindowStart,
			cus.Name.Coalesce(""),
			placeLabel(pickupLoc),
			placeLabel(destLoc),
			driver,
			sp.Status,
			a.PrimaryWorkerID.IsNotNull(),
		).
		Where(
			stp.Type.In(shipmentvocab.PickupStopTypes()...),
			stp.Status.Ne(shipmentvocab.StopStatusCanceled),
			sm.Status.Ne(shipmentvocab.MoveStatusCanceled),
			sp.Status.Ne(shipmentvocab.StatusCanceled),
			stp.ScheduledWindowStart.Gte(from),
			stp.ScheduledWindowStart.Lt(to),
		).
		Order(stp.ScheduledWindowStart.Asc(), sp.ProNumber.Asc()).
		Limit(limit).
		Offset(offset)
}

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
			tomorrowsPickupsQuery(tomorrowStart.Unix(), tomorrowEnd.Unix(), limit, req.offset),
		)
		if err != nil {
			return nil, err
		}

		return buildTomorrowsPickupsCard(tomorrowStart, rows), nil
	})
}

func (p *Provider) getLaneHeatmap(ctx context.Context, windowDays int) (*LaneHeatmapCard, error) {
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
					s.Status.Ne(shipmentvocab.StatusCanceled),
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
	case shipmentvocab.StatusNew, shipmentvocab.StatusPartiallyAssigned:
		return TomorrowPickupStatusTentative
	case shipmentvocab.StatusAssigned, shipmentvocab.StatusInTransit:
		return TomorrowPickupStatusConfirmed
	case shipmentvocab.StatusDelayed,
		shipmentvocab.StatusPartiallyCompleted,
		shipmentvocab.StatusReadyToInvoice,
		shipmentvocab.StatusCompleted,
		shipmentvocab.StatusInvoiced,
		shipmentvocab.StatusCanceled:
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
