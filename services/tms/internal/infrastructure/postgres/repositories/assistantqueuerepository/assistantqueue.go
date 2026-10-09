package assistantqueuerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const entityName = "Queued message"

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.AssistantQueueRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.assistant-queue-repository"),
	}
}

func scopeSelect(
	q *bun.SelectQuery,
	scope *repositories.AssistantQueueScope,
) *bun.SelectQuery {
	cols := buncolgen.QueuedMessageColumns
	q = buncolgen.QueuedMessageScopeTenant(q, scope.TenantInfo).
		Where(cols.ThreadID.Eq(), scope.ThreadID)
	if scope.UserID.IsNotNil() {
		q = q.Where(cols.UserID.Eq(), scope.UserID)
	}

	return q
}

func scopeUpdate(
	q *bun.UpdateQuery,
	scope *repositories.AssistantQueueScope,
) *bun.UpdateQuery {
	cols := buncolgen.QueuedMessageColumns
	q = buncolgen.QueuedMessageScopeTenantUpdate(q, scope.TenantInfo).
		Where(cols.ThreadID.Eq(), scope.ThreadID)
	if scope.UserID.IsNotNil() {
		q = q.Where(cols.UserID.Eq(), scope.UserID)
	}

	return q
}

func scopeDelete(
	q *bun.DeleteQuery,
	scope *repositories.AssistantQueueScope,
) *bun.DeleteQuery {
	cols := buncolgen.QueuedMessageColumns
	q = buncolgen.QueuedMessageScopeTenantDelete(q, scope.TenantInfo).
		Where(cols.ThreadID.Eq(), scope.ThreadID)
	if scope.UserID.IsNotNil() {
		q = q.Where(cols.UserID.Eq(), scope.UserID)
	}

	return q
}

func inOrder(q *bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.QueuedMessageColumns

	return q.Order(
		cols.Position.OrderAsc(),
		cols.CreatedAt.OrderAsc(),
		cols.ID.OrderAsc(),
	)
}

func (r *repository) List(
	ctx context.Context,
	scope *repositories.AssistantQueueScope,
) ([]*conversation.QueuedMessage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*conversation.QueuedMessage, error) {
		items := make([]*conversation.QueuedMessage, 0, conversation.MaxQueuedPerThread)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&items).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return scopeSelect(sq, scope)
			}).
			Apply(inOrder).
			Scan(ctx)
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
	req *repositories.QueuedMessageRequest,
) (*conversation.QueuedMessage, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*conversation.QueuedMessage, error) {
		entity := new(conversation.QueuedMessage)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return scopeSelect(sq, &req.Scope).
					Where(buncolgen.QueuedMessageColumns.ID.Eq(), req.ID)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, entityName)
		}

		return entity, nil
	})
}

func (r *repository) Insert(
	ctx context.Context,
	entity *conversation.QueuedMessage,
) (*conversation.QueuedMessage, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*conversation.QueuedMessage, error) {
		db := r.db.DBForContext(ctx)
		if err := lockThread(ctx, db, entity.ThreadID); err != nil {
			return nil, err
		}

		cols := buncolgen.QueuedMessageColumns
		var tally struct {
			Count int   `bun:"queued"`
			Last  int64 `bun:"last"`
		}
		err := db.NewSelect().
			Model((*conversation.QueuedMessage)(nil)).
			ColumnExpr(buncolgen.Count("queued")).
			ColumnExpr(cols.Position.Expr("COALESCE(MAX({}), 0) AS last")).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return scopeSelect(sq, &repositories.AssistantQueueScope{
					ThreadID:   entity.ThreadID,
					TenantInfo: tenantOf(entity),
				})
			}).
			Scan(ctx, &tally)
		if err != nil {
			return nil, fmt.Errorf("read the conversation's queue: %w", err)
		}
		if tally.Count >= conversation.MaxQueuedPerThread {
			return nil, repositories.ErrQueueFull
		}

		entity.Position = tally.Last + 1
		if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
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
		cols := buncolgen.QueuedMessageColumns
		previous := entity.Version
		entity.Version++
		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			Column(cols.Content.Bare(), cols.Request.Bare(), cols.Version.Bare(),
				cols.UpdatedAt.Bare()).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return scopeUpdate(uq, &repositories.AssistantQueueScope{
					ThreadID:   entity.ThreadID,
					UserID:     entity.UserID,
					TenantInfo: tenantOf(entity),
				}).
					Where(cols.ID.Eq(), entity.ID).
					Where(cols.Version.Eq(), previous)
			}).
			Returning("*").
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to edit a queued message",
				zap.String("message", entity.ID.String()),
				zap.Error(err),
			)
			return nil, err
		}
		if err = dberror.CheckRowsAffected(result, entityName, entity.ID.String()); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) MarkSteer(
	ctx context.Context,
	req *repositories.QueuedMessageRequest,
) (*conversation.QueuedMessage, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*conversation.QueuedMessage, error) {
		cols := buncolgen.QueuedMessageColumns
		entity := new(conversation.QueuedMessage)
		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			Set(cols.Steer.Set(), true).
			Set(cols.Version.Inc(1)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return scopeUpdate(uq, &req.Scope).Where(cols.ID.Eq(), req.ID)
			}).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, err
		}
		if err = dberror.CheckFound(result, entityName); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Delete(ctx context.Context, req *repositories.QueuedMessageRequest) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		result, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*conversation.QueuedMessage)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return scopeDelete(dq, &req.Scope).
					Where(buncolgen.QueuedMessageColumns.ID.Eq(), req.ID)
			}).
			Exec(ctx)
		if err != nil {
			return err
		}

		return dberror.CheckFound(result, entityName)
	})
}

