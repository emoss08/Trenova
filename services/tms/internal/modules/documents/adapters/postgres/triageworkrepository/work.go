package triageworkrepository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentaiextraction"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/documentaiextractioncol"
	"github.com/emoss08/trenova/pkg/buncolgen/documentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/documentcontentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/documenttypecol"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"
	"github.com/emoss08/trenova/pkg/buncolgen/triagecol"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

const shipmentResourceType = "shipment"

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) inboundmessageports.TriageWorkRepository {
	return &repository{db: p.DB}
}

func (r *repository) ContentsByDocumentIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	documentIDs []pulid.ID,
) (map[pulid.ID]*documentcontent.Content, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]*documentcontent.Content, error) {
			byDocument := make(map[pulid.ID]*documentcontent.Content, len(documentIDs))
			if len(documentIDs) == 0 {
				return byDocument, nil
			}

			cols := documentcontentcol.ContentColumns
			entities := make([]*documentcontent.Content, 0, len(documentIDs))
			if err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Column(
					cols.ID.Bare(),
					cols.DocumentID.Bare(),
					cols.OrganizationID.Bare(),
					cols.BusinessUnitID.Bare(),
					cols.Status.Bare(),
					cols.PageCount.Bare(),
					cols.DetectedDocumentKind.Bare(),
					cols.ClassificationConfidence.Bare(),
					cols.StructuredData.Bare(),
				).
				Where(cols.DocumentID.In(), bun.List(documentIDs)).
				Apply(documentcontentcol.ContentApplyTenant(tenantInfo)).
				Scan(ctx); err != nil {
				return nil, err
			}

			for _, entity := range entities {
				byDocument[entity.DocumentID] = entity
			}

			return byDocument, nil
		},
	)
}

func (r *repository) DocumentTypesOnShipments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) (map[pulid.ID][]string, error) {
	type typeRow struct {
		ResourceID string `bun:"resource_id"`
		Name       string `bun:"name"`
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID][]string, error) {
		byShipment := make(map[pulid.ID][]string, len(shipmentIDs))
		if len(shipmentIDs) == 0 {
			return byShipment, nil
		}

		resourceIDs := make([]string, 0, len(shipmentIDs))
		for _, id := range shipmentIDs {
			resourceIDs = append(resourceIDs, id.String())
		}

		doc := documentcol.DocumentColumns
		dt := documenttypecol.DocumentTypeColumns
		rows := make([]typeRow, 0, len(shipmentIDs)*2)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*document.Document)(nil)).
			ColumnExpr(doc.ResourceID.Qualified()).
			ColumnExpr(dt.Name.As("name")).
			Join("JOIN "+documenttypecol.DocumentTypeTable.Name+" AS "+documenttypecol.DocumentTypeTable.Alias+
				" ON "+dt.ID.EqColumn(doc.DocumentTypeID)+
				" AND "+dt.OrganizationID.EqColumn(doc.OrganizationID)+
				" AND "+dt.BusinessUnitID.EqColumn(doc.BusinessUnitID)).
			Where(doc.ResourceType.Eq(), shipmentResourceType).
			Where(doc.ResourceID.In(), bun.List(resourceIDs)).
			Where(doc.IsCurrentVersion.IsTrue()).
			Where(doc.Status.Eq(), document.StatusActive).
			Apply(documentcol.DocumentApplyTenant(tenantInfo)).
			GroupExpr(doc.ResourceID.Qualified()+", "+dt.Name.Qualified()).
			OrderExpr(doc.ResourceID.Qualified()+", "+dt.Name.Qualified()).
			Scan(ctx, &rows); err != nil {
			return nil, err
		}

		for _, row := range rows {
			id, err := pulid.Parse(row.ResourceID)
			if err != nil {
				continue
			}
			byShipment[id] = append(byShipment[id], row.Name)
		}

		return byShipment, nil
	})
}

func (r *repository) CurrentShipmentDocuments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	documentTypeID pulid.ID,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		doc := documentcol.DocumentColumns
		ids := make([]pulid.ID, 0, 2)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*document.Document)(nil)).
			Column(doc.ID.Bare()).
			Where(doc.ResourceType.Eq(), shipmentResourceType).
			Where(doc.ResourceID.Eq(), shipmentID.String()).
			Where(doc.DocumentTypeID.Eq(), documentTypeID).
			Where(doc.IsCurrentVersion.IsTrue()).
			Where(doc.Status.Eq(), document.StatusActive).
			Apply(documentcol.DocumentApplyTenant(tenantInfo)).
			Scan(ctx, &ids); err != nil {
			return nil, err
		}

		return ids, nil
	})
}

