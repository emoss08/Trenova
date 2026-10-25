package extractionshadowrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractionshadow"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractionshadow/extractionshadowdb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionshadowports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var settingsErrors = dberror.Mapping{Entity: "ExtractionShadowSettings"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type settingsRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSettings(p Params) extractionshadowports.ExtractionShadowSettingsRepository {
	return &settingsRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.extractionshadow-settings-repository"),
	}
}

func (r *settingsRepository) Get(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*extractionshadow.ShadowSettings, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*extractionshadow.ShadowSettings, error) {
			entity, err := extractionshadowdb.ShadowSettingses.Cache(resultcache.Settings()...).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, settingsErrors.Map(err, "")
			}

			return entity, nil
		},
	)
}

func (r *settingsRepository) Save(
	ctx context.Context,
	entity *extractionshadow.ShadowSettings,
) (*extractionshadow.ShadowSettings, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*extractionshadow.ShadowSettings, error) {
			s := extractionshadowdb.ShadowSettingses
			db := r.db.Keel(ctx)

			if entity.ID.IsNil() {
				if err := s.Create(ctx, db, entity); err != nil {
					if dberror.IsUniqueConstraintViolation(err) {
						return nil, errortypes.NewBusinessError(
							"Someone else changed the shadow settings; reload them and try again",
						).WithInternal(err)
					}
					r.l.Error("failed to create extraction shadow settings", zap.Error(err))

					return nil, fmt.Errorf("create extraction shadow settings: %w", err)
				}

				return entity, nil
			}

			if err := s.Save(ctx, db, entity); err != nil {
				r.l.Error("failed to save extraction shadow settings", zap.Error(err))

				return nil, settingsErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}
