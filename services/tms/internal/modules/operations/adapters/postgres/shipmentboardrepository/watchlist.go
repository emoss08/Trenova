package shipmentboardrepository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/buncolgen/locationcol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	deliveryAlias      = "ld"
	pickupAlias        = "fp"
	dwellAlias         = "dw"
	maxDwellingStops   = 1000
	maxBillingCustomer = 25
)

var (
	ldDeliveryAt    = buncolgen.NewColumn("delivery_at", deliveryAlias)
	ldActualArrival = buncolgen.NewColumn("actual_arrival", deliveryAlias)
	ldCutoff        = buncolgen.NewColumn("cutoff", deliveryAlias)
	ldCity          = buncolgen.NewColumn("city", deliveryAlias)
	fpPickupAt      = buncolgen.NewColumn("pickup_at", pickupAlias)
	fpCity          = buncolgen.NewColumn("city", pickupAlias)
	dwStopID        = buncolgen.NewColumn("stop_id", dwellAlias)
	dwMoveID        = buncolgen.NewColumn("move_id", dwellAlias)
	dwFacility      = buncolgen.NewColumn("facility_name", dwellAlias)
	dwArrival       = buncolgen.NewColumn("actual_arrival", dwellAlias)
)

func withStopLocation(q *bun.SelectQuery) *bun.SelectQuery {
	stp := shipmentcol.StopColumns
	loc := locationcol.LocationColumns

	return q.
		Join("LEFT JOIN " + locationcol.LocationTable.As(locationcol.LocationTable.Alias)).
		JoinOn(loc.ID.EqColumn(stp.LocationID)).
		JoinOn(loc.OrganizationID.EqColumn(stp.OrganizationID)).
		JoinOn(loc.BusinessUnitID.EqColumn(stp.BusinessUnitID))
}

func withCustomer(q *bun.SelectQuery) *bun.SelectQuery {
	sp := shipmentcol.ShipmentColumns
	cus := customercol.CustomerColumns

	return q.
		Join("JOIN " + customercol.CustomerTable.As(customercol.CustomerTable.Alias)).
		JoinOn(cus.ID.EqColumn(sp.CustomerID)).
		JoinOn(cus.OrganizationID.EqColumn(sp.OrganizationID)).
		JoinOn(cus.BusinessUnitID.EqColumn(sp.BusinessUnitID))
}

func lastDeliveryLateral(dba bun.IDB) *bun.SelectQuery {
	stp := shipmentcol.StopColumns
	loc := locationcol.LocationColumns

	return withStopLocation(shipmentrepository.LastDeliveryStopQuery(dba)).
		ColumnExpr(shipmentrepository.DeliveryAtExpr() + " AS " + ldDeliveryAt.Name).
		ColumnExpr(stp.ActualArrival.As(ldActualArrival.Name)).
		ColumnExpr(buncolgen.Expr(
			"COALESCE({0}, {1}) AS "+ldCutoff.Name,
			stp.ScheduledWindowEnd,
			stp.ScheduledWindowStart,
		)).
		ColumnExpr(loc.City.As(ldCity.Name))
}

func (r *Repository) ListDeliveriesToday(
	ctx context.Context,
	req *shipmentports.ShipmentWatchlistRequest,
) ([]*shipmentports.ShipmentDeliveryRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentports.ShipmentDeliveryRow, error) {
			dba := r.db.DBForContext(ctx)
			sp := shipmentcol.ShipmentColumns
			cus := customercol.CustomerColumns

			today, err := shipmentrepository.QuickFilterCondition(
				dba,
				req.Basis,
				shipment.Quick(shipment.QuickFilterDeliveringToday),
			)
			if err != nil {
				return nil, err
			}

			rows := make([]*shipmentports.ShipmentDeliveryRow, 0, 64)
			q := dba.NewSelect().
				Model((*shipment.Shipment)(nil)).
				ColumnExpr(sp.ID.As("shipment_id")).
				ColumnExpr(sp.ProNumber.As("pro_number")).
				ColumnExpr(sp.StageRank.As("stage_rank")).
				ColumnExpr(cus.Name.As("customer_name")).
				ColumnExpr(ldDeliveryAt.As(ldDeliveryAt.Name)).
				ColumnExpr(ldActualArrival.As(ldActualArrival.Name)).
				ColumnExpr(ldCutoff.As(ldCutoff.Name)).
				ColumnExpr(ldCity.Expr("COALESCE({}, '') AS "+ldCity.Name)).
				Apply(withCustomer).
				Join("JOIN LATERAL (?) AS ? ON TRUE", lastDeliveryLateral(dba), bun.Ident(deliveryAlias)).
				Apply(shipmentcol.ShipmentApplyTenant(req.TenantInfo)).
				Where("?", today).
				Order(ldDeliveryAt.OrderAsc())

			if err = q.Scan(ctx, &rows); err != nil {
				r.l.Error("failed to list deliveries today", zap.Error(err))
				return nil, err
			}

			return rows, nil
		},
	)
}

