package agentmemorysubjectrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
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

func New(p Params) agentports.AgentMemorySubjectRepository {
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
	req *agentports.ListMemoryRecordLinksRequest,
) ([]agentports.MemoryRecordLink, error) {
	if req == nil {
		return nil, errors.New("memory record links request is required")
	}
	if len(req.IDs) == 0 {
		return []agentports.MemoryRecordLink{}, nil
	}

	var (
		links []agentports.MemoryRecordLink
		err   error
	)
	switch req.Kind {
	case agent.MemoryRecordShipment:
		links, err = r.shipmentLinks(ctx, req.IDs)
	case agent.MemoryRecordShipmentMove:
		links, err = r.moveLinks(ctx, req.IDs)
	case agent.MemoryRecordInvoice:
		links, err = r.invoiceLinks(ctx, req.IDs)
	case agent.MemoryRecordBillingQueueItem:
		links, err = r.billingQueueLinks(ctx, req.IDs)
	case agent.MemoryRecordInboundMessage:
		links, err = r.inboundMessageLinks(ctx, req.IDs)
	case agent.MemoryRecordDocument:
		links, err = r.documentLinks(ctx, req.IDs)
	case agent.MemoryRecordCustomer,
		agent.MemoryRecordLocation,
		agent.MemoryRecordWorker,
		agent.MemoryRecordCarrier:
		return []agentports.MemoryRecordLink{}, nil
	default:
		return []agentports.MemoryRecordLink{}, nil
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
) ([]agentports.MemoryRecordLink, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]agentports.MemoryRecordLink, error) {
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
					asn.Status.Ne(shipment.AssignmentStatusCanceled),
				).
				Order(asnMove.Sequence.Asc()), &workers),
			keel.IntoSlice(casn.
				Pick(casnMove.ShipmentID, casn.CarrierID).
				Join(casnMove, casnMove.ID.EqCol(casn.ShipmentMoveID)).
				Where(
					casnMove.ShipmentID.In(ids...),
					casn.Status.Ne(shipment.CarrierAssignmentStatusCanceled),
				).
				Order(casnMove.Sequence.Asc()), &carriers),
		)
		if err != nil {
			return nil, fmt.Errorf("shipment links: %w", err)
		}

		links := make(
			[]agentports.MemoryRecordLink,
			0,
			len(customers)*2+len(stops)+len(workers)*2+len(carriers),
		)
		links = appendPairs(links, customers, agent.MemoryRecordCustomer)
		links = appendLinks(links, stops, agent.MemoryRecordLocation)
		links = appendPairs(links, workers, agent.MemoryRecordWorker)

		return appendLinks(links, carriers, agent.MemoryRecordCarrier), nil
	})
}

func (r *repository) moveLinks(
	ctx context.Context,
	ids []pulid.ID,
) ([]agentports.MemoryRecordLink, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]agentports.MemoryRecordLink, error) {
		sm := shipmentdb.ShipmentMoves
		rows, err := keel.Into[linkRow](ctx, r.db.Keel(ctx), sm.
			Pick(sm.ID, sm.ShipmentID).
			Where(sm.ID.In(ids...)))
		if err != nil {
			return nil, fmt.Errorf("move shipments: %w", err)
		}

		return appendLinks(
			make([]agentports.MemoryRecordLink, 0, len(rows)),
			rows,
			agent.MemoryRecordShipment,
		), nil
	})
}

func (r *repository) invoiceLinks(
	ctx context.Context,
	ids []pulid.ID,
) ([]agentports.MemoryRecordLink, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]agentports.MemoryRecordLink, error) {
		inv := invoicedb.Invoices
		rows, err := keel.Into[linkRow](ctx, r.db.Keel(ctx), inv.
			Pick(inv.ID, inv.CustomerID).
			Where(inv.ID.In(ids...)))
		if err != nil {
			return nil, fmt.Errorf("invoice customers: %w", err)
		}

		return appendLinks(
			make([]agentports.MemoryRecordLink, 0, len(rows)),
			rows,
			agent.MemoryRecordCustomer,
		), nil
	})
}

