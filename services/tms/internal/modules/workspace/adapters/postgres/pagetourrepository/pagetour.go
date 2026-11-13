package pagetourrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagetour"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagetour/pagetourdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/pagetourports"
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

func New(p Params) pagetourports.Repository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.page-tour-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*pagetour.View, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*pagetour.View, error) {
		v := pagetourdb.Views
		views, err := v.Where(v.UserID.Eq(tenantInfo.UserID)).
			Order(v.Tour.Asc()).
			Limit(pagetour.MaxViewsPerUser).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list page tour views",
				zap.String("userID", tenantInfo.UserID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		return views, nil
	})
}

func (r *repository) Exists(ctx context.Context, req *pagetourports.ViewRequest) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		v := pagetourdb.Views
		found, err := v.Where(v.UserID.Eq(req.TenantInfo.UserID), v.Tour.Eq(req.Tour)).
			Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to look up a page tour view",
				zap.String("tour", req.Tour),
				zap.Error(err),
			)
			return false, err
		}

		return found, nil
	})
}

func (r *repository) CountForUser(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		v := pagetourdb.Views
		count, err := v.Where(v.UserID.Eq(tenantInfo.UserID)).
			CountUpTo(ctx, r.db.Keel(ctx), pagetour.MaxViewsPerUser)
		if err != nil {
			r.l.Error("failed to count page tour views",
				zap.String("userID", tenantInfo.UserID.String()),
				zap.Error(err),
			)
			return 0, err
		}

		return int(count), nil
	})
}

func (r *repository) Upsert(ctx context.Context, entity *pagetour.View) (*pagetour.View, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*pagetour.View, error) {
		v := pagetourdb.Views
		db := r.db.Keel(ctx)
		inserted, err := v.Upsert(
			ctx,
			db,
			[]*pagetour.View{entity},
			keel.OnConflict(v.OrganizationID, v.BusinessUnitID, v.UserID, v.Tour).
				Update(v.TourVersion, v.Outcome, v.SeenAt),
		)
		if err == nil && !inserted[0] {
			var stored *pagetour.View
			stored, err = v.Where(v.UserID.Eq(entity.UserID), v.Tour.Eq(entity.Tour)).One(ctx, db)
			if err == nil {
				entity = stored
			}
		}
		if err != nil {
			r.l.Error("failed to record a page tour view",
				zap.String("tour", entity.Tour),
				zap.Error(err),
			)
			return nil, err
		}

		return entity, nil
	})
}
