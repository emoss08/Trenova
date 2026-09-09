package shipmentmovejurisdictionmilerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	completedMovesCTE      = "mc"
	completedMoveIDCol     = "mc.id"
	completedMoveMilesCol  = "mc.miles"
	completedAtCol         = "mc.completed_at"
	kilometersPerMileLabel = "1.609344"
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

func New(p Params) repositories.ShipmentMoveJurisdictionMileRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.shipment-move-jurisdiction-mile-repository"),
	}
}

func (r *repository) ReplaceForMove(ctx context.Context, move *shipment.ShipmentMove) error {
	if move == nil || move.ID.IsNil() {
		return errortypes.NewBusinessError("Shipment move is required")
	}
	tenantInfo := pagination.TenantInfo{
		OrgID: move.OrganizationID,
		BuID:  move.BusinessUnitID,
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

	cols := buncolgen.ShipmentMoveJurisdictionMileColumns
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().
			Model((*shipment.ShipmentMoveJurisdictionMile)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.ShipmentMoveJurisdictionMileScopeTenantDelete(dq, tenantInfo).
					Where(cols.ShipmentMoveID.Eq(), move.ID)
			}).
			Exec(c); err != nil {
			return fmt.Errorf("delete jurisdiction miles for move %s: %w", move.ID, err)
		}
		if len(rows) == 0 {
			return nil
		}
		if _, err := tx.NewInsert().Model(&rows).Exec(c); err != nil {
			return fmt.Errorf("insert jurisdiction miles for move %s: %w", move.ID, err)
		}
		return nil
	})
	if err != nil {
		r.l.Error("failed to replace jurisdiction miles",
			zap.String("moveId", move.ID.String()),
			zap.Error(err))
		return err
	}
	move.JurisdictionMiles = rows
	move.JurisdictionMilesDirty = false
	return nil
}

func (r *repository) ListByMoveIDs(
	ctx context.Context,
	req repositories.ListJurisdictionMilesByMoveIDsRequest,
) ([]*shipment.ShipmentMoveJurisdictionMile, error) {
	entities := make([]*shipment.ShipmentMoveJurisdictionMile, 0, len(req.MoveIDs)*4)
	if len(req.MoveIDs) == 0 {
		return entities, nil
	}
	cols := buncolgen.ShipmentMoveJurisdictionMileColumns
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShipmentMoveJurisdictionMileScopeTenant(sq, req.TenantInfo).
				Where(cols.ShipmentMoveID.In(), bun.List(req.MoveIDs))
		}).
		Order(cols.ShipmentMoveID.OrderAsc()).
		Order(cols.Sequence.OrderAsc()).
		Order(cols.JurisdictionCode.OrderAsc()).
		Scan(ctx); err != nil {
		r.l.Error("failed to list jurisdiction miles by move", zap.Error(err))
		return nil, fmt.Errorf("list jurisdiction miles by move: %w", err)
	}
	return entities, nil
}

func (r *repository) ListUnattributedMoves(
	ctx context.Context,
	req repositories.UnattributedMovesRequest,
) (*repositories.UnattributedMovesPage, error) {
	if req.Start >= req.End {
		return nil, errortypes.NewBusinessError("Period start must be before period end")
	}
	dba := r.db.DBForContext(ctx)

	var totals struct {
		TotalMoves int             `bun:"total_moves"`
		TotalMiles decimal.Decimal `bun:"total_miles"`
	}
	if err := dba.NewSelect().
		With(completedMovesCTE, r.completedMovesWithoutBreakdown(dba, req.TenantInfo)).
		TableExpr(completedMovesCTE).
		ColumnExpr("COUNT(*) AS total_moves").
		ColumnExpr("ROUND(COALESCE(SUM(" + completedMoveMilesCol + "), 0)::numeric, 2) AS total_miles").
		Where(completedAtCol+" >= ?", req.Start).
		Where(completedAtCol+" < ?", req.End).
		Scan(ctx, &totals); err != nil {
		r.l.Error("failed to count unattributed moves", zap.Error(err))
		return nil, fmt.Errorf("count unattributed moves: %w", err)
	}

	page := &repositories.UnattributedMovesPage{
		MoveIDs:    make([]pulid.ID, 0, req.PageSize()),
		TotalMoves: totals.TotalMoves,
		TotalMiles: totals.TotalMiles,
	}
	if totals.TotalMoves == 0 || req.Offset >= totals.TotalMoves {
		return page, nil
	}

	if err := dba.NewSelect().
		With(completedMovesCTE, r.completedMovesWithoutBreakdown(dba, req.TenantInfo)).
		TableExpr(completedMovesCTE).
		ColumnExpr(completedMoveIDCol).
		Where(completedAtCol+" >= ?", req.Start).
		Where(completedAtCol+" < ?", req.End).
		OrderExpr(completedMoveIDCol + " ASC").
		Limit(req.PageSize()).
		Offset(req.Offset).
		Scan(ctx, &page.MoveIDs); err != nil {
		r.l.Error("failed to list unattributed moves", zap.Error(err))
		return nil, fmt.Errorf("list unattributed moves: %w", err)
	}
	return page, nil
}

func (r *repository) completedMovesWithoutBreakdown(
	dba bun.IDB,
	tenantInfo pagination.TenantInfo,
) *bun.SelectQuery {
	sm := buncolgen.ShipmentMoveColumns
	st := buncolgen.StopColumns
	jm := buncolgen.ShipmentMoveJurisdictionMileColumns

	breakdownExists := dba.NewSelect().
		Model((*shipment.ShipmentMoveJurisdictionMile)(nil)).
		ColumnExpr("1").
		Where(jm.ShipmentMoveID.EqColumn(sm.ID)).
		Where(jm.OrganizationID.EqColumn(sm.OrganizationID)).
		Where(jm.BusinessUnitID.EqColumn(sm.BusinessUnitID))

	return dba.NewSelect().
		Model((*shipment.ShipmentMove)(nil)).
		ColumnExpr(sm.ID.As("id")).
		ColumnExpr(buncolgen.Expr(
			"CASE WHEN {0} = ? THEN {1} / "+kilometersPerMileLabel+" ELSE {1} END AS miles",
			sm.DistanceUnits,
			sm.Distance,
		), shipment.JurisdictionDistanceUnitsKilometers).
		ColumnExpr(buncolgen.Expr(
			"COALESCE(MAX(COALESCE({0}, {1})), {2}) AS completed_at",
			st.ActualDeparture,
			st.ActualArrival,
			sm.UpdatedAt,
		)).
		Join("LEFT JOIN "+buncolgen.StopTable.As(buncolgen.StopTable.Alias)+
			" ON "+st.ShipmentMoveID.EqColumn(sm.ID)+
			" AND "+st.OrganizationID.EqColumn(sm.OrganizationID)+
			" AND "+st.BusinessUnitID.EqColumn(sm.BusinessUnitID)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShipmentMoveScopeTenant(sq, tenantInfo).
				Where(sm.Status.Eq(), shipment.MoveStatusCompleted).
				Where(sm.Distance.Gt(), 0).
				Where("NOT EXISTS (?)", breakdownExists)
		}).
		GroupExpr(sm.ID.Qualified() + ", " + sm.OrganizationID.Qualified() + ", " +
			sm.BusinessUnitID.Qualified() + ", " + sm.Distance.Qualified() + ", " +
			sm.DistanceUnits.Qualified() + ", " + sm.UpdatedAt.Qualified())
}
