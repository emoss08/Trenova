package conversationrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultThreadLimit = 50

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.ConversationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.conversation-repository"),
	}
}

func (r *repository) CreateThread(
	ctx context.Context,
	thread *conversation.Thread,
) (*conversation.Thread, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(thread).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create thread", zap.Error(err))
		return nil, err
	}

	return thread, nil
}

// GetThread scopes by user as well as tenant. A thread reveals what someone was
// investigating, so belonging to the same organization is not enough to read it.
func (r *repository) GetThread(
	ctx context.Context,
	req repositories.GetThreadRequest,
) (*conversation.Thread, error) {
	cols := buncolgen.ThreadColumns
	entity := new(conversation.Thread)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ThreadScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID).
				Where(cols.UserID.Eq(), req.UserID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Thread")
	}

	return entity, nil
}

func (r *repository) GetThreadOwned(
	ctx context.Context,
	req repositories.GetThreadOwnedRequest,
) (*conversation.Thread, error) {
	cols := buncolgen.ThreadColumns
	entity := new(conversation.Thread)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ThreadScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Thread")
	}

	return entity, nil
}

func (r *repository) ListThreads(
	ctx context.Context,
	req repositories.ListThreadsRequest,
) (*pagination.ListResult[*conversation.Thread], error) {
	cols := buncolgen.ThreadColumns

	limit := req.Limit
	if limit <= 0 || limit > defaultThreadLimit {
		limit = defaultThreadLimit
	}

	entities := make([]*conversation.Thread, 0, limit)
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.ThreadScopeTenant(sq, req.TenantInfo).
				Where(cols.UserID.Eq(), req.UserID)
			if !req.IncludeUnlisted {
				sq = sq.Where(cols.Origin.NotIn(), bun.In(conversation.UnlistedOrigins()))
			}

			return sq
		}).
		Order(cols.Pinned.OrderDesc(), cols.LastMessageAt.OrderDesc(), cols.CreatedAt.OrderDesc()).
		Limit(limit).
		Offset(req.Offset).
		ScanAndCount(ctx)
	if err != nil {
		r.l.Error("failed to list threads", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*conversation.Thread]{Items: entities, Total: total}, nil
}

func (r *repository) UpdateThread(
	ctx context.Context,
	thread *conversation.Thread,
) (*conversation.Thread, error) {
	cols := buncolgen.ThreadColumns
	ov := thread.Version
	thread.Version++

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(thread).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ThreadScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: thread.OrganizationID,
				BuID:  thread.BusinessUnitID,
			}).Where(cols.ID.Eq(), thread.ID).
				Where(cols.UserID.Eq(), thread.UserID).
				Where(cols.Version.Eq(), ov)
		}).
		Set(cols.Title.Set(), thread.Title).
		Set(cols.Status.Set(), thread.Status).
		Set(cols.PreferredProviderID.Set(), thread.PreferredProviderID).
		Set(cols.Origin.Set(), thread.Origin).
		Set(cols.Pinned.Set(), thread.Pinned).
		Set(cols.SubjectType.Set(), thread.SubjectType).
		Set(cols.SubjectID.Set(), thread.SubjectID).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Set(cols.Version.Set(), thread.Version).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update thread: %w", err)
	}

	if err = dberror.CheckRowsAffected(res, "Thread", thread.ID.String()); err != nil {
		return nil, err
	}

	return thread, nil
}

func (r *repository) MarkThreadTainted(
	ctx context.Context,
	req repositories.MarkThreadTaintedRequest,
) error {
	if !req.Taint.Tainted() {
		return nil
	}

	cols := buncolgen.ThreadColumns
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*conversation.Thread)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ThreadScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ThreadID)
		}).
		Set(cols.Taint.Set(), req.Taint).
		Set(cols.TaintedAt.SetExpr("COALESCE({}, ?)"), req.TaintedAt).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("mark thread tainted: %w", err)
	}

	return dberror.CheckRowsAffected(res, "Thread", req.ThreadID.String())
}

