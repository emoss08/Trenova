package checkcallrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/carriercol"
	"github.com/emoss08/trenova/pkg/buncolgen/checkcallcol"
	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/trackingeventcol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultListLimit  = 200
	maxListLimit      = 500
	maxMonitoredMoves = 5000
	contactBatchSize  = 500
	lastContactColumn = "last_at"
	contactMoveColumn = "move_id"
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

func New(p Params) checkcallports.CheckCallRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.check-call-repository"),
	}
}

func (r *repository) Insert(
	ctx context.Context,
	entity *checkcall.CheckCall,
) (*checkcall.CheckCall, bool, error) {
	return dbtx.Write2(
		ctx,
		r.db,
		func(ctx context.Context) (*checkcall.CheckCall, bool, error) {
			result, err := r.db.DBForContext(ctx).NewInsert().
				Model(entity).
				On("CONFLICT (organization_id, business_unit_id, source, source_key) DO NOTHING").
				Exec(ctx)
			if err != nil {
				return nil, false, fmt.Errorf("insert check call: %w", err)
			}
			rows, err := result.RowsAffected()
			if err != nil {
				return nil, false, fmt.Errorf("insert check call rows affected: %w", err)
			}
			if rows > 0 {
				return entity, true, nil
			}

			cols := checkcallcol.CheckCallColumns
			existing := new(checkcall.CheckCall)
			err = r.db.DBForContext(ctx).NewSelect().
				Model(existing).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return checkcallcol.CheckCallScopeTenant(sq, pagination.TenantInfo{
						OrgID: entity.OrganizationID,
						BuID:  entity.BusinessUnitID,
					}).
						Where(cols.Source.Eq(), entity.Source).
						Where(cols.SourceKey.Eq(), entity.SourceKey)
				}).
				Scan(ctx)
			if err != nil {
				return nil, false, dberror.HandleNotFoundError(err, "Check call")
			}
			return existing, false, nil
		},
	)
}

func (r *repository) ListByShipment(
	ctx context.Context,
	req *checkcallports.ListCheckCallsByShipmentRequest,
) ([]*checkcall.CheckCall, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*checkcall.CheckCall, error) {
		cols := checkcallcol.CheckCallColumns
		entities := make([]*checkcall.CheckCall, 0)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return checkcallcol.CheckCallScopeTenant(sq, req.TenantInfo).
					Where(cols.ShipmentID.Eq(), req.ShipmentID)
			}).
			Order(cols.ReportedAt.OrderDesc(), cols.ID.OrderDesc()).
			Limit(int64(listLimit(req.Limit))).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list check calls by shipment: %w", err)
		}
		return entities, nil
	})
}

func (r *repository) LatestForMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (*checkcall.CheckCall, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*checkcall.CheckCall, error) {
		cols := checkcallcol.CheckCallColumns
		entities := make([]*checkcall.CheckCall, 0, 1)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return checkcallcol.CheckCallScopeTenant(sq, tenantInfo).
					Where(cols.ShipmentMoveID.Eq(), moveID)
			}).
			Order(cols.ReceivedAt.OrderDesc(), cols.ID.OrderDesc()).
			Limit(1).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("latest check call for move: %w", err)
		}
		if len(entities) == 0 {
			return nil, nil
		}
		return entities[0], nil
	})
}

func (r *repository) LatestForMoves(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveIDs []pulid.ID,
) (map[pulid.ID]*checkcall.CheckCall, error) {
	out := make(map[pulid.ID]*checkcall.CheckCall, len(moveIDs))
	if len(moveIDs) == 0 {
		return out, nil
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]*checkcall.CheckCall, error) {
			cols := checkcallcol.CheckCallColumns
			for start := 0; start < len(moveIDs); start += contactBatchSize {
				batch := moveIDs[start:min(start+contactBatchSize, len(moveIDs))]
				entities := make([]*checkcall.CheckCall, 0, len(batch))
				err := r.db.DBForContext(ctx).NewSelect().
					Model(&entities).
					DistinctOn(cols.ShipmentMoveID.Qualified()).
					WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
						return checkcallcol.CheckCallScopeTenant(sq, tenantInfo).
							Where(cols.ShipmentMoveID.In(), bun.List(batch))
					}).
					Order(
						cols.ShipmentMoveID.OrderAsc(),
						cols.ReportedAt.OrderDesc(),
						cols.ID.OrderDesc(),
					).
					Scan(ctx)
				if err != nil {
					return nil, fmt.Errorf("latest check calls for moves: %w", err)
				}
				for _, entity := range entities {
					out[entity.ShipmentMoveID] = entity
				}
			}
			return out, nil
		},
	)
}

