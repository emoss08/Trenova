package integrationrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration/integrationdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/integrationports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var integrationErrors = dberror.Mapping{Entity: "Integration"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) integrationports.IntegrationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.integration-repository"),
	}
}

func (r *repository) ListByTenant(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*integration.Integration, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*integration.Integration, error) {
		i := integrationdb.Integrations
		return i.Order(i.Type.Asc()).All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) ListEnabledByType(
	ctx context.Context,
	typ integration.Type,
) ([]*integration.Integration, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations with an integration type enabled")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*integration.Integration, error) {
		i := integrationdb.Integrations
		return i.AllTenants().
			Where(i.Type.Eq(typ), i.Enabled.IsTrue()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) GetByType(
	ctx context.Context,
	_ pagination.TenantInfo,
	typ integration.Type,
) (*integration.Integration, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*integration.Integration, error) {
		i := integrationdb.Integrations
		rows, err := i.Where(i.Type.Eq(typ)).
			Limit(1).
			Cache(resultcache.Settings()...).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, integrationErrors.Map(err, string(typ))
		}
		if len(rows) == 0 {
			return nil, integrationErrors.Map(keel.ErrNotFound, string(typ))
		}

		return rows[0], nil
	})
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *integration.Integration,
) (*integration.Integration, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*integration.Integration, error) {
		i := integrationdb.Integrations
		db := r.db.Keel(ctx)
		existing, err := i.Where(i.Type.Eq(entity.Type)).One(ctx, db)
		if dberror.IsNotFoundError(err) {
			if err = i.Create(ctx, db, entity); err != nil {
				return nil, err
			}

			return entity, nil
		}
		if err != nil {
			return nil, err
		}

		entity.ID = existing.ID
		entity.CreatedAt = existing.CreatedAt
		entity.Version = existing.Version
		if err = i.Save(ctx, db, entity); err != nil {
			return nil, integrationErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}