func (r *repository) DeleteThread(
	ctx context.Context,
	req repositories.GetThreadRequest,
) error {
	cols := buncolgen.ThreadColumns

	// Messages are removed explicitly rather than by cascade: they carry no
	// foreign key to the thread, since a thread's primary key is composite and a
	// per-message reference to all three columns would buy nothing.
	err := r.db.DB().RunInTx(ctx, nil, func(txCtx context.Context, tx bun.Tx) error {
		if _, txErr := tx.NewDelete().
			Model((*conversation.Message)(nil)).
			Where(buncolgen.MessageColumns.ThreadID.Eq(), req.ID).
			Where(buncolgen.MessageColumns.OrganizationID.Eq(), req.TenantInfo.OrgID).
			Where(buncolgen.MessageColumns.BusinessUnitID.Eq(), req.TenantInfo.BuID).
			Exec(txCtx); txErr != nil {
			return txErr
		}

		res, txErr := tx.NewDelete().
			Model((*conversation.Thread)(nil)).
			Where(cols.ID.Eq(), req.ID).
			Where(cols.UserID.Eq(), req.UserID).
			Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
			Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
			Exec(txCtx)
		if txErr != nil {
			return txErr
		}

		return dberror.CheckRowsAffected(res, "Thread", req.ID.String())
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) ListMessages(
	ctx context.Context,
	req repositories.ListMessagesRequest,
) ([]conversation.Message, error) {
	cols := buncolgen.MessageColumns

	messages := make([]conversation.Message, 0, 32)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&messages).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.MessageScopeTenant(sq, req.TenantInfo).
				Where(cols.ThreadID.Eq(), req.ThreadID)
		})

	if req.BeforeSequence != nil {
		query = query.Where(cols.Sequence.Lt(), *req.BeforeSequence)
	}

	if len(req.ExcludeKinds) > 0 {
		query = query.Where(cols.Kind.NotIn(), bun.List(req.ExcludeKinds))
	}

	if len(req.Kinds) > 0 {
		query = query.Where(cols.Kind.In(), bun.List(req.Kinds))
	}

	if req.Limit > 0 {
		// Taking the newest N and reversing keeps the most recent context rather
		// than the oldest, which is what a long conversation needs.
		if err := query.Order(cols.Sequence.OrderDesc()).Limit(req.Limit).Scan(ctx); err != nil {
			return nil, err
		}
		reverse(messages)

		return messages, nil
	}

	if err := query.Order(cols.Sequence.OrderAsc()).Scan(ctx); err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *repository) CountMessages(
	ctx context.Context,
	req repositories.CountMessagesRequest,
) (int, error) {
	cols := buncolgen.MessageColumns

	count, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*conversation.Message)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.MessageScopeTenant(sq, req.TenantInfo).
				Where(cols.ThreadID.Eq(), req.ThreadID)
		}).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count messages: %w", err)
	}

	return count, nil
}

func reverse(messages []conversation.Message) {
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
}

