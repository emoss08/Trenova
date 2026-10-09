package tablelayoutrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tablelayout"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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

func New(p Params) repositories.TableLayoutRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.table-layout-repository"),
	}
}

func (r *repository) Get(
	ctx context.Context,
	req *repositories.TableLayoutRequest,
) (*tablelayout.TableLayout, bool, error) {
	return dbtx.Read2(ctx, r.db, func(ctx context.Context) (*tablelayout.TableLayout, bool, error) {
		cols := buncolgen.TableLayoutColumns
		entity := new(tablelayout.TableLayout)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.TableLayoutScopeTenant(sq, req.TenantInfo).
					Where(cols.UserID.Eq(), req.TenantInfo.UserID).
					Where(cols.Resource.Eq(), req.Resource)
			}).
			Scan(ctx)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, false, nil
			}
			r.l.Error("failed to get table layout",
				zap.String("resource", req.Resource),
				zap.String("userID", req.TenantInfo.UserID.String()),
				zap.Error(err),
			)
			return nil, false, err
		}

		return entity, true, nil
	})
}

func (r *repository) CountForUser(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		count, err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*tablelayout.TableLayout)(nil)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.TableLayoutScopeTenant(sq, tenantInfo).
					Where(buncolgen.TableLayoutColumns.UserID.Eq(), tenantInfo.UserID)
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count table layouts",
				zap.String("userID", tenantInfo.UserID.String()),
				zap.Error(err),
			)
			return 0, err
		}

		return count, nil
	})
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *tablelayout.TableLayout,
) (*tablelayout.TableLayout, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tablelayout.TableLayout, error) {
		cols := buncolgen.TableLayoutColumns
		_, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			On("CONFLICT (" +
				cols.OrganizationID.Bare() + ", " +
				cols.BusinessUnitID.Bare() + ", " +
				cols.UserID.Bare() + ", " +
				cols.Resource.Bare() + ") DO UPDATE").
			Set(cols.Layout.SetExcluded()).
			Set(cols.UpdatedAt.SetExcluded()).
			Set(cols.Version.IncConflict(1)).
			Returning("*").
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to save table layout",
				zap.String("resource", entity.Resource),
				zap.String("userID", entity.UserID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Delete(ctx context.Context, req *repositories.TableLayoutRequest) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.TableLayoutColumns
		_, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*tablelayout.TableLayout)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.TableLayoutScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.UserID.Eq(), req.TenantInfo.UserID).
					Where(cols.Resource.Eq(), req.Resource)
			}).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to reset table layout",
				zap.String("resource", req.Resource),
				zap.String("userID", req.TenantInfo.UserID.String()),
				zap.Error(err),
			)
			return err
		}

		return nil
	})
}
