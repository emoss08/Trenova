package insightrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// The aggregates behind the detectors.
//
// These are read-only reporting queries over whole periods, closer in shape to
// the analytics providers than to an entity repository, and they are written as
// explicit column expressions for the same reason those are: a FILTER clause
// counting two windows in one pass does not decompose into column helpers
// without becoming harder to read than the SQL it replaces.
//
// Every one is tenant-scoped in its WHERE clause on the driving table and again
// on each join, because a join that forgets it is the way cross-tenant data
// leaks into a report.

type MetricsParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type metricsRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewMetrics(p MetricsParams) repositories.InsightMetricsRepository {
	return &metricsRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.insight-metrics-repository"),
	}
}

// onTimeCondition is the definition of a delivery made on time: arrival no later
// than the close of its scheduled window, falling back to the window start where
// no end was given. It matches what the shipment analytics provider already
// reports, so the insight panel and the dashboard cannot disagree about a number
// a person can see in both places.
const onTimeCondition = `stp.actual_arrival <= COALESCE(stp.scheduled_window_end, stp.scheduled_window_start)`

// minOnTimeStops mirrors the detector's volume floor so the query does not carry
// back hundreds of customers the detector will immediately discard. The detector
// enforces it too; this is a cost optimisation, not the rule.
const minOnTimeStops = 12

func (r *metricsRepository) CustomerOnTimeComparison(
	ctx context.Context,
	req repositories.InsightWindowRequest,
) ([]repositories.CustomerOnTimeRow, error) {
	log := r.l.With(zap.String("operation", "CustomerOnTimeComparison"))

	// The prior period is the same length immediately before the window, so the
	// two are comparable without a seasonality argument.
	priorStart := req.WindowStart - (req.WindowEnd - req.WindowStart)

	rows := make([]repositories.CustomerOnTimeRow, 0)

	err := r.db.DB().NewSelect().
		TableExpr("stops AS stp").
		ColumnExpr("cus.id AS customer_id").
		ColumnExpr("cus.name AS customer_name").
		ColumnExpr(
			"COUNT(*) FILTER (WHERE stp.actual_arrival >= ? AND stp.actual_arrival < ?)::bigint AS current_total",
			req.WindowStart, req.WindowEnd,
		).
		ColumnExpr(
			"COUNT(*) FILTER (WHERE stp.actual_arrival >= ? AND stp.actual_arrival < ? AND "+onTimeCondition+")::bigint AS current_on_time",
			req.WindowStart, req.WindowEnd,
		).
		ColumnExpr(
			"COUNT(*) FILTER (WHERE stp.actual_arrival >= ? AND stp.actual_arrival < ?)::bigint AS prior_total",
			priorStart, req.WindowStart,
		).
		ColumnExpr(
			"COUNT(*) FILTER (WHERE stp.actual_arrival >= ? AND stp.actual_arrival < ? AND "+onTimeCondition+")::bigint AS prior_on_time",
			priorStart, req.WindowStart,
		).
		Join("JOIN shipment_moves AS smv ON smv.id = stp.shipment_move_id AND smv.organization_id = stp.organization_id AND smv.business_unit_id = stp.business_unit_id").
		Join("JOIN shipments AS sp ON sp.id = smv.shipment_id AND sp.organization_id = smv.organization_id AND sp.business_unit_id = smv.business_unit_id").
		Join("JOIN customers AS cus ON cus.id = sp.customer_id AND cus.organization_id = sp.organization_id AND cus.business_unit_id = sp.business_unit_id").
		Where("stp.organization_id = ?", req.TenantInfo.OrgID).
		Where("stp.business_unit_id = ?", req.TenantInfo.BuID).
		Where("stp.status = ?", shipment.StopStatusCompleted).
		Where("stp.type IN (?)", bun.In([]shipment.StopType{
			shipment.StopTypeDelivery,
			shipment.StopTypeSplitDelivery,
		})).
		Where("stp.actual_arrival IS NOT NULL").
		Where("stp.actual_arrival > 0").
		// A stop with no scheduled window cannot be late, because nothing was
		// promised. Counting it as on time would flatter the number.
		Where("stp.scheduled_window_start > 0").
		Where("stp.actual_arrival >= ?", priorStart).
		Where("stp.actual_arrival < ?", req.WindowEnd).
		GroupExpr("cus.id, cus.name").
		Having("COUNT(*) FILTER (WHERE stp.actual_arrival >= ? AND stp.actual_arrival < ?) >= ?",
			req.WindowStart, req.WindowEnd, minOnTimeStops).
		Scan(ctx, &rows)
	if err != nil {
		log.Error("failed to read on-time comparison", zap.Error(err))

		return nil, err
	}

	return rows, nil
}

