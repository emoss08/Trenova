package shipmentboardrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentpredicate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/quickfilter"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
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
			q, err := deliveriesTodayQuery(req.Basis)
			if err != nil {
				return nil, err
			}

			rows, err := keel.Into[shipmentboardports.ShipmentDeliveryRow](ctx, r.db.Keel(ctx), q)
			if err != nil {
				r.l.Error("failed to list deliveries today", zap.Error(err))
				return nil, err
			}

			return pointers(rows), nil
		},
	)
}

// deliveriesTodayQuery lists the shipments delivering today in delivery order.
// The day and the order read the delivery time stored on the shipment, so the
// last delivery stop is looked up only for the shipments listed.
func deliveriesTodayQuery(
	basis *shipmentports.ShipmentQuickFilterBasis,
) (keel.Query[*shipment.Shipment], error) {
	today, err := shipmentpredicate.QuickFilterCondition(
		basis,
		quickfilter.Quick(quickfilter.QuickFilterDeliveringToday),
	)
	if err != nil {
		return nil, err
	}

	sp := shipmentdb.Shipments
	ld := shipmentdb.Stops.As(deliveryAlias)
	loc := locationdb.Locations.As(deliveryLocationAlias)
	delivery := shipmentpredicate.LastDeliveryStop(
		ld,
		shipmentdb.ShipmentMoves.As(deliveryMoveAlias),
	).Pick(ld.ID, ld.LocationID, ld.ActualArrival, ld.ScheduledWindowStart, ld.ScheduledWindowEnd)

	return sp.
		JoinLateral(delivery).
		LeftJoin(loc, loc.ID.EqCol(ld.LocationID)).
		Pick(
			sp.ID,
			sp.ProNumber,
			sp.StageRank,
			sp.Customer().Name,
			loc.City.Coalesce(""),
			shipmentpredicate.DeliveryAt(ld),
			ld.ActualArrival,
			ld.ScheduledWindowEnd.Coalesce(ld.ScheduledWindowStart),
		).
		Where(today).
		Order(sp.DeliveryAt.Asc(), sp.ID.Asc()), nil
}

func (r *Repository) NextUncoveredPickup(
	ctx context.Context,
	req *shipmentboardports.ShipmentWatchlistRequest,
) (*shipmentboardports.ShipmentPickupRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*shipmentboardports.ShipmentPickupRow, error) {
			rows, err := keel.Into[shipmentboardports.ShipmentPickupRow](
				ctx,
				r.db.Keel(ctx),
				nextUncoveredPickupQuery(req.Basis.Now.Unix()),
			)
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

// nextUncoveredPickupQuery reads the uncovered shipment picked up soonest after
// now. It filters and orders by the pickup time stored on the shipment, so the
// stage and pickup index yields the one row and its stops are looked up once.
func nextUncoveredPickupQuery(now int64) keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments
	fp := shipmentdb.Stops.As(pickupAlias)
	fpLoc := locationdb.Locations.As(pickupLocationAlias)
	ld := shipmentdb.Stops.As(deliveryAlias)
	ldLoc := locationdb.Locations.As(deliveryLocationAlias)

	pickup := shipmentpredicate.FirstPickupStop(
		fp,
		shipmentdb.ShipmentMoves.As(pickupMoveAlias),
	).Pick(fp.ID, fp.LocationID, fp.ScheduledWindowStart)
	delivery := shipmentpredicate.LastDeliveryStop(
		ld,
		shipmentdb.ShipmentMoves.As(deliveryMoveAlias),
	).Pick(ld.ID, ld.LocationID)

	return sp.
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
			shipmentpredicate.StageCondition(shipmentvocab.StageNeedsCoverage),
			sp.PickupAt.Gte(now),
		).
		Order(sp.PickupAt.Asc(), sp.ID.Asc()).
		Limit(1)
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
				return nil, shipmentpredicate.ErrQuickFilterDetentionMissing
			}

			rows, err := keel.Into[shipmentboardports.ShipmentDwellRow](
				ctx,
				r.db.Keel(ctx),
				dwellingStopsQuery(req.Basis.Now.Unix(), req.Basis.Detention.ThresholdMinutes),
			)
			if err != nil {
				r.l.Error("failed to list dwelling stops", zap.Error(err))
				return nil, err
			}

			return pointers(rows), nil
		},
	)
}

// dwellingStopsQuery lists the stops dwelling past the threshold, longest
// first. It reads from the dwelling stops, a small set the stops index holds,
// and joins their shipments, rather than searching every shipment's stops for
// one that is dwelling.
func dwellingStopsQuery(now, thresholdMinutes int64) keel.Query[*shipment.Stop] {
	sp := shipmentdb.Shipments
	dw := shipmentdb.Stops.As(dwellAlias)
	dwm := shipmentdb.ShipmentMoves.As(dwellMoveAlias)
	loc := locationdb.Locations.As(dwellLocationAlias)

	return shipmentpredicate.DwellingStops(dw, dwm, now, thresholdMinutes).
		Join(sp, sp.ID.EqCol(dwm.ShipmentID)).
		LeftJoin(loc, loc.ID.EqCol(dw.LocationID)).
		Pick(
			sp.ID,
			dw.ShipmentMoveID,
			dw.ID,
			loc.Name.Coalesce(""),
			dw.ActualArrival,
		).
		Order(dw.ActualArrival.Asc(), dw.ID.Asc()).
		Limit(maxDwellingStops)
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
					Where(shipmentpredicate.BillingTransferCandidateCondition()).
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
