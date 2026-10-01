package extractionshadowrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/extractionshadow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const settingsEntity = "ExtractionShadowSettings"

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type settingsRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSettings(p Params) repositories.ExtractionShadowSettingsRepository {
	return &settingsRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.extractionshadow-settings-repository"),
	}
}

func (r *settingsRepository) Get(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*extractionshadow.ShadowSettings, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*extractionshadow.ShadowSettings, error) {
		entity := new(extractionshadow.ShadowSettings)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ShadowSettingsScopeTenant(sq, tenant)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, settingsEntity)
		}

		return entity, nil
	})
}

func (r *settingsRepository) Save(
	ctx context.Context,
	entity *extractionshadow.ShadowSettings,
) (*extractionshadow.ShadowSettings, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*extractionshadow.ShadowSettings, error) {
		if entity.ID.IsNil() {
			return r.insert(ctx, entity)
		}

		cols := buncolgen.ShadowSettingsColumns
		previous := entity.Version
		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ShadowSettingsScopeTenantUpdate(uq, pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				}).
					Where(cols.ID.Eq(), entity.ID).
					Where(cols.Version.Eq(), previous)
			}).
			ExcludeColumn(
				cols.ID.Bare(),
				cols.OrganizationID.Bare(),
				cols.BusinessUnitID.Bare(),
				cols.CreatedAt.Bare(),
				cols.Version.Bare(),
			).
			Set(cols.Version.Inc(1)).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to save extraction shadow settings", zap.Error(err))

			return nil, fmt.Errorf("save extraction shadow settings: %w", err)
		}
		if err = dberror.CheckRowsAffected(res, settingsEntity, entity.ID.String()); err != nil {
			return nil, err
		}
		entity.Version = previous + 1

		return entity, nil
	})
}

func (r *settingsRepository) insert(
	ctx context.Context,
	entity *extractionshadow.ShadowSettings,
) (*extractionshadow.ShadowSettings, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*extractionshadow.ShadowSettings, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, errortypes.NewBusinessError(
					"Someone else changed the shadow settings; reload them and try again",
				).WithInternal(err)
			}
			r.l.Error("failed to create extraction shadow settings", zap.Error(err))

			return nil, fmt.Errorf("create extraction shadow settings: %w", err)
		}

		return entity, nil
	})
}
