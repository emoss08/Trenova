package inboundreplyrepository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"
	"github.com/emoss08/trenova/pkg/buncolgen/triagecol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

const (
	replyEntity            = "InboundReply"
	repliesPerMessageHint  = 2
	statusRankSent         = 0
	statusRankSending      = 1
	statusRankOther        = 2
	threadKeyLookupMaximum = 50
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) inboundmessageports.InboundReplyRepository {
	return &repository{db: p.DB}
}

func (r *repository) Create(ctx context.Context, entity *triage.Reply) (*triage.Reply, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.Reply, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Update(ctx context.Context, entity *triage.Reply) (*triage.Reply, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.Reply, error) {
		previous := entity.Version
		entity.Version++

		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(triagecol.ReplyColumns.Version.Eq(), previous).
			Returning("*").
			Exec(ctx)
		if err != nil {
			entity.Version = previous

			return nil, err
		}
		if err = dberror.CheckRowsAffected(results, replyEntity, entity.ID.String()); err != nil {
			entity.Version = previous

			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req inboundmessageports.GetInboundReplyRequest,
) (*triage.Reply, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triage.Reply, error) {
		entity := new(triage.Reply)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Where(triagecol.ReplyColumns.ID.Eq(), req.ID).
			Apply(triagecol.ReplyApplyTenant(req.TenantInfo)).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, replyEntity)
		}

		return entity, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req inboundmessageports.GetInboundRepliesRequest,
) ([]*triage.Reply, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triage.Reply, error) {
		entities := make([]*triage.Reply, 0, len(req.IDs))
		if len(req.IDs) == 0 {
			return entities, nil
		}

		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Where(triagecol.ReplyColumns.ID.In(), bun.List(req.IDs)).
			Apply(triagecol.ReplyApplyTenant(req.TenantInfo)).
			Scan(ctx); err != nil {
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) GetByIdempotencyKey(
	ctx context.Context,
	req inboundmessageports.GetInboundReplyByKeyRequest,
) (*triage.Reply, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triage.Reply, error) {
		entity := new(triage.Reply)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Where(triagecol.ReplyColumns.IdempotencyKey.Eq(), req.IdempotencyKey).
			Apply(triagecol.ReplyApplyTenant(req.TenantInfo)).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, replyEntity)
		}

		return entity, nil
	})
}

func (r *repository) UpsertDraft(ctx context.Context, entity *triage.Reply) (*triage.Reply, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.Reply, error) {
		tenantInfo := pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		}
		if err := r.lockMessage(ctx, tenantInfo, entity.MessageID); err != nil {
			return nil, err
		}

		existing := new(triage.Reply)
		cols := triagecol.ReplyColumns
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(existing).
			Where(cols.MessageID.Eq(), entity.MessageID).
			Where(cols.Status.Eq(), triage.ReplyDraft).
			Apply(triagecol.ReplyApplyTenant(tenantInfo)).
			Order(cols.CreatedAt.OrderDesc()).
			Limit(1).
			Scan(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if err != nil {
			entity.Status = triage.ReplyDraft
			if _, err = r.db.DBForContext(ctx).
				NewInsert().
				Model(entity).
				Returning("*").
				Exec(ctx); err != nil {
				return nil, err
			}

			return entity, nil
		}

		existing.Subject = entity.Subject
		existing.Body = entity.Body
		existing.DraftedByNova = entity.DraftedByNova
		existing.TemplateID = entity.TemplateID
		existing.ToAddress = entity.ToAddress
		existing.FromAddress = entity.FromAddress
		existing.CcAddresses = entity.CcAddresses
		if entity.CreatedByID.IsNotNil() {
			existing.CreatedByID = entity.CreatedByID
		}

		return r.Update(ctx, existing)
	})
}

func (r *repository) lockMessage(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	messageID pulid.ID,
) error {
	var id pulid.ID
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*inboundmessage.InboundMessage)(nil)).
		Column(inboundmessagecol.InboundMessageColumns.ID.Bare()).
		Where(inboundmessagecol.InboundMessageColumns.ID.Eq(), messageID).
		Apply(inboundmessagecol.InboundMessageApplyTenant(tenantInfo)).
		For("UPDATE").
		Scan(ctx, &id); err != nil {
		return dberror.HandleNotFoundError(err, "InboundMessage")
	}

	return nil
}

func (r *repository) DeleteDraft(
	ctx context.Context,
	req inboundmessageports.InboundReplyMessageRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := triagecol.ReplyColumns
		result, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*triage.Reply)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return triagecol.ReplyScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.MessageID.Eq(), req.MessageID).
					Where(cols.Status.Eq(), triage.ReplyDraft)
			}).
			Exec(ctx)
		if err != nil {
			return 0, err
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}

		return int(affected), nil
	})
}