// AppendTurn numbers and writes a turn's messages in one transaction.
//
// Sequence is allocated from the current maximum inside the transaction, and the
// unique index on (thread_id, sequence) is what makes that safe: two concurrent
// turns cannot both claim the same number, and the loser fails rather than
// silently interleaving. Writing the messages together matters because a turn
// saved halfway would leave a tool call with no result, which breaks the next
// replay.
func (r *repository) AppendTurn(
	ctx context.Context,
	req repositories.AppendTurnRequest,
) ([]conversation.Message, error) {
	if len(req.Messages) == 0 {
		return nil, nil
	}

	saved := make([]conversation.Message, 0, len(req.Messages))

	err := r.db.DB().RunInTx(ctx, nil, func(txCtx context.Context, tx bun.Tx) error {
		cols := buncolgen.MessageColumns

		var maxSequence int
		err := tx.NewSelect().
			Model((*conversation.Message)(nil)).
			ColumnExpr("COALESCE(MAX(?), -1)", bun.Ident(cols.Sequence.Bare())).
			Where(cols.ThreadID.Eq(), req.ThreadID).
			Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
			Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
			Scan(txCtx, &maxSequence)
		if err != nil {
			return fmt.Errorf("read current sequence: %w", err)
		}

		now := timeutils.NowUnix()
		batch := make([]conversation.Message, 0, len(req.Messages))
		for idx := range req.Messages {
			msg := req.Messages[idx]
			msg.ThreadID = req.ThreadID
			msg.OrganizationID = req.TenantInfo.OrgID
			msg.BusinessUnitID = req.TenantInfo.BuID
			msg.Sequence = maxSequence + 1 + idx
			batch = append(batch, msg)
		}
		conversation.StampUnstamped(batch, now)

		if _, err = tx.NewInsert().Model(&batch).Returning("*").Exec(txCtx); err != nil {
			return fmt.Errorf("insert turn: %w", err)
		}

		threadCols := buncolgen.ThreadColumns
		if _, err = tx.NewUpdate().
			Model((*conversation.Thread)(nil)).
			Where(threadCols.ID.Eq(), req.ThreadID).
			Where(threadCols.OrganizationID.Eq(), req.TenantInfo.OrgID).
			Where(threadCols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
			Set(threadCols.LastMessageAt.Set(), now).
			Set(threadCols.UpdatedAt.Set(), now).
			Exec(txCtx); err != nil {
			return fmt.Errorf("touch thread: %w", err)
		}

		saved = batch

		return nil
	})
	if err != nil {
		return nil, err
	}

	return saved, nil
}

// DeleteStaleThreads removes unkept conversations of one origin whose last
// activity is older than the cut-off. Threads are picked first, bounded by
// the limit, then their messages and the threads go in one transaction;
// artifacts follow the thread by cascade.
func (r *repository) DeleteStaleThreads(
	ctx context.Context,
	req repositories.DeleteStaleThreadsRequest,
) (int, error) {
	if req.Origin == "" || req.Before <= 0 {
		return 0, nil
	}
	limit := req.Limit
	if limit <= 0 || limit > defaultThreadLimit*10 {
		limit = defaultThreadLimit * 10
	}

	cols := buncolgen.ThreadColumns

	deleted := 0
	err := r.db.DB().RunInTx(ctx, nil, func(txCtx context.Context, tx bun.Tx) error {
		var stale []conversation.Thread
		q := tx.NewSelect().
			Model(&stale).
			Column(cols.ID.Bare(), cols.OrganizationID.Bare(), cols.BusinessUnitID.Bare()).
			Where(cols.Origin.Eq(), req.Origin).
			Where(cols.LastMessageAt.Lt(), req.Before).
			Where(cols.CreatedAt.Lt(), req.Before)
		if req.SubjectlessOnly {
			q = q.Where(cols.SubjectID.IsNull())
		}
		if err := q.Order(cols.LastMessageAt.OrderAsc()).
			Limit(limit).
			Scan(txCtx); err != nil {
			return err
		}
		if len(stale) == 0 {
			return nil
		}

		for i := range stale {
			thread := &stale[i]
			if _, err := tx.NewDelete().
				Model((*conversation.Message)(nil)).
				Where(buncolgen.MessageColumns.ThreadID.Eq(), thread.ID).
				Where(buncolgen.MessageColumns.OrganizationID.Eq(), thread.OrganizationID).
				Where(buncolgen.MessageColumns.BusinessUnitID.Eq(), thread.BusinessUnitID).
				Exec(txCtx); err != nil {
				return err
			}
			res, err := tx.NewDelete().
				Model((*conversation.Thread)(nil)).
				Where(cols.ID.Eq(), thread.ID).
				Where(cols.OrganizationID.Eq(), thread.OrganizationID).
				Where(cols.BusinessUnitID.Eq(), thread.BusinessUnitID).
				Exec(txCtx)
			if err != nil {
				return err
			}
			if affected, _ := res.RowsAffected(); affected > 0 {
				deleted++
			}
		}

		return nil
	})
	if err != nil {
		r.l.Error("failed to delete stale threads",
			zap.String("origin", string(req.Origin)),
			zap.Error(err),
		)

		return 0, err
	}

	return deleted, nil
}
