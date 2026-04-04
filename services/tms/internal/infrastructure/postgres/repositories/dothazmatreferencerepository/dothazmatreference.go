package dothazmatreferencerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dothazmatreference"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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

func New(p Params) repositories.DotHazmatReferenceRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.dot-hazmat-reference-repository"),
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetDotHazmatReferenceByIDRequest,
) (*dothazmatreference.DotHazmatReference, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.DotHazmatReferenceID.String()),
	)

	entity := new(dothazmatreference.DotHazmatReference)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		Where("dhr.id = ?", req.DotHazmatReferenceID).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get dot hazmat reference", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "DotHazmatReference")
	}

	return entity, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*dothazmatreference.DotHazmatReference], error) {
	entities := make([]*dothazmatreference.DotHazmatReference, 0, req.Pagination.SafeLimit())

	q := r.db.DB().
		NewSelect().
		Model(&entities).
		Column("id", "un_number", "proper_shipping_name", "hazard_class", "subsidiary_hazard", "packing_group", "special_provisions", "erg_guide", "symbols").
		Limit(req.Pagination.Limit).
		Offset(req.Pagination.Offset)

	if req.Query != "" {
		q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereOr("dhr.un_number LIKE ?", req.Query+"%").
				WhereOr("LOWER(dhr.proper_shipping_name) LIKE LOWER(?)", "%"+req.Query+"%")
		})
	}

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*dothazmatreference.DotHazmatReference]{
		Items: entities,
		Total: total,
	}, nil
}
