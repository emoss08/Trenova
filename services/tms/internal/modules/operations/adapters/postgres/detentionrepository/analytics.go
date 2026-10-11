package detentionrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/buncolgen/detentioncol"
	"github.com/emoss08/trenova/pkg/buncolgen/locationcol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type AnalyticsParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type analyticsRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewAnalyticsRepository(p AnalyticsParams) detentionports.DetentionAnalyticsRepository {
	return &analyticsRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.detention-analytics-repository"),
	}
}

// ListStopFacts pulls the raw dwell history a backtest replays against. Only
// completed stops with both timestamps are returned, because a stop the driver
// has not left has no settled duration to re-price.
func (r *analyticsRepository) ListStopFacts(
	ctx context.Context,
	req *detentionports.ListStopFactsRequest,
) ([]*detentionports.StopFact, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*detentionports.StopFact, error) {
		stopCols := shipmentcol.StopColumns
		moveCols := shipmentcol.ShipmentMoveColumns
		shipCols := shipmentcol.ShipmentColumns

		facts := make([]*detentionports.StopFact, 0, 256)

		q := r.db.DBForContext(ctx).
			NewSelect().
			Model((*shipment.Stop)(nil)).
			ColumnExpr(stopCols.ID.As("stop_id")).
			ColumnExpr(stopCols.LocationID.As("location_id")).
			ColumnExpr(stopCols.Type.As("stop_type")).
			ColumnExpr(stopCols.ScheduleType.As("schedule_type")).
			ColumnExpr(stopCols.ScheduledWindowStart.As("scheduled_window_start")).
			ColumnExpr(stopCols.ScheduledWindowEnd.As("scheduled_window_end")).
			ColumnExpr(stopCols.ActualArrival.As("actual_arrival")).
			ColumnExpr(stopCols.ActualDeparture.As("actual_departure")).
			ColumnExpr(stopCols.CountDetentionOverride.As("count_detention_override")).
			ColumnExpr(moveCols.ShipmentID.As("shipment_id")).
			ColumnExpr(shipCols.CustomerID.As("customer_id")).
			ColumnExpr(shipCols.ShipmentTypeID.As("shipment_type_id")).
			ColumnExpr(shipCols.ServiceTypeID.As("service_type_id")).
			Join("JOIN shipment_moves AS sm ON ?", bun.Safe(
				moveCols.ID.EqColumn(stopCols.ShipmentMoveID)+
					" AND "+moveCols.OrganizationID.EqColumn(stopCols.OrganizationID),
			)).
			Join("JOIN shipments AS sp ON ?", bun.Safe(
				shipCols.ID.EqColumn(moveCols.ShipmentID)+
					" AND "+shipCols.OrganizationID.EqColumn(moveCols.OrganizationID),
			)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return shipmentcol.StopScopeTenant(sq, req.TenantInfo).
					Where(stopCols.ActualArrival.IsNotNull()).
					Where(stopCols.ActualDeparture.IsNotNull()).
					Where(stopCols.Status.Ne(), shipment.StopStatusCanceled)
			}).
			Order(stopCols.ActualArrival.OrderDesc())

		if req.From > 0 {
			q = q.Where(stopCols.ActualArrival.Gte(), req.From)
		}
		if req.To > 0 {
			q = q.Where(stopCols.ActualArrival.Lte(), req.To)
		}
		if req.CustomerID != nil && !req.CustomerID.IsNil() {
			q = q.Where(shipCols.CustomerID.Eq(), *req.CustomerID)
		}
		if req.LocationID != nil && !req.LocationID.IsNil() {
			q = q.Where(stopCols.LocationID.Eq(), *req.LocationID)
		}
		if req.Limit > 0 {
			q = q.Limit(req.Limit)
		}

		if err := q.Scan(ctx, &facts); err != nil {
			r.l.Error("failed to list stop facts", zap.Error(err))
			return nil, err
		}

		return facts, nil
	})
}

