//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package inboundmessagerepository

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const newerAlias = "nwr"

var (
	messageErrors    = dberror.Mapping{Entity: "InboundMessage"}
	attachmentErrors = dberror.Mapping{Entity: "InboundAttachment"}
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

func New(p Params) inboundmessageports.InboundMessageRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.inbound-message-repository"),
	}
}

var needsReviewStatuses = []inboundmessage.Status{
	inboundmessage.StatusInReview,
	inboundmessage.StatusQuarantined,
}

var settledStatuses = []inboundmessage.Status{
	inboundmessage.StatusActioned,
	inboundmessage.StatusIgnored,
}

func narrow(req *inboundmessageports.ListInboundMessagesRequest) []keel.Pred {
	m := inboundmessagedb.InboundMessages

	preds := applyFolder(m, req)
	preds = append(preds, keel.When(req.ThreadKey != "", m.ThreadKey.Eq(req.ThreadKey)))
	if req.CollapseThreads && req.ThreadKey == "" {
		preds = append(preds, keel.NotExists(newerInFolder(req)))
	}

	return preds
}

func applyFolder(
	m *inboundmessagedb.InboundMessagesTable,
	req *inboundmessageports.ListInboundMessagesRequest,
) []keel.Pred {
	return []keel.Pred{
		lane(m, req.Lane, req.ViewerID),
		keel.When(len(req.Statuses) > 0, m.Status.In(req.Statuses...)),
		keel.When(req.Classification != "", m.Classification.Eq(req.Classification)),
		keel.When(req.MailboxID.IsNotNil(), m.MailboxID.Eq(req.MailboxID)),
		keel.When(req.ShipmentID.IsNotNil(), m.MatchedShipmentID.Eq(req.ShipmentID)),
		keel.When(req.OwnerID.IsNotNil(), m.OwnerID.Eq(req.OwnerID)),
		keel.When(req.Since > 0, m.ReceivedAt.Gte(req.Since)),
	}
}

func lane(
	m *inboundmessagedb.InboundMessagesTable,
	l inboundmessageports.Lane,
	viewerID pulid.ID,
) keel.Pred {
	switch l {
	case inboundmessageports.LaneWaiting:
		return keel.And(m.Status.In(needsReviewStatuses...), m.Parked.IsNull())
	case inboundmessageports.LaneAssignedToMe:
		return keel.And(
			m.Status.In(needsReviewStatuses...),
			m.Parked.IsNull(),
			m.OwnerID.Eq(viewerID),
		)
	case inboundmessageports.LaneWaitingOnThem:
		return m.Parked.Eq(inboundmessage.ParkingAwaitingReply)
	case inboundmessageports.LaneSnoozed:
		return m.Parked.Eq(inboundmessage.ParkingSnoozed)
	case inboundmessageports.LaneHandled:
		return m.Status.Eq(inboundmessage.StatusActioned)
	case inboundmessageports.LaneIgnored:
		return m.Status.Eq(inboundmessage.StatusIgnored)
	case inboundmessageports.LaneEverything:
		return nil
	default:
		return nil
	}
}

func newerInFolder(
	req *inboundmessageports.ListInboundMessagesRequest,
) *keel.Query[inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages
	nwr := inboundmessagedb.InboundMessages.As(newerAlias)

	probe := nwr.Pick(nwr.ID).Where(
		nwr.ThreadKey.EqCol(m.ThreadKey),
		keel.DynRow(nwr.ReceivedAt, nwr.ID).Gt(keel.DynRow(m.ReceivedAt, m.ID)),
	).Where(applyFolder(nwr, req)...)
	if req.Filter != nil && strings.TrimSpace(req.Filter.Query) != "" {
		probe = searchPredicate(probe, req.Filter.Query)
	}

	return probe
}

func searchPredicate(
	q *keel.Query[inboundmessage.InboundMessage],
	query string,
) *keel.Query[inboundmessage.InboundMessage] {
	return listquery.Filter(q, &pagination.QueryOptions{Query: query})
}

