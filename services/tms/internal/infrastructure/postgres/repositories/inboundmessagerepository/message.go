//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package inboundmessagerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
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

func New(p Params) repositories.InboundMessageRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.inbound-message-repository"),
	}
}

// narrow applies the filters the inbox lanes are built from. It is shared by
// the offset and cursor lists so the two cannot drift into showing different
// rows for the same lane.
func narrow(
	sq *bun.SelectQuery,
	req *repositories.ListInboundMessagesRequest,
) *bun.SelectQuery {
	cols := buncolgen.InboundMessageColumns

	if len(req.Statuses) > 0 {
		sq = sq.Where(cols.Status.In(), bun.In(req.Statuses))
	}
	if req.Classification != "" {
		sq = sq.Where(cols.Classification.Eq(), req.Classification)
	}
	if req.MailboxID.IsNotNil() {
		sq = sq.Where(cols.MailboxID.Eq(), req.MailboxID)
	}
	if req.ShipmentID.IsNotNil() {
		sq = sq.Where(cols.MatchedShipmentID.Eq(), req.ShipmentID)
	}
	if req.Since > 0 {
		sq = sq.Where(cols.ReceivedAt.Gte(), req.Since)
	}

	return sq
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListInboundMessagesRequest,
) (*pagination.ListResult[*inboundmessage.InboundMessage], error) {
	entities := make([]*inboundmessage.InboundMessage, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.InboundMessageColumns

	query := narrow(
		r.db.DBForContext(ctx).NewSelect().Model(&entities).Relation("Mailbox"),
		req,
	)

	total, err := querybuilder.ApplyFilters(
		query, "imsg", req.Filter, (*inboundmessage.InboundMessage)(nil),
	).
		Order(cols.ReceivedAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*inboundmessage.InboundMessage]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) ListCursor(
	ctx context.Context,
	req *repositories.ListInboundMessagesRequest,
) (*pagination.CursorListResult[*inboundmessage.InboundMessage], error) {
	dba := r.db.DBForContext(ctx)

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*inboundmessage.InboundMessage)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return narrow(querybuilder.ApplyFiltersWithoutSort(
					sq, "imsg", req.Filter, (*inboundmessage.InboundMessage)(nil),
				), req)
			}).
			Count(ctx)
		if err != nil {
			return nil, err
		}
		totalCount = &total
	}

	return dbhelper.CursorList(ctx, dbhelper.CursorListParams[*inboundmessage.InboundMessage]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: totalCount,
		Query: func(entities *[]*inboundmessage.InboundMessage) *bun.SelectQuery {
			return dba.NewSelect().Model(entities).Relation("Mailbox")
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			sq, applyErr := querybuilder.ApplyCursorFilters(
				sq, "imsg", req.Filter, req.Cursor, (*inboundmessage.InboundMessage)(nil),
			)
			if applyErr != nil {
				return sq, applyErr
			}

			return narrow(sq, req), nil
		},
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	entity := new(inboundmessage.InboundMessage)
	cols := buncolgen.InboundMessageColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("Mailbox").
		Where(cols.ID.Eq(), req.ID).
		Apply(buncolgen.InboundMessageApplyTenant(req.TenantInfo))
	if req.IncludeAttachments {
		query = query.Relation("Attachments")
	}

	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "InboundMessage")
	}

	return entity, nil
}

