package integrationrepository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
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

func New(p Params) repositories.IntegrationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.integration-repository"),
	}
}

func (r *repository) ListByTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*integration.Integration, error) {
	entities := make([]*integration.Integration, 0)

	if err := r.db.DB().NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("integ.organization_id = ?", tenantInfo.OrgID).
				Where("integ.business_unit_id = ?", tenantInfo.BuID)
		}).
		OrderExpr("integ.type ASC").
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByType(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*integration.Integration, error) {
	entity := new(integration.Integration)

	err := r.db.DB().NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("integ.organization_id = ?", tenantInfo.OrgID).
				Where("integ.business_unit_id = ?", tenantInfo.BuID).
				Where("integ.type = ?", typ)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Integration")
	}

	return entity, nil
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *integration.Integration,
) (*integration.Integration, error) {
	existing := new(integration.Integration)
	err := r.db.DB().NewSelect().
		Model(existing).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("integ.organization_id = ?", entity.OrganizationID).
				Where("integ.business_unit_id = ?", entity.BusinessUnitID).
				Where("integ.type = ?", entity.Type)
		}).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if _, insertErr := r.db.DB().NewInsert().
				Model(entity).
				Returning("*").
				Exec(ctx); insertErr != nil {
				return nil, insertErr
			}

			return entity, nil
		}

		return nil, err
	}

	entity.ID = existing.ID
	entity.CreatedAt = existing.CreatedAt
	entity.Version = existing.Version + 1

	result, err := r.db.DB().NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", existing.Version).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	if err = dberror.CheckRowsAffected(result, "Integration", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}
