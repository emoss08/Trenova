package tcaallowlistrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert/tablechangealertdb"
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
	_ pagination.TenantInfo,
) ([]*tablechangealert.TCAAllowlistedTable, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*tablechangealert.TCAAllowlistedTable, error) {
			a := tablechangealertdb.TCAAllowlistedTables
			entities, err := a.Where(a.Enabled.IsTrue()).
				Order(a.DisplayName.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list tca allowlisted tables", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) IsTableAllowed(
	ctx context.Context,
	tableName string,
	_ pagination.TenantInfo,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		a := tablechangealertdb.TCAAllowlistedTables
		exists, err := a.Where(a.TableName.Eq(tableName), a.Enabled.IsTrue()).
			Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to check if table is allowed",
				zap.String("tableName", tableName),
				zap.Error(err),
			)
			return false, err
		}

		return exists, nil
	})
}