// GetByProviderID is the idempotency read. It is scoped by mailbox rather than
// by tenant because the mailbox is what the delivery resolved to, and the
// unique index it rides on is `(mailbox_id, provider_message_id)`.
func (r *repository) GetByProviderID(
	ctx context.Context,
	req repositories.GetInboundMessageByProviderIDRequest,
) (*inboundmessage.InboundMessage, error) {
	entity := new(inboundmessage.InboundMessage)
	cols := buncolgen.InboundMessageColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.MailboxID.Eq(), req.MailboxID).
		Where(cols.ProviderMessageID.Eq(), req.ProviderMessageID).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "InboundMessage")
	}

	return entity, nil
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
	err := r.db.DBForContext(ctx).RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(entity).Returning("*").Exec(c); iErr != nil {
			return iErr
		}
		if len(attachments) == 0 {
			return nil
		}

		for _, attachment := range attachments {
			attachment.MessageID = entity.ID
			attachment.OrganizationID = entity.OrganizationID
			attachment.BusinessUnitID = entity.BusinessUnitID
		}
		if _, iErr := tx.NewInsert().Model(&attachments).Returning("*").Exec(c); iErr != nil {
			return iErr
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	entity.Attachments = attachments

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *inboundmessage.InboundMessage,
) (*inboundmessage.InboundMessage, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		entity.Version = ov

		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "InboundMessage", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

// UpdateAttachment carries no version because an attachment row is only ever
// written by the pipeline that owns the message, one step at a time.
func (r *repository) UpdateAttachment(
	ctx context.Context,
	entity *inboundmessage.InboundAttachment,
) (*inboundmessage.InboundAttachment, error) {
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(
		results, "InboundAttachment", entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListAttachments(
	ctx context.Context,
	messageID pulid.ID,
	tenantInfo pagination.TenantInfo,
) ([]*inboundmessage.InboundAttachment, error) {
	entities := make([]*inboundmessage.InboundAttachment, 0, 4)
	cols := buncolgen.InboundAttachmentColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.MessageID.Eq(), messageID).
		Apply(buncolgen.InboundAttachmentApplyTenant(tenantInfo)).
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

// CountBreakdown is the inbox's census in one aggregate: a row per status,
// reading and mailbox that has any mail. The service folds it into lanes,
// kinds and mailboxes, so the counts in the rail cannot disagree with each
// other the way three separate queries run a moment apart could.
func (r *repository) CountBreakdown(
	ctx context.Context,
	req repositories.CountInboundMessagesRequest,
) ([]repositories.InboundMessageCount, error) {
	rows := make([]repositories.InboundMessageCount, 0, len(inboundmessage.AllStatuses()))

	if err := countBreakdownQuery(r.db.DBForContext(ctx), req).Scan(ctx, &rows); err != nil {
		return nil, err
	}

	return rows, nil
}

func countBreakdownQuery(
	db bun.IDB,
	req repositories.CountInboundMessagesRequest,
) *bun.SelectQuery {
	cols := buncolgen.InboundMessageColumns

	query := db.NewSelect().
		Model((*inboundmessage.InboundMessage)(nil)).
		ColumnExpr(cols.Status.Qualified()).
		ColumnExpr("COALESCE(?, '') AS classification", bun.Safe(cols.Classification.Qualified())).
		ColumnExpr(cols.MailboxID.Qualified()).
		ColumnExpr("COUNT(*) AS count").
		Apply(buncolgen.InboundMessageApplyTenant(req.TenantInfo))
	if len(req.Statuses) > 0 {
		query = query.Where(cols.Status.In(), bun.In(req.Statuses))
	}
	if req.Since > 0 {
		query = query.Where(cols.ReceivedAt.Gte(), req.Since)
	}

	return query.
		GroupExpr(cols.Status.Qualified()).
		GroupExpr(cols.Classification.Qualified()).
		GroupExpr(cols.MailboxID.Qualified())
}

// CountAttachmentsByMessageIDs answers the paperclip on a page of list rows in
// one query, rather than one per row.
func (r *repository) CountAttachmentsByMessageIDs(
	ctx context.Context,
	req repositories.CountInboundAttachmentsRequest,
) (map[pulid.ID]int, error) {
	counts := make(map[pulid.ID]int, len(req.MessageIDs))
	if len(req.MessageIDs) == 0 {
		return counts, nil
	}

	cols := buncolgen.InboundAttachmentColumns
	rows := make([]struct {
		MessageID pulid.ID `bun:"message_id"`
		Count     int      `bun:"count"`
	}, 0, len(req.MessageIDs))

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*inboundmessage.InboundAttachment)(nil)).
		ColumnExpr(cols.MessageID.Qualified()).
		ColumnExpr("COUNT(*) AS count").
		Where(cols.MessageID.In(), bun.In(req.MessageIDs)).
		Apply(buncolgen.InboundAttachmentApplyTenant(req.TenantInfo)).
		GroupExpr(cols.MessageID.Qualified()).
		Scan(ctx, &rows); err != nil {
		return nil, err
	}

	for _, row := range rows {
		counts[row.MessageID] = row.Count
	}

	return counts, nil
}

// ListRecentForReview feeds the watchtower's nightly snapshot: the messages
// still waiting on a person.
func (r *repository) ListRecentForReview(
	ctx context.Context,
	req repositories.ListRecentInboundMessagesForReviewRequest,
) ([]*inboundmessage.InboundMessage, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	entities := make([]*inboundmessage.InboundMessage, 0, limit)
	cols := buncolgen.InboundMessageColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.Status.In(), bun.In([]inboundmessage.Status{
			inboundmessage.StatusInReview,
			inboundmessage.StatusQuarantined,
		})).
		Apply(buncolgen.InboundMessageApplyTenant(req.TenantInfo)).
		Order(cols.ReceivedAt.OrderDesc()).
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

// DeleteBefore removes messages that are finished with and old enough.
//
// Only terminal ones go: a message still in review is work somebody has not
// done yet, and retention is not the thing that should decide it never will be.
func (r *repository) DeleteBefore(
	ctx context.Context,
	req repositories.DeleteInboundMessagesBeforeRequest,
) (int64, error) {
	cols := buncolgen.InboundMessageColumns

	ids := r.db.DBForContext(ctx).
		NewSelect().
		Model((*inboundmessage.InboundMessage)(nil)).
		Column("id").
		Where(cols.ReceivedAt.Lt(), req.Before).
		Where(cols.Status.In(), bun.In([]inboundmessage.Status{
			inboundmessage.StatusActioned,
			inboundmessage.StatusIgnored,
			inboundmessage.StatusQuarantined,
		})).
		Apply(buncolgen.InboundMessageApplyTenant(req.TenantInfo)).
		Limit(req.Limit)

	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*inboundmessage.InboundMessage)(nil)).
		Where("id IN (?)", ids).
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	return results.RowsAffected()
}
