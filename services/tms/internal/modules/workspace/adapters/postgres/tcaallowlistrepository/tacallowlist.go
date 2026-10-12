package tcaallowlistrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/tablechangealertcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/tablechangealertports"
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

func New(p Params) tablechangealertports.TCAAllowlistRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.tca-allowlist-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*tablechangealert.TCAAllowlistedTable, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*tablechangealert.TCAAllowlistedTable, error) {
			log := r.l.With(zap.String("operation", "List"))
			cols := tablechangealertcol.TCAAllowlistedTableColumns

			entities := make([]*tablechangealert.TCAAllowlistedTable, 0)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Apply(tablechangealertcol.TCAAllowlistedTableApplyTenant(tenantInfo)).
				Where(cols.Enabled.Eq(), true).
				Order(cols.DisplayName.OrderAsc()).
				Scan(ctx)
			if err != nil {
				log.Error("failed to list tca allowlisted tables", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) IsTableAllowed(
	ctx context.Context,
	tableName string,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		log := r.l.With(
			zap.String("operation", "IsTableAllowed"),
			zap.String("tableName", tableName),
		)
		cols := tablechangealertcol.TCAAllowlistedTableColumns

		exists, err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*tablechangealert.TCAAllowlistedTable)(nil)).
			Apply(tablechangealertcol.TCAAllowlistedTableApplyTenant(tenantInfo)).
			Where(cols.TableName.Eq(), tableName).
			Where(cols.Enabled.Eq(), true).
			Exists(ctx)
		if err != nil {
			log.Error("failed to check if table is allowed", zap.Error(err))
			return false, err
		}

		return exists, nil
	})
}
