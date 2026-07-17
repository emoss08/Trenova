package sidebarpreferencerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/sidebarpreference"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
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

func New(p Params) repositories.SidebarPreferenceRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.sidebar-preference-repository"),
	}
}

func (r *repository) Get(
	ctx context.Context,
	req *repositories.GetSidebarPreferenceRequest,
) (*sidebarpreference.SidebarPreference, bool, error) {
	log := r.l.With(
		zap.String("operation", "Get"),
		zap.String("userID", req.TenantInfo.UserID.String()),
	)

	entity := new(sidebarpreference.SidebarPreference)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("sbp.user_id = ?", req.TenantInfo.UserID).
				Where("sbp.organization_id = ?", req.TenantInfo.OrgID).
				Where("sbp.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, false, nil
		}
		log.Error("failed to get sidebar preference", zap.Error(err))
		return nil, false, err
	}

	return entity, true, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *sidebarpreference.SidebarPreference,
) (*sidebarpreference.SidebarPreference, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("userID", entity.UserID.String()),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, dberror.CreateVersionMismatchError(
				"SidebarPreference",
				entity.UserID.String(),
			)
		}
		log.Error("failed to create sidebar preference", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *sidebarpreference.SidebarPreference,
) (*sidebarpreference.SidebarPreference, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	result, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where("sbp.version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update sidebar preference", zap.Error(err))
		return nil, err
	}

	err = dberror.CheckRowsAffected(result, "SidebarPreference", entity.ID.String())
	if err != nil {
		return nil, err
	}

	return entity, nil
}