// BaselineByStop returns what each stop actually billed in one pass, so a
// backtest can diff against reality without a query per stop.
func (r *analyticsRepository) BaselineByStop(
	ctx context.Context,
	req *detentionports.DetentionStatsRequest,
) ([]*detentionports.StopBaseline, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*detentionports.StopBaseline, error) {
		cols := detentioncol.DetentionOccurrenceColumns
		baselines := make([]*detentionports.StopBaseline, 0, 256)

		q := r.db.DBForContext(ctx).
			NewSelect().
			Model((*detention.DetentionOccurrence)(nil)).
			ColumnExpr(cols.StopID.As("stop_id")).
			ColumnExpr(cols.BillableAmount.As("billable_amount")).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return detentioncol.DetentionOccurrenceScopeTenant(sq, req.TenantInfo)
			})

		if req.From > 0 {
			q = q.Where(cols.ClockStartAt.Gte(), req.From)
		}
		if req.To > 0 {
			q = q.Where(cols.ClockStartAt.Lte(), req.To)
		}

		if err := q.Scan(ctx, &baselines); err != nil {
			r.l.Error("failed to load detention baselines", zap.Error(err))
			return nil, err
		}

		return baselines, nil
	})
}

// FacilityStats aggregates settled detention by facility so the worst docks are
// identifiable at planning and negotiation time rather than only at billing.
func (r *analyticsRepository) FacilityStats(
	ctx context.Context,
	req *detentionports.DetentionStatsRequest,
) ([]*detentionports.FacilityDetentionStat, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detentionports.FacilityDetentionStat, error) {
			cols := detentioncol.DetentionOccurrenceColumns
			locCols := locationcol.LocationColumns
			stats := make([]*detentionports.FacilityDetentionStat, 0, 64)

			q := r.db.DBForContext(ctx).
				NewSelect().
				Model((*detention.DetentionOccurrence)(nil)).
				ColumnExpr(cols.LocationID.As("location_id")).
				ColumnExpr(
					"COALESCE(?, '') AS location_name",
					bun.Safe(locCols.Name.Qualified()),
				).
				Join("LEFT JOIN locations AS loc ON ?", bun.Safe(
					locCols.ID.EqColumn(cols.LocationID)+
						" AND "+locCols.OrganizationID.EqColumn(cols.OrganizationID),
				)).
				ColumnExpr("COUNT(*) AS stop_count").
				ColumnExpr("COUNT(*) FILTER (WHERE ? > 0) AS breach_count",
					bun.Safe(cols.RoundedMinutes.Qualified())).
				ColumnExpr("COALESCE(ROUND(AVG(?)), 0) AS avg_dwell_minutes",
					bun.Safe(cols.RawDwellMinutes.Qualified())).
				ColumnExpr(
					"COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY ?), 0) AS median_dwell_minutes",
					bun.Safe(cols.RawDwellMinutes.Qualified())).
				ColumnExpr(
					"COALESCE(PERCENTILE_CONT(0.9) WITHIN GROUP (ORDER BY ?), 0) AS p90_dwell_minutes",
					bun.Safe(cols.RawDwellMinutes.Qualified())).
				ColumnExpr("COALESCE(SUM(?), 0) AS billed_amount",
					bun.Safe(cols.BillableAmount.Qualified())).
				ColumnExpr("COALESCE(SUM(?), 0) AS driver_pay_amount",
					bun.Safe(cols.DriverPayAmount.Qualified())).
				ColumnExpr("COALESCE(SUM(?), 0) AS net_margin",
					bun.Safe(cols.NetMargin.Qualified())).
				ColumnExpr("COALESCE(SUM(?), 0) AS waived_amount",
					bun.Safe(cols.WaivedAmount.Qualified())).
				ColumnExpr("COUNT(*) FILTER (WHERE ? = 'Disputed') AS dispute_count",
					bun.Safe(cols.Status.Qualified())).
				ColumnExpr("COUNT(*) FILTER (WHERE ?) AS suppressed_count",
					bun.Safe(cols.SuppressedByGate.Qualified())).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return detentioncol.DetentionOccurrenceScopeTenant(sq, req.TenantInfo).
						Where(cols.IsOpen.IsFalse())
				}).
				GroupExpr(cols.LocationID.Qualified()).
				GroupExpr(locCols.Name.Qualified()).
				OrderExpr("billed_amount DESC")

			if req.From > 0 {
				q = q.Where(cols.ClockStartAt.Gte(), req.From)
			}
			if req.To > 0 {
				q = q.Where(cols.ClockStartAt.Lte(), req.To)
			}
			if req.Limit > 0 {
				q = q.Limit(req.Limit)
			}

			if err := q.Scan(ctx, &stats); err != nil {
				r.l.Error("failed to aggregate facility detention stats", zap.Error(err))
				return nil, err
			}

			return stats, nil
		},
	)
}

