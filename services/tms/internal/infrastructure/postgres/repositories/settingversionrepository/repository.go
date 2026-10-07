package settingversionrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/settingversion"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
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

func New(p Params) repositories.SettingVersionRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.setting-version-repository")}
}

func (r *repository) Create(ctx context.Context, version *settingversion.SettingVersion) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(version).Exec(ctx); err != nil {
			return fmt.Errorf("record setting version: %w", err)
		}
		return nil
	})
}

func (r *repository) LatestAt(
	ctx context.Context,
	req *repositories.GetSettingVersionAtRequest,
) (*settingversion.SettingVersion, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*settingversion.SettingVersion, error) {
		cols := buncolgen.SettingVersionColumns
		version := new(settingversion.SettingVersion)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(version).
			Relation(buncolgen.SettingVersionRelations.Author).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.SettingVersionScopeTenant(sq, req.TenantInfo).
					Where(cols.Kind.Eq(), req.Kind).
					Where(cols.SubjectID.Eq(), req.SubjectID).
					Where(cols.Version.Lte(), req.Version)
			}).
			Order(cols.Version.OrderDesc()).
			Limit(1).
			Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // a setting saved before versions were kept has none
		}
		if err != nil {
			return nil, fmt.Errorf("read setting version: %w", err)
		}

		return version, nil
	})
}