func (r *metricsRepository) UnbilledDeliveredShipments(
	ctx context.Context,
	req repositories.UnbilledShipmentsRequest,
) ([]repositories.UnbilledShipmentRow, error) {
	log := r.l.With(zap.String("operation", "UnbilledDeliveredShipments"))

	rows := make([]repositories.UnbilledShipmentRow, 0)

	err := r.db.DB().NewSelect().
		TableExpr("shipments AS sp").
		ColumnExpr("cus.id AS customer_id").
		ColumnExpr("cus.name AS customer_name").
		ColumnExpr("COUNT(*)::bigint AS shipment_count").
		ColumnExpr("COALESCE(SUM(sp.total_charge_amount), 0)::text AS total_amount").
		ColumnExpr("MIN(sp.actual_delivery_date)::bigint AS oldest_delivery").
		Join("JOIN customers AS cus ON cus.id = sp.customer_id AND cus.organization_id = sp.organization_id AND cus.business_unit_id = sp.business_unit_id").
		Where("sp.organization_id = ?", req.TenantInfo.OrgID).
		Where("sp.business_unit_id = ?", req.TenantInfo.BuID).
		Where("sp.actual_delivery_date IS NOT NULL").
		Where("sp.actual_delivery_date > 0").
		Where("sp.actual_delivery_date < ?", req.DeliveredBefore).
		// Billed is the only thing that ends this: a shipment marked ready to
		// bill but never invoiced is exactly the case worth reporting.
		Where("sp.billed_at IS NULL").
		Where("sp.status IN (?)", bun.In(shipment.BillingTransferCandidateStatuses())).
		GroupExpr("cus.id, cus.name").
		Scan(ctx, &rows)
	if err != nil {
		log.Error("failed to read unbilled shipments", zap.Error(err))

		return nil, err
	}

	return rows, nil
}

func (r *metricsRepository) UnbilledDetention(
	ctx context.Context,
	req repositories.InsightWindowRequest,
) ([]repositories.UnbilledDetentionRow, error) {
	log := r.l.With(zap.String("operation", "UnbilledDetention"))

	rows := make([]repositories.UnbilledDetentionRow, 0)

	err := r.db.DB().NewSelect().
		TableExpr("detention_occurrences AS dto").
		ColumnExpr("loc.id AS location_id").
		ColumnExpr("loc.name AS location_name").
		ColumnExpr("COUNT(*)::bigint AS occurrence_count").
		ColumnExpr("COALESCE(SUM(dto.billable_amount), 0)::text AS unbilled_amount").
		ColumnExpr("COALESCE(SUM(dto.raw_dwell_minutes), 0)::bigint AS total_dwell_minutes").
		Join("JOIN locations AS loc ON loc.id = dto.location_id AND loc.organization_id = dto.organization_id AND loc.business_unit_id = dto.business_unit_id").
		Where("dto.organization_id = ?", req.TenantInfo.OrgID).
		Where("dto.business_unit_id = ?", req.TenantInfo.BuID).
		Where("dto.clock_start_at >= ?", req.WindowStart).
		Where("dto.clock_start_at < ?", req.WindowEnd).
		// Still accruing means the clock is running and nobody has decided yet,
		// which is not money lost. These four are decisions that ended with the
		// charge unbilled.
		Where("dto.status IN (?)", bun.In([]detention.OccurrenceStatus{
			detention.OccurrenceStatusWaived,
			detention.OccurrenceStatusNotBillable,
			detention.OccurrenceStatusPending,
			detention.OccurrenceStatusApproved,
		})).
		Where("dto.is_open = ?", false).
		Where("dto.billable_amount > 0").
		GroupExpr("loc.id, loc.name").
		Scan(ctx, &rows)
	if err != nil {
		log.Error("failed to read unbilled detention", zap.Error(err))

		return nil, err
	}

	return rows, nil
}