// CustomerStats aggregates settled detention by customer for margin review and
// rate negotiation.
func (r *analyticsRepository) CustomerStats(
	ctx context.Context,
	req *detentionports.DetentionStatsRequest,
) ([]*detentionports.CustomerDetentionStat, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detentionports.CustomerDetentionStat, error) {
			cols := detentioncol.DetentionOccurrenceColumns
			cusCols := customercol.CustomerColumns
			stats := make([]*detentionports.CustomerDetentionStat, 0, 64)

			q := r.db.DBForContext(ctx).
				NewSelect().
				Model((*detention.DetentionOccurrence)(nil)).
				ColumnExpr(cols.CustomerID.As("customer_id")).
				ColumnExpr(
					"COALESCE(?, '') AS customer_name",
					bun.Safe(cusCols.Name.Qualified()),
				).
				Join("LEFT JOIN customers AS cus ON ?", bun.Safe(
					cusCols.ID.EqColumn(cols.CustomerID)+
						" AND "+cusCols.OrganizationID.EqColumn(cols.OrganizationID),
				)).
				ColumnExpr("COUNT(*) AS stop_count").
				ColumnExpr("COUNT(*) FILTER (WHERE ? > 0) AS breach_count",
					bun.Safe(cols.RoundedMinutes.Qualified())).
				ColumnExpr("COALESCE(SUM(?), 0) AS billed_amount",
					bun.Safe(cols.BillableAmount.Qualified())).
				ColumnExpr("COALESCE(SUM(?), 0) AS driver_pay_amount",
					bun.Safe(cols.DriverPayAmount.Qualified())).
				ColumnExpr("COALESCE(SUM(?), 0) AS net_margin",
					bun.Safe(cols.NetMargin.Qualified())).
				ColumnExpr("COALESCE(SUM(?), 0) AS waived_amount",
					bun.Safe(cols.WaivedAmount.Qualified())).
				ColumnExpr("COUNT(*) FILTER (WHERE ? = 'Disputed') AS dispute_count",
					bun.Safe(cols.Status.Qualified())).
				ColumnExpr("COUNT(*) FILTER (WHERE ?) AS suppressed_count",
					bun.Safe(cols.SuppressedByGate.Qualified())).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return detentioncol.DetentionOccurrenceScopeTenant(sq, req.TenantInfo).
						Where(cols.IsOpen.IsFalse())
				}).
				GroupExpr(cols.CustomerID.Qualified()).
				GroupExpr(cusCols.Name.Qualified()).
				OrderExpr("billed_amount DESC")

			if req.From > 0 {
				q = q.Where(cols.ClockStartAt.Gte(), req.From)
			}
			if req.To > 0 {
				q = q.Where(cols.ClockStartAt.Lte(), req.To)
			}
			if req.Limit > 0 {
				q = q.Limit(req.Limit)
			}

			if err := q.Scan(ctx, &stats); err != nil {
				r.l.Error("failed to aggregate customer detention stats", zap.Error(err))
				return nil, err
			}

			return stats, nil
		},
	)
}