type monitoredMoveRow struct {
	MoveID                pulid.ID `bun:"move_id"`
	ShipmentID            pulid.ID `bun:"shipment_id"`
	ProNumber             string   `bun:"pro_number"`
	OwnerID               pulid.ID `bun:"owner_id"`
	CarrierID             pulid.ID `bun:"carrier_id"`
	CarrierName           string   `bun:"carrier_name"`
	CustomerIntervalHours *int16   `bun:"customer_interval_hours"`
	ConfirmedAt           *int64   `bun:"confirmed_at"`
}

type contactRow struct {
	MoveID pulid.ID `bun:"move_id"`
	LastAt *int64   `bun:"last_at"`
}

func (r *repository) ListMonitoredMoves(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	limit int,
) ([]*checkcallports.MonitoredMove, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*checkcallports.MonitoredMove, error) {
		rows, err := r.listMonitoredRows(ctx, tenantInfo, limit)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return []*checkcallports.MonitoredMove{}, nil
		}

		moveIDs := make([]pulid.ID, 0, len(rows))
		for _, row := range rows {
			moveIDs = append(moveIDs, row.MoveID)
		}
		contacts, err := r.lastContacts(ctx, tenantInfo, moveIDs)
		if err != nil {
			return nil, err
		}

		moves := make([]*checkcallports.MonitoredMove, 0, len(rows))
		for _, row := range rows {
			last := contacts[row.MoveID]
			if row.ConfirmedAt != nil && *row.ConfirmedAt > last {
				last = *row.ConfirmedAt
			}
			moves = append(moves, &checkcallports.MonitoredMove{
				MoveID:                row.MoveID,
				ShipmentID:            row.ShipmentID,
				ProNumber:             row.ProNumber,
				OwnerID:               row.OwnerID,
				CarrierID:             row.CarrierID,
				CarrierName:           row.CarrierName,
				CustomerIntervalHours: row.CustomerIntervalHours,
				LastContactAt:         last,
			})
		}
		return moves, nil
	})
}

func (r *repository) listMonitoredRows(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	limit int,
) ([]monitoredMoveRow, error) {
	sm := shipmentcol.ShipmentMoveColumns
	sp := shipmentcol.ShipmentColumns
	ca := shipmentcol.CarrierAssignmentColumns
	car := carriercol.CarrierColumns
	cus := customercol.CustomerColumns
	spTable := shipmentcol.ShipmentTable
	caTable := shipmentcol.CarrierAssignmentTable
	carTable := carriercol.CarrierTable
	cusTable := customercol.CustomerTable

	if limit <= 0 || limit > maxMonitoredMoves {
		limit = maxMonitoredMoves
	}

	rows := make([]monitoredMoveRow, 0)
	err := r.db.DBForContext(ctx).NewSelect().
		Model((*shipment.ShipmentMove)(nil)).
		ColumnExpr(sm.ID.As("move_id")).
		ColumnExpr(sm.ShipmentID.As("shipment_id")).
		ColumnExpr(sp.ProNumber.As("pro_number")).
		ColumnExpr(sp.OwnerID.As("owner_id")).
		ColumnExpr(ca.CarrierID.As("carrier_id")).
		ColumnExpr(car.Name.Expr("COALESCE({}, '')")+" AS carrier_name").
		ColumnExpr(cus.CheckCallIntervalHours.As("customer_interval_hours")).
		ColumnExpr(ca.ConfirmedAt.As("confirmed_at")).
		Join("JOIN "+spTable.As(spTable.Alias)).
		JoinOn(sp.ID.EqColumn(sm.ShipmentID)).
		JoinOn(sp.OrganizationID.EqColumn(sm.OrganizationID)).
		JoinOn(sp.BusinessUnitID.EqColumn(sm.BusinessUnitID)).
		Join("JOIN "+caTable.As(caTable.Alias)).
		JoinOn(ca.ShipmentMoveID.EqColumn(sm.ID)).
		JoinOn(ca.OrganizationID.EqColumn(sm.OrganizationID)).
		JoinOn(ca.BusinessUnitID.EqColumn(sm.BusinessUnitID)).
		JoinOn(ca.Status.Ne(), shipment.CarrierAssignmentStatusCanceled).
		Join("LEFT JOIN "+carTable.As(carTable.Alias)).
		JoinOn(car.ID.EqColumn(ca.CarrierID)).
		JoinOn(car.OrganizationID.EqColumn(ca.OrganizationID)).
		JoinOn(car.BusinessUnitID.EqColumn(ca.BusinessUnitID)).
		Join("LEFT JOIN "+cusTable.As(cusTable.Alias)).
		JoinOn(cus.ID.EqColumn(sp.CustomerID)).
		JoinOn(cus.OrganizationID.EqColumn(sp.OrganizationID)).
		JoinOn(cus.BusinessUnitID.EqColumn(sp.BusinessUnitID)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return shipmentcol.ShipmentMoveScopeTenant(sq, tenantInfo).
				Where(sm.Status.Eq(), shipment.MoveStatusInTransit).
				Where(sm.CoverageType.Eq(), shipment.MoveCoverageTypeCarrier)
		}).
		Order(sm.ID.OrderAsc()).
		Limit(int64(limit)).
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("list monitored moves: %w", err)
	}
	return rows, nil
}

