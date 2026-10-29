package agentmemorysubjectrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
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

func New(p Params) agentmemoryports.AgentMemorySubjectRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentmemorysubject-repository"),
	}
}

type linkRow struct {
	FromID   pulid.ID
	LinkedID *pulid.ID
}

type pairRow struct {
	FromID pulid.ID
	First  *pulid.ID
	Second *pulid.ID
}

type tripleRow struct {
	FromID pulid.ID
	First  *pulid.ID
	Second *pulid.ID
	Third  *pulid.ID
}

type ownerRow struct {
	FromID       pulid.ID
	ResourceType string
	ResourceID   string
}

func (r *repository) ListRecordLinks(
	ctx context.Context,
	req *agentmemoryports.ListMemoryRecordLinksRequest,
) ([]agentmemoryports.MemoryRecordLink, error) {
	if req == nil {
		return nil, errors.New("memory record links request is required")
	}
	if len(req.IDs) == 0 {
		return []agentmemoryports.MemoryRecordLink{}, nil
	}

	var (
		links []agentmemoryports.MemoryRecordLink
		err   error
	)
	switch req.Kind {
	case agentmemory.MemoryRecordShipment:
		links, err = r.shipmentLinks(ctx, req.IDs)
	case agentmemory.MemoryRecordShipmentMove:
		links, err = r.moveLinks(ctx, req.IDs)
	case agentmemory.MemoryRecordInvoice:
		links, err = r.invoiceLinks(ctx, req.IDs)
	case agentmemory.MemoryRecordBillingQueueItem:
		links, err = r.billingQueueLinks(ctx, req.IDs)
	case agentmemory.MemoryRecordInboundMessage:
		links, err = r.inboundMessageLinks(ctx, req.IDs)
	case agentmemory.MemoryRecordDocument:
		links, err = r.documentLinks(ctx, req.IDs)
	case agentmemory.MemoryRecordCustomer,
		agentmemory.MemoryRecordLocation,
		agentmemory.MemoryRecordWorker,
		agentmemory.MemoryRecordCarrier:
		return []agentmemoryports.MemoryRecordLink{}, nil
	default:
		return []agentmemoryports.MemoryRecordLink{}, nil
	}
	if err != nil {
		r.l.Error("failed to read the records a memory subject names",
			zap.String("kind", string(req.Kind)),
			zap.Error(err),
		)

		return nil, fmt.Errorf("list %s memory record links: %w", req.Kind, err)
	}

	return links, nil
}

func (r *repository) shipmentLinks(
	ctx context.Context,
	ids []pulid.ID,
) ([]agentmemoryports.MemoryRecordLink, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]agentmemoryports.MemoryRecordLink, error) {
			sp := shipmentdb.Shipments
			stp := shipmentdb.Stops
			stpMove := shipmentdb.ShipmentMoves.As("stop_move")
			asn := shipmentdb.Assignments
			asnMove := shipmentdb.ShipmentMoves.As("assignment_move")
			casn := shipmentdb.CarrierAssignments
			casnMove := shipmentdb.ShipmentMoves.As("carrier_move")

			var (
				customers []pairRow
				stops     []linkRow
				workers   []pairRow
				carriers  []linkRow
			)
			err := keel.Batch(ctx, r.db.Keel(ctx),
				keel.IntoSlice(sp.
					Pick(sp.ID, sp.CustomerID, sp.BillToCustomerID).
					Where(sp.ID.In(ids...)), &customers),
				keel.IntoSlice(stp.
					Pick(stpMove.ShipmentID, stp.LocationID).
					Join(stpMove, stpMove.ID.EqCol(stp.ShipmentMoveID)).
					Where(stpMove.ShipmentID.In(ids...)).
					Order(stpMove.Sequence.Asc(), stp.Sequence.Asc()), &stops),
				keel.IntoSlice(asn.
					Pick(asnMove.ShipmentID, asn.PrimaryWorkerID, asn.SecondaryWorkerID).
					Join(asnMove, asnMove.ID.EqCol(asn.ShipmentMoveID)).
					Where(
						asnMove.ShipmentID.In(ids...),
						asn.Status.Ne(shipmentvocab.AssignmentStatusCanceled),
					).
					Order(asnMove.Sequence.Asc()), &workers),
				keel.IntoSlice(casn.
					Pick(casnMove.ShipmentID, casn.CarrierID).
					Join(casnMove, casnMove.ID.EqCol(casn.ShipmentMoveID)).
					Where(
						casnMove.ShipmentID.In(ids...),
						casn.Status.Ne(shipmentvocab.CarrierAssignmentStatusCanceled),
					).
					Order(casnMove.Sequence.Asc()), &carriers),
			)
			if err != nil {
				return nil, fmt.Errorf("shipment links: %w", err)
			}

			links := make(
				[]agentmemoryports.MemoryRecordLink,
				0,
				len(customers)*2+len(stops)+len(workers)*2+len(carriers),
			)
			links = appendPairs(links, customers, agentmemory.MemoryRecordCustomer)
			links = appendLinks(links, stops, agentmemory.MemoryRecordLocation)
			links = appendPairs(links, workers, agentmemory.MemoryRecordWorker)

			return appendLinks(links, carriers, agentmemory.MemoryRecordCarrier), nil
		},
	)
}