// WaiverStats surfaces where discretionary revenue is going: which coded reason,
// how much, and who approved it.
func (r *analyticsRepository) WaiverStats(
	ctx context.Context,
	req *detentionports.DetentionStatsRequest,
) ([]*detentionports.WaiverLeakageStat, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*detentionports.WaiverLeakageStat, error) {
			cols := detentioncol.DetentionOccurrenceColumns
			stats := make([]*detentionports.WaiverLeakageStat, 0, 16)

			q := r.db.DBForContext(ctx).
				NewSelect().
				Model((*detention.DetentionOccurrence)(nil)).
				ColumnExpr(cols.WaiverReason.As("reason")).
				ColumnExpr("COUNT(*) AS waiver_count").
				ColumnExpr("COUNT(DISTINCT ?) AS approver_count",
					bun.Safe(cols.WaivedByID.Qualified())).
				ColumnExpr("COALESCE(SUM(?), 0) AS waived_amount",
					bun.Safe(cols.WaivedAmount.Qualified())).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return detentioncol.DetentionOccurrenceScopeTenant(sq, req.TenantInfo).
						Where(cols.Status.Eq(), detention.OccurrenceStatusWaived)
				}).
				GroupExpr(cols.WaiverReason.Qualified()).
				OrderExpr("waived_amount DESC")

			if req.From > 0 {
				q = q.Where(cols.WaivedAt.Gte(), req.From)
			}
			if req.To > 0 {
				q = q.Where(cols.WaivedAt.Lte(), req.To)
			}

			if err := q.Scan(ctx, &stats); err != nil {
				r.l.Error("failed to aggregate detention waiver stats", zap.Error(err))
				return nil, err
			}

			return stats, nil
		},
	)
}

type entityNameRow struct {
	ID   pulid.ID `bun:"id"`
	Name string   `bun:"name"`
}

// EntityNames resolves display names for the location and customer IDs a
// backtest groups by, so its buckets can be labeled without a query per row.
func (r *analyticsRepository) EntityNames(
	ctx context.Context,
	req *detentionports.DetentionEntityNamesRequest,
) (*detentionports.DetentionEntityNames, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*detentionports.DetentionEntityNames, error) {
			names := &detentionports.DetentionEntityNames{
				Locations: make(map[pulid.ID]string, len(req.LocationIDs)),
				Customers: make(map[pulid.ID]string, len(req.CustomerIDs)),
			}

			if len(req.LocationIDs) > 0 {
				locCols := locationcol.LocationColumns
				rows := make([]entityNameRow, 0, len(req.LocationIDs))

				err := r.db.DBForContext(ctx).
					NewSelect().
					Model((*location.Location)(nil)).
					ColumnExpr(locCols.ID.As("id")).
					ColumnExpr(locCols.Name.As("name")).
					WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
						return locationcol.LocationScopeTenant(sq, req.TenantInfo).
							Where(locCols.ID.In(), bun.List(req.LocationIDs))
					}).
					Scan(ctx, &rows)
				if err != nil {
					r.l.Error("failed to resolve location names", zap.Error(err))
					return nil, err
				}

				for _, row := range rows {
					names.Locations[row.ID] = row.Name
				}
			}

			if len(req.CustomerIDs) > 0 {
				cusCols := customercol.CustomerColumns
				rows := make([]entityNameRow, 0, len(req.CustomerIDs))

				err := r.db.DBForContext(ctx).
					NewSelect().
					Model((*customer.Customer)(nil)).
					ColumnExpr(cusCols.ID.As("id")).
					ColumnExpr(cusCols.Name.As("name")).
					WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
						return customercol.CustomerScopeTenant(sq, req.TenantInfo).
							Where(cusCols.ID.In(), bun.List(req.CustomerIDs))
					}).
					Scan(ctx, &rows)
				if err != nil {
					r.l.Error("failed to resolve customer names", zap.Error(err))
					return nil, err
				}

				for _, row := range rows {
					names.Customers[row.ID] = row.Name
				}
			}

			return names, nil
		},
	)
}
