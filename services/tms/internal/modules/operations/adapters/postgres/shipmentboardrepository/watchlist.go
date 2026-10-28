package shipmentboardrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentboardports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"go.uber.org/zap"
)

const (
	deliveryAlias         = "ld"
	deliveryMoveAlias     = "ld_sm"
	deliveryLocationAlias = "ld_loc"
	pickupAlias           = "fp"
	pickupMoveAlias       = "fp_sm"
	pickupLocationAlias   = "fp_loc"
	dwellAlias            = "dw"
	dwellMoveAlias        = "dw_sm"
	dwellLocationAlias    = "dw_loc"
	maxDwellingStops      = 1000
	maxBillingCustomer    = 25
)

func (r *Repository) ListDeliveriesToday(
	ctx context.Context,
	req *shipmentboardports.ShipmentWatchlistRequest,
) ([]*shipmentboardports.ShipmentDeliveryRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentboardports.ShipmentDeliveryRow, error) {
			today, err := shipmentrepository.QuickFilterCondition(
				req.Basis,
				shipment.Quick(shipment.QuickFilterDeliveringToday),
			)
			if err != nil {
				return nil, err
			}

			sp := shipmentdb.Shipments
			ld := shipmentdb.Stops.As(deliveryAlias)
			loc := locationdb.Locations.As(deliveryLocationAlias)
			delivery := shipmentrepository.LastDeliveryStop(
				ld,
				shipmentdb.ShipmentMoves.As(deliveryMoveAlias),
			).Pick(ld.ID, ld.LocationID, ld.ActualArrival, ld.ScheduledWindowStart, ld.ScheduledWindowEnd)
			deliveryAt := shipmentrepository.DeliveryAt(ld)

			rows, err := keel.Into[shipmentboardports.ShipmentDeliveryRow](ctx, r.db.Keel(ctx), sp.
				JoinLateral(delivery).
				LeftJoin(loc, loc.ID.EqCol(ld.LocationID)).
				Pick(
					sp.ID,
					sp.ProNumber,
					sp.StageRank,
					sp.Customer().Name,
					loc.City.Coalesce(""),
					deliveryAt,
					ld.ActualArrival,
					ld.ScheduledWindowEnd.Coalesce(ld.ScheduledWindowStart),
				).
				Where(today).
				Order(keel.Asc(deliveryAt)))
			if err != nil {
				r.l.Error("failed to list deliveries today", zap.Error(err))
				return nil, err
			}

			return pointers(rows), nil
		},
	)
}

func (r *Repository) NextUncoveredPickup(
	ctx context.Context,
	req *shipmentboardports.ShipmentWatchlistRequest,
) (*shipmentboardports.ShipmentPickupRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*shipmentboardports.ShipmentPickupRow, error) {
			sp := shipmentdb.Shipments
			fp := shipmentdb.Stops.As(pickupAlias)
			fpLoc := locationdb.Locations.As(pickupLocationAlias)
			ld := shipmentdb.Stops.As(deliveryAlias)
			ldLoc := locationdb.Locations.As(deliveryLocationAlias)

			pickup := shipmentrepository.FirstPickupStop(
				fp,
				shipmentdb.ShipmentMoves.As(pickupMoveAlias),
			).Pick(fp.ID, fp.LocationID, fp.ScheduledWindowStart)
			delivery := shipmentrepository.LastDeliveryStop(
				ld,
				shipmentdb.ShipmentMoves.As(deliveryMoveAlias),
			).Pick(ld.ID, ld.LocationID)

			rows, err := keel.Into[shipmentboardports.ShipmentPickupRow](ctx, r.db.Keel(ctx), sp.
				JoinLateral(pickup).
				LeftJoin(fpLoc, fpLoc.ID.EqCol(fp.LocationID)).
				LeftJoinLateral(delivery).
				LeftJoin(ldLoc, ldLoc.ID.EqCol(ld.LocationID)).
				Pick(
					sp.ID,
					fp.ScheduledWindowStart,
					fpLoc.City.Coalesce(""),
					ldLoc.City.Coalesce(""),
				).
				Where(
					shipmentrepository.StageCondition(shipment.StageNeedsCoverage),
					fp.ScheduledWindowStart.Gte(req.Basis.Now.Unix()),
				).
				Order(fp.ScheduledWindowStart.Asc(), sp.ID.Asc()).
				Limit(1))
			if err != nil {
				r.l.Error("failed to read next uncovered pickup", zap.Error(err))
				return nil, err
			}
			if len(rows) == 0 {
				return nil, nil
			}

			return &rows[0], nil
		},
	)
}

func (r *Repository) ListDwellingStops(
	ctx context.Context,
	req *shipmentboardports.ShipmentWatchlistRequest,
) ([]*shipmentboardports.ShipmentDwellRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentboardports.ShipmentDwellRow, error) {
			if req.Basis == nil || req.Basis.Detention == nil {
				return nil, shipmentrepository.ErrQuickFilterDetentionMissing
			}

			sp := shipmentdb.Shipments
			dw := shipmentdb.Stops.As(dwellAlias)
			loc := locationdb.Locations.As(dwellLocationAlias)
			dwelling := shipmentrepository.DwellingStops(
				dw,
				shipmentdb.ShipmentMoves.As(dwellMoveAlias),
				req.Basis.Now.Unix(),
				req.Basis.Detention.ThresholdMinutes,
			).Pick(dw.ID, dw.ShipmentMoveID, dw.LocationID, dw.ActualArrival)

			rows, err := keel.Into[shipmentboardports.ShipmentDwellRow](ctx, r.db.Keel(ctx), sp.
				JoinLateral(dwelling).
				LeftJoin(loc, loc.ID.EqCol(dw.LocationID)).
				Pick(
					sp.ID,
					dw.ShipmentMoveID,
					dw.ID,
					loc.Name.Coalesce(""),
					dw.ActualArrival,
				).
				Order(dw.ActualArrival.Asc(), dw.ID.Asc()).
				Limit(maxDwellingStops))
			if err != nil {
				r.l.Error("failed to list dwelling stops", zap.Error(err))
				return nil, err
			}

			return pointers(rows), nil
		},
	)
}

func (r *Repository) ListReadyToBillCustomers(
	ctx context.Context,
	req *shipmentboardports.ListReadyToBillCustomersRequest,
) ([]*shipmentboardports.ShipmentBillingCustomerRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentboardports.ShipmentBillingCustomerRow, error) {
			limit := req.Limit
			if limit <= 0 || limit > maxBillingCustomer {
				limit = maxBillingCustomer
			}

			sp := shipmentdb.Shipments
			cus := sp.Customer()
			total := revenueSum()

			rows, err := keel.Into[shipmentboardports.ShipmentBillingCustomerRow](
				ctx,
				r.db.Keel(ctx),
				sp.Group(sp.CustomerID, cus.Name).
					Pick(sp.CustomerID, cus.Name, keel.Count(), total, keel.Count().OverWindow(keel.Window{})).
					Where(shipmentrepository.BillingTransferCandidateCondition()).
					Order(keel.Desc(total), sp.CustomerID.Asc()).
					Limit(limit),
			)
			if err != nil {
				r.l.Error("failed to list ready to bill customers", zap.Error(err))
				return nil, err
			}

			return pointers(rows), nil
		},
	)
}
