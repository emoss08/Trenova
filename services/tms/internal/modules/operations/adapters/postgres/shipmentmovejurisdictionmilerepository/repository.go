package shipmentmovejurisdictionmilerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	completedMovesCTE  = "mc"
	completedMoveID    = "id"
	completedMoveMiles = "miles"
	completedAt        = "completed_at"
	kilometersPerMile  = 1.609344
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) shipmentports.ShipmentMoveJurisdictionMileRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-move-jurisdiction-mile-repository"),
	}
}

func (r *repository) ReplaceForMove(ctx context.Context, move *shipment.ShipmentMove) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if move == nil || move.ID.IsNil() {
			return errortypes.NewBusinessError("Shipment move is required")
		}
		rows := make([]*shipment.ShipmentMoveJurisdictionMile, 0, len(move.JurisdictionMiles))
		for _, row := range move.JurisdictionMiles {
			if row == nil {
				continue
			}
			row.OrganizationID = move.OrganizationID
			row.BusinessUnitID = move.BusinessUnitID
			row.ShipmentMoveID = move.ID
			row.ShipmentID = move.ShipmentID
			rows = append(rows, row)
		}

		if err := r.replace(ctx, move.ID, rows); err != nil {
			r.l.Error("failed to replace jurisdiction miles",
				zap.String("moveId", move.ID.String()),
				zap.Error(err))
			return err
		}
		move.JurisdictionMiles = rows
		move.JurisdictionMilesDirty = false
		return nil
	})
}

func (r *repository) replace(
	ctx context.Context,
	moveID pulid.ID,
	rows []*shipment.ShipmentMoveJurisdictionMile,
) error {
	jm := shipmentdb.ShipmentMoveJurisdictionMiles
	if _, err := jm.Where(jm.ShipmentMoveID.Eq(moveID)).Delete(ctx, r.db.Keel(ctx)); err != nil {
		return fmt.Errorf("delete jurisdiction miles for move %s: %w", moveID, err)
	}
	if len(rows) == 0 {
		return nil
	}
	if err := jm.CreateMany(ctx, r.db.Keel(ctx), rows); err != nil {
		return fmt.Errorf("insert jurisdiction miles for move %s: %w", moveID, err)
	}
	return nil
}

func (r *repository) ListByMoveIDs(
	ctx context.Context,
	req shipmentports.ListJurisdictionMilesByMoveIDsRequest,
) ([]*shipment.ShipmentMoveJurisdictionMile, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipment.ShipmentMoveJurisdictionMile, error) {
			if len(req.MoveIDs) == 0 {
				return []*shipment.ShipmentMoveJurisdictionMile{}, nil
			}
			jm := shipmentdb.ShipmentMoveJurisdictionMiles
			entities, err := jm.Where(jm.ShipmentMoveID.In(req.MoveIDs...)).
				Order(jm.ShipmentMoveID.Asc(), jm.Sequence.Asc(), jm.JurisdictionCode.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list jurisdiction miles by move", zap.Error(err))
				return nil, fmt.Errorf("list jurisdiction miles by move: %w", err)
			}
			return entities, nil
		},
	)
}

type unattributedTotals struct {
	TotalMoves int64
	TotalMiles decimal.Decimal
}

func (r *repository) ListUnattributedMoves(
	ctx context.Context,
	req shipmentports.UnattributedMovesRequest,
) (*shipmentports.UnattributedMovesPage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*shipmentports.UnattributedMovesPage, error) {
			if req.Start >= req.End {
				return nil, errortypes.NewBusinessError("Period start must be before period end")
			}

			mc := completedMovesWithoutBreakdown()
			inPeriod := []keel.Pred{
				mc.Col(completedAt).Gte(req.Start),
				mc.Col(completedAt).Lt(req.End),
			}

			totals, err := keel.IntoOne[unattributedTotals](ctx, r.db.Keel(ctx), mc.Query().
				With(mc).
				Pick(
					keel.Count(),
					keel.CastNumeric(keel.NumericOf(mc.Col(completedMoveMiles).Sum()).Coalesce(0)).
						RoundTo(2),
				).
				Where(inPeriod...))
			if err != nil {
				r.l.Error("failed to count unattributed moves", zap.Error(err))
				return nil, fmt.Errorf("count unattributed moves: %w", err)
			}

			page := &shipmentports.UnattributedMovesPage{
				MoveIDs:    make([]pulid.ID, 0, req.PageSize()),
				TotalMoves: int(totals.TotalMoves),
				TotalMiles: totals.TotalMiles,
			}
			if totals.TotalMoves == 0 || req.Offset >= page.TotalMoves {
				return page, nil
			}

			ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), mc.Query().
				With(mc).
				Pick(mc.Col(completedMoveID)).
				Where(inPeriod...).
				Order(mc.Col(completedMoveID).Asc()).
				Limit(req.PageSize()).
				Offset(req.Offset))
			if err != nil {
				r.l.Error("failed to list unattributed moves", zap.Error(err))
				return nil, fmt.Errorf("list unattributed moves: %w", err)
			}
			page.MoveIDs = append(page.MoveIDs, ids...)
			return page, nil
		},
	)
}

func completedMovesWithoutBreakdown() *keel.CTETable {
	sm := shipmentdb.ShipmentMoves
	st := shipmentdb.Stops
	jm := shipmentdb.ShipmentMoveJurisdictionMiles

	body := sm.LeftJoin(st, st.ShipmentMoveID.EqCol(sm.ID)).
		Where(
			sm.Status.Eq(shipment.MoveStatusCompleted),
			sm.Distance.Gt(0),
			keel.NotExists(jm.Where(jm.ShipmentMoveID.EqCol(sm.ID))),
		).
		Group(sm.ID, sm.OrganizationID, sm.BusinessUnitID, sm.Distance, sm.DistanceUnits, sm.UpdatedAt).
		Pick(
			sm.ID,
			keel.CaseFloat64().
				When(
					sm.DistanceUnits.Eq(shipment.JurisdictionDistanceUnitsKilometers),
					sm.Distance.Div(keel.Float64(kilometersPerMile)),
				).
				Else(sm.Distance),
			keel.Int64Of(st.ActualDeparture.Coalesce(st.ActualArrival).Max()).
				Coalesce(sm.UpdatedAt),
		)

	return keel.CTE(
		completedMovesCTE,
		[]string{completedMoveID, completedMoveMiles, completedAt},
		body,
	)
}
