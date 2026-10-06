package shipmentboardrepository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/shipmentrepository"
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
	stp := buncolgen.StopColumns
	loc := buncolgen.LocationColumns

	return q.
		Join("LEFT JOIN " + buncolgen.LocationTable.As(buncolgen.LocationTable.Alias)).
		JoinOn(loc.ID.EqColumn(stp.LocationID)).
		JoinOn(loc.OrganizationID.EqColumn(stp.OrganizationID)).
		JoinOn(loc.BusinessUnitID.EqColumn(stp.BusinessUnitID))
}

func withCustomer(q *bun.SelectQuery) *bun.SelectQuery {
	sp := buncolgen.ShipmentColumns
	cus := buncolgen.CustomerColumns

	return q.
		Join("JOIN " + buncolgen.CustomerTable.As(buncolgen.CustomerTable.Alias)).
		JoinOn(cus.ID.EqColumn(sp.CustomerID)).
		JoinOn(cus.OrganizationID.EqColumn(sp.OrganizationID)).
		JoinOn(cus.BusinessUnitID.EqColumn(sp.BusinessUnitID))
}

func lastDeliveryLateral(dba bun.IDB) *bun.SelectQuery {
	stp := buncolgen.StopColumns
	loc := buncolgen.LocationColumns

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
	req *repositories.ShipmentWatchlistRequest,
) ([]*repositories.ShipmentDeliveryRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*repositories.ShipmentDeliveryRow, error) {
			dba := r.db.DBForContext(ctx)
			sp := buncolgen.ShipmentColumns
			cus := buncolgen.CustomerColumns

			today, err := shipmentrepository.QuickFilterCondition(
				dba,
				req.Basis,
				shipment.Quick(shipment.QuickFilterDeliveringToday),
			)
			if err != nil {
				return nil, err
			}

			rows := make([]*repositories.ShipmentDeliveryRow, 0, 64)
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
				Apply(buncolgen.ShipmentApplyTenant(req.TenantInfo)).
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
	req *repositories.ShipmentWatchlistRequest,
) (*repositories.ShipmentPickupRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*repositories.ShipmentPickupRow, error) {
		dba := r.db.DBForContext(ctx)
		sp := buncolgen.ShipmentColumns
		stp := buncolgen.StopColumns
		loc := buncolgen.LocationColumns

		pickup := withStopLocation(shipmentrepository.FirstPickupStopQuery(dba)).
			ColumnExpr(stp.ScheduledWindowStart.As(fpPickupAt.Name)).
			ColumnExpr(loc.City.As(fpCity.Name))

		row := new(repositories.ShipmentPickupRow)
		err := dba.NewSelect().
			Model((*shipment.Shipment)(nil)).
			ColumnExpr(sp.ID.As("shipment_id")).
			ColumnExpr(fpPickupAt.As(fpPickupAt.Name)).
			ColumnExpr(fpCity.Expr("COALESCE({}, '') AS origin_city")).
			ColumnExpr(ldCity.Expr("COALESCE({}, '') AS destination_city")).
			Join("JOIN LATERAL (?) AS ? ON TRUE", pickup, bun.Ident(pickupAlias)).
			Join("LEFT JOIN LATERAL (?) AS ? ON TRUE", lastDeliveryLateral(dba), bun.Ident(deliveryAlias)).
			Apply(buncolgen.ShipmentApplyTenant(req.TenantInfo)).
			Where("?", shipmentrepository.StageCondition(shipment.StageNeedsCoverage)).
			Where(fpPickupAt.Gte(), req.Basis.Now.Unix()).
			Order(fpPickupAt.OrderAsc(), sp.ID.OrderAsc()).
			Limit(1).
			Scan(ctx, row)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil //nolint:nilnil // no upcoming uncovered pickup
			}
			r.l.Error("failed to read next uncovered pickup", zap.Error(err))
			return nil, err
		}

		return row, nil
	})
}

func (r *Repository) ListDwellingStops(
	ctx context.Context,
	req *repositories.ShipmentWatchlistRequest,
) ([]*repositories.ShipmentDwellRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*repositories.ShipmentDwellRow, error) {
			if req.Basis == nil || req.Basis.Detention == nil {
				return nil, shipmentrepository.ErrQuickFilterDetentionMissing
			}

			dba := r.db.DBForContext(ctx)
			sp := buncolgen.ShipmentColumns
			stp := buncolgen.StopColumns
			sm := buncolgen.ShipmentMoveColumns
			loc := buncolgen.LocationColumns

			dwelling := withStopLocation(shipmentrepository.DwellingStopsQuery(
				dba,
				req.Basis.Now.Unix(),
				req.Basis.Detention.ThresholdMinutes,
			)).
				ColumnExpr(stp.ID.As(dwStopID.Name)).
				ColumnExpr(sm.ID.As(dwMoveID.Name)).
				ColumnExpr(loc.Name.Expr("COALESCE({}, '') AS " + dwFacility.Name)).
				ColumnExpr(stp.ActualArrival.As(dwArrival.Name))

			rows := make([]*repositories.ShipmentDwellRow, 0, 16)
			err := dba.NewSelect().
				Model((*shipment.Shipment)(nil)).
				ColumnExpr(sp.ID.As("shipment_id")).
				ColumnExpr(dwMoveID.As(dwMoveID.Name)).
				ColumnExpr(dwStopID.As(dwStopID.Name)).
				ColumnExpr(dwFacility.As(dwFacility.Name)).
				ColumnExpr(dwArrival.As(dwArrival.Name)).
				Join("JOIN LATERAL (?) AS ? ON TRUE", dwelling, bun.Ident(dwellAlias)).
				Apply(buncolgen.ShipmentApplyTenant(req.TenantInfo)).
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
	req *repositories.ListReadyToBillCustomersRequest,
) ([]*repositories.ShipmentBillingCustomerRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*repositories.ShipmentBillingCustomerRow, error) {
			sp := buncolgen.ShipmentColumns
			cus := buncolgen.CustomerColumns

			limit := req.Limit
			if limit <= 0 || limit > maxBillingCustomer {
				limit = maxBillingCustomer
			}

			rows := make([]*repositories.ShipmentBillingCustomerRow, 0, limit)
			err := r.db.DBForContext(ctx).NewSelect().
				Model((*shipment.Shipment)(nil)).
				ColumnExpr(sp.CustomerID.As("customer_id")).
				ColumnExpr(cus.Name.As("name")).
				ColumnExpr(buncolgen.Count("count")).
				ColumnExpr(sp.TotalChargeAmount.Expr("COALESCE(SUM({}), 0) AS total")).
				ColumnExpr("COUNT(*) OVER () AS total_customers").
				Apply(withCustomer).
				Apply(buncolgen.ShipmentApplyTenant(req.TenantInfo)).
				Where("?", shipmentrepository.BillingTransferCandidateCondition()).
				GroupExpr(sp.CustomerID.Qualified()).
				GroupExpr(cus.Name.Qualified()).
				OrderExpr("total DESC").
				OrderExpr(sp.CustomerID.OrderAsc()).
				Limit(limit).
				Scan(ctx, &rows)
			if err != nil {
				r.l.Error("failed to list ready to bill customers", zap.Error(err))
				return nil, err
			}

			return rows, nil
		},
	)
}
