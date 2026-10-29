package iftarepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta/iftadb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/iftaports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	kilometersPerMile    = 1.609344
	milesPrecision       = 3
	mismatchFloorMiles   = 1
	mismatchTolerance    = 0.02
	inputMovesCTE        = "inp"
	jurisdictionMilesCTE = "jm"
)

type moveMiles struct {
	MoveID pulid.ID
	Miles  decimal.Decimal
}

type accumulationSource struct {
	inp *keel.CTETable
	jm  *keel.CTETable
}

func inMiles(distance keel.Float64Col, units keel.StringCol) keel.Float64Expr {
	return keel.CaseFloat64().
		When(
			units.Eq(shipmentvocab.JurisdictionDistanceUnitsKilometers),
			distance.Div(keel.Float64(kilometersPerMile)),
		).
		Else(distance)
}

func roundMiles(e keel.Expr) keel.NumericExpr {
	return keel.CastNumeric(keel.NumericOf(e)).RoundTo(milesPrecision)
}

func newAccumulationSource(req *iftaports.AccumulateMilesRequest) accumulationSource {
	sm := shipmentdb.ShipmentMoves
	stp := shipmentdb.Stops
	smjm := shipmentdb.ShipmentMoveJurisdictionMiles
	me := iftadb.JurisdictionMileageEntries

	completedAt := keel.Int64Of(stp.ActualDeparture.Coalesce(stp.ActualArrival).Max()).
		Coalesce(sm.UpdatedAt)

	inp := keel.CTE(
		inputMovesCTE,
		[]string{"id", "loaded", "distance", "completed_at"},
		sm.LeftJoin(stp, stp.ShipmentMoveID.EqCol(sm.ID)).
			Pick(sm.ID, sm.Loaded, inMiles(sm.Distance, sm.DistanceUnits), completedAt).
			Where(sm.Status.Eq(shipmentvocab.MoveStatusCompleted)).
			Group(sm.ID, sm.Loaded, sm.Distance, sm.DistanceUnits, sm.UpdatedAt).
			Having(completedAt.Gte(req.Start), completedAt.Lt(req.End)),
	)

	jm := keel.CTE(
		jurisdictionMilesCTE,
		[]string{
			"shipment_move_id",
			"country_code",
			"jurisdiction_code",
			"loaded",
			"move_distance",
			"miles",
		},
		smjm.Join(inp, inp.Col("id").EqCol(smjm.ShipmentMoveID)).
			Pick(
				smjm.ShipmentMoveID,
				smjm.CountryCode,
				smjm.JurisdictionCode,
				inp.Col("loaded"),
				inp.Col("distance"),
				inMiles(smjm.Distance, smjm.DistanceUnits),
			).
			Where(keel.NotExists(me.Where(me.ShipmentMoveID.EqCol(smjm.ShipmentMoveID)))),
	)

	return accumulationSource{inp: inp, jm: jm}
}

func (s accumulationSource) routeRows() *keel.Query[*keel.Row] {
	jm := s.jm
	a := shipmentdb.Assignments
	ifj := iftadb.Jurisdictions
	miles := jm.Col("miles").Sum()
	loaded := keel.BoolOf(jm.Col("loaded"))

	return jm.Query().
		With(s.inp, jm).
		LeftJoin(a, a.ShipmentMoveID.EqCol(jm.Col("shipment_move_id")), a.ArchivedAt.IsNull()).
		LeftJoin(ifj, ifj.CountryCode.EqCol(jm.Col("country_code")), ifj.Code.EqCol(jm.Col("jurisdiction_code"))).
		Pick(
			a.TractorID,
			jm.Col("country_code"),
			jm.Col("jurisdiction_code"),
			ifj.ID,
			roundMiles(keel.NumericOf(miles).Coalesce(0)),
			roundMiles(keel.NumericOf(miles.Filter(loaded.IsTrue())).Coalesce(0)),
			roundMiles(keel.NumericOf(miles.Filter(loaded.IsFalse())).Coalesce(0)),
			jm.Col("shipment_move_id").CountDistinct(),
		).
		Group(a.TractorID, jm.Col("country_code"), jm.Col("jurisdiction_code"), ifj.ID).
		Order(
			a.TractorID.Asc().NullsFirst(),
			jm.Col("country_code").Asc(),
			jm.Col("jurisdiction_code").Asc(),
		)
}

func (s accumulationSource) noTractor() *keel.Query[*keel.Row] {
	jm := s.jm
	a := shipmentdb.Assignments

	return jm.Query().
		With(s.inp, jm).
		LeftJoin(a, a.ShipmentMoveID.EqCol(jm.Col("shipment_move_id")), a.ArchivedAt.IsNull()).
		Pick(jm.Col("shipment_move_id"), roundMiles(keel.NumericOf(jm.Col("miles").Sum()).Coalesce(0))).
		Where(a.TractorID.IsNull()).
		Group(jm.Col("shipment_move_id")).
		Order(jm.Col("shipment_move_id").Asc())
}