func (r *repository) LatestExtractionID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	documentID pulid.ID,
) (pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (pulid.ID, error) {
		cols := documentaiextractioncol.ExtractionColumns
		var id pulid.ID
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*documentaiextraction.Extraction)(nil)).
			Column(cols.ID.Bare()).
			Where(cols.DocumentID.Eq(), documentID).
			Where(cols.Status.In(), bun.List([]documentaiextraction.Status{
				documentaiextraction.StatusCompleted,
				documentaiextraction.StatusApplied,
			})).
			Apply(documentaiextractioncol.ExtractionApplyTenant(tenantInfo)).
			Order(cols.ExtractedAt.OrderDesc()).
			Limit(1).
			Scan(ctx, &id)
		if errors.Is(err, sql.ErrNoRows) {
			return pulid.Nil, nil
		}

		return id, err
	})
}

func (r *repository) AlternateAddresses(
	ctx context.Context,
	req *inboundmessageports.AlternateAddressRequest,
) ([]string, error) {
	domain := strings.ToLower(strings.TrimSpace(req.Domain))
	exclude := strings.ToLower(strings.TrimSpace(req.Exclude))

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]string, error) {
		if domain == "" || req.Limit <= 0 {
			return []string{}, nil
		}

		db := r.db.DBForContext(ctx)
		msg := inboundmessagecol.InboundMessageColumns
		reply := triagecol.ReplyColumns

		senders := db.NewSelect().
			Model((*inboundmessage.InboundMessage)(nil)).
			ColumnExpr(msg.FromAddress.Expr("lower({})")+" AS address").
			ColumnExpr(buncolgen.Max(msg.ReceivedAt, "seen_at")).
			Where(msg.SenderDomain.Eq(), domain).
			Where(msg.ReceivedAt.Gte(), req.Since).
			Where(msg.FromAddress.Expr("lower({})")+" <> ?", exclude).
			Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
			GroupExpr(msg.FromAddress.Expr("lower({})"))

		delivered := db.NewSelect().
			Model((*triage.Reply)(nil)).
			ColumnExpr(reply.ToAddress.Expr("lower({})")+" AS address").
			ColumnExpr(buncolgen.Max(reply.SentAt, "seen_at")).
			Where(reply.Status.Eq(), triage.ReplySent).
			Where(reply.ToAddress.Expr("lower(split_part({}, '@', 2))")+" = ?", domain).
			Where(reply.SentAt.Gte(), req.Since).
			Where(reply.ToAddress.Expr("lower({})")+" <> ?", exclude).
			Apply(triagecol.ReplyApplyTenant(req.TenantInfo)).
			GroupExpr(reply.ToAddress.Expr("lower({})"))

		addresses := make([]string, 0, req.Limit)
		if err := db.NewSelect().
			With("candidates", senders.UnionAll(delivered)).
			Table("candidates").
			Column("address").
			Group("address").
			OrderExpr("MAX(seen_at) DESC, address ASC").
			Limit(int64(req.Limit)).
			Scan(ctx, &addresses); err != nil {
			return nil, err
		}

		return addresses, nil
	})
}

func (r *repository) CustomerTriage(
	ctx context.Context,
	req *inboundmessageports.CustomerTriageRequest,
) (*inboundmessageports.CustomerTriage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*inboundmessageports.CustomerTriage, error) {
			db := r.db.DBForContext(ctx)
			cols := inboundmessagecol.InboundMessageColumns

			waiting, err := db.NewSelect().
				Model((*inboundmessage.InboundMessage)(nil)).
				Where(cols.MatchedCustomerID.Eq(), req.CustomerID).
				Where(cols.Status.In(), bun.List([]inboundmessage.Status{
					inboundmessage.StatusInReview,
					inboundmessage.StatusQuarantined,
				})).
				Where(cols.Parked.IsNull()).
				Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
				Count(ctx)
			if err != nil {
				return nil, err
			}

			messages := make([]*inboundmessage.InboundMessage, 0, req.Limit)
			if req.Limit > 0 {
				if err = db.NewSelect().
					Model(&messages).
					Column(
						cols.ID.Bare(),
						cols.FromName.Bare(),
						cols.FromAddress.Bare(),
						cols.Subject.Bare(),
						cols.Classification.Bare(),
						cols.ReceivedAt.Bare(),
					).
					Where(cols.MatchedCustomerID.Eq(), req.CustomerID).
					Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
					Order(cols.ReceivedAt.OrderDesc(), cols.ID.OrderDesc()).
					Limit(int64(req.Limit)).
					Scan(ctx); err != nil {
					return nil, err
				}
			}

			recent := make([]*inboundmessageports.Sample, 0, len(messages))
			for _, message := range messages {
				sample := &inboundmessageports.Sample{
					MessageID:   message.ID,
					FromName:    message.FromName,
					FromAddress: message.FromAddress,
					Subject:     message.Subject,
					ReceivedAt:  message.ReceivedAt,
				}
				if message.Classification != "" {
					class := message.Classification
					sample.Classification = &class
				}
				recent = append(recent, sample)
			}

			return &inboundmessageports.CustomerTriage{Waiting: int(waiting), Recent: recent}, nil
		},
	)
}
