package capturerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type coverSheetRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewCoverSheetRepository(p Params) repositories.CaptureCoverSheetRepository {
	return &coverSheetRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.capture-cover-sheet-repository"),
	}
}

func (r *coverSheetRepository) CreateMany(
	ctx context.Context,
	entities []*capture.CaptureCoverSheet,
) error {
	if len(entities) == 0 {
		return nil
	}

	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&entities).
		Returning("*").
		Exec(ctx)

	return err
}

func (r *coverSheetRepository) GetByID(
	ctx context.Context,
	req repositories.GetCaptureCoverSheetByIDRequest,
) (*capture.CaptureCoverSheet, error) {
	entity := new(capture.CaptureCoverSheet)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Apply(buncolgen.CaptureCoverSheetApplyTenant(req.TenantInfo)).
		Where(buncolgen.CaptureCoverSheetColumns.ID.Eq(), req.ID).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Cover sheet")
	}

	return entity, nil
}

func (r *coverSheetRepository) GetByTokenHash(
	ctx context.Context,
	req repositories.GetCaptureCoverSheetByTokenRequest,
) (*capture.CaptureCoverSheet, error) {
	entity := new(capture.CaptureCoverSheet)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Apply(buncolgen.CaptureCoverSheetApplyTenant(req.TenantInfo)).
		Where(buncolgen.CaptureCoverSheetColumns.TokenHash.Eq(), req.TokenHash).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Cover sheet")
	}

	return entity, nil
}

func (r *coverSheetRepository) MarkUsed(
	ctx context.Context,
	req repositories.MarkCaptureCoverSheetUsedRequest,
) error {
	cols := buncolgen.CaptureCoverSheetColumns

	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*capture.CaptureCoverSheet)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CaptureCoverSheetScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Set(cols.LastUsedAt.Set(), req.UsedAt).
		Set(cols.UseCount.Inc(1)).
		Exec(ctx)

	return err
}
