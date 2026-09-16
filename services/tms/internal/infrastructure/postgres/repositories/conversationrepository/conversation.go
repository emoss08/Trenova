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
			return buncolgen.ThreadScopeTenant(sq, req.TenantInfo).
				Where(cols.UserID.Eq(), req.UserID)
		}).
		Order(cols.LastMessageAt.OrderDesc(), cols.CreatedAt.OrderDesc()).
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
			msg.CreatedAt = now
			batch = append(batch, msg)
		}

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