func (r *repository) lastContacts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveIDs []pulid.ID,
) (map[pulid.ID]int64, error) {
	contacts := make(map[pulid.ID]int64, len(moveIDs))
	for start := 0; start < len(moveIDs); start += contactBatchSize {
		end := min(start+contactBatchSize, len(moveIDs))
		batch := moveIDs[start:end]
		for _, read := range []func(context.Context, pagination.TenantInfo, []pulid.ID) ([]contactRow, error){
			r.lastCheckCalls,
			r.lastTrackingEvents,
			r.latestStopVisits,
		} {
			rows, err := read(ctx, tenantInfo, batch)
			if err != nil {
				return nil, err
			}
			for _, row := range rows {
				if row.LastAt != nil && *row.LastAt > contacts[row.MoveID] {
					contacts[row.MoveID] = *row.LastAt
				}
			}
		}
	}
	return contacts, nil
}

func (r *repository) lastCheckCalls(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveIDs []pulid.ID,
) ([]contactRow, error) {
	cols := checkcallcol.CheckCallColumns
	rows := make([]contactRow, 0, len(moveIDs))
	err := r.db.DBForContext(ctx).NewSelect().
		Model((*checkcall.CheckCall)(nil)).
		ColumnExpr(cols.ShipmentMoveID.As(contactMoveColumn)).
		ColumnExpr(buncolgen.Max(cols.ReceivedAt, lastContactColumn)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return checkcallcol.CheckCallScopeTenant(sq, tenantInfo).
				Where(cols.ShipmentMoveID.In(), bun.List(moveIDs))
		}).
		GroupExpr(cols.ShipmentMoveID.Qualified()).
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("last check calls: %w", err)
	}
	return rows, nil
}

func (r *repository) lastTrackingEvents(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveIDs []pulid.ID,
) ([]contactRow, error) {
	cols := trackingeventcol.TrackingEventColumns
	rows := make([]contactRow, 0, len(moveIDs))
	err := r.db.DBForContext(ctx).NewSelect().
		Model((*trackingevent.TrackingEvent)(nil)).
		ColumnExpr(cols.ShipmentMoveID.As(contactMoveColumn)).
		ColumnExpr(buncolgen.Max(cols.ReceivedAt, lastContactColumn)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return trackingeventcol.TrackingEventScopeTenant(sq, tenantInfo).
				Where(cols.ShipmentMoveID.In(), bun.List(moveIDs))
		}).
		GroupExpr(cols.ShipmentMoveID.Qualified()).
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("last tracking events: %w", err)
	}
	return rows, nil
}

func (r *repository) latestStopVisits(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveIDs []pulid.ID,
) ([]contactRow, error) {
	cols := shipmentcol.StopColumns
	rows := make([]contactRow, 0, len(moveIDs))
	err := r.db.DBForContext(ctx).NewSelect().
		Model((*shipment.Stop)(nil)).
		ColumnExpr(cols.ShipmentMoveID.As(contactMoveColumn)).
		ColumnExpr("MAX("+buncolgen.Expr(
			"GREATEST(COALESCE({0}, 0), COALESCE({1}, 0))",
			cols.ActualArrival,
			cols.ActualDeparture,
		)+") AS "+lastContactColumn).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return shipmentcol.StopScopeTenant(sq, tenantInfo).
				Where(cols.ShipmentMoveID.In(), bun.List(moveIDs))
		}).
		GroupExpr(cols.ShipmentMoveID.Qualified()).
		Scan(ctx, &rows)
	if err != nil {
		return nil, fmt.Errorf("latest stop visits: %w", err)
	}
	return rows, nil
}

func listLimit(limit int) int {
	if limit <= 0 {
		return defaultListLimit
	}
	return min(limit, maxListLimit)
}
