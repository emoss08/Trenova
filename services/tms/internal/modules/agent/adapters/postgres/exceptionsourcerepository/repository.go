package exceptionsourcerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender/tenderdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/fx"
)

const (
	defaultLimit = 500
	maxLimit     = 2000

	newerTenderAlias     = "newer_tnd"
	latestStopAlias      = "last_stp"
	latestMoveAlias      = "last_smv"
	shipmentResourceType = "shipment"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) watchtowerports.ExceptionSourceReader {
	return &repository{db: p.DB}
}

func boundedLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	return min(limit, maxLimit)
}

func openShipmentStatuses() []shipmentvocab.Status {
	return []shipmentvocab.Status{
		shipmentvocab.StatusNew,
		shipmentvocab.StatusPartiallyAssigned,
		shipmentvocab.StatusAssigned,
		shipmentvocab.StatusInTransit,
		shipmentvocab.StatusDelayed,
		shipmentvocab.StatusPartiallyCompleted,
	}
}

func deliveredShipmentStatuses() []shipmentvocab.Status {
	return []shipmentvocab.Status{
		shipmentvocab.StatusPartiallyCompleted,
		shipmentvocab.StatusCompleted,
		shipmentvocab.StatusReadyToInvoice,
	}
}

func missedStopsQuery(req *watchtowerports.ListMissedStopsRequest) *keel.Query[*shipment.Stop] {
	stp := shipmentdb.Stops
	sm := shipmentdb.ShipmentMoves
	sp := shipmentdb.Shipments
	loc := locationdb.Locations
	ust := loc.State()
	cutoff := stp.ScheduledWindowEnd.Coalesce(stp.ScheduledWindowStart)

	return stp.Join(sm, sm.ID.EqCol(stp.ShipmentMoveID)).
		Join(sp, sp.ID.EqCol(sm.ShipmentID)).
		LeftJoin(loc, loc.ID.EqCol(stp.LocationID)).
		Pick(
			stp.ID,
			sm.ID,
			sp.ID,
			sp.ProNumber,
			sp.OwnerID,
			stp.Type,
			cutoff,
			loc.Name.Coalesce(""),
			loc.City.Coalesce(""),
			ust.Abbreviation.Coalesce(""),
		).
		Where(
			stp.Type.In(req.StopTypes...),
			stp.ActualArrival.IsNull(),
			stp.Status.NotIn(shipmentvocab.StopStatusCompleted, shipmentvocab.StopStatusCanceled),
			sm.Status.Ne(shipmentvocab.MoveStatusCanceled),
			sp.Status.In(openShipmentStatuses()...),
			cutoff.Lt(req.ClosedBefore),
			keel.When(len(req.ShipmentIDs) > 0, sp.ID.In(req.ShipmentIDs...)),
		).
		Order(cutoff.Asc(), stp.ID.Asc()).
		Limit(boundedLimit(req.Limit))
}

func lastDeliveryQuery(sp *shipmentdb.ShipmentsTable) *keel.Query[*shipment.Stop] {
	stp := shipmentdb.Stops.As(latestStopAlias)
	sm := shipmentdb.ShipmentMoves.As(latestMoveAlias)

	return stp.Pick(stp.ActualArrival).
		Join(sm, sm.ID.EqCol(stp.ShipmentMoveID)).
		Where(
			sm.ShipmentID.EqCol(sp.ID),
			sm.Status.Ne(shipmentvocab.MoveStatusCanceled),
			stp.Type.In(shipmentvocab.StopTypeDelivery, shipmentvocab.StopTypeSplitDelivery),
			stp.Status.Ne(shipmentvocab.StopStatusCanceled),
			stp.ActualArrival.IsNotNull(),
		).
		Order(sm.Sequence.Desc(), stp.Sequence.Desc()).
		Limit(1)
}

func podOnFile(sp *shipmentdb.ShipmentsTable) keel.Pred {
	doc := documentdb.Documents

	return keel.Exists(doc.Where(
		doc.ResourceType.Eq(shipmentResourceType),
		doc.ResourceID.EqCol(sp.ID),
		doc.IsCurrentVersion.IsTrue(),
		doc.Status.NotIn(document.StatusRejected, document.StatusArchived),
		doc.DocumentType().Code.Eq(documenttype.CodePOD),
	))
}

