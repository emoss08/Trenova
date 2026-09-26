//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
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

const maxRequestsForTarget = 50

type requestRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewRequestRepository(p Params) repositories.CaptureRequestRepository {
	return &requestRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.capture-request-repository"),
	}
}

var (
	openRequestStatuses = []capture.RequestStatus{
		capture.RequestPending,
		capture.RequestDelivered,
		capture.RequestInProgress,
	}
	waitingRequestStatuses = []capture.RequestStatus{
		capture.RequestPending,
		capture.RequestDelivered,
	}
)

func (r *requestRepository) Create(
	ctx context.Context,
	entity *capture.CaptureRequest,
) (*capture.CaptureRequest, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *requestRepository) Update(
	ctx context.Context,
	entity *capture.CaptureRequest,
) (*capture.CaptureRequest, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.CaptureRequestColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		entity.Version = ov

		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "Capture request", entity.ID.String()); err != nil {
		entity.Version = ov

		return nil, err
	}

	return entity, nil
}

func (r *requestRepository) GetByID(
	ctx context.Context,
	req repositories.GetCaptureRequestByIDRequest,
) (*capture.CaptureRequest, error) {
	entity := new(capture.CaptureRequest)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.CaptureRequestRelations.CaptureProfile).
		Where(buncolgen.CaptureRequestColumns.ID.Eq(), req.ID).
		Apply(buncolgen.CaptureRequestApplyTenant(req.TenantInfo)).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Capture request")
	}

	return entity, nil
}

func (r *requestRepository) ListOpen(
	ctx context.Context,
	req repositories.ListOpenCaptureRequestsRequest,
) ([]*capture.CaptureRequest, error) {
	entities := make([]*capture.CaptureRequest, 0)
	cols := buncolgen.CaptureRequestColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.CaptureRequestRelations.CaptureProfile).
		Apply(buncolgen.CaptureRequestApplyTenant(req.TenantInfo)).
		Where(cols.DeviceID.Eq(), req.DeviceID).
		Where(cols.Status.In(), bun.List(openRequestStatuses)).
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *requestRepository) ListForTarget(
	ctx context.Context,
	req repositories.ListCaptureRequestsForTargetRequest,
) ([]*capture.CaptureRequest, error) {
	limit := req.Limit
	if limit <= 0 || limit > maxRequestsForTarget {
		limit = maxRequestsForTarget
	}

	entities := make([]*capture.CaptureRequest, 0, limit)
	cols := buncolgen.CaptureRequestColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(buncolgen.CaptureRequestApplyTenant(req.TenantInfo)).
		Where(cols.TargetType.Eq(), req.TargetType).
		Where(cols.TargetID.Eq(), req.TargetID)
	if req.UserID.IsNotNil() {
		query = query.Where(cols.UserID.Eq(), req.UserID)
	}

	if err := query.
		Order(cols.CreatedAt.OrderDesc()).
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *requestRepository) ListExpired(
	ctx context.Context,
	now int64,
	limit int,
) ([]*capture.CaptureRequest, error) {
	entities := make([]*capture.CaptureRequest, 0, limit)
	cols := buncolgen.CaptureRequestColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.Status.In(), bun.List(waitingRequestStatuses)).
		Where(cols.ExpiresAt.Lte(), now).
		Order(cols.ExpiresAt.OrderAsc()).
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}
