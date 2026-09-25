package capturerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

type pageRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPageRepository(p Params) repositories.CapturePageRepository {
	return &pageRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.capture-page-repository"),
	}
}

// Insert relies on the (batch, sequence) unique index rather than a read
// first: two retries of the same page racing each other both reach the index,
// and exactly one of them wins.
func (r *pageRepository) Insert(
	ctx context.Context,
	entity *capture.CapturePage,
) (*capture.CapturePage, bool, error) {
	result, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On("CONFLICT (batch_id, business_unit_id, organization_id, sequence) DO NOTHING").
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, false, err
	}
	if affected > 0 {
		return entity, true, nil
	}

	existing, err := r.GetBySequence(ctx, repositories.GetCapturePageBySequenceRequest{
		BatchID: entity.BatchID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		Sequence: entity.Sequence,
	})
	if err != nil {
		return nil, false, err
	}

	return existing, false, nil
}

func (r *pageRepository) Update(
	ctx context.Context,
	entity *capture.CapturePage,
) (*capture.CapturePage, error) {
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "Capture page", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *pageRepository) GetByID(
	ctx context.Context,
	req repositories.GetCapturePageByIDRequest,
) (*capture.CapturePage, error) {
	entity := new(capture.CapturePage)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Apply(buncolgen.CapturePageApplyTenant(req.TenantInfo)).
		Where(buncolgen.CapturePageColumns.ID.Eq(), req.ID).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Capture page")
	}

	return entity, nil
}

func (r *pageRepository) GetBySequence(
	ctx context.Context,
	req repositories.GetCapturePageBySequenceRequest,
) (*capture.CapturePage, error) {
	entity := new(capture.CapturePage)
	cols := buncolgen.CapturePageColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Apply(buncolgen.CapturePageApplyTenant(req.TenantInfo)).
		Where(cols.BatchID.Eq(), req.BatchID).
		Where(cols.Sequence.Eq(), req.Sequence).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Capture page")
	}

	return entity, nil
}

func (r *pageRepository) ListByBatch(
	ctx context.Context,
	req repositories.ListCapturePagesRequest,
) ([]*capture.CapturePage, error) {
	entities := make([]*capture.CapturePage, 0)
	cols := buncolgen.CapturePageColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(buncolgen.CapturePageApplyTenant(req.TenantInfo)).
		Where(cols.BatchID.Eq(), req.BatchID).
		Order(cols.Sequence.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}
