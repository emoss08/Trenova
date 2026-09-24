package agentmemorysubjectrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
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

func New(p Params) repositories.AgentMemorySubjectRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentmemorysubject-repository"),
	}
}

type linkRow struct {
	FromID   pulid.ID  `bun:"from_id"`
	LinkedID *pulid.ID `bun:"linked_id"`
}

type pairRow struct {
	FromID pulid.ID  `bun:"from_id"`
	First  *pulid.ID `bun:"first_id"`
	Second *pulid.ID `bun:"second_id"`
}

type tripleRow struct {
	FromID pulid.ID  `bun:"from_id"`
	First  *pulid.ID `bun:"first_id"`
	Second *pulid.ID `bun:"second_id"`
	Third  *pulid.ID `bun:"third_id"`
}

type ownerRow struct {
	FromID       pulid.ID `bun:"from_id"`
	ResourceType string   `bun:"resource_type"`
	ResourceID   string   `bun:"resource_id"`
}

func (r *repository) ListRecordLinks(
	ctx context.Context,
	req repositories.ListMemoryRecordLinksRequest,
) ([]repositories.MemoryRecordLink, error) {
	if len(req.IDs) == 0 {
		return []repositories.MemoryRecordLink{}, nil
	}

	var (
		links []repositories.MemoryRecordLink
		err   error
	)
	switch req.Kind {
	case agent.MemoryRecordShipment:
		links, err = r.shipmentLinks(ctx, req.TenantInfo, req.IDs)
	case agent.MemoryRecordShipmentMove:
		links, err = r.moveLinks(ctx, req.TenantInfo, req.IDs)
	case agent.MemoryRecordInvoice:
		links, err = r.invoiceLinks(ctx, req.TenantInfo, req.IDs)
	case agent.MemoryRecordBillingQueueItem:
		links, err = r.billingQueueLinks(ctx, req.TenantInfo, req.IDs)
	case agent.MemoryRecordInboundMessage:
		links, err = r.inboundMessageLinks(ctx, req.TenantInfo, req.IDs)
	case agent.MemoryRecordDocument:
		links, err = r.documentLinks(ctx, req.TenantInfo, req.IDs)
	default:
		return []repositories.MemoryRecordLink{}, nil
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
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) ([]repositories.MemoryRecordLink, error) {
	sp := buncolgen.ShipmentColumns
	sm := buncolgen.ShipmentMoveColumns
	stp := buncolgen.StopColumns
	asn := buncolgen.AssignmentColumns
	casn := buncolgen.CarrierAssignmentColumns
	dba := r.db.DBForContext(ctx)

	customers := make([]pairRow, 0, len(ids))
	if err := dba.NewSelect().
		Model((*shipment.Shipment)(nil)).
		ColumnExpr(sp.ID.As("from_id")).
		ColumnExpr(sp.CustomerID.As("first_id")).
		ColumnExpr(sp.BillToCustomerID.As("second_id")).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShipmentScopeTenant(sq, tenant).Where(sp.ID.In(), bun.List(ids))
		}).
		Scan(ctx, &customers); err != nil {
		return nil, fmt.Errorf("shipment customers: %w", err)
	}

	stops := make([]linkRow, 0, len(ids)*2)
	if err := dba.NewSelect().
		Model((*shipment.Stop)(nil)).
		ColumnExpr(sm.ShipmentID.As("from_id")).
		ColumnExpr(stp.LocationID.As("linked_id")).
		Join(joinMove(stp.ShipmentMoveID, stp.OrganizationID, stp.BusinessUnitID)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.StopScopeTenant(sq, tenant).Where(sm.ShipmentID.In(), bun.List(ids))
		}).
		OrderExpr(sm.Sequence.OrderAsc()).
		OrderExpr(stp.Sequence.OrderAsc()).
		Scan(ctx, &stops); err != nil {
		return nil, fmt.Errorf("shipment stop locations: %w", err)
	}

	workers := make([]pairRow, 0, len(ids))
	if err := dba.NewSelect().
		Model((*shipment.Assignment)(nil)).
		ColumnExpr(sm.ShipmentID.As("from_id")).
		ColumnExpr(asn.PrimaryWorkerID.As("first_id")).
		ColumnExpr(asn.SecondaryWorkerID.As("second_id")).
		Join(joinMove(asn.ShipmentMoveID, asn.OrganizationID, asn.BusinessUnitID)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AssignmentScopeTenant(sq, tenant).
				Where(sm.ShipmentID.In(), bun.List(ids)).
				Where(asn.Status.NotEq(), shipment.AssignmentStatusCanceled)
		}).
		OrderExpr(sm.Sequence.OrderAsc()).
		Scan(ctx, &workers); err != nil {
		return nil, fmt.Errorf("shipment assigned workers: %w", err)
	}

	carriers := make([]linkRow, 0, len(ids))
	if err := dba.NewSelect().
		Model((*shipment.CarrierAssignment)(nil)).
		ColumnExpr(sm.ShipmentID.As("from_id")).
		ColumnExpr(casn.CarrierID.As("linked_id")).
		Join(joinMove(casn.ShipmentMoveID, casn.OrganizationID, casn.BusinessUnitID)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierAssignmentScopeTenant(sq, tenant).
				Where(sm.ShipmentID.In(), bun.List(ids)).
				Where(casn.Status.NotEq(), shipment.CarrierAssignmentStatusCanceled)
		}).
		OrderExpr(sm.Sequence.OrderAsc()).
		Scan(ctx, &carriers); err != nil {
		return nil, fmt.Errorf("shipment carriers: %w", err)
	}

	links := make(
		[]repositories.MemoryRecordLink,
		0,
		len(customers)*2+len(stops)+len(workers)*2+len(carriers),
	)
	links = appendPairs(links, customers, agent.MemoryRecordCustomer)
	links = appendLinks(links, stops, agent.MemoryRecordLocation)
	links = appendPairs(links, workers, agent.MemoryRecordWorker)

	return appendLinks(links, carriers, agent.MemoryRecordCarrier), nil
}

