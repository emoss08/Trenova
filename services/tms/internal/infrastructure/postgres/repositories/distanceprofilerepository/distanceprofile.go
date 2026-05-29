package distanceprofilerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
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

func New(p Params) repositories.DistanceProfileRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.distance-profile-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListDistanceProfileRequest,
) (*pagination.ListResult[*distanceprofile.DistanceProfile], error) {
	entities := make([]*distanceprofile.DistanceProfile, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).NewSelect().
		Model(&entities).
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			q = querybuilder.ApplyFilters(q, "dp", req.Filter, (*distanceprofile.DistanceProfile)(nil))
			return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
		}).
		ScanAndCount(ctx)
	if err != nil {
		r.l.Error("failed to list distance profiles", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*distanceprofile.DistanceProfile]{Items: entities, Total: total}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetDistanceProfileByIDRequest,
) (*distanceprofile.DistanceProfile, error) {
	entity := new(distanceprofile.DistanceProfile)
	err := r.db.DBForContext(ctx).NewSelect().
		Model(entity).
		Where("dp.id = ?", req.ID).
		Where("dp.organization_id = ?", req.TenantInfo.OrgID).
		Where("dp.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "DistanceProfile")
	}
	return entity, nil
}

func (r *repository) GetDefault(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*distanceprofile.DistanceProfile, error) {
	entity := new(distanceprofile.DistanceProfile)
	err := r.db.DBForContext(ctx).NewSelect().
		Model(entity).
		Where("dp.organization_id = ?", tenantInfo.OrgID).
		Where("dp.business_unit_id = ?", tenantInfo.BuID).
		Where("dp.is_default = true").
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "DistanceProfile")
	}
	return entity, nil
}

func (r *repository) EnsureDefault(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*distanceprofile.DistanceProfile, error) {
	existing, err := r.GetDefault(ctx, tenantInfo)
	if err == nil {
		return existing, nil
	}
	if !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	entity := distanceprofile.NewDefault(tenantInfo.OrgID, tenantInfo.BuID)
	result, err := r.db.DBForContext(ctx).NewInsert().
		Model(entity).
		On("CONFLICT DO NOTHING").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if rowsAffected, rowsErr := result.RowsAffected(); rowsErr == nil && rowsAffected == 0 {
		return r.GetDefault(ctx, tenantInfo)
	}
	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *distanceprofile.DistanceProfile,
) (*distanceprofile.DistanceProfile, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		if entity.IsDefault {
			if _, err := r.db.DBForContext(c).NewUpdate().
				Model((*distanceprofile.DistanceProfile)(nil)).
				Set("is_default = false").
				Where("organization_id = ?", entity.OrganizationID).
				Where("business_unit_id = ?", entity.BusinessUnitID).
				Exec(c); err != nil {
				return err
			}
		}
		_, err := r.db.DBForContext(c).NewInsert().Model(entity).Returning("*").Exec(c)
		return err
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(err, "Distance profile is busy. Retry the request.")
	}
	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *distanceprofile.DistanceProfile,
) (*distanceprofile.DistanceProfile, error) {
	previousVersion := entity.Version
	entity.Version++
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		if entity.IsDefault {
			if _, err := r.db.DBForContext(c).NewUpdate().
				Model((*distanceprofile.DistanceProfile)(nil)).
				Set("is_default = false").
				Where("organization_id = ?", entity.OrganizationID).
				Where("business_unit_id = ?", entity.BusinessUnitID).
				Where("id <> ?", entity.ID).
				Exec(c); err != nil {
				return err
			}
		}
		result, err := r.db.DBForContext(c).NewUpdate().
			Model(entity).
			WherePK().
			Where("version = ?", previousVersion).
			Returning("*").
			Exec(c)
		if err != nil {
			return err
		}
		return dberror.CheckRowsAffected(result, "DistanceProfile", entity.ID.String())
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(err, "Distance profile is busy. Retry the request.")
	}
	return entity, nil
}

func (r *repository) Delete(ctx context.Context, req repositories.DeleteDistanceProfileRequest) error {
	result, err := r.db.DBForContext(ctx).NewDelete().
		Model((*distanceprofile.DistanceProfile)(nil)).
		Where("dp.id = ?", req.ID).
		Where("dp.organization_id = ?", req.TenantInfo.OrgID).
		Where("dp.business_unit_id = ?", req.TenantInfo.BuID).
		Exec(ctx)
	if err != nil {
		return err
	}
	return dberror.CheckRowsAffected(result, "DistanceProfile", req.ID.String())
}

func (r *repository) SetDefault(
	ctx context.Context,
	req repositories.GetDistanceProfileByIDRequest,
) (*distanceprofile.DistanceProfile, error) {
	var entity *distanceprofile.DistanceProfile
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		current, err := r.GetByID(c, req)
		if err != nil {
			return err
		}
		if current.Status != distanceprofile.StatusActive {
			return errortypes.NewValidationError(
				"isDefault",
				errortypes.ErrInvalid,
				"Default profile must be active",
			)
		}
		if _, err = r.db.DBForContext(c).NewUpdate().
			Model((*distanceprofile.DistanceProfile)(nil)).
			Set("is_default = false").
			Where("organization_id = ?", req.TenantInfo.OrgID).
			Where("business_unit_id = ?", req.TenantInfo.BuID).
			Exec(c); err != nil {
			return err
		}
		current.IsDefault = true
		current.Version++
		if _, err = r.db.DBForContext(c).NewUpdate().
			Model(current).
			Column("is_default", "version", "updated_at").
			WherePK().
			Returning("*").
			Exec(c); err != nil {
			return err
		}
		entity = current
		return nil
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(err, "Distance profile is busy. Retry the request.")
	}
	return entity, nil
}
