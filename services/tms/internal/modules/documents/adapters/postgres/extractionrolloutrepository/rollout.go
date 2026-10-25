package extractionrolloutrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractionrollout"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractionrollout/extractionrolloutdb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionrolloutports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var rolloutErrors = dberror.Mapping{Entity: "ExtractionRollout"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type rolloutRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewRollouts(p Params) extractionrolloutports.ExtractionRolloutRepository {
	return &rolloutRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.extractionrollout-repository"),
	}
}

func (r *rolloutRepository) Get(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*extractionrollout.ExtractionRollout, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*extractionrollout.ExtractionRollout, error) {
			entity, err := extractionrolloutdb.ExtractionRollouts.Query().One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, rolloutErrors.Map(err, "")
			}

			return entity, nil
		},
	)
}

func (r *rolloutRepository) Save(
	ctx context.Context,
	entity *extractionrollout.ExtractionRollout,
) (*extractionrollout.ExtractionRollout, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*extractionrollout.ExtractionRollout, error) {
			ro := extractionrolloutdb.ExtractionRollouts
			db := r.db.Keel(ctx)

			if entity.ID.IsNil() {
				if err := ro.Create(ctx, db, entity); err != nil {
					if dberror.IsUniqueConstraintViolation(err) {
						return nil, errortypes.NewBusinessError(
							"Someone else changed the rollout; reload it and try again",
						).WithInternal(err)
					}
					r.l.Error("failed to create extraction rollout", zap.Error(err))
					return nil, fmt.Errorf("create extraction rollout: %w", err)
				}

				return entity, nil
			}

			if err := ro.Save(ctx, db, entity); err != nil {
				r.l.Error("failed to save extraction rollout", zap.Error(err))
				return nil, rolloutErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}