func (r *repository) moveLinks(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) ([]repositories.MemoryRecordLink, error) {
	sm := buncolgen.ShipmentMoveColumns

	rows := make([]linkRow, 0, len(ids))
	if err := r.db.DBForContext(ctx).NewSelect().
		Model((*shipment.ShipmentMove)(nil)).
		ColumnExpr(sm.ID.As("from_id")).
		ColumnExpr(sm.ShipmentID.As("linked_id")).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShipmentMoveScopeTenant(sq, tenant).Where(sm.ID.In(), bun.List(ids))
		}).
		Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("move shipments: %w", err)
	}

	return appendLinks(
		make([]repositories.MemoryRecordLink, 0, len(rows)),
		rows,
		agent.MemoryRecordShipment,
	), nil
}

func (r *repository) invoiceLinks(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) ([]repositories.MemoryRecordLink, error) {
	inv := buncolgen.InvoiceColumns

	rows := make([]linkRow, 0, len(ids))
	if err := r.db.DBForContext(ctx).NewSelect().
		Model((*invoice.Invoice)(nil)).
		ColumnExpr(inv.ID.As("from_id")).
		ColumnExpr(inv.CustomerID.As("linked_id")).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.InvoiceScopeTenant(sq, tenant).Where(inv.ID.In(), bun.List(ids))
		}).
		Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("invoice customers: %w", err)
	}

	return appendLinks(
		make([]repositories.MemoryRecordLink, 0, len(rows)),
		rows,
		agent.MemoryRecordCustomer,
	), nil
}

func (r *repository) billingQueueLinks(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) ([]repositories.MemoryRecordLink, error) {
	bqi := buncolgen.BillingQueueItemColumns

	rows := make([]linkRow, 0, len(ids))
	if err := r.db.DBForContext(ctx).NewSelect().
		Model((*billingqueue.BillingQueueItem)(nil)).
		ColumnExpr(bqi.ID.As("from_id")).
		ColumnExpr(bqi.BillToCustomerID.As("linked_id")).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.BillingQueueItemScopeTenant(sq, tenant).
				Where(bqi.ID.In(), bun.List(ids))
		}).
		Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("billing queue item customers: %w", err)
	}

	return appendLinks(
		make([]repositories.MemoryRecordLink, 0, len(rows)),
		rows,
		agent.MemoryRecordCustomer,
	), nil
}

