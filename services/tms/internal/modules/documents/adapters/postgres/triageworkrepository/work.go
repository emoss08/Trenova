package triageworkrepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentaiextraction"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentaiextraction/documentaiextractiondb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent/documentcontentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage/triagedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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
	_ pagination.TenantInfo,
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

			dc := documentcontentdb.Contents
			entities, err := dc.Pick(
				dc.ID,
				dc.DocumentID,
				dc.OrganizationID,
				dc.BusinessUnitID,
				dc.Status,
				dc.PageCount,
				dc.DetectedDocumentKind,
				dc.ClassificationConfidence,
				dc.StructuredData,
			).
				Where(dc.DocumentID.In(documentIDs...)).
				Limit(len(documentIDs)).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
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
	_ pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) (map[pulid.ID][]string, error) {
	type typeRow struct {
		ResourceID string
		Name       string
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

		d := documentdb.Documents
		dt := d.DocumentType()
		rows, err := keel.Into[typeRow](ctx, r.db.Keel(ctx), d.
			Pick(d.ResourceID, dt.Name).
			Where(
				dt.ID.IsNotNull(),
				d.ResourceType.Eq(shipmentResourceType),
				d.ResourceID.In(resourceIDs...),
				d.IsCurrentVersion.IsTrue(),
				d.Status.Eq(document.StatusActive),
			).
			Group(d.ResourceID, dt.Name).
			Order(d.ResourceID.Asc(), dt.Name.Asc()))
		if err != nil {
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
	_ pagination.TenantInfo,
	shipmentID pulid.ID,
	documentTypeID pulid.ID,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		d := documentdb.Documents
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), d.
			Pick(d.ID).
			Where(
				d.ResourceType.Eq(shipmentResourceType),
				d.ResourceID.Eq(shipmentID.String()),
				d.DocumentTypeID.Eq(documentTypeID),
				d.IsCurrentVersion.IsTrue(),
				d.Status.Eq(document.StatusActive),
			))
		if err != nil {
			return nil, err
		}
		if ids == nil {
			ids = []pulid.ID{}
		}

		return ids, nil
	})
}

func (r *repository) LatestExtractionID(
	ctx context.Context,
	_ pagination.TenantInfo,
	documentID pulid.ID,
) (pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (pulid.ID, error) {
		ex := documentaiextractiondb.Extractions
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), ex.
			Pick(ex.ID).
			Where(
				ex.DocumentID.Eq(documentID),
				ex.Status.In(
					documentaiextraction.StatusCompleted,
					documentaiextraction.StatusApplied,
				),
			).
			Order(ex.ExtractedAt.Desc()).
			Limit(1))
		if err != nil {
			return pulid.Nil, err
		}
		if len(ids) == 0 {
			return pulid.Nil, nil
		}

		return ids[0], nil
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

		m := inboundmessagedb.InboundMessages
		rp := triagedb.Replies

		senders := m.Pick(m.FromAddress.Lower(), m.ReceivedAt.Max()).
			Where(
				m.SenderDomain.Eq(domain),
				m.ReceivedAt.Gte(req.Since),
				m.FromAddress.Lower().Ne(exclude),
			).
			Group(m.FromAddress.Lower())

		delivered := rp.Pick(rp.ToAddress.Lower(), rp.SentAt.Max()).
			Where(
				rp.Status.Eq(triage.ReplySent),
				keel.StringOf(
					keel.DynFunc("split_part", rp.ToAddress, keel.DynConst("@"), keel.DynConst(2)),
				).Lower().Eq(domain),
				rp.SentAt.Gte(req.Since),
				rp.ToAddress.Lower().Ne(exclude),
			).
			Group(rp.ToAddress.Lower())

		candidates := keel.CTE(
			"candidates",
			[]string{"address", "seen_at"},
			keel.UnionAll(senders, delivered),
		)
		address := candidates.Col("address")

		addresses, err := keel.Into[string](ctx, r.db.Keel(ctx), candidates.Query().
			With(candidates).
			Pick(address).
			Group(address).
			Order(keel.Desc(keel.Max(candidates.Col("seen_at"))), address.Asc()).
			Limit(req.Limit))
		if err != nil {
			return nil, err
		}
		if addresses == nil {
			addresses = []string{}
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
			db := r.db.Keel(ctx)
			m := inboundmessagedb.InboundMessages

			waiting, err := m.Where(
				m.MatchedCustomerID.Eq(req.CustomerID),
				m.Status.In(inboundmessage.StatusInReview, inboundmessage.StatusQuarantined),
				m.Parked.IsNull(),
			).Count(ctx, db)
			if err != nil {
				return nil, err
			}

			var messages []*inboundmessage.InboundMessage
			if req.Limit > 0 {
				messages, err = m.Pick(
					m.ID,
					m.FromName,
					m.FromAddress,
					m.Subject,
					m.Classification,
					m.ReceivedAt,
				).
					Where(m.MatchedCustomerID.Eq(req.CustomerID)).
					Order(m.ReceivedAt.Desc(), m.ID.Desc()).
					Limit(req.Limit).
					List(ctx, db)
				if err != nil {
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
