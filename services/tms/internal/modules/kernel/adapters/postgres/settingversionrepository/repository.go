package settingversionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/settingversion"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/settingversion/settingversiondb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/settingversionports"
	"github.com/emoss08/trenova/pkg/dberror"
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

func New(p Params) settingversionports.SettingVersionRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.setting-version-repository")}
}

func (r *repository) Create(ctx context.Context, version *settingversion.SettingVersion) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := settingversiondb.SettingVersions.Create(ctx, r.db.Keel(ctx), version); err != nil {
			return fmt.Errorf("record setting version: %w", err)
		}
		return nil
	})
}

func (r *repository) LatestAt(
	ctx context.Context,
	req *settingversionports.GetSettingVersionAtRequest,
) (*settingversion.SettingVersion, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*settingversion.SettingVersion, error) {
		sv := settingversiondb.SettingVersions
		version, err := sv.Where(
			sv.Kind.Eq(req.Kind),
			sv.SubjectID.Eq(req.SubjectID),
			sv.Version.Lte(req.Version),
		).
			Load(sv.Author()).
			Order(sv.Version.Desc()).
			First(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // a setting saved before versions were kept has none
		}
		if err != nil {
			return nil, fmt.Errorf("read setting version: %w", err)
		}

		return version, nil
	})
}