func (r *repository) DeleteMany(
	ctx context.Context,
	req *repositories.DeleteQueuedMessagesRequest,
) error {
	if len(req.IDs) == 0 {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		_, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*conversation.QueuedMessage)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return scopeDelete(dq, &req.Scope).
					Where(buncolgen.QueuedMessageColumns.ID.In(), bun.List(req.IDs))
			}).
			Exec(ctx)
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
	req *repositories.ReorderQueuedMessagesRequest,
) ([]*conversation.QueuedMessage, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*conversation.QueuedMessage, error) {
		db := r.db.DBForContext(ctx)
		if err := lockThread(ctx, db, req.Scope.ThreadID); err != nil {
			return nil, err
		}

		cols := buncolgen.QueuedMessageColumns
		ids := pulid.Map(req.IDs, func(id pulid.ID) string { return id.String() })
		items := make([]*conversation.QueuedMessage, 0, len(req.IDs))
		_, err := db.NewUpdate().
			Model((*conversation.QueuedMessage)(nil)).
			Set(
				cols.Position.SetExpr("array_position(?::varchar[], "+cols.ID.Bare()+")"),
				pgdialect.Array(ids),
			).
			Set(cols.Version.Inc(1)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return scopeUpdate(uq, &req.Scope).Where(cols.ID.In(), bun.List(req.IDs))
			}).
			Returning("*").
			Exec(ctx, &items)
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
	scope *repositories.AssistantQueueScope,
) (*conversation.QueuedMessage, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*conversation.QueuedMessage, error) {
		db := r.db.DBForContext(ctx)
		cols := buncolgen.QueuedMessageColumns
		head := db.NewSelect().
			Model((*conversation.QueuedMessage)(nil)).
			Column(cols.ID.Bare()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return scopeSelect(sq, scope)
			}).
			Apply(inOrder).
			Limit(1).
			For("UPDATE SKIP LOCKED")

		return r.claim(ctx, db, scope, func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where(cols.ID.Qualified()+" = (?)", head)
		})
	})
}

func (r *repository) Claim(
	ctx context.Context,
	req *repositories.QueuedMessageRequest,
) (*conversation.QueuedMessage, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*conversation.QueuedMessage, error) {
		return r.claim(ctx, r.db.DBForContext(ctx), &req.Scope,
			func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return dq.Where(buncolgen.QueuedMessageColumns.ID.Eq(), req.ID)
			})
	})
}

func (r *repository) claim(
	ctx context.Context,
	db bun.IDB,
	scope *repositories.AssistantQueueScope,
	pick func(*bun.DeleteQuery) *bun.DeleteQuery,
) (*conversation.QueuedMessage, error) {
	entity := new(conversation.QueuedMessage)
	result, err := db.NewDelete().
		Model(entity).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return pick(scopeDelete(dq, scope))
		}).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to take a queued message",
			zap.String("thread", scope.ThreadID.String()),
			zap.Error(err),
		)
		return nil, err
	}
	taken, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("count the queued message taken: %w", err)
	}
	if taken == 0 {
		return nil, repositories.ErrQueueEmpty
	}

	return entity, nil
}

func (r *repository) Restore(ctx context.Context, entity *conversation.QueuedMessage) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		_, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			On("CONFLICT DO NOTHING").
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to put a queued message back",
				zap.String("message", entity.ID.String()),
				zap.Error(err),
			)
		}

		return err
	})
}

func lockThread(ctx context.Context, db bun.IDB, threadID pulid.ID) error {
	if _, err := db.NewRaw(
		"SELECT pg_advisory_xact_lock(hashtextextended(?, 0))",
		"assistant-queue:"+threadID.String(),
	).Exec(ctx); err != nil {
		return fmt.Errorf("hold the conversation's queue: %w", err)
	}

	return nil
}

func tenantOf(entity *conversation.QueuedMessage) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}
