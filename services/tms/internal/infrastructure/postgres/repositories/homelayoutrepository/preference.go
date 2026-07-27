package homelayoutrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func (r *repository) GetPreference(
	ctx context.Context,
	req *repositories.GetHomePreferenceRequest,
) (*homelayout.Preference, bool, error) {
	log := r.l.With(
		zap.String("operation", "GetPreference"),
		zap.String("userID", req.TenantInfo.UserID.String()),
	)

	entity := new(homelayout.Preference)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PreferenceScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.PreferenceColumns.UserID.Eq(), req.TenantInfo.UserID)
		}).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, false, nil
		}
		log.Error("failed to get home preference", zap.Error(err))
		return nil, false, err
	}

	return entity, true, nil
}

func (r *repository) CreatePreference(
	ctx context.Context,
	entity *homelayout.Preference,
) (*homelayout.Preference, error) {
	log := r.l.With(
		zap.String("operation", "CreatePreference"),
		zap.String("userID", entity.UserID.String()),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, dberror.CreateVersionMismatchError(
				"HomePreference",
				entity.UserID.String(),
			)
		}
		log.Error("failed to create home preference", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdatePreference(
	ctx context.Context,
	entity *homelayout.Preference,
) (*homelayout.Preference, error) {
	log := r.l.With(
		zap.String("operation", "UpdatePreference"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	result, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.PreferenceColumns.Version.Eq(), ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update home preference", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "HomePreference", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}
