package patternconfigrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
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

func NewRepository(p Params) repositories.PatternConfigRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.patternconfig-repository"),
	}
}

func (r *repository) GetAll(ctx context.Context) ([]*dedicatedlane.PatternConfig, error) {
	log := r.l.With(
		zap.String("operation", "GetAll"),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*dedicatedlane.PatternConfig, 0)

	if err = db.NewSelect().Model(&entities).Relation("Organization").Scan(ctx); err != nil {
		log.Error("failed to scan pattern configs", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByOrgID(
	ctx context.Context,
	req repositories.GetPatternConfigRequest,
) (*dedicatedlane.PatternConfig, error) {
	log := r.l.With(
		zap.String("operation", "GetByOrgID"),
		zap.String("orgID", req.OrgID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(dedicatedlane.PatternConfig)

	if err = db.NewSelect().Model(entity).Where("organization_id = ?", req.OrgID).Scan(ctx); err != nil {
		log.Error("failed to scan pattern config", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	pc *dedicatedlane.PatternConfig,
) (*dedicatedlane.PatternConfig, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("pcID", pc.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := pc.Version
	pc.Version++

	results, err := db.NewUpdate().
		Model(pc).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update pattern config", zap.Error(err))
		return nil, err
	}

	roErr := dberror.CheckRowsAffected(results, "Pattern Config", pc.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return pc, nil
}
