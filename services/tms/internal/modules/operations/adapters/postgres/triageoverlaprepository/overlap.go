package triageoverlaprepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const maxLoadsChecked = 20

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) triageports.TenderOverlapFinder {
	return &repository{db: p.DB}
}

func (r *repository) FindOverlappingLoads(
	ctx context.Context,
	req *triageports.FindOverlappingLoadsRequest,
) ([]triageports.OverlappingLoad, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]triageports.OverlappingLoad, error) {
		found := make([]triageports.OverlappingLoad, 0, len(req.Loads))
		if req.CustomerID.IsNil() {
			return found, nil
		}

		checked := make([]*triageports.TenderLoad, 0, min(len(req.Loads), maxLoadsChecked))
		for i := range req.Loads {
			if i >= maxLoadsChecked {
				break
			}
			if load := &req.Loads[i]; checkable(load) {
				checked = append(checked, load)
			}
		}
		if len(checked) == 0 {
			return found, nil
		}

		matches := make([][]pulid.ID, len(checked))
		items := make([]keel.BatchItem, len(checked))
		for i, load := range checked {
			items[i] = keel.IntoSlice(overlapQuery(req.CustomerID, load), &matches[i])
		}
		if err := keel.Batch(ctx, r.db.Keel(ctx), items...); err != nil {
			return nil, err
		}

		for i, ids := range matches {
			if len(ids) > 0 {
				found = append(found, triageports.OverlappingLoad{
					LoadIndex:  checked[i].Index,
					ShipmentID: ids[0],
				})
			}
		}

		return found, nil
	})
}

func checkable(load *triageports.TenderLoad) bool {
	return load.OriginCity != "" && load.OriginState != "" &&
		load.DestinationCity != "" && load.DestinationState != "" &&
		load.PickupFrom > 0 && load.PickupTo > load.PickupFrom
}

func overlapQuery(
	customerID pulid.ID,
	load *triageports.TenderLoad,
) *keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments

	pickup := stopAt(sp, "pick", stopAtRequest{
		types: []shipment.StopType{shipment.StopTypePickup, shipment.StopTypeSplitPickup},
		city:  load.OriginCity,
		state: load.OriginState,
		from:  load.PickupFrom,
		to:    load.PickupTo,
	})
	delivery := stopAt(sp, "drop", stopAtRequest{
		types: []shipment.StopType{shipment.StopTypeDelivery, shipment.StopTypeSplitDelivery},
		city:  load.DestinationCity,
		state: load.DestinationState,
	})

	return sp.Pick(sp.ID).
		Where(
			sp.CustomerID.Eq(customerID),
			sp.Status.Ne(shipment.StatusCanceled),
			keel.Exists(pickup),
			keel.Exists(delivery),
		).
		Order(sp.CreatedAt.Desc()).
		Limit(1)
}

type stopAtRequest struct {
	types []shipment.StopType
	city  string
	state string
	from  int64
	to    int64
}

func stopAt(
	sp *shipmentdb.ShipmentsTable,
	alias string,
	req stopAtRequest,
) *keel.Query[*shipment.ShipmentMove] {
	move := shipmentdb.ShipmentMoves.As(alias + "_sm")
	stop := shipmentdb.Stops.As(alias + "_stp")
	loc := locationdb.Locations.As(alias + "_loc")
	state := usstatedb.UsStates.As(alias + "_st")

	return move.
		Join(stop, stop.ShipmentMoveID.EqCol(move.ID)).
		Join(loc, loc.ID.EqCol(stop.LocationID)).
		Join(state, state.ID.EqCol(loc.StateID)).
		Where(
			move.ShipmentID.EqCol(sp.ID),
			stop.Type.In(req.types...),
			stop.Status.Ne(shipment.StopStatusCanceled),
			keel.DynFunc("lower", keel.DynFunc("btrim", loc.City)).
				Eq(strings.ToLower(strings.TrimSpace(req.city))),
			keel.DynFunc("upper", state.Abbreviation).Eq(strings.ToUpper(req.state)),
			keel.When(req.from > 0 && req.to > req.from, stop.ScheduledWindowStart.Gte(req.from)),
			keel.When(req.from > 0 && req.to > req.from, stop.ScheduledWindowStart.Lt(req.to)),
		)
}
