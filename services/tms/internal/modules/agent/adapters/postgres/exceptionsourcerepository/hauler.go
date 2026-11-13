package exceptionsourcerepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/ports/haulerports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender/tenderdb"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// NewHaulers is the read a driver status request plans from.
func NewHaulers(p Params) haulerports.Reader {
	return &repository{db: p.DB}
}

type haulerMove struct {
	ID         pulid.ID
	ShipmentID pulid.ID
	Status     shipmentvocab.MoveStatus
}

// maxShipmentMoves bounds the read of one shipment's moves.
const maxShipmentMoves = 16

// Hauler reads who has the load: the move the request names, or the
// shipment's move under way, then its driver or carrier.
func (r *repository) Hauler(
	ctx context.Context,
	req *haulerports.HaulerRequest,
) (*haulerports.Hauler, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*haulerports.Hauler, error) {
		db := r.db.Keel(ctx)
		move, err := r.haulingMove(ctx, db, req)
		if err != nil || move == nil {
			return nil, err
		}

		sp := shipmentdb.Shipments
		pro, err := keel.IntoOne[string](ctx, db,
			sp.Pick(sp.ProNumber.Coalesce("")).Where(sp.ID.Eq(move.ShipmentID)).Limit(1))
		if err != nil && !errors.Is(err, keel.ErrNotFound) {
			return nil, fmt.Errorf("read the load's PRO number: %w", err)
		}

		haulers, err := r.haulers(ctx, map[pulid.ID]pulid.ID{move.ShipmentID: move.ID})
		if err != nil {
			return nil, err
		}

		return &haulerports.Hauler{
			ShipmentID: move.ShipmentID,
			ProNumber:  pro,
			Driver:     haulers.drivers[move.ID],
			Carrier:    haulers.carriers[move.ID],
		}, nil
	})
}

func (r *repository) haulingMove(
	ctx context.Context,
	db *keel.DB,
	req *haulerports.HaulerRequest,
) (*haulerMove, error) {
	mv := shipmentdb.ShipmentMoves
	moveID := req.MoveID
	if moveID.IsNil() && req.StopID.IsNotNil() {
		st := shipmentdb.Stops
		stopMove, err := keel.IntoOne[pulid.ID](ctx, db,
			st.Pick(st.ShipmentMoveID).Where(st.ID.Eq(req.StopID)).Limit(1))
		if err != nil && !errors.Is(err, keel.ErrNotFound) {
			return nil, fmt.Errorf("read the stop's move: %w", err)
		}
		moveID = stopMove
	}
	if moveID.IsNotNil() {
		move, err := keel.IntoOne[haulerMove](ctx, db,
			mv.Pick(mv.ID, mv.ShipmentID, mv.Status).Where(mv.ID.Eq(moveID)).Limit(1))
		if errors.Is(err, keel.ErrNotFound) {
			return nil, nil //nolint:nilnil // a move that is gone has no hauler
		}
		if err != nil {
			return nil, fmt.Errorf("read the move: %w", err)
		}

		return &move, nil
	}
	if req.ShipmentID.IsNil() {
		return nil, nil //nolint:nilnil // nothing names a load
	}

	moves, err := keel.Into[haulerMove](ctx, db,
		mv.Pick(mv.ID, mv.ShipmentID, mv.Status).
			Where(mv.ShipmentID.Eq(req.ShipmentID), mv.Status.Ne(shipmentvocab.MoveStatusCanceled)).
			Order(mv.Sequence.Asc()).
			Limit(maxShipmentMoves))
	if err != nil {
		return nil, fmt.Errorf("read the shipment's moves: %w", err)
	}

	return moveUnderWay(moves), nil
}

// moveUnderWay is the move a load is on now: the one in transit, else the
// first not yet finished, else the last one.
func moveUnderWay(moves []haulerMove) *haulerMove {
	if len(moves) == 0 {
		return nil
	}
	for i := range moves {
		if moves[i].Status == shipmentvocab.MoveStatusInTransit {
			return &moves[i]
		}
	}
	for i := range moves {
		if moves[i].Status != shipmentvocab.MoveStatusCompleted {
			return &moves[i]
		}
	}

	return &moves[len(moves)-1]
}

// TenderMove reads the move a tender offered to carriers.
func (r *repository) TenderMove(
	ctx context.Context,
	_ pagination.TenantInfo,
	tenderID pulid.ID,
) (pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (pulid.ID, error) {
		td := tenderdb.Tenders
		moveID, err := keel.IntoOne[pulid.ID](ctx, r.db.Keel(ctx),
			td.Pick(td.ShipmentMoveID).Where(td.ID.Eq(tenderID)).Limit(1))
		if errors.Is(err, keel.ErrNotFound) {
			return pulid.Nil, nil
		}
		if err != nil {
			return pulid.Nil, fmt.Errorf("read the tender's move: %w", err)
		}

		return moveID, nil
	})
}