func (r *repository) moveLinks(
	ctx context.Context,
	ids []pulid.ID,
) ([]agentmemoryports.MemoryRecordLink, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]agentmemoryports.MemoryRecordLink, error) {
			sm := shipmentdb.ShipmentMoves
			rows, err := keel.Into[linkRow](ctx, r.db.Keel(ctx), sm.
				Pick(sm.ID, sm.ShipmentID).
				Where(sm.ID.In(ids...)))
			if err != nil {
				return nil, fmt.Errorf("move shipments: %w", err)
			}

			return appendLinks(
				make([]agentmemoryports.MemoryRecordLink, 0, len(rows)),
				rows,
				agentmemory.MemoryRecordShipment,
			), nil
		},
	)
}

func (r *repository) invoiceLinks(
	ctx context.Context,
	ids []pulid.ID,
) ([]agentmemoryports.MemoryRecordLink, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]agentmemoryports.MemoryRecordLink, error) {
			inv := invoicedb.Invoices
			rows, err := keel.Into[linkRow](ctx, r.db.Keel(ctx), inv.
				Pick(inv.ID, inv.CustomerID).
				Where(inv.ID.In(ids...)))
			if err != nil {
				return nil, fmt.Errorf("invoice customers: %w", err)
			}

			return appendLinks(
				make([]agentmemoryports.MemoryRecordLink, 0, len(rows)),
				rows,
				agentmemory.MemoryRecordCustomer,
			), nil
		},
	)
}

func (r *repository) billingQueueLinks(
	ctx context.Context,
	ids []pulid.ID,
) ([]agentmemoryports.MemoryRecordLink, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]agentmemoryports.MemoryRecordLink, error) {
			bqi := billingqueuedb.BillingQueueItems
			rows, err := keel.Into[linkRow](ctx, r.db.Keel(ctx), bqi.
				Pick(bqi.ID, bqi.BillToCustomerID).
				Where(bqi.ID.In(ids...)))
			if err != nil {
				return nil, fmt.Errorf("billing queue item customers: %w", err)
			}

			return appendLinks(
				make([]agentmemoryports.MemoryRecordLink, 0, len(rows)),
				rows,
				agentmemory.MemoryRecordCustomer,
			), nil
		},
	)
}

func (r *repository) inboundMessageLinks(
	ctx context.Context,
	ids []pulid.ID,
) ([]agentmemoryports.MemoryRecordLink, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]agentmemoryports.MemoryRecordLink, error) {
			imsg := inboundmessagedb.InboundMessages
			rows, err := keel.Into[tripleRow](ctx, r.db.Keel(ctx), imsg.
				Pick(
					imsg.ID,
					imsg.MatchedCustomerID,
					imsg.MatchedCarrierID,
					imsg.MatchedShipmentID,
				).
				Where(imsg.ID.In(ids...)))
			if err != nil {
				return nil, fmt.Errorf("inbound message matches: %w", err)
			}

			links := make([]agentmemoryports.MemoryRecordLink, 0, len(rows)*3)
			for _, row := range rows {
				links = appendLink(links, row.FromID, agentmemory.MemoryRecordCustomer, row.First)
				links = appendLink(links, row.FromID, agentmemory.MemoryRecordCarrier, row.Second)
				links = appendLink(links, row.FromID, agentmemory.MemoryRecordShipment, row.Third)
			}

			return links, nil
		},
	)
}

func (r *repository) documentLinks(
	ctx context.Context,
	ids []pulid.ID,
) ([]agentmemoryports.MemoryRecordLink, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]agentmemoryports.MemoryRecordLink, error) {
			doc := documentdb.Documents
			rows, err := keel.Into[ownerRow](ctx, r.db.Keel(ctx), doc.
				Pick(doc.ID, doc.ResourceType, doc.ResourceID).
				Where(doc.ID.In(ids...)))
			if err != nil {
				return nil, fmt.Errorf("document owners: %w", err)
			}

			links := make([]agentmemoryports.MemoryRecordLink, 0, len(rows))
			for _, row := range rows {
				owner, ok := agentmemory.MemoryRecordRefOf(row.ResourceType, row.ResourceID)
				if !ok || owner.Kind == agentmemory.MemoryRecordDocument {
					continue
				}
				links = append(links, agentmemoryports.MemoryRecordLink{
					From: row.FromID,
					Kind: owner.Kind,
					ID:   owner.ID,
				})
			}

			return links, nil
		},
	)
}

func appendLink(
	links []agentmemoryports.MemoryRecordLink,
	from pulid.ID,
	kind agentmemory.MemoryRecordKind,
	id *pulid.ID,
) []agentmemoryports.MemoryRecordLink {
	if id == nil || id.IsNil() {
		return links
	}

	return append(links, agentmemoryports.MemoryRecordLink{From: from, Kind: kind, ID: *id})
}

func appendLinks(
	links []agentmemoryports.MemoryRecordLink,
	rows []linkRow,
	kind agentmemory.MemoryRecordKind,
) []agentmemoryports.MemoryRecordLink {
	for _, row := range rows {
		links = appendLink(links, row.FromID, kind, row.LinkedID)
	}

	return links
}

func appendPairs(
	links []agentmemoryports.MemoryRecordLink,
	rows []pairRow,
	kind agentmemory.MemoryRecordKind,
) []agentmemoryports.MemoryRecordLink {
	for _, row := range rows {
		links = appendLink(links, row.FromID, kind, row.First)
		links = appendLink(links, row.FromID, kind, row.Second)
	}

	return links
}