func (r *Repository) NextUncoveredPickup(
	ctx context.Context,
	req *shipmentports.ShipmentWatchlistRequest,
) (*shipmentports.ShipmentPickupRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*shipmentports.ShipmentPickupRow, error) {
			dba := r.db.DBForContext(ctx)
			sp := shipmentcol.ShipmentColumns
			stp := shipmentcol.StopColumns
			loc := locationcol.LocationColumns

			pickup := withStopLocation(shipmentrepository.FirstPickupStopQuery(dba)).
				ColumnExpr(stp.ScheduledWindowStart.As(fpPickupAt.Name)).
				ColumnExpr(loc.City.As(fpCity.Name))

			row := new(shipmentports.ShipmentPickupRow)
			err := dba.NewSelect().
				Model((*shipment.Shipment)(nil)).
				ColumnExpr(sp.ID.As("shipment_id")).
				ColumnExpr(fpPickupAt.As(fpPickupAt.Name)).
				ColumnExpr(fpCity.Expr("COALESCE({}, '') AS origin_city")).
				ColumnExpr(ldCity.Expr("COALESCE({}, '') AS destination_city")).
				Join("JOIN LATERAL (?) AS ? ON TRUE", pickup, bun.Ident(pickupAlias)).
				Join("LEFT JOIN LATERAL (?) AS ? ON TRUE", lastDeliveryLateral(dba), bun.Ident(deliveryAlias)).
				Apply(shipmentcol.ShipmentApplyTenant(req.TenantInfo)).
				Where("?", shipmentrepository.StageCondition(shipment.StageNeedsCoverage)).
				Where(fpPickupAt.Gte(), req.Basis.Now.Unix()).
				Order(fpPickupAt.OrderAsc(), sp.ID.OrderAsc()).
				Limit(1).
				Scan(ctx, row)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, nil
				}
				r.l.Error("failed to read next uncovered pickup", zap.Error(err))
				return nil, err
			}

			return row, nil
		},
	)
}

func (r *Repository) ListDwellingStops(
	ctx context.Context,
	req *shipmentports.ShipmentWatchlistRequest,
) ([]*shipmentports.ShipmentDwellRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentports.ShipmentDwellRow, error) {
			if req.Basis == nil || req.Basis.Detention == nil {
				return nil, shipmentrepository.ErrQuickFilterDetentionMissing
			}

			dba := r.db.DBForContext(ctx)
			sp := shipmentcol.ShipmentColumns
			stp := shipmentcol.StopColumns
			sm := shipmentcol.ShipmentMoveColumns
			loc := locationcol.LocationColumns

			dwelling := withStopLocation(shipmentrepository.DwellingStopsQuery(
				dba,
				req.Basis.Now.Unix(),
				req.Basis.Detention.ThresholdMinutes,
			)).
				ColumnExpr(stp.ID.As(dwStopID.Name)).
				ColumnExpr(sm.ID.As(dwMoveID.Name)).
				ColumnExpr(loc.Name.Expr("COALESCE({}, '') AS " + dwFacility.Name)).
				ColumnExpr(stp.ActualArrival.As(dwArrival.Name))

			rows := make([]*shipmentports.ShipmentDwellRow, 0, 16)
			err := dba.NewSelect().
				Model((*shipment.Shipment)(nil)).
				ColumnExpr(sp.ID.As("shipment_id")).
				ColumnExpr(dwMoveID.As(dwMoveID.Name)).
				ColumnExpr(dwStopID.As(dwStopID.Name)).
				ColumnExpr(dwFacility.As(dwFacility.Name)).
				ColumnExpr(dwArrival.As(dwArrival.Name)).
				Join("JOIN LATERAL (?) AS ? ON TRUE", dwelling, bun.Ident(dwellAlias)).
				Apply(shipmentcol.ShipmentApplyTenant(req.TenantInfo)).
				Order(dwArrival.OrderAsc(), dwStopID.OrderAsc()).
				Limit(maxDwellingStops).
				Scan(ctx, &rows)
			if err != nil {
				r.l.Error("failed to list dwelling stops", zap.Error(err))
				return nil, err
			}

			return rows, nil
		},
	)
}

func (r *Repository) ListReadyToBillCustomers(
	ctx context.Context,
	req *shipmentports.ListReadyToBillCustomersRequest,
) ([]*shipmentports.ShipmentBillingCustomerRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentports.ShipmentBillingCustomerRow, error) {
			sp := shipmentcol.ShipmentColumns
			cus := customercol.CustomerColumns

			limit := req.Limit
			if limit <= 0 || limit > maxBillingCustomer {
				limit = maxBillingCustomer
			}

			rows := make([]*shipmentports.ShipmentBillingCustomerRow, 0, limit)
			err := r.db.DBForContext(ctx).NewSelect().
				Model((*shipment.Shipment)(nil)).
				ColumnExpr(sp.CustomerID.As("customer_id")).
				ColumnExpr(cus.Name.As("name")).
				ColumnExpr(buncolgen.Count("count")).
				ColumnExpr(sp.TotalChargeAmount.Expr("COALESCE(SUM({}), 0) AS total")).
				ColumnExpr("COUNT(*) OVER () AS total_customers").
				Apply(withCustomer).
				Apply(shipmentcol.ShipmentApplyTenant(req.TenantInfo)).
				Where("?", shipmentrepository.BillingTransferCandidateCondition()).
				GroupExpr(sp.CustomerID.Qualified()).
				GroupExpr(cus.Name.Qualified()).
				OrderExpr("total DESC").
				OrderExpr(sp.CustomerID.OrderAsc()).
				Limit(int64(limit)).
				Scan(ctx, &rows)
			if err != nil {
				r.l.Error("failed to list ready to bill customers", zap.Error(err))
				return nil, err
			}

			return rows, nil
		},
	)
}
