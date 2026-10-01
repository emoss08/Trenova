package extractionrolloutrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/extractionrollout"
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

const rolloutEntity = "ExtractionRollout"

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type rolloutRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewRollouts(p Params) repositories.ExtractionRolloutRepository {
	return &rolloutRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.extractionrollout-repository"),
	}
}

func (r *rolloutRepository) Get(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*extractionrollout.ExtractionRollout, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*extractionrollout.ExtractionRollout, error) {
		entity := new(extractionrollout.ExtractionRollout)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ExtractionRolloutScopeTenant(sq, tenant)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, rolloutEntity)
		}

		return entity, nil
	})
}

func (r *rolloutRepository) Save(
	ctx context.Context,
	entity *extractionrollout.ExtractionRollout,
) (*extractionrollout.ExtractionRollout, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*extractionrollout.ExtractionRollout, error) {
		if entity.ID.IsNil() {
			return r.insert(ctx, entity)
		}

		cols := buncolgen.ExtractionRolloutColumns
		previous := entity.Version
		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ExtractionRolloutScopeTenantUpdate(uq, pagination.TenantInfo{
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
			r.l.Error("failed to save extraction rollout", zap.Error(err))

			return nil, fmt.Errorf("save extraction rollout: %w", err)
		}
		if err = dberror.CheckRowsAffected(res, rolloutEntity, entity.ID.String()); err != nil {
			return nil, err
		}
		entity.Version = previous + 1

		return entity, nil
	})
}

func (r *rolloutRepository) insert(
	ctx context.Context,
	entity *extractionrollout.ExtractionRollout,
) (*extractionrollout.ExtractionRollout, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*extractionrollout.ExtractionRollout, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, errortypes.NewBusinessError(
					"Someone else changed the rollout; reload it and try again",
				).WithInternal(err)
			}
			r.l.Error("failed to create extraction rollout", zap.Error(err))

			return nil, fmt.Errorf("create extraction rollout: %w", err)
		}

		return entity, nil
	})
}