func (r *repository) List(
	ctx context.Context,
	req *inboundmessageports.ListInboundMessagesRequest,
) (*pagination.ListResult[*inboundmessage.InboundMessage], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*inboundmessage.InboundMessage], error) {
			m := inboundmessagedb.InboundMessages
			return listquery.List(
				ctx,
				r.db.Keel(ctx),
				m.Where(narrow(req)...),
				&listquery.Request[inboundmessage.InboundMessage]{
					Filter: req.Filter,
					Shape:  withMailbox,
					Then:   []keel.Order{m.ReceivedAt.Desc()},
				},
			)
		},
	)
}

func withMailbox(
	q *keel.Query[inboundmessage.InboundMessage],
) *keel.Query[inboundmessage.InboundMessage] {
	return q.Load(inboundmessagedb.InboundMessages.Mailbox())
}

func (r *repository) ListCursor(
	ctx context.Context,
	req *inboundmessageports.ListInboundMessagesRequest,
) (*pagination.CursorListResult[*inboundmessage.InboundMessage], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*inboundmessage.InboundMessage], error) {
			return listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				inboundmessagedb.InboundMessages.Where(narrow(req)...),
				&listquery.Request[inboundmessage.InboundMessage]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape:  withMailbox,
				},
			)
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req inboundmessageports.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*inboundmessage.InboundMessage, error) {
		m := inboundmessagedb.InboundMessages
		loads := []keel.Loader{m.Mailbox()}
		if req.IncludeAttachments {
			loads = append(loads, m.Attachments())
		}

		entity, err := m.Where(m.ID.Eq(req.ID)).Load(loads...).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, messageErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

// GetByProviderID is the idempotency read. It is scoped by mailbox rather than
// by tenant because the mailbox is what the delivery resolved to, and the
// unique index it rides on is `(mailbox_id, provider_message_id)`.
func (r *repository) GetByProviderID(
	ctx context.Context,
	req inboundmessageports.GetInboundMessageByProviderIDRequest,
) (*inboundmessage.InboundMessage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*inboundmessage.InboundMessage, error) {
		m := inboundmessagedb.InboundMessages
		entity, err := m.Where(
			m.MailboxID.Eq(req.MailboxID),
			m.ProviderMessageID.Eq(req.ProviderMessageID),
		).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, messageErrors.Map(err, req.ProviderMessageID)
		}

		return entity, nil
	})
}

// Create writes the message and its attachments as one transaction.
//
// A message whose attachment rows failed to write would be a message the
// pipeline believes has nothing to extract, and the files would be lost with no
// record that they ever arrived.
func (r *repository) Create(
	ctx context.Context,
	entity *inboundmessage.InboundMessage,
	attachments []*inboundmessage.InboundAttachment,
) (*inboundmessage.InboundMessage, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*inboundmessage.InboundMessage, error) {
		db := r.db.Keel(ctx)
		if err := inboundmessagedb.InboundMessages.Create(ctx, db, entity); err != nil {
			return nil, err
		}

		if len(attachments) > 0 {
			for _, attachment := range attachments {
				attachment.MessageID = entity.ID
				attachment.OrganizationID = entity.OrganizationID
				attachment.BusinessUnitID = entity.BusinessUnitID
			}
			if err := inboundmessagedb.InboundAttachments.CreateMany(
				ctx,
				db,
				attachments,
			); err != nil {
				return nil, err
			}
		}

		entity.Attachments = attachments

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *inboundmessage.InboundMessage,
) (*inboundmessage.InboundMessage, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*inboundmessage.InboundMessage, error) {
		if err := inboundmessagedb.InboundMessages.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, messageErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

// UpdateAttachment carries no version because an attachment row is only ever
// written by the pipeline that owns the message, one step at a time.
func (r *repository) UpdateAttachment(
	ctx context.Context,
	entity *inboundmessage.InboundAttachment,
) (*inboundmessage.InboundAttachment, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*inboundmessage.InboundAttachment, error) {
			err := inboundmessagedb.InboundAttachments.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				return nil, attachmentErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) ListAttachments(
	ctx context.Context,
	messageID pulid.ID,
	_ pagination.TenantInfo,
) ([]*inboundmessage.InboundAttachment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*inboundmessage.InboundAttachment, error) {
			a := inboundmessagedb.InboundAttachments
			return orEmptyAttachments(a.Where(a.MessageID.Eq(messageID)).
				Order(a.CreatedAt.Asc()).
				All(ctx, r.db.Keel(ctx)))
		},
	)
}

