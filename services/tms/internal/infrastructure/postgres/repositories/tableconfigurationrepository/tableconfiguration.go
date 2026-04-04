package tableconfigurationrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) repositories.TableConfigurationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.table-configuration-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListTableConfigurationsRequest,
) *bun.SelectQuery {
	log := r.l.With(
		zap.String("operation", "filterQuery"),
		zap.Any("req", req),
	)

	q = querybuilder.ApplyFilters(
		q,
		"tc",
		req.Filter,
		(*tableconfiguration.TableConfiguration)(nil),
	)

	q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.
			WhereGroup(" AND ", func(privateSq *bun.SelectQuery) *bun.SelectQuery {
				return privateSq.
					Where("tc.user_id = ?", req.Filter.TenantInfo.UserID).
					Where("tc.visibility = ?", tableconfiguration.VisibilityPrivate)
			}).
			WhereGroup(" OR ", func(publicSq *bun.SelectQuery) *bun.SelectQuery {
				return publicSq.
					Where("tc.organization_id = ?", req.Filter.TenantInfo.OrgID).
					Where("tc.visibility = ?", tableconfiguration.VisibilityPublic)
			})
	})

	q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where("tc.organization_id = ?", req.Filter.TenantInfo.OrgID).
			Where("tc.business_unit_id = ?", req.Filter.TenantInfo.BuID)
	})

	if req.Resource != "" {
		q = q.Where("tc.resource = ?", req.Resource)
	}

	if req.Visibility != "" {
		v, err := tableconfiguration.VisibilityFromString(req.Visibility)
		if err != nil {
			log.Error("failed to parse visibility", zap.Error(err))
			return q
		}
		q = q.Where("tc.visibility = ?", v)
	}

	q = q.Order("tc.is_default DESC", "tc.created_at DESC")

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListTableConfigurationsRequest,
) (*pagination.ListResult[*tableconfiguration.TableConfiguration], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*tableconfiguration.TableConfiguration, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count table configurations", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*tableconfiguration.TableConfiguration]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *tableconfiguration.TableConfiguration,
) (*tableconfiguration.TableConfiguration, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("name", entity.Name),
	)

	_, err := r.db.DB().NewInsert().Model(entity).Exec(ctx)
	if err != nil {
		log.Error("failed to create table configuration", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *tableconfiguration.TableConfiguration,
) (*tableconfiguration.TableConfiguration, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	_, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update table configuration", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetTableConfigurationByIDRequest,
) (*tableconfiguration.TableConfiguration, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("configurationID", req.ConfigurationID.String()),
	)

	entity := new(tableconfiguration.TableConfiguration)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("tc.id = ?", req.ConfigurationID).
				Where("tc.organization_id = ?", req.TenantInfo.OrgID).
				Where("tc.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get table configuration", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "TableConfiguration")
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	configurationID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("configurationID", configurationID.String()),
	)

	result, err := r.db.DB().
		NewDelete().
		Model((*tableconfiguration.TableConfiguration)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where("tc.id = ?", configurationID).
				Where("tc.organization_id = ?", tenantInfo.OrgID).
				Where("tc.business_unit_id = ?", tenantInfo.BuID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete table configuration", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(result, "TableConfiguration", configurationID.String())
}

func (r *repository) GetDefaultForResource(
	ctx context.Context,
	req repositories.GetDefaultTableConfigurationRequest,
) (*tableconfiguration.TableConfiguration, error) {
	log := r.l.With(
		zap.String("operation", "GetDefaultForResource"),
		zap.String("resource", req.Resource),
	)

	entity := new(tableconfiguration.TableConfiguration)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("tc.user_id = ?", req.TenantInfo.UserID).
				Where("tc.resource = ?", req.Resource).
				Where("tc.is_default = ?", true).
				Where("tc.organization_id = ?", req.TenantInfo.OrgID).
				Where("tc.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Debug("no default table configuration found", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "TableConfiguration")
	}

	return entity, nil
}

func (r *repository) ClearDefaultForResource(
	ctx context.Context,
	userID pulid.ID,
	resource string,
	tenantInfo pagination.TenantInfo,
) error {
	log := r.l.With(
		zap.String("operation", "ClearDefaultForResource"),
		zap.String("userID", userID.String()),
		zap.String("resource", resource),
	)

	_, err := r.db.DB().
		NewUpdate().
		Model((*tableconfiguration.TableConfiguration)(nil)).
		Set("is_default = ?", false).
		Set("updated_at = extract(epoch from current_timestamp)::bigint").
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("tc.user_id = ?", userID).
				Where("tc.resource = ?", resource).
				Where("tc.is_default = ?", true).
				Where("tc.organization_id = ?", tenantInfo.OrgID).
				Where("tc.business_unit_id = ?", tenantInfo.BuID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to clear default table configuration", zap.Error(err))
		return err
	}

	return nil
}
