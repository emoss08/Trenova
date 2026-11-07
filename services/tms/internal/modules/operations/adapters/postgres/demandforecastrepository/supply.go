package demandforecastrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/driverstart"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const maxDriverStarts = 10_000

type bookedRow struct {
	Latitude  float64
	Longitude float64
	SlotStart int64
	Total     int64
	Open      int64
}

func (r *repository) ListBooked(
	ctx context.Context,
	req *demandforecastports.BookedRequest,
) ([]demandforecastports.BookedSlot, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]demandforecastports.BookedSlot, error) {
			l := loadsBetween(req.From, req.Until)
			loc := locationdb.Locations.As("bk_loc")
			slotStart := l.start.Sub(
				l.sinceMonday().Mod(keel.Int64(dispatchmarket.SlotSeconds)),
			)
			open := keel.Count().Filter(l.sm.Status.Eq(shipmentvocab.MoveStatusNew))

			rows, err := keel.Into[bookedRow](ctx, r.db.Keel(ctx), l.q.
				Join(loc, loc.ID.EqCol(l.st.LocationID)).
				Pick(loc.Latitude, loc.Longitude, slotStart, keel.Count(), open).
				Where(loc.Latitude.IsNotNull(), loc.Longitude.IsNotNull()).
				Group(loc.Latitude, loc.Longitude, slotStart))
			if err != nil {
				return nil, fmt.Errorf("read booked loads: %w", err)
			}

			out := make([]demandforecastports.BookedSlot, 0, len(rows))
			for _, row := range rows {
				out = append(out, demandforecastports.BookedSlot(row))
			}

			return out, nil
		},
	)
}

type startRow struct {
	WorkerID  pulid.ID
	Latitude  *float64
	Longitude *float64
	FreeAt    int64
}

func (r *repository) ListDriverStarts(
	ctx context.Context,
	req *demandforecastports.DriverStartsRequest,
) ([]demandforecastports.DriverStart, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]demandforecastports.DriverStart, error) {
			w := workerdb.Workers
			wt := tractordb.Tractors.As("ds_trac")
			primary := wt.
				Pick(wt.ID).
				Where(wt.PrimaryWorkerID.EqCol(w.ID)).
				Order(keel.Desc(wt.Status.Eq(domaintypes.EquipmentStatusAvailable)), wt.Code.Asc()).
				Limit(1)

			q, start := driverstart.Join(w.LeftJoinLateral(primary), w, wt, req.Now)
			rows, err := keel.Into[startRow](ctx, r.db.Keel(ctx), q.
				Pick(w.ID, start.Latitude, start.Longitude, start.FreeAt).
				Where(
					w.Status.Eq(domaintypes.StatusActive),
					w.CanBeAssigned.IsTrue(),
					w.AvailableForDispatch.IsTrue(),
				).
				Order(w.ID.Asc()).
				Limit(maxDriverStarts))
			if err != nil {
				return nil, fmt.Errorf("read driver starts: %w", err)
			}

			out := make([]demandforecastports.DriverStart, 0, len(rows))
			for _, row := range rows {
				if row.Latitude == nil || row.Longitude == nil {
					continue
				}
				out = append(out, demandforecastports.DriverStart{
					WorkerID:  row.WorkerID,
					Latitude:  *row.Latitude,
					Longitude: *row.Longitude,
					FreeAt:    row.FreeAt,
				})
			}

			return out, nil
		},
	)
}

type customerNameRow struct {
	ID   pulid.ID
	Name string
}

func (r *repository) CustomerNames(
	ctx context.Context,
	req *demandforecastports.CustomerNamesRequest,
) (map[pulid.ID]string, error) {
	if len(req.CustomerIDs) == 0 {
		return map[pulid.ID]string{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		c := customerdb.Customers
		rows, err := keel.Into[customerNameRow](ctx, r.db.Keel(ctx), c.
			Pick(c.ID, c.Name).
			Where(c.ID.In(req.CustomerIDs...)).
			Limit(len(req.CustomerIDs)))
		if err != nil {
			return nil, fmt.Errorf("read customer names: %w", err)
		}
		names := make(map[pulid.ID]string, len(rows))
		for _, row := range rows {
			names[row.ID] = row.Name
		}

		return names, nil
	})
}
