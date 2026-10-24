package sidebarpreferencerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/sidebarpreference"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/sidebarpreference/sidebarpreferencedb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/sidebarpreferenceports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const entityName = "SidebarPreference"

var sidebarPreferenceErrors = dberror.Mapping{Entity: entityName}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) sidebarpreferenceports.SidebarPreferenceRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.sidebar-preference-repository"),
	}
}

func (r *repository) Get(
	ctx context.Context,
	req *sidebarpreferenceports.GetSidebarPreferenceRequest,
) (*sidebarpreference.SidebarPreference, bool, error) {
	return dbtx.Read2(
		ctx,
		r.db,
		func(ctx context.Context) (*sidebarpreference.SidebarPreference, bool, error) {
			sp := sidebarpreferencedb.SidebarPreferences
			entity, err := sp.Where(sp.UserID.Eq(req.TenantInfo.UserID)).One(ctx, r.db.Keel(ctx))
			if dberror.IsNotFoundError(err) {
				return nil, false, nil
			}
			if err != nil {
				r.l.Error(
					"failed to get sidebar preference",
					zap.String("userID", req.TenantInfo.UserID.String()),
					zap.Error(err),
				)
				return nil, false, err
			}

			return entity, true, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *sidebarpreference.SidebarPreference,
) (*sidebarpreference.SidebarPreference, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*sidebarpreference.SidebarPreference, error) {
			err := sidebarpreferencedb.SidebarPreferences.Create(ctx, r.db.Keel(ctx), entity)
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, dberror.CreateVersionMismatchError(entityName, entity.UserID.String())
			}
			if err != nil {
				r.l.Error(
					"failed to create sidebar preference",
					zap.String("userID", entity.UserID.String()),
					zap.Error(err),
				)
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *sidebarpreference.SidebarPreference,
) (*sidebarpreference.SidebarPreference, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*sidebarpreference.SidebarPreference, error) {
			err := sidebarpreferencedb.SidebarPreferences.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to update sidebar preference",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, sidebarPreferenceErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}