func missingPODsQuery(req *watchtowerports.ListMissingPODsRequest) *keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments
	delivered := shipmentdb.Stops.As(latestStopAlias).ActualArrival

	return sp.JoinLateral(lastDeliveryQuery(sp)).
		Pick(
			sp.ID,
			sp.ProNumber,
			sp.OwnerID,
			sp.Customer().Name.Coalesce(""),
			delivered,
		).
		Where(
			sp.Status.In(deliveredShipmentStatuses()...),
			delivered.Lt(req.DeliveredBefore),
			delivered.Gte(req.DeliveredAfter),
			keel.Not(podOnFile(sp)),
			keel.When(len(req.ShipmentIDs) > 0, sp.ID.In(req.ShipmentIDs...)),
		).
		Order(delivered.Asc(), sp.ID.Asc()).
		Limit(boundedLimit(req.Limit))
}

func exhaustedTendersQuery(
	req *watchtowerports.ListExhaustedTendersRequest,
) *keel.Query[*tender.Tender] {
	tc := tenderdb.Tenders
	sm := shipmentdb.ShipmentMoves
	sp := shipmentdb.Shipments
	ca := shipmentdb.CarrierAssignments
	newer := tenderdb.Tenders.As(newerTenderAlias)

	newerTender := newer.Where(
		newer.ShipmentMoveID.EqCol(tc.ShipmentMoveID),
		keel.Tuple(newer.CreatedAt, newer.ID).Gt(keel.Tuple(tc.CreatedAt, tc.ID)),
	)
	covered := ca.Where(
		ca.ShipmentMoveID.EqCol(sm.ID),
		ca.Status.Ne(shipmentvocab.CarrierAssignmentStatusCanceled),
	)

	return tc.Join(sm, sm.ID.EqCol(tc.ShipmentMoveID)).
		Join(sp, sp.ID.EqCol(sm.ShipmentID)).
		Pick(
			tc.ID,
			sm.ID,
			sp.ID,
			sp.ProNumber,
			sp.OwnerID,
			tc.ExhaustedAt.Coalesce(tc.UpdatedAt),
		).
		Where(
			tc.Status.Eq(tender.StatusExhausted),
			sm.Status.Eq(shipmentvocab.MoveStatusNew),
			sp.Status.In(openShipmentStatuses()...),
			keel.NotExists(newerTender),
			keel.NotExists(covered),
			keel.When(len(req.ShipmentIDs) > 0, sp.ID.In(req.ShipmentIDs...)),
		).
		Order(tc.CreatedAt.Asc(), tc.ID.Asc()).
		Limit(boundedLimit(req.Limit))
}

func (r *repository) ListMissedStops(
	ctx context.Context,
	req *watchtowerports.ListMissedStopsRequest,
) ([]*watchtowerports.MissedStop, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*watchtowerports.MissedStop, error) {
		rows, err := keel.Into[watchtowerports.MissedStop](
			ctx,
			r.db.Keel(ctx),
			missedStopsQuery(req),
		)
		if err != nil {
			return nil, fmt.Errorf("list missed stops: %w", err)
		}
		return sliceutils.Pointers(rows), nil
	})
}

func (r *repository) ListMissingPODs(
	ctx context.Context,
	req *watchtowerports.ListMissingPODsRequest,
) ([]*watchtowerports.MissingPOD, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*watchtowerports.MissingPOD, error) {
		rows, err := keel.Into[watchtowerports.MissingPOD](
			ctx,
			r.db.Keel(ctx),
			missingPODsQuery(req),
		)
		if err != nil {
			return nil, fmt.Errorf("list shipments missing a pod: %w", err)
		}
		return sliceutils.Pointers(rows), nil
	})
}

func (r *repository) ListExhaustedTenders(
	ctx context.Context,
	req *watchtowerports.ListExhaustedTendersRequest,
) ([]*watchtowerports.ExhaustedTender, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*watchtowerports.ExhaustedTender, error) {
			rows, err := keel.Into[watchtowerports.ExhaustedTender](
				ctx,
				r.db.Keel(ctx),
				exhaustedTendersQuery(req),
			)
			if err != nil {
				return nil, fmt.Errorf("list exhausted tenders: %w", err)
			}
			return sliceutils.Pointers(rows), nil
		},
	)
}
