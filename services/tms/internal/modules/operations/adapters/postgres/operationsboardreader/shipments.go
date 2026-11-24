package operationsboardreader

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/operationsboard"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta/stopetadb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

var shipmentKind = recordkind.Kind(permission.ResourceShipment)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Shipments struct {
	db *postgres.Connection
}

type Lateness struct {
	db *postgres.Connection
}

type Revenue struct {
	db *postgres.Connection
}

func NewShipments(p Params) *Shipments {
	return &Shipments{db: p.DB}
}

func NewLateness(p Params) *Lateness {
	return &Lateness{db: p.DB}
}

func NewRevenue(p Params) *Revenue {
	return &Revenue{db: p.DB}
}

func (*Shipments) Kind() recordkind.Kind { return shipmentKind }

func (*Lateness) Kind() recordkind.Kind { return shipmentKind }

func (*Lateness) Measure() string { return operationsboard.MeasureLateness }

func (*Revenue) Kind() recordkind.Kind { return shipmentKind }

func (*Revenue) Measure() string { return operationsboard.MeasureRevenue }

type shipmentRow struct {
	ID      pulid.ID
	Label   string
	Status  string
	Revenue decimal.NullDecimal
}

type stopRow struct {
	ShipmentID   pulid.ID
	MoveSequence int64
	Sequence     int64
	Type         string
	Status       string
	WindowStart  int64
	WindowEnd    *int64
	Arrival      *int64
	Estimate     *int64
	City         string
	State        string
}

func text(e keel.Expr) keel.StringExpr {
	return keel.CastText(e).Coalesce("")
}

func (r *Shipments) Summaries(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Summary, error) {
	if len(req.IDs) == 0 {
		return map[pulid.ID]boardreading.Summary{}, nil
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Summary, error) {
			rows, err := readShipments(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}
			routes, err := readRoutes(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}
			out := make(map[pulid.ID]boardreading.Summary, len(rows))
			for _, row := range rows {
				status, tone := shipmentStatus(ctx, row.Status)
				out[row.ID] = boardreading.Summary{
					Label:      strings.TrimSpace(row.Label),
					Subtitle:   lane(routes[row.ID]),
					Status:     status,
					StatusTone: tone,
				}
			}

			return out, nil
		},
	)
}

func (r *Revenue) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	if len(req.IDs) == 0 {
		return map[pulid.ID]boardreading.Reading{}, nil
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Reading, error) {
			rows, err := readShipments(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}
			out := make(map[pulid.ID]boardreading.Reading, len(rows))
			for _, row := range rows {
				if !row.Revenue.Valid {
					continue
				}
				value := boardcatalog.Amount(row.Revenue.Decimal, "")
				out[row.ID] = boardreading.Reading{
					Value:    value,
					Standing: boardreading.Words{Text: slot, Value: &value},
				}
			}

			return out, nil
		},
	)
}

func (r *Lateness) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	if len(req.IDs) == 0 {
		return map[pulid.ID]boardreading.Reading{}, nil
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Reading, error) {
			routes, err := readRoutes(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}
			out := make(map[pulid.ID]boardreading.Reading, len(routes))
			for id, route := range routes {
				if reading, ok := lateness(ctx, route, req.Now); ok {
					out[id] = reading
				}
			}

			return out, nil
		},
	)
}

func readShipments(
	ctx context.Context,
	db *postgres.Connection,
	ids []pulid.ID,
) ([]shipmentRow, error) {
	sp := shipmentdb.Shipments

	return keel.Into[shipmentRow](ctx, db.Keel(ctx),
		sp.Pick(sp.ID, text(sp.ProNumber), text(sp.Status), sp.TotalChargeAmount).
			Where(sp.ID.In(ids...)))
}

func readRoutes(
	ctx context.Context,
	db *postgres.Connection,
	ids []pulid.ID,
) (map[pulid.ID][]stopRow, error) {
	stp := shipmentdb.Stops
	sm := shipmentdb.ShipmentMoves
	loc := locationdb.Locations
	st := usstatedb.UsStates
	eta := stopetadb.StopETAs

	sp := shipmentdb.Shipments
	stops, err := keel.Into[stopRow](ctx, db.Keel(ctx),
		stp.Join(sm, sm.ID.EqCol(stp.ShipmentMoveID)).
			Join(sp, sp.ID.EqCol(sm.ShipmentID)).
			LeftJoin(loc, loc.ID.EqCol(stp.LocationID)).
			LeftJoin(st, st.ID.EqCol(loc.StateID)).
			LeftJoin(eta, eta.StopID.EqCol(stp.ID)).
			Pick(
				sm.ShipmentID, sm.Sequence, stp.Sequence, text(stp.Type), text(stp.Status),
				stp.ScheduledWindowStart, stp.ScheduledWindowEnd, stp.ActualArrival,
				eta.EstimatedArrival, text(loc.City), text(st.Abbreviation),
			).
			Where(sm.ShipmentID.In(ids...), stp.Status.Ne(shipmentvocab.StopStatusCanceled)))
	if err != nil {
		return nil, err
	}
	routes := make(map[pulid.ID][]stopRow, len(ids))
	for idx := range stops {
		routes[stops[idx].ShipmentID] = append(routes[stops[idx].ShipmentID], stops[idx])
	}
	for id := range routes {
		slices.SortFunc(routes[id], func(a, b stopRow) int {
			return cmp.Or(
				cmp.Compare(a.MoveSequence, b.MoveSequence),
				cmp.Compare(a.Sequence, b.Sequence),
			)
		})
	}

	return routes, nil
}

func pickup(stop *stopRow) bool {
	return stop.Type == string(shipmentvocab.StopTypePickup) ||
		stop.Type == string(shipmentvocab.StopTypeSplitPickup)
}

func place(stop *stopRow) string {
	return strings.Join(slices.DeleteFunc([]string{
		strings.TrimSpace(stop.City), strings.TrimSpace(stop.State),
	}, func(part string) bool { return part == "" }), ", ")
}

func lane(route []stopRow) string {
	var from, to string
	for idx := range route {
		if pickup(&route[idx]) {
			if from == "" {
				from = place(&route[idx])
			}
			continue
		}
		to = place(&route[idx])
	}
	if from != "" && to != "" {
		return from + " → " + to
	}

	return cmp.Or(from, to)
}

func windowEnd(stop *stopRow) int64 {
	if stop.WindowEnd != nil && *stop.WindowEnd > 0 {
		return *stop.WindowEnd
	}

	return stop.WindowStart
}
