package insightrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/jackc/pgx/v5"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type MetricsParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type metricsRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewMetrics(p MetricsParams) insightports.InsightMetricsRepository {
	return &metricsRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.insight-metrics-repository"),
	}
}

const onTimeCondition = `stp.actual_arrival <= COALESCE(stp.scheduled_window_end, stp.scheduled_window_start)`

const minOnTimeStops = 12

const customerOnTimeSQL = `
SELECT
	cus.id AS customer_id,
	cus.name AS customer_name,
	COUNT(*) FILTER (WHERE stp.actual_arrival >= $7 AND stp.actual_arrival < $6)::bigint AS current_total,
	COUNT(*) FILTER (WHERE stp.actual_arrival >= $7 AND stp.actual_arrival < $6 AND ` + onTimeCondition + `)::bigint AS current_on_time,
	COUNT(*) FILTER (WHERE stp.actual_arrival >= $5 AND stp.actual_arrival < $7)::bigint AS prior_total,
	COUNT(*) FILTER (WHERE stp.actual_arrival >= $5 AND stp.actual_arrival < $7 AND ` + onTimeCondition + `)::bigint AS prior_on_time
FROM stops AS stp
JOIN shipment_moves AS smv ON smv.id = stp.shipment_move_id
	AND smv.organization_id = stp.organization_id AND smv.business_unit_id = stp.business_unit_id
JOIN shipments AS sp ON sp.id = smv.shipment_id
	AND sp.organization_id = smv.organization_id AND sp.business_unit_id = smv.business_unit_id
JOIN customers AS cus ON cus.id = sp.customer_id
	AND cus.organization_id = sp.organization_id AND cus.business_unit_id = sp.business_unit_id
WHERE stp.organization_id = $1
	AND stp.business_unit_id = $2
	AND stp.status = $3
	AND stp.type = ANY($4)
	AND stp.actual_arrival IS NOT NULL
	AND stp.actual_arrival > 0
	AND stp.scheduled_window_start > 0
	AND stp.actual_arrival >= $5
	AND stp.actual_arrival < $6
GROUP BY cus.id, cus.name
HAVING COUNT(*) FILTER (WHERE stp.actual_arrival >= $7 AND stp.actual_arrival < $6) >= $8`

func (r *metricsRepository) CustomerOnTimeComparison(
	ctx context.Context,
	req insightports.InsightWindowRequest,
) ([]insightports.CustomerOnTimeRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]insightports.CustomerOnTimeRow, error) {
			priorStart := req.WindowStart - (req.WindowEnd - req.WindowStart)

			rows, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToStructByName[insightports.CustomerOnTimeRow],
				customerOnTimeSQL,
				req.TenantInfo.OrgID.String(),
				req.TenantInfo.BuID.String(),
				string(shipment.StopStatusCompleted),
				sliceutils.Strings([]shipment.StopType{
					shipment.StopTypeDelivery,
					shipment.StopTypeSplitDelivery,
				}),
				priorStart,
				req.WindowEnd,
				req.WindowStart,
				minOnTimeStops,
			)
			if err != nil {
				r.l.Error("failed to read on-time comparison",
					zap.String("operation", "CustomerOnTimeComparison"),
					zap.Error(err))

				return nil, err
			}

			return nonNil(rows), nil
		},
	)
}

const unbilledShipmentsSQL = `
SELECT
	cus.id AS customer_id,
	cus.name AS customer_name,
	COUNT(*)::bigint AS shipment_count,
	COALESCE(SUM(sp.total_charge_amount), 0)::text AS total_amount,
	MIN(sp.actual_delivery_date)::bigint AS oldest_delivery
FROM shipments AS sp
JOIN customers AS cus ON cus.id = sp.customer_id
	AND cus.organization_id = sp.organization_id AND cus.business_unit_id = sp.business_unit_id
WHERE sp.organization_id = $1
	AND sp.business_unit_id = $2
	AND sp.actual_delivery_date IS NOT NULL
	AND sp.actual_delivery_date > 0
	AND sp.actual_delivery_date < $3
	AND sp.billed_at IS NULL
	AND sp.status = ANY($4)
GROUP BY cus.id, cus.name`

func (r *metricsRepository) UnbilledDeliveredShipments(
	ctx context.Context,
	req insightports.UnbilledShipmentsRequest,
) ([]insightports.UnbilledShipmentRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]insightports.UnbilledShipmentRow, error) {
			rows, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToStructByName[insightports.UnbilledShipmentRow],
				unbilledShipmentsSQL,
				req.TenantInfo.OrgID.String(),
				req.TenantInfo.BuID.String(),
				req.DeliveredBefore,
				sliceutils.Strings(shipment.BillingTransferCandidateStatuses()),
			)
			if err != nil {
				r.l.Error("failed to read unbilled shipments",
					zap.String("operation", "UnbilledDeliveredShipments"),
					zap.Error(err))

				return nil, err
			}

			return nonNil(rows), nil
		},
	)
}

const unbilledDetentionSQL = `
SELECT
	loc.id AS location_id,
	loc.name AS location_name,
	COUNT(*)::bigint AS occurrence_count,
	COALESCE(SUM(dto.billable_amount), 0)::text AS unbilled_amount,
	COALESCE(SUM(dto.raw_dwell_minutes), 0)::bigint AS total_dwell_minutes
FROM detention_occurrences AS dto
JOIN locations AS loc ON loc.id = dto.location_id
	AND loc.organization_id = dto.organization_id AND loc.business_unit_id = dto.business_unit_id
WHERE dto.organization_id = $1
	AND dto.business_unit_id = $2
	AND dto.clock_start_at >= $3
	AND dto.clock_start_at < $4
	AND dto.status = ANY($5)
	AND dto.is_open = FALSE
	AND dto.billable_amount > 0
GROUP BY loc.id, loc.name`

