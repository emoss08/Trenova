package inboundreplyrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage/triagedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const (
	statusRankSent         = 0
	statusRankSending      = 1
	statusRankOther        = 2
	threadKeyLookupMaximum = 50
)

var (
	replyErrors   = dberror.Mapping{Entity: "InboundReply"}
	messageErrors = dberror.Mapping{Entity: "InboundMessage"}
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
		if err := triagedb.Replies.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, replyErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(ctx context.Context, entity *triage.Reply) (*triage.Reply, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.Reply, error) {
		return save(ctx, r.db.Keel(ctx), entity)
	})
}

func save(ctx context.Context, db *keel.DB, entity *triage.Reply) (*triage.Reply, error) {
	if err := triagedb.Replies.Save(ctx, db, entity); err != nil {
		return nil, replyErrors.Map(err, entity.ID.String())
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req inboundmessageports.GetInboundReplyRequest,
) (*triage.Reply, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triage.Reply, error) {
		entity, err := triagedb.Replies.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, replyErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req inboundmessageports.GetInboundRepliesRequest,
) ([]*triage.Reply, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triage.Reply, error) {
		if len(req.IDs) == 0 {
			return []*triage.Reply{}, nil
		}

		rp := triagedb.Replies
		entities, err := rp.Where(rp.ID.In(req.IDs...)).
			Limit(len(req.IDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}
		if entities == nil {
			entities = []*triage.Reply{}
		}

		return entities, nil
	})
}

func (r *repository) GetByIdempotencyKey(
	ctx context.Context,
	req inboundmessageports.GetInboundReplyByKeyRequest,
) (*triage.Reply, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triage.Reply, error) {
		rp := triagedb.Replies
		entity, err := rp.Where(rp.IdempotencyKey.Eq(req.IdempotencyKey)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, replyErrors.Map(err, req.IdempotencyKey)
		}

		return entity, nil
	})
}

func (r *repository) UpsertDraft(ctx context.Context, entity *triage.Reply) (*triage.Reply, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.Reply, error) {
		db := r.db.Keel(ctx)
		if err := lockMessage(ctx, db, entity.MessageID); err != nil {
			return nil, err
		}

		rp := triagedb.Replies
		drafts, err := rp.Where(rp.MessageID.Eq(entity.MessageID), rp.Status.Eq(triage.ReplyDraft)).
			Order(rp.CreatedAt.Desc()).
			Limit(1).
			List(ctx, db)
		if err != nil {
			return nil, err
		}
		if len(drafts) == 0 {
			entity.Status = triage.ReplyDraft
			if err = rp.Create(ctx, db, entity); err != nil {
				return nil, replyErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		}

		existing := drafts[0]
		existing.Subject = entity.Subject
		existing.Body = entity.Body
		existing.DraftedByNova = entity.DraftedByNova
		existing.TemplateID = entity.TemplateID
		existing.ToAddress = entity.ToAddress
		existing.FromAddress = entity.FromAddress
		existing.CcAddresses = entity.CcAddresses
		existing.Attachments = entity.Attachments
		if entity.CreatedByID.IsNotNil() {
			existing.CreatedByID = entity.CreatedByID
		}

		return save(ctx, db, existing)
	})
}

func lockMessage(ctx context.Context, db *keel.DB, messageID pulid.ID) error {
	m := inboundmessagedb.InboundMessages
	if _, err := m.Pick(m.ID).Where(m.ID.Eq(messageID)).Lock(keel.ForUpdate).One(ctx, db); err != nil {
		return messageErrors.Map(err, messageID.String())
	}

	return nil
}

func (r *repository) DeleteDraft(
	ctx context.Context,
	req inboundmessageports.InboundReplyMessageRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		rp := triagedb.Replies
		n, err := rp.Where(rp.MessageID.Eq(req.MessageID), rp.Status.Eq(triage.ReplyDraft)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, err
		}

		return int(n), nil
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

		rp := triagedb.Replies
		entities, err := rp.Where(
			rp.MessageID.In(req.MessageIDs...),
			rp.Status.Ne(triage.ReplyDraft),
		).
			Order(rp.MessageID.Asc(), rp.CreatedAt.Asc(), rp.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
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
	return r.distinctByMessage(ctx, req.MessageIDs, true)
}

func (r *repository) LatestByMessageIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	messageIDs []pulid.ID,
) (map[pulid.ID]*triage.Reply, error) {
	return r.distinctByMessage(ctx, messageIDs, false)
}

func (r *repository) distinctByMessage(
	ctx context.Context,
	messageIDs []pulid.ID,
	draftsOnly bool,
) (map[pulid.ID]*triage.Reply, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]*triage.Reply, error) {
		latest := make(map[pulid.ID]*triage.Reply, len(messageIDs))
		if len(messageIDs) == 0 {
			return latest, nil
		}

		rp := triagedb.Replies
		entities, err := rp.DistinctOn(rp.MessageID).
			Where(
				rp.MessageID.In(messageIDs...),
				keel.When(draftsOnly, rp.Status.Eq(triage.ReplyDraft)),
			).
			Order(
				rp.MessageID.Asc(),
				keel.Asc(keel.Raw(
					"CASE ? WHEN ? THEN ?::int WHEN ? THEN ?::int ELSE ?::int END",
					rp.Status,
					string(triage.ReplySent), statusRankSent,
					string(triage.ReplySending), statusRankSending,
					statusRankOther,
				)),
				rp.CreatedAt.Desc(),
				rp.ID.Desc(),
			).
			Limit(len(messageIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
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
	_ pagination.TenantInfo,
	messageKeys []string,
) (threadKey string, found bool, err error) {
	if len(messageKeys) == 0 {
		return "", false, nil
	}
	if len(messageKeys) > threadKeyLookupMaximum {
		messageKeys = messageKeys[:threadKeyLookupMaximum]
	}

	type threadRow struct {
		MessageKey string
		ThreadKey  string
	}

	return dbtx.Read2(ctx, r.db, func(ctx context.Context) (string, bool, error) {
		rp := triagedb.Replies
		rows, scanErr := keel.Into[threadRow](ctx, r.db.Keel(ctx), rp.
			Pick(rp.MessageKey, rp.ThreadKey).
			Where(rp.MessageKey.In(messageKeys...), rp.ThreadKey.Ne("")))
		if scanErr != nil {
			return "", false, scanErr
		}
		if len(rows) == 0 {
			return "", false, nil
		}

		byKey := make(map[string]string, len(rows))
		for _, row := range rows {
			byKey[row.MessageKey] = row.ThreadKey
		}
		for _, key := range messageKeys {
			if known, ok := byKey[key]; ok {
				return known, true, nil
			}
		}

		return "", false, nil
	})
}