// CountBreakdown is the inbox's census in one aggregate: a row per status,
// reading and mailbox that has any mail. The service folds it into lanes,
// kinds and mailboxes, so the counts in the rail cannot disagree with each
// other the way three separate queries run a moment apart could.
func (r *repository) CountBreakdown(
	ctx context.Context,
	req inboundmessageports.CountInboundMessagesRequest,
) ([]inboundmessageports.InboundMessageCount, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]inboundmessageports.InboundMessageCount, error) {
			rows, err := keel.Into[inboundmessageports.InboundMessageCount](
				ctx,
				r.db.Keel(ctx),
				countBreakdownQuery(req),
			)
			if err != nil {
				return nil, err
			}
			if rows == nil {
				rows = []inboundmessageports.InboundMessageCount{}
			}

			return rows, nil
		},
	)
}

func countBreakdownQuery(req inboundmessageports.CountInboundMessagesRequest) keel.Projection {
	open, settled := censusStatuses(req.Statuses)
	if len(settled) == 0 {
		return openCensusQuery(req, open)
	}
	if len(open) == 0 {
		return settledCensusQuery(req, settled)
	}

	return keel.UnionAll(openCensusQuery(req, open), settledCensusQuery(req, settled))
}

func censusStatuses(
	requested []inboundmessage.Status,
) (open, settled []inboundmessage.Status) {
	if len(requested) == 0 {
		requested = inboundmessage.AllStatuses()
	}
	open = make([]inboundmessage.Status, 0, len(requested))
	settled = make([]inboundmessage.Status, 0, len(settledStatuses))
	for _, status := range requested {
		if slices.Contains(settledStatuses, status) {
			settled = append(settled, status)

			continue
		}
		open = append(open, status)
	}

	return open, settled
}

func censusBase(
	req inboundmessageports.CountInboundMessagesRequest,
	statuses []inboundmessage.Status,
) *keel.Query[inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages

	return m.Where(
		m.Status.In(statuses...),
		keel.When(req.Since > 0, m.ReceivedAt.Gte(req.Since)),
	)
}

func openCensusQuery(
	req inboundmessageports.CountInboundMessagesRequest,
	statuses []inboundmessage.Status,
) *keel.Query[inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages
	classification := keel.DynOf(m.Classification).Coalesce("")
	parked := keel.DynOf(m.Parked).Coalesce("")
	mine := keel.DynOf(m.OwnerID).Eq(keel.DynLit(req.ViewerID.String())).Coalesce(false)
	overdue := keel.DynOf(m.DueAt).Lt(keel.DynLit(req.Now)).Coalesce(false)

	return censusBase(req, statuses).
		Pick(m.Status, classification, m.MailboxID, parked, mine, overdue, keel.Count()).
		Group(m.Status, classification, m.MailboxID, parked, mine, overdue)
}

func settledCensusQuery(
	req inboundmessageports.CountInboundMessagesRequest,
	statuses []inboundmessage.Status,
) *keel.Query[inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages

	return censusBase(req, statuses).
		Pick(
			m.Status,
			keel.DynOf(m.Classification).Coalesce(""),
			m.MailboxID,
			keel.DynLit(""),
			keel.DynLit(false),
			keel.DynLit(false),
			keel.Count(),
		).
		Group(m.Status, m.Classification, m.MailboxID)
}