func (r *metricsRepository) UnbilledDetention(
	ctx context.Context,
	req insightports.InsightWindowRequest,
) ([]insightports.UnbilledDetentionRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]insightports.UnbilledDetentionRow, error) {
			rows, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToStructByName[insightports.UnbilledDetentionRow],
				unbilledDetentionSQL,
				req.TenantInfo.OrgID.String(),
				req.TenantInfo.BuID.String(),
				req.WindowStart,
				req.WindowEnd,
				sliceutils.Strings([]detention.OccurrenceStatus{
					detention.OccurrenceStatusWaived,
					detention.OccurrenceStatusNotBillable,
					detention.OccurrenceStatusPending,
					detention.OccurrenceStatusApproved,
				}),
			)
			if err != nil {
				r.l.Error("failed to read unbilled detention",
					zap.String("operation", "UnbilledDetention"),
					zap.Error(err))

				return nil, err
			}

			return nonNil(rows), nil
		},
	)
}

const customerEmptyMilesSQL = `
SELECT
	cus.id AS customer_id,
	cus.name AS customer_name,
	COALESCE(SUM(smv.distance) FILTER (WHERE smv.loaded = FALSE), 0)::text AS empty_miles,
	COALESCE(SUM(smv.distance), 0)::text AS total_miles,
	COUNT(*)::bigint AS move_count,
	SUM(COALESCE(SUM(smv.distance) FILTER (WHERE smv.loaded = FALSE), 0)) OVER ()::text AS org_empty_miles,
	SUM(COALESCE(SUM(smv.distance), 0)) OVER ()::text AS org_total_miles
FROM shipment_moves AS smv
JOIN shipments AS sp ON sp.id = smv.shipment_id
	AND sp.organization_id = smv.organization_id AND sp.business_unit_id = smv.business_unit_id
JOIN customers AS cus ON cus.id = sp.customer_id
	AND cus.organization_id = sp.organization_id AND cus.business_unit_id = sp.business_unit_id
WHERE smv.organization_id = $1
	AND smv.business_unit_id = $2
	AND smv.status = $3
	AND smv.distance IS NOT NULL
	AND smv.distance > 0
	AND sp.actual_delivery_date >= $4
	AND sp.actual_delivery_date < $5
GROUP BY cus.id, cus.name`

func (r *metricsRepository) CustomerEmptyMiles(
	ctx context.Context,
	req insightports.InsightWindowRequest,
) ([]insightports.CustomerEmptyMilesRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]insightports.CustomerEmptyMilesRow, error) {
			rows, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToStructByName[insightports.CustomerEmptyMilesRow],
				customerEmptyMilesSQL,
				req.TenantInfo.OrgID.String(),
				req.TenantInfo.BuID.String(),
				string(shipment.MoveStatusCompleted),
				req.WindowStart,
				req.WindowEnd,
			)
			if err != nil {
				r.l.Error("failed to read empty miles",
					zap.String("operation", "CustomerEmptyMiles"),
					zap.Error(err))

				return nil, err
			}

			return nonNil(rows), nil
		},
	)
}

const expiringCredentialsSQL = `
SELECT
	wct.id AS credential_type_id,
	wct.name AS credential_type_name,
	bool_or(wct.is_required) AS is_required,
	COUNT(DISTINCT wcred.worker_id)::bigint AS worker_count,
	MIN(wcred.expires_at)::bigint AS earliest_expiry,
	COUNT(DISTINCT wcred.worker_id) FILTER (WHERE wcred.expires_at < $3)::bigint AS already_expired
FROM worker_credentials AS wcred
JOIN worker_credential_types AS wct ON wct.id = wcred.credential_type_id
	AND wct.organization_id = wcred.organization_id AND wct.business_unit_id = wcred.business_unit_id
JOIN workers AS wrk ON wrk.id = wcred.worker_id
	AND wrk.organization_id = wcred.organization_id AND wrk.business_unit_id = wcred.business_unit_id
WHERE wcred.organization_id = $1
	AND wcred.business_unit_id = $2
	AND wcred.expires_at IS NOT NULL
	AND wcred.expires_at > 0
	AND wcred.expires_at < $4
	AND wcred.archived_at IS NULL
	AND wrk.status = $5
GROUP BY wct.id, wct.name`

func (r *metricsRepository) ExpiringCredentials(
	ctx context.Context,
	req insightports.ExpiringCredentialsRequest,
) ([]insightports.ExpiringCredentialRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]insightports.ExpiringCredentialRow, error) {
			rows, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToStructByName[insightports.ExpiringCredentialRow],
				expiringCredentialsSQL,
				req.TenantInfo.OrgID.String(),
				req.TenantInfo.BuID.String(),
				req.From,
				req.Through,
				string(domaintypes.StatusActive),
			)
			if err != nil {
				r.l.Error("failed to read expiring credentials",
					zap.String("operation", "ExpiringCredentials"),
					zap.Error(err))

				return nil, err
			}

			return nonNil(rows), nil
		},
	)
}

func nonNil[T any](rows []T) []T {
	if rows == nil {
		return []T{}
	}

	return rows
}