func (r *repository) ListByMessageIDs(
	ctx context.Context,
	req inboundmessageports.InboundRepliesByMessagesRequest,
) (map[pulid.ID][]*triage.Reply, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID][]*triage.Reply, error) {
		grouped := make(map[pulid.ID][]*triage.Reply, len(req.MessageIDs))
		if len(req.MessageIDs) == 0 {
			return grouped, nil
		}

		cols := triagecol.ReplyColumns
		entities := make([]*triage.Reply, 0, len(req.MessageIDs)*repliesPerMessageHint)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Where(cols.MessageID.In(), bun.List(req.MessageIDs)).
			Where(cols.Status.Ne(), triage.ReplyDraft).
			Apply(triagecol.ReplyApplyTenant(req.TenantInfo)).
			Order(cols.MessageID.OrderAsc(), cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc()).
			Scan(ctx); err != nil {
			return nil, err
		}

		for _, entity := range entities {
			grouped[entity.MessageID] = append(grouped[entity.MessageID], entity)
		}

		return grouped, nil
	})
}

func (r *repository) DraftsByMessageIDs(
	ctx context.Context,
	req inboundmessageports.InboundRepliesByMessagesRequest,
) (map[pulid.ID]*triage.Reply, error) {
	return r.distinctByMessage(ctx, &distinctRequest{
		tenantInfo: req.TenantInfo,
		messageIDs: req.MessageIDs,
		draftsOnly: true,
	})
}

func (r *repository) LatestByMessageIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	messageIDs []pulid.ID,
) (map[pulid.ID]*triage.Reply, error) {
	return r.distinctByMessage(ctx, &distinctRequest{
		tenantInfo: tenantInfo,
		messageIDs: messageIDs,
	})
}

type distinctRequest struct {
	tenantInfo pagination.TenantInfo
	messageIDs []pulid.ID
	draftsOnly bool
}

func (r *repository) distinctByMessage(
	ctx context.Context,
	req *distinctRequest,
) (map[pulid.ID]*triage.Reply, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]*triage.Reply, error) {
		latest := make(map[pulid.ID]*triage.Reply, len(req.messageIDs))
		if len(req.messageIDs) == 0 {
			return latest, nil
		}

		cols := triagecol.ReplyColumns
		entities := make([]*triage.Reply, 0, len(req.messageIDs))
		query := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			DistinctOn(cols.MessageID.Qualified()).
			Where(cols.MessageID.In(), bun.List(req.messageIDs)).
			Apply(triagecol.ReplyApplyTenant(req.tenantInfo))
		if req.draftsOnly {
			query = query.Where(cols.Status.Eq(), triage.ReplyDraft)
		}
		if err := query.
			OrderExpr(cols.MessageID.OrderAsc()).
			OrderExpr(
				"CASE "+cols.Status.Qualified()+" WHEN ? THEN ? WHEN ? THEN ? ELSE ? END",
				triage.ReplySent, statusRankSent,
				triage.ReplySending, statusRankSending,
				statusRankOther,
			).
			OrderExpr(cols.CreatedAt.OrderDesc()).
			OrderExpr(cols.ID.OrderDesc()).
			Scan(ctx); err != nil {
			return nil, err
		}

		for _, entity := range entities {
			latest[entity.MessageID] = entity
		}

		return latest, nil
	})
}

func (r *repository) ThreadKeyForMessageKeys(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	messageKeys []string,
) (string, bool, error) {
	if len(messageKeys) == 0 {
		return "", false, nil
	}
	if len(messageKeys) > threadKeyLookupMaximum {
		messageKeys = messageKeys[:threadKeyLookupMaximum]
	}

	type threadRow struct {
		MessageKey string `bun:"message_key"`
		ThreadKey  string `bun:"thread_key"`
	}

	return dbtx.Read2(ctx, r.db, func(ctx context.Context) (string, bool, error) {
		cols := triagecol.ReplyColumns
		rows := make([]threadRow, 0, len(messageKeys))
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*triage.Reply)(nil)).
			Column(cols.MessageKey.Bare(), cols.ThreadKey.Bare()).
			Where(cols.MessageKey.In(), bun.List(messageKeys)).
			Where(cols.ThreadKey.Ne(), "").
			Apply(triagecol.ReplyApplyTenant(tenantInfo)).
			Scan(ctx, &rows); err != nil {
			return "", false, err
		}
		if len(rows) == 0 {
			return "", false, nil
		}

		byKey := make(map[string]string, len(rows))
		for _, row := range rows {
			byKey[row.MessageKey] = row.ThreadKey
		}
		for _, key := range messageKeys {
			if threadKey, ok := byKey[key]; ok {
				return threadKey, true, nil
			}
		}

		return "", false, nil
	})
}