// CountAttachmentsByMessageIDs answers the paperclip on a page of list rows in
// one query, rather than one per row.
func (r *repository) CountAttachmentsByMessageIDs(
	ctx context.Context,
	req inboundmessageports.CountInboundAttachmentsRequest,
) (map[pulid.ID]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]int, error) {
		counts := make(map[pulid.ID]int, len(req.MessageIDs))
		if len(req.MessageIDs) == 0 {
			return counts, nil
		}

		type attachmentCount struct {
			MessageID pulid.ID
			Count     int64
		}

		a := inboundmessagedb.InboundAttachments
		rows, err := keel.Into[attachmentCount](ctx, r.db.Keel(ctx), a.
			Pick(a.MessageID, keel.Count()).
			Where(a.MessageID.In(req.MessageIDs...)).
			Group(a.MessageID))
		if err != nil {
			return nil, err
		}

		for _, row := range rows {
			counts[row.MessageID] = int(row.Count)
		}

		return counts, nil
	})
}

// ListRecentForReview feeds the watchtower's nightly snapshot: the messages
// still waiting on a person.
func (r *repository) ListRecentForReview(
	ctx context.Context,
	req inboundmessageports.ListRecentInboundMessagesForReviewRequest,
) ([]*inboundmessage.InboundMessage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*inboundmessage.InboundMessage, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = 10
			}

			m := inboundmessagedb.InboundMessages
			return orEmptyMessages(m.Where(m.Status.In(needsReviewStatuses...)).
				Order(m.ReceivedAt.Desc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx)))
		},
	)
}

// DeleteBefore removes messages that are finished with and old enough.
//
// Only terminal ones go: a message still in review is work somebody has not
// done yet, and retention is not the thing that should decide it never will be.
func (r *repository) DeleteBefore(
	ctx context.Context,
	req inboundmessageports.DeleteInboundMessagesBeforeRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		m := inboundmessagedb.InboundMessages
		expired := inboundmessagedb.InboundMessages.As("expired")

		return m.Where(m.ID.InQuery(expired.
			Pick(expired.ID).
			Where(
				expired.ReceivedAt.Lt(req.Before),
				expired.Status.In(
					inboundmessage.StatusActioned,
					inboundmessage.StatusIgnored,
					inboundmessage.StatusQuarantined,
				),
			).
			Limit(max(req.Limit, 1)),
		)).Delete(ctx, r.db.Keel(ctx))
	})
}

// ListSettledBefore reads the oldest settled messages first, through the lane
// index, returning only what retention needs: the id and the stored body key.
func (r *repository) ListSettledBefore(
	ctx context.Context,
	req inboundmessageports.ListSettledInboundMessagesRequest,
) ([]*inboundmessage.InboundMessage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*inboundmessage.InboundMessage, error) {
			return orEmptyMessages(settledBeforeQuery(req).List(ctx, r.db.Keel(ctx)))
		},
	)
}

func settledBeforeQuery(
	req inboundmessageports.ListSettledInboundMessagesRequest,
) *keel.Query[inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages

	return m.Pick(m.ID, m.OrganizationID, m.BusinessUnitID, m.HTMLKey).
		Where(m.Status.In(settledStatuses...), m.ReceivedAt.Lt(req.Before)).
		Order(m.ReceivedAt.Asc()).
		Limit(max(req.Limit, 1))
}

// DeleteByIDs removes messages inside one tenant. Their attachment rows go
// with them through the foreign key's cascade.
func (r *repository) DeleteByIDs(
	ctx context.Context,
	req inboundmessageports.DeleteInboundMessagesRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if len(req.IDs) == 0 {
			return 0, nil
		}

		m := inboundmessagedb.InboundMessages
		deleted, err := m.Where(m.ID.In(req.IDs...)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, err
		}

		return int(deleted), nil
	})
}

func orEmptyMessages(
	entities []*inboundmessage.InboundMessage,
	err error,
) ([]*inboundmessage.InboundMessage, error) {
	if err != nil {
		return nil, err
	}
	if entities == nil {
		return []*inboundmessage.InboundMessage{}, nil
	}

	return entities, nil
}

func orEmptyAttachments(
	entities []*inboundmessage.InboundAttachment,
	err error,
) ([]*inboundmessage.InboundAttachment, error) {
	if err != nil {
		return nil, err
	}
	if entities == nil {
		return []*inboundmessage.InboundAttachment{}, nil
	}

	return entities, nil
}
