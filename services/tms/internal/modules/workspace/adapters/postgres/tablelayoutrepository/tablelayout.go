package tablelayoutrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablelayout"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablelayout/tablelayoutdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/tablelayoutports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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

func New(p Params) tablelayoutports.TableLayoutRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.table-layout-repository"),
	}
}

func layoutOf(req *tablelayoutports.TableLayoutRequest) *keel.Query[*tablelayout.TableLayout] {
	tl := tablelayoutdb.TableLayouts
	return tl.Where(tl.UserID.Eq(req.TenantInfo.UserID), tl.Resource.Eq(req.Resource))
}

func (r *repository) Get(
	ctx context.Context,
	req *tablelayoutports.TableLayoutRequest,
) (*tablelayout.TableLayout, bool, error) {
	return dbtx.Read2(ctx, r.db, func(ctx context.Context) (*tablelayout.TableLayout, bool, error) {
		entity, err := layoutOf(req).One(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return nil, false, nil
		}
		if err != nil {
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
		tl := tablelayoutdb.TableLayouts
		count, err := tl.Where(tl.UserID.Eq(tenantInfo.UserID)).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count table layouts",
				zap.String("userID", tenantInfo.UserID.String()),
				zap.Error(err),
			)
			return 0, err
		}

		return int(count), nil
	})
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *tablelayout.TableLayout,
) (*tablelayout.TableLayout, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tablelayout.TableLayout, error) {
		tl := tablelayoutdb.TableLayouts
		db := r.db.Keel(ctx)
		inserted, err := tl.Upsert(
			ctx,
			db,
			[]*tablelayout.TableLayout{entity},
			keel.OnConflict(tl.OrganizationID, tl.BusinessUnitID, tl.UserID, tl.Resource).
				Update(tl.Layout),
		)
		if err == nil && !inserted[0] {
			var stored *tablelayout.TableLayout
			stored, err = tl.Where(tl.UserID.Eq(entity.UserID), tl.Resource.Eq(entity.Resource)).
				One(ctx, db)
			if err == nil {
				entity = stored
			}
		}
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

func (r *repository) Delete(ctx context.Context, req *tablelayoutports.TableLayoutRequest) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if _, err := layoutOf(req).Delete(ctx, r.db.Keel(ctx)); err != nil {
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