func (r *metricsRepository) CustomerEmptyMiles(
	ctx context.Context,
	req repositories.InsightWindowRequest,
) ([]repositories.CustomerEmptyMilesRow, error) {
	log := r.l.With(zap.String("operation", "CustomerEmptyMiles"))

	rows := make([]repositories.CustomerEmptyMilesRow, 0)

	// The organization's own averages ride along on every row as a window
	// function rather than a second query, so the customer and the baseline are
	// guaranteed to describe the same set of moves. Computing them separately is
	// how a comparison quietly starts comparing different periods.
	err := r.db.DB().NewSelect().
		TableExpr("shipment_moves AS smv").
		ColumnExpr("cus.id AS customer_id").
		ColumnExpr("cus.name AS customer_name").
		ColumnExpr("COALESCE(SUM(smv.distance) FILTER (WHERE smv.loaded = FALSE), 0)::text AS empty_miles").
		ColumnExpr("COALESCE(SUM(smv.distance), 0)::text AS total_miles").
		ColumnExpr("COUNT(*)::bigint AS move_count").
		ColumnExpr("SUM(COALESCE(SUM(smv.distance) FILTER (WHERE smv.loaded = FALSE), 0)) OVER ()::text AS org_empty_miles").
		ColumnExpr("SUM(COALESCE(SUM(smv.distance), 0)) OVER ()::text AS org_total_miles").
		Join("JOIN shipments AS sp ON sp.id = smv.shipment_id AND sp.organization_id = smv.organization_id AND sp.business_unit_id = smv.business_unit_id").
		Join("JOIN customers AS cus ON cus.id = sp.customer_id AND cus.organization_id = sp.organization_id AND cus.business_unit_id = sp.business_unit_id").
		Where("smv.organization_id = ?", req.TenantInfo.OrgID).
		Where("smv.business_unit_id = ?", req.TenantInfo.BuID).
		Where("smv.status = ?", shipment.MoveStatusCompleted).
		Where("smv.distance IS NOT NULL").
		Where("smv.distance > 0").
		Where("sp.actual_delivery_date >= ?", req.WindowStart).
		Where("sp.actual_delivery_date < ?", req.WindowEnd).
		GroupExpr("cus.id, cus.name").
		Scan(ctx, &rows)
	if err != nil {
		log.Error("failed to read empty miles", zap.Error(err))

		return nil, err
	}

	return rows, nil
}

func (r *metricsRepository) ExpiringCredentials(
	ctx context.Context,
	req repositories.ExpiringCredentialsRequest,
) ([]repositories.ExpiringCredentialRow, error) {
	log := r.l.With(zap.String("operation", "ExpiringCredentials"))

	rows := make([]repositories.ExpiringCredentialRow, 0)

	err := r.db.DB().NewSelect().
		TableExpr("worker_credentials AS wcred").
		ColumnExpr("wct.id AS credential_type_id").
		ColumnExpr("wct.name AS credential_type_name").
		ColumnExpr("bool_or(wct.is_required) AS is_required").
		ColumnExpr("COUNT(DISTINCT wcred.worker_id)::bigint AS worker_count").
		ColumnExpr("MIN(wcred.expires_at)::bigint AS earliest_expiry").
		ColumnExpr(
			"COUNT(DISTINCT wcred.worker_id) FILTER (WHERE wcred.expires_at < ?)::bigint AS already_expired",
			req.From,
		).
		Join("JOIN worker_credential_types AS wct ON wct.id = wcred.credential_type_id AND wct.organization_id = wcred.organization_id AND wct.business_unit_id = wcred.business_unit_id").
		Join("JOIN workers AS wrk ON wrk.id = wcred.worker_id AND wrk.organization_id = wcred.organization_id AND wrk.business_unit_id = wcred.business_unit_id").
		Where("wcred.organization_id = ?", req.TenantInfo.OrgID).
		Where("wcred.business_unit_id = ?", req.TenantInfo.BuID).
		Where("wcred.expires_at IS NOT NULL").
		Where("wcred.expires_at > 0").
		Where("wcred.expires_at < ?", req.Through).
		// An archived credential has been superseded or the worker has left; it
		// is not a compliance exposure.
		Where("wcred.archived_at IS NULL").
		// A worker who is no longer active cannot be taken off the road by an
		// expiring credential.
		Where("wrk.status = ?", domaintypes.StatusActive).
		GroupExpr("wct.id, wct.name").
		Scan(ctx, &rows)
	if err != nil {
		log.Error("failed to read expiring credentials", zap.Error(err))

		return nil, err
	}

	return rows, nil
}
