package assistantqueuerepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const entityName = "Queued message"

var queuedMessageErrors = dberror.Mapping{Entity: entityName}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentports.AssistantQueueRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.assistant-queue-repository"),
	}
}

type queueTally struct {
	Count int64
	Last  int64
}

func inScope(scope *agentports.AssistantQueueScope) keel.Pred {
	q := conversationdb.QueuedMessages

	return keel.And(
		q.ThreadID.Eq(scope.ThreadID),
		keel.When(scope.UserID.IsNotNil(), q.UserID.Eq(scope.UserID)),
	)
}

func inOrder() []keel.Order {
	q := conversationdb.QueuedMessages

	return []keel.Order{q.Position.Asc(), q.CreatedAt.Asc(), q.ID.Asc()}
}

func listQuery(scope *agentports.AssistantQueueScope) *keel.Query[conversation.QueuedMessage] {
	return conversationdb.QueuedMessages.Where(inScope(scope)).Order(inOrder()...)
}

func getQuery(req *agentports.QueuedMessageRequest) *keel.Query[conversation.QueuedMessage] {
	q := conversationdb.QueuedMessages

	return q.Where(inScope(&req.Scope), q.ID.Eq(req.ID))
}

func tallyQuery(threadID pulid.ID) *keel.Query[conversation.QueuedMessage] {
	q := conversationdb.QueuedMessages

	return q.Pick(keel.Count(), keel.DynOf(q.Position).Max().Coalesce(0)).
		Where(inScope(&agentports.AssistantQueueScope{ThreadID: threadID}))
}

func editQuery(
	entity *conversation.QueuedMessage,
	version int64,
) *keel.Query[conversation.QueuedMessage] {
	q := conversationdb.QueuedMessages

	return q.Where(
		inScope(&agentports.AssistantQueueScope{ThreadID: entity.ThreadID, UserID: entity.UserID}),
		q.ID.Eq(entity.ID),
		q.Version.Eq(version),
	)
}

func reorderQuery(
	req *agentports.ReorderQueuedMessagesRequest,
) *keel.Query[conversation.QueuedMessage] {
	q := conversationdb.QueuedMessages

	return q.Where(inScope(&req.Scope), q.ID.In(req.IDs...))
}

func reorderPosition(ids []pulid.ID) keel.Assignment {
	q := conversationdb.QueuedMessages

	return q.Position.ToExpr(keel.DynFunc(
		"array_position",
		pulid.Map(ids, func(id pulid.ID) string { return id.String() }),
		q.ID,
	))
}

func headQuery(scope *agentports.AssistantQueueScope) *keel.Query[conversation.QueuedMessage] {
	return listQuery(scope).Limit(1).Lock(keel.ForUpdate, keel.SkipLocked)
}

func claimQuery(req *agentports.QueuedMessageRequest) *keel.Query[conversation.QueuedMessage] {
	return getQuery(req).Lock(keel.ForUpdate)
}

func (r *repository) List(
	ctx context.Context,
	scope *agentports.AssistantQueueScope,
) ([]*conversation.QueuedMessage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*conversation.QueuedMessage, error) {
		items, err := listQuery(scope).All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list queued messages",
				zap.String("thread", scope.ThreadID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		return items, nil
	})
}

func (r *repository) Get(
	ctx context.Context,
	req *agentports.QueuedMessageRequest,
) (*conversation.QueuedMessage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*conversation.QueuedMessage, error) {
		entity, err := getQuery(req).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, queuedMessageErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Insert(
	ctx context.Context,
	entity *conversation.QueuedMessage,
) (*conversation.QueuedMessage, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*conversation.QueuedMessage, error) {
		db := r.db.Keel(ctx)
		if err := lockThread(ctx, db, entity.ThreadID); err != nil {
			return nil, err
		}

		tally, err := keel.IntoOne[queueTally](ctx, db, tallyQuery(entity.ThreadID))
		if err != nil {
			return nil, fmt.Errorf("read the conversation's queue: %w", err)
		}
		if tally.Count >= conversation.MaxQueuedPerThread {
			return nil, agentports.ErrQueueFull
		}

		entity.Position = tally.Last + 1
		if err = conversationdb.QueuedMessages.Create(ctx, db, entity); err != nil {
			r.l.Error("failed to queue a message",
				zap.String("thread", entity.ThreadID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *conversation.QueuedMessage,
) (*conversation.QueuedMessage, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*conversation.QueuedMessage, error) {
		q := conversationdb.QueuedMessages
		rows, err := editQuery(entity, entity.Version).UpdateReturning(
			ctx,
			r.db.Keel(ctx),
			q.Content.To(entity.Content),
			q.Request.To(entity.Request),
		)
		if err != nil {
			r.l.Error("failed to edit a queued message",
				zap.String("message", entity.ID.String()),
				zap.Error(err),
			)
			return nil, err
		}
		if len(rows) == 0 {
			return nil, dberror.CreateVersionMismatchError(entityName, entity.ID.String())
		}

		return rows[0], nil
	})
}

func (r *repository) MarkSteer(
	ctx context.Context,
	req *agentports.QueuedMessageRequest,
) (*conversation.QueuedMessage, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*conversation.QueuedMessage, error) {
		q := conversationdb.QueuedMessages
		rows, err := getQuery(req).UpdateReturning(ctx, r.db.Keel(ctx), q.Steer.To(true))
		if err != nil {
			return nil, err
		}
		if err = dberror.CheckFoundCount(int64(len(rows)), entityName); err != nil {
			return nil, err
		}

		return rows[0], nil
	})
}