func (s accumulationSource) unattributed() *keel.Query[*keel.Row] {
	inp := s.inp
	smjm := shipmentdb.ShipmentMoveJurisdictionMiles
	me := iftadb.JurisdictionMileageEntries

	return inp.Query().
		With(inp).
		Pick(inp.Col("id"), roundMiles(inp.Col("distance"))).
		Where(
			inp.Col("distance").Gt(0),
			keel.NotExists(smjm.Where(smjm.ShipmentMoveID.EqCol(inp.Col("id")))),
			keel.NotExists(me.Where(me.ShipmentMoveID.EqCol(inp.Col("id")))),
		).
		Order(inp.Col("completed_at").Asc(), inp.Col("id").Asc())
}

func (s accumulationSource) mismatches() *keel.Query[*keel.Row] {
	jm := s.jm
	moveDistance := keel.Float64Of(jm.Col("move_distance"))
	attributed := keel.Float64Of(jm.Col("miles").Sum())

	return jm.Query().
		With(s.inp, jm).
		Pick(jm.Col("shipment_move_id"), roundMiles(moveDistance), roundMiles(attributed)).
		Where(moveDistance.IsNotNull()).
		Group(jm.Col("shipment_move_id"), jm.Col("move_distance")).
		Having(
			attributed.Sub(moveDistance).Abs().GtExpr(
				keel.Float64(mismatchFloorMiles).
					Greatest(moveDistance.Mul(keel.Float64(mismatchTolerance))),
			),
		).
		Order(jm.Col("shipment_move_id").Asc())
}

func (r *repository) AccumulateMiles(
	ctx context.Context,
	req *iftaports.AccumulateMilesRequest,
) (*iftaports.MileAccumulation, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*iftaports.MileAccumulation, error) {
		db := r.db.Keel(ctx)
		src := newAccumulationSource(req)

		routeRows, err := keel.Into[iftaports.MileRow](ctx, db, src.routeRows())
		if err != nil {
			r.l.Error("failed to accumulate route miles", zap.Error(err))
			return nil, fmt.Errorf("accumulate route miles: %w", err)
		}

		noTractor, err := keel.Into[moveMiles](ctx, db, src.noTractor())
		if err != nil {
			r.l.Error("failed to accumulate no-tractor miles", zap.Error(err))
			return nil, fmt.Errorf("accumulate no-tractor miles: %w", err)
		}

		unattributed, err := keel.Into[moveMiles](ctx, db, src.unattributed())
		if err != nil {
			r.l.Error("failed to list unattributed moves", zap.Error(err))
			return nil, fmt.Errorf("list unattributed moves: %w", err)
		}

		mismatches, err := keel.Into[iftaports.MoveMismatch](ctx, db, src.mismatches())
		if err != nil {
			r.l.Error("failed to detect jurisdiction mileage mismatches", zap.Error(err))
			return nil, fmt.Errorf("detect mileage mismatches: %w", err)
		}

		manual, err := r.manualRows(ctx, req)
		if err != nil {
			return nil, err
		}

		out := &iftaports.MileAccumulation{
			RouteRows:  nonNil(sliceutils.Pointers(routeRows)),
			ManualRows: manual,
			Mismatches: nonNil(mismatches),
		}
		for i := range noTractor {
			out.NoTractor.Add(noTractor[i].MoveID, noTractor[i].Miles)
		}
		for i := range unattributed {
			out.Unattributed.Add(unattributed[i].MoveID, unattributed[i].Miles)
		}

		return out, nil
	})
}

func (r *repository) manualRows(
	ctx context.Context,
	req *iftaports.AccumulateMilesRequest,
) ([]*iftaports.MileRow, error) {
	me := iftadb.JurisdictionMileageEntries
	j := iftadb.Jurisdictions
	rows, err := keel.Into[iftaports.MileRow](
		ctx,
		r.db.Keel(ctx),
		me.Join(j, j.ID.EqCol(me.JurisdictionID)).
			Pick(
				me.TractorID,
				j.CountryCode,
				j.Code,
				me.JurisdictionID,
				keel.NumericOf(me.Miles.Sum()).Coalesce(0),
				keel.NumericOf(me.Miles.Sum().Filter(me.Loaded.IsTrue())).Coalesce(0),
				keel.NumericOf(me.Miles.Sum().Filter(me.Loaded.IsFalse())).Coalesce(0),
				keel.Count(),
			).
			Where(
				me.Year.Eq(int64(req.Year)),
				me.Quarter.Eq(int64(req.Quarter)),
			).
			Group(me.TractorID, j.CountryCode, j.Code, me.JurisdictionID).
			Order(me.TractorID.Asc(), j.CountryCode.Asc(), j.Code.Asc()),
	)
	if err != nil {
		r.l.Error("failed to accumulate manual miles", zap.Error(err))
		return nil, fmt.Errorf("accumulate manual miles: %w", err)
	}

	out := make([]*iftaports.MileRow, len(rows))
	for i := range rows {
		out[i] = &rows[i]
	}

	return out, nil
}

func nonNil[T any](items []T) []T {
	if items == nil {
		return []T{}
	}

	return items
}