func (r *repository) inboundMessageLinks(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) ([]repositories.MemoryRecordLink, error) {
	imsg := buncolgen.InboundMessageColumns

	rows := make([]tripleRow, 0, len(ids))
	if err := r.db.DBForContext(ctx).NewSelect().
		Model((*inboundmessage.InboundMessage)(nil)).
		ColumnExpr(imsg.ID.As("from_id")).
		ColumnExpr(imsg.MatchedCustomerID.As("first_id")).
		ColumnExpr(imsg.MatchedCarrierID.As("second_id")).
		ColumnExpr(imsg.MatchedShipmentID.As("third_id")).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.InboundMessageScopeTenant(sq, tenant).
				Where(imsg.ID.In(), bun.List(ids))
		}).
		Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("inbound message matches: %w", err)
	}

	links := make([]repositories.MemoryRecordLink, 0, len(rows)*3)
	for _, row := range rows {
		links = appendLink(links, row.FromID, agent.MemoryRecordCustomer, row.First)
		links = appendLink(links, row.FromID, agent.MemoryRecordCarrier, row.Second)
		links = appendLink(links, row.FromID, agent.MemoryRecordShipment, row.Third)
	}

	return links, nil
}

func (r *repository) documentLinks(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
) ([]repositories.MemoryRecordLink, error) {
	doc := buncolgen.DocumentColumns

	rows := make([]ownerRow, 0, len(ids))
	if err := r.db.DBForContext(ctx).NewSelect().
		Model((*document.Document)(nil)).
		ColumnExpr(doc.ID.As("from_id")).
		ColumnExpr(doc.ResourceType.As("resource_type")).
		ColumnExpr(doc.ResourceID.As("resource_id")).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentScopeTenant(sq, tenant).Where(doc.ID.In(), bun.List(ids))
		}).
		Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("document owners: %w", err)
	}

	links := make([]repositories.MemoryRecordLink, 0, len(rows))
	for _, row := range rows {
		owner, ok := agent.MemoryRecordRefOf(row.ResourceType, row.ResourceID)
		if !ok || owner.Kind == agent.MemoryRecordDocument {
			continue
		}
		links = append(links, repositories.MemoryRecordLink{
			From: row.FromID,
			Kind: owner.Kind,
			ID:   owner.ID,
		})
	}

	return links, nil
}

func joinMove(moveID, organizationID, businessUnitID buncolgen.Column) string {
	sm := buncolgen.ShipmentMoveColumns

	return "JOIN " + buncolgen.ShipmentMoveTable.As(buncolgen.ShipmentMoveTable.Alias) +
		" ON " + sm.ID.EqColumn(moveID) +
		" AND " + sm.OrganizationID.EqColumn(organizationID) +
		" AND " + sm.BusinessUnitID.EqColumn(businessUnitID)
}

func appendLink(
	links []repositories.MemoryRecordLink,
	from pulid.ID,
	kind agent.MemoryRecordKind,
	id *pulid.ID,
) []repositories.MemoryRecordLink {
	if id == nil || id.IsNil() {
		return links
	}

	return append(links, repositories.MemoryRecordLink{From: from, Kind: kind, ID: *id})
}

func appendLinks(
	links []repositories.MemoryRecordLink,
	rows []linkRow,
	kind agent.MemoryRecordKind,
) []repositories.MemoryRecordLink {
	for _, row := range rows {
		links = appendLink(links, row.FromID, kind, row.LinkedID)
	}

	return links
}

func appendPairs(
	links []repositories.MemoryRecordLink,
	rows []pairRow,
	kind agent.MemoryRecordKind,
) []repositories.MemoryRecordLink {
	for _, row := range rows {
		links = appendLink(links, row.FromID, kind, row.First)
		links = appendLink(links, row.FromID, kind, row.Second)
	}

	return links
}
