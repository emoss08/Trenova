package shipmentsearchcardrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentpredicate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type SearchCardParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type searchCardReader struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSearchCardReader(p SearchCardParams) shipmentports.ShipmentSearchCardReader {
	return &searchCardReader{db: p.DB, l: p.Logger.Named("postgres.shipment-search-card-reader")}
}

const (
	driverAssignAlias  = "drv_asg"
	driverMoveAlias    = "drv_sm"
	driverSeatAlias    = "drv_seat"
	driverWorkerAlias  = "drv"
	seatSourceName     = "seat"
	seatSourceWorkerID = "worker_id"
)

var liveShipmentStatuses = []shipment.Status{
	shipment.StatusNew,
	shipment.StatusPartiallyAssigned,
	shipment.StatusAssigned,
	shipment.StatusInTransit,
	shipment.StatusDelayed,
	shipment.StatusPartiallyCompleted,
}

var openMoveStatuses = []shipment.MoveStatus{
	shipment.MoveStatusNew,
	shipment.MoveStatusAssigned,
	shipment.MoveStatusInTransit,
}

func (r *searchCardReader) ListSearchCards(
	ctx context.Context,
	req *shipmentports.ListShipmentSearchCardsRequest,
) ([]*shipmentports.ShipmentSearchCard, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentports.ShipmentSearchCard, error) {
			if len(req.ShipmentIDs) == 0 {
				return nil, nil
			}

			sp := shipmentdb.Shipments
			origin := shipmentpredicate.FirstPickupPlace()
			dest := shipmentpredicate.LastDeliveryPlace()
			asg := shipmentdb.Assignments.As(driverAssignAlias)
			sm := shipmentdb.ShipmentMoves.As(driverMoveAlias)
			seat := workerdb.Workers.As(driverSeatAlias)
			drv := workerdb.Workers.As(driverWorkerAlias)
			driver := asg.Join(sm, sm.ID.EqCol(asg.ShipmentMoveID)).
				Join(seat, seat.ID.EqCol(asg.PrimaryWorkerID)).
				Where(
					sm.ShipmentID.EqCol(sp.ID),
					sm.Status.Ne(shipment.MoveStatusCanceled),
					asg.ArchivedAt.IsNull(),
					asg.Status.Ne(shipment.AssignmentStatusCanceled),
				).
				Order(keel.Asc(sm.Status.Eq(shipment.MoveStatusCompleted)), sm.Sequence.Asc()).
				Limit(1).
				Pick(asg.PrimaryWorkerID)

			q := dest.Join(origin.Join(sp.Query())).
				LeftJoinLateral(driver).
				LeftJoin(drv, drv.ID.EqCol(asg.PrimaryWorkerID)).
				Pick(
					sp.ID,
					sp.ProNumber.Coalesce(""),
					sp.BOL.Coalesce(""),
					sp.Status,
					sp.CustomerID.Coalesce(""),
					sp.Customer().Name.Coalesce(""),
					sp.Customer().Code.Coalesce(""),
					sp.ServiceType().Code.Coalesce(""),
					origin.City(),
					origin.StateCode(),
					dest.City(),
					dest.StateCode(),
					drv.ID.Coalesce(""),
					keel.ConcatWS(" ", drv.FirstName, drv.LastName).Coalesce(""),
				).
				Where(sp.ID.In(req.ShipmentIDs...))

			cards, err := keel.Into[shipmentports.ShipmentSearchCard](ctx, r.db.Keel(ctx), q)
			if err != nil {
				r.l.Error("failed to list shipment search cards", zap.Error(err))
				return nil, fmt.Errorf("list shipment search cards: %w", err)
			}

			return sliceutils.Pointers(cards), nil
		},
	)
}

func (r *searchCardReader) ListPartyLinks(
	ctx context.Context,
	req *shipmentports.ListShipmentPartyLinksRequest,
) ([]*shipmentports.ShipmentPartyLink, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentports.ShipmentPartyLink, error) {
			links := make([]*shipmentports.ShipmentPartyLink, 0)

			if len(req.CustomerIDs) > 0 && req.CustomerLimit > 0 {
				customerLinks, err := r.customerLinks(ctx, req)
				if err != nil {
					return nil, err
				}
				links = append(links, customerLinks...)
			}

			if len(req.WorkerIDs) > 0 {
				workerLinks, err := r.workerLinks(ctx, req)
				if err != nil {
					return nil, err
				}
				links = append(links, workerLinks...)
			}

			return links, nil
		},
	)
}

func (r *searchCardReader) customerLinks(
	ctx context.Context,
	req *shipmentports.ListShipmentPartyLinksRequest,
) ([]*shipmentports.ShipmentPartyLink, error) {
	sp := shipmentdb.Shipments
	links, err := keel.Into[shipmentports.ShipmentPartyLink](ctx, r.db.Keel(ctx), sp.
		Pick(sp.ID, sp.CustomerID, keel.Text("")).
		Where(
			sp.CustomerID.In(req.CustomerIDs...),
			sp.Status.Ne(shipment.StatusCanceled),
		).
		Order(keel.Desc(sp.Status.In(liveShipmentStatuses...)), sp.UpdatedAt.Desc()).
		Limit(req.CustomerLimit))
	if err != nil {
		r.l.Error("failed to list customer shipments for search", zap.Error(err))
		return nil, fmt.Errorf("list customer shipments for search: %w", err)
	}

	return sliceutils.Pointers(links), nil
}

func (r *searchCardReader) workerLinks(
	ctx context.Context,
	req *shipmentports.ListShipmentPartyLinksRequest,
) ([]*shipmentports.ShipmentPartyLink, error) {
	asg := shipmentdb.Assignments
	sm := shipmentdb.ShipmentMoves
	sp := shipmentdb.Shipments
	seat := keel.Rows(
		seatSourceName,
		[]string{seatSourceWorkerID},
		keel.ArrayOfString(asg.PrimaryWorkerID, asg.SecondaryWorkerID).Unnest(),
	)
	workerID := seat.Col(seatSourceWorkerID)

	links, err := keel.Into[shipmentports.ShipmentPartyLink](ctx, r.db.Keel(ctx), asg.
		Join(sm, sm.ID.EqCol(asg.ShipmentMoveID)).
		Join(sp, sp.ID.EqCol(sm.ShipmentID)).
		JoinLateral(seat).
		DistinctOn(workerID).
		Pick(sp.ID, keel.Text(""), workerID).
		Where(
			asg.ArchivedAt.IsNull(),
			asg.Status.Ne(shipment.AssignmentStatusCanceled),
			sm.Status.In(openMoveStatuses...),
			sp.Status.In(liveShipmentStatuses...),
			workerID.In(keel.Strings(req.WorkerIDs)),
		).
		Order(
			workerID.Asc(),
			keel.Desc(sm.Status.Eq(shipment.MoveStatusInTransit)),
			sm.Sequence.Asc(),
			sp.UpdatedAt.Desc(),
		))
	if err != nil {
		r.l.Error("failed to list worker loads for search", zap.Error(err))
		return nil, fmt.Errorf("list worker loads for search: %w", err)
	}

	return sliceutils.Pointers(links), nil
}
