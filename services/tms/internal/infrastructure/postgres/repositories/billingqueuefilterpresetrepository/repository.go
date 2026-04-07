package billingqueuefilterpresetrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueuefilterpreset"
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

func New(p Params) repositories.BillingQueueFilterPresetRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.billing-queue-filter-preset-repository"),
	}
}

func (r *repository) ListByUserID(
	ctx context.Context,
	req *repositories.ListBillingQueueFilterPresetsRequest,
) ([]*billingqueuefilterpreset.BillingQueueFilterPreset, error) {
	log := r.l.With(
		zap.String("operation", "ListByUserID"),
		zap.String("userID", req.UserID.String()),
	)

	entities := make([]*billingqueuefilterpreset.BillingQueueFilterPreset, 0)
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("bqfp.user_id = ?", req.UserID).
				Where("bqfp.organization_id = ?", req.TenantInfo.OrgID).
				Where("bqfp.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Order("bqfp.created_at DESC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to list billing queue filter presets", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *billingqueuefilterpreset.BillingQueueFilterPreset,
) (*billingqueuefilterpreset.BillingQueueFilterPreset, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("name", entity.Name),
	)

	_, err := r.db.DB().NewInsert().Model(entity).Exec(ctx)
	if err != nil {
		log.Error("failed to create billing queue filter preset", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *billingqueuefilterpreset.BillingQueueFilterPreset,
) (*billingqueuefilterpreset.BillingQueueFilterPreset, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	result, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where("bqfp.user_id = ?", entity.UserID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to update billing queue filter preset", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "BillingQueueFilterPreset", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req *repositories.DeleteBillingQueueFilterPresetRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.PresetID.String()),
	)

	result, err := r.db.DB().
		NewDelete().
		Model((*billingqueuefilterpreset.BillingQueueFilterPreset)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where("bqfp.id = ?", req.PresetID).
				Where("bqfp.user_id = ?", req.UserID).
				Where("bqfp.organization_id = ?", req.TenantInfo.OrgID).
				Where("bqfp.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete billing queue filter preset", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(result, "BillingQueueFilterPreset", req.PresetID.String())
}