func (r *repository) Delete(ctx context.Context, req *agentports.QueuedMessageRequest) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		n, err := getQuery(req).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return err
		}

		return dberror.CheckFoundCount(n, entityName)
	})
}

func (r *repository) DeleteMany(
	ctx context.Context,
	req *agentports.DeleteQueuedMessagesRequest,
) error {
	if len(req.IDs) == 0 {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		q := conversationdb.QueuedMessages
		_, err := q.Where(inScope(&req.Scope), q.ID.In(req.IDs...)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to clear read queued messages",
				zap.String("thread", req.Scope.ThreadID.String()),
				zap.Error(err),
			)
		}

		return err
	})
}

func (r *repository) Reorder(
	ctx context.Context,
	req *agentports.ReorderQueuedMessagesRequest,
) ([]*conversation.QueuedMessage, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*conversation.QueuedMessage, error) {
		db := r.db.Keel(ctx)
		if err := lockThread(ctx, db, req.Scope.ThreadID); err != nil {
			return nil, err
		}

		items, err := reorderQuery(req).UpdateReturning(ctx, db, reorderPosition(req.IDs))
		if err != nil {
			r.l.Error("failed to reorder queued messages",
				zap.String("thread", req.Scope.ThreadID.String()),
				zap.Error(err),
			)
			return nil, err
		}
		if len(items) != len(req.IDs) {
			return nil, dberror.CreateBulkVersionMismatchError(entityName, req.IDs)
		}

		return items, nil
	})
}

func (r *repository) ClaimNext(
	ctx context.Context,
	scope *agentports.AssistantQueueScope,
) (*conversation.QueuedMessage, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*conversation.QueuedMessage, error) {
		return r.claim(ctx, scope, headQuery(scope))
	})
}

func (r *repository) Claim(
	ctx context.Context,
	req *agentports.QueuedMessageRequest,
) (*conversation.QueuedMessage, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*conversation.QueuedMessage, error) {
		return r.claim(ctx, &req.Scope, claimQuery(req))
	})
}

func (r *repository) claim(
	ctx context.Context,
	scope *agentports.AssistantQueueScope,
	pick *keel.Query[conversation.QueuedMessage],
) (*conversation.QueuedMessage, error) {
	db := r.db.Keel(ctx)
	entity, err := pick.First(ctx, db)
	if errors.Is(err, keel.ErrNotFound) {
		return nil, agentports.ErrQueueEmpty
	}
	if err == nil {
		err = conversationdb.QueuedMessages.DeleteRow(ctx, db, entity)
		if errors.Is(err, keel.ErrNotFound) {
			return nil, agentports.ErrQueueEmpty
		}
	}
	if err != nil {
		r.l.Error("failed to take a queued message",
			zap.String("thread", scope.ThreadID.String()),
			zap.Error(err),
		)
		return nil, err
	}

	return entity, nil
}

func (r *repository) Restore(ctx context.Context, entity *conversation.QueuedMessage) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		_, err := conversationdb.QueuedMessages.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*conversation.QueuedMessage{entity},
		)
		if err != nil {
			r.l.Error("failed to put a queued message back",
				zap.String("message", entity.ID.String()),
				zap.Error(err),
			)
		}

		return err
	})
}

func lockThread(ctx context.Context, db *keel.DB, threadID pulid.ID) error {
	if err := keel.LockKey(ctx, db, "assistant-queue:"+threadID.String()); err != nil {
		return fmt.Errorf("hold the conversation's queue: %w", err)
	}

	return nil
}