func (r *repository) billingQueueLinks(
	ctx context.Context,
	ids []pulid.ID,
) ([]agentports.MemoryRecordLink, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]agentports.MemoryRecordLink, error) {
		bqi := billingqueuedb.BillingQueueItems
		rows, err := keel.Into[linkRow](ctx, r.db.Keel(ctx), bqi.
			Pick(bqi.ID, bqi.BillToCustomerID).
			Where(bqi.ID.In(ids...)))
		if err != nil {
			return nil, fmt.Errorf("billing queue item customers: %w", err)
		}

		return appendLinks(
			make([]agentports.MemoryRecordLink, 0, len(rows)),
			rows,
			agent.MemoryRecordCustomer,
		), nil
	})
}

func (r *repository) inboundMessageLinks(
	ctx context.Context,
	ids []pulid.ID,
) ([]agentports.MemoryRecordLink, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]agentports.MemoryRecordLink, error) {
		imsg := inboundmessagedb.InboundMessages
		rows, err := keel.Into[tripleRow](ctx, r.db.Keel(ctx), imsg.
			Pick(imsg.ID, imsg.MatchedCustomerID, imsg.MatchedCarrierID, imsg.MatchedShipmentID).
			Where(imsg.ID.In(ids...)))
		if err != nil {
			return nil, fmt.Errorf("inbound message matches: %w", err)
		}

		links := make([]agentports.MemoryRecordLink, 0, len(rows)*3)
		for _, row := range rows {
			links = appendLink(links, row.FromID, agent.MemoryRecordCustomer, row.First)
			links = appendLink(links, row.FromID, agent.MemoryRecordCarrier, row.Second)
			links = appendLink(links, row.FromID, agent.MemoryRecordShipment, row.Third)
		}

		return links, nil
	})
}

func (r *repository) documentLinks(
	ctx context.Context,
	ids []pulid.ID,
) ([]agentports.MemoryRecordLink, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]agentports.MemoryRecordLink, error) {
		doc := documentdb.Documents
		rows, err := keel.Into[ownerRow](ctx, r.db.Keel(ctx), doc.
			Pick(doc.ID, doc.ResourceType, doc.ResourceID).
			Where(doc.ID.In(ids...)))
		if err != nil {
			return nil, fmt.Errorf("document owners: %w", err)
		}

		links := make([]agentports.MemoryRecordLink, 0, len(rows))
		for _, row := range rows {
			owner, ok := agent.MemoryRecordRefOf(row.ResourceType, row.ResourceID)
			if !ok || owner.Kind == agent.MemoryRecordDocument {
				continue
			}
			links = append(links, agentports.MemoryRecordLink{
				From: row.FromID,
				Kind: owner.Kind,
				ID:   owner.ID,
			})
		}

		return links, nil
	})
}

func appendLink(
	links []agentports.MemoryRecordLink,
	from pulid.ID,
	kind agent.MemoryRecordKind,
	id *pulid.ID,
) []agentports.MemoryRecordLink {
	if id == nil || id.IsNil() {
		return links
	}

	return append(links, agentports.MemoryRecordLink{From: from, Kind: kind, ID: *id})
}

func appendLinks(
	links []agentports.MemoryRecordLink,
	rows []linkRow,
	kind agent.MemoryRecordKind,
) []agentports.MemoryRecordLink {
	for _, row := range rows {
		links = appendLink(links, row.FromID, kind, row.LinkedID)
	}

	return links
}

func appendPairs(
	links []agentports.MemoryRecordLink,
	rows []pairRow,
	kind agent.MemoryRecordKind,
) []agentports.MemoryRecordLink {
	for _, row := range rows {
		links = appendLink(links, row.FromID, kind, row.First)
		links = appendLink(links, row.FromID, kind, row.Second)
	}

	return links
}
