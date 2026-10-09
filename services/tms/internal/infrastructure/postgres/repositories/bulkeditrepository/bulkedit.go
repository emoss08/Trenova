package bulkeditrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/bulkedit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxListed = 50

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.BulkEditRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.bulk-edit-repository")}
}

func (r *repository) Create(
	ctx context.Context,
	entity *bulkedit.BulkEdit,
) (*bulkedit.BulkEdit, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*bulkedit.BulkEdit, error) {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
			r.l.Error("failed to create bulk edit", zap.Error(err))
			return nil, err
		}
		return entity, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetBulkEditRequest,
) (*bulkedit.BulkEdit, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*bulkedit.BulkEdit, error) {
		entity := new(bulkedit.BulkEdit)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.BulkEditScopeTenant(sq, req.TenantInfo).
					Where(buncolgen.BulkEditColumns.ID.Eq(), req.ID)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "Bulk edit")
		}
		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *bulkedit.BulkEdit,
) (*bulkedit.BulkEdit, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*bulkedit.BulkEdit, error) {
		previous := entity.Version
		entity.Version++

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(buncolgen.BulkEditColumns.Version.Eq(), previous).
			Returning("*").
			Exec(ctx)
		if err != nil {
			entity.Version = previous
			r.l.Error("failed to update bulk edit", zap.Error(err))
			return nil, err
		}
		if err = dberror.CheckRowsAffected(result, "Bulk edit", entity.ID.String()); err != nil {
			entity.Version = previous
			return nil, err
		}
		return entity, nil
	})
}

func (r *repository) ListForUser(
	ctx context.Context,
	req *repositories.ListBulkEditsRequest,
) ([]*bulkedit.BulkEdit, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*bulkedit.BulkEdit, error) {
		cols := buncolgen.BulkEditColumns
		limit := req.Limit
		if limit <= 0 || limit > maxListed {
			limit = maxListed
		}

		entities := make([]*bulkedit.BulkEdit, 0, limit)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			ExcludeColumn(cols.Targets.Bare()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = buncolgen.BulkEditScopeTenant(sq, req.TenantInfo).
					Where(cols.UserID.Eq(), req.TenantInfo.UserID)
				if req.Resource != "" {
					sq = sq.Where(cols.Resource.Eq(), req.Resource)
				}
				return sq
			}).
			Order(cols.CreatedAt.OrderDesc()).
			Limit(limit).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list bulk edits", zap.Error(err))
			return nil, err
		}
		return entities, nil
	})
}
