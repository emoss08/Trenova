package checkcallrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall/checkcalldb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent/trackingeventdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultListLimit  = 200
	maxListLimit      = 500
	maxMonitoredMoves = 5000
	contactBatchSize  = 500
)

var checkCallErrors = dberror.Mapping{Entity: "Check call"}

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
			cc := checkcalldb.CheckCalls
			db := r.db.Keel(ctx)
			inserted, err := cc.CreateOrSkip(
				ctx,
				db,
				[]*checkcall.CheckCall{entity},
				cc.OrganizationID,
				cc.BusinessUnitID,
				cc.Source,
				cc.SourceKey,
			)
			if err != nil {
				return nil, false, fmt.Errorf(
					"insert check call: %w",
					checkCallErrors.Map(err, entity.ID.String()),
				)
			}
			if inserted > 0 {
				return entity, true, nil
			}

			existing, err := cc.Where(
				cc.Source.Eq(entity.Source),
				cc.SourceKey.Eq(entity.SourceKey),
			).One(ctx, db)
			if err != nil {
				return nil, false, checkCallErrors.Map(err, entity.SourceKey)
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
		cc := checkcalldb.CheckCalls
		entities, err := cc.Where(cc.ShipmentID.Eq(req.ShipmentID)).
			Order(cc.ReportedAt.Desc(), cc.ID.Desc()).
			Limit(listLimit(req.Limit)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list check calls by shipment: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) LatestForMove(
	ctx context.Context,
	_ pagination.TenantInfo,
	moveID pulid.ID,
) (*checkcall.CheckCall, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*checkcall.CheckCall, error) {
		cc := checkcalldb.CheckCalls
		entity, err := cc.Where(cc.ShipmentMoveID.Eq(moveID)).
			Order(cc.ReceivedAt.Desc(), cc.ID.Desc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("latest check call for move: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) LatestForMoves(
	ctx context.Context,
	_ pagination.TenantInfo,
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
			cc := checkcalldb.CheckCalls
			for start := 0; start < len(moveIDs); start += contactBatchSize {
				batch := moveIDs[start:min(start+contactBatchSize, len(moveIDs))]
				entities, err := cc.DistinctOn(cc.ShipmentMoveID).
					Where(cc.ShipmentMoveID.In(batch...)).
					Order(cc.ShipmentMoveID.Asc(), cc.ReportedAt.Desc(), cc.ID.Desc()).
					All(ctx, r.db.Keel(ctx))
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
	MoveID                pulid.ID
	ShipmentID            pulid.ID
	ProNumber             string
	OwnerID               pulid.ID
	CarrierID             pulid.ID
	CarrierName           string
	CustomerIntervalHours *int16
	ConfirmedAt           *int64
}

type contactRow struct {
	MoveID pulid.ID
	LastAt *int64
}

func (r *repository) ListMonitoredMoves(
	ctx context.Context,
	_ pagination.TenantInfo,
	limit int,
) ([]*checkcallports.MonitoredMove, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*checkcallports.MonitoredMove, error) {
		db := r.db.Keel(ctx)
		rows, err := listMonitoredRows(ctx, db, limit)
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
		contacts, err := lastContacts(ctx, db, moveIDs)
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

func listMonitoredRows(ctx context.Context, db *keel.DB, limit int) ([]monitoredMoveRow, error) {
	if limit <= 0 || limit > maxMonitoredMoves {
		limit = maxMonitoredMoves
	}

	sm := shipmentdb.ShipmentMoves
	sp := shipmentdb.Shipments
	ca := shipmentdb.CarrierAssignments
	car := carrierdb.Carriers
	cus := customerdb.Customers

	rows, err := keel.Into[monitoredMoveRow](ctx, db, sm.
		Join(sp, sp.ID.EqCol(sm.ShipmentID)).
		Join(ca, ca.ShipmentMoveID.EqCol(sm.ID), ca.Status.Ne(shipment.CarrierAssignmentStatusCanceled)).
		LeftJoin(car, car.ID.EqCol(ca.CarrierID)).
		LeftJoin(cus, cus.ID.EqCol(sp.CustomerID)).
		Pick(
			sm.ID,
			sm.ShipmentID,
			sp.ProNumber,
			keel.DynOf(sp.OwnerID).Coalesce(""),
			ca.CarrierID,
			keel.DynOf(car.Name).Coalesce(""),
			cus.CheckCallIntervalHours,
			ca.ConfirmedAt,
		).
		Where(
			sm.Status.Eq(shipment.MoveStatusInTransit),
			sm.CoverageType.Eq(shipment.MoveCoverageTypeCarrier),
		).
		Order(sm.ID.Asc()).
		Limit(limit))
	if err != nil {
		return nil, fmt.Errorf("list monitored moves: %w", err)
	}

	return rows, nil
}

func lastContacts(
	ctx context.Context,
	db *keel.DB,
	moveIDs []pulid.ID,
) (map[pulid.ID]int64, error) {
	cc := checkcalldb.CheckCalls
	te := trackingeventdb.TrackingEvents
	st := shipmentdb.Stops

	contacts := make(map[pulid.ID]int64, len(moveIDs))
	for start := 0; start < len(moveIDs); start += contactBatchSize {
		batch := moveIDs[start:min(start+contactBatchSize, len(moveIDs))]
		var calls, events, visits []contactRow
		err := keel.Batch(ctx, db,
			keel.IntoSlice(cc.Group(cc.ShipmentMoveID).
				Pick(cc.ShipmentMoveID, cc.ReceivedAt.Max()).
				Where(cc.ShipmentMoveID.In(batch...)), &calls),
			keel.IntoSlice(te.Group(te.ShipmentMoveID).
				Pick(te.ShipmentMoveID, te.ReceivedAt.Max()).
				Where(te.ShipmentMoveID.In(batch...)), &events),
			keel.IntoSlice(st.Group(st.ShipmentMoveID).
				Pick(st.ShipmentMoveID, keel.DynFunc(
					"greatest",
					keel.DynOf(st.ActualArrival).Coalesce(0),
					keel.DynOf(st.ActualDeparture).Coalesce(0),
				).Max()).
				Where(st.ShipmentMoveID.In(batch...)), &visits),
		)
		if err != nil {
			return nil, fmt.Errorf("last contacts: %w", err)
		}

		for _, rows := range [][]contactRow{calls, events, visits} {
			for _, row := range rows {
				if row.LastAt != nil && *row.LastAt > contacts[row.MoveID] {
					contacts[row.MoveID] = *row.LastAt
				}
			}
		}
	}

	return contacts, nil
}

func listLimit(limit int) int {
	if limit <= 0 {
		return defaultListLimit
	}
	return min(limit, maxListLimit)
}
