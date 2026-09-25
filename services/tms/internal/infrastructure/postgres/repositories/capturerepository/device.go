//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package capturerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type deviceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewDeviceRepository(p Params) repositories.CaptureDeviceRepository {
	return &deviceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.capture-device-repository"),
	}
}

func (r *deviceRepository) Create(
	ctx context.Context,
	entity *capture.CaptureDevice,
) (*capture.CaptureDevice, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *deviceRepository) Update(
	ctx context.Context,
	entity *capture.CaptureDevice,
) (*capture.CaptureDevice, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.CaptureDeviceColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		entity.Version = ov

		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "Device", entity.ID.String()); err != nil {
		entity.Version = ov

		return nil, err
	}

	return entity, nil
}

func (r *deviceRepository) GetByID(
	ctx context.Context,
	req repositories.GetCaptureDeviceByIDRequest,
) (*capture.CaptureDevice, error) {
	entity := new(capture.CaptureDevice)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(buncolgen.CaptureDeviceColumns.ID.Eq(), req.ID).
		Apply(buncolgen.CaptureDeviceApplyTenant(req.TenantInfo)).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Device")
	}

	return entity, nil
}

func (r *deviceRepository) List(
	ctx context.Context,
	req *repositories.ListCaptureDevicesRequest,
) (*pagination.ListResult[*capture.CaptureDevice], error) {
	entities := make([]*capture.CaptureDevice, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.CaptureDeviceColumns

	query := r.db.DBForContext(ctx).NewSelect().Model(&entities)
	if req.UserID.IsNotNil() {
		query = query.Where(cols.UserID.Eq(), req.UserID)
	}
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}

	total, err := querybuilder.ApplyFilters(
		query,
		buncolgen.CaptureDeviceTable.Alias,
		req.Filter,
		(*capture.CaptureDevice)(nil),
	).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*capture.CaptureDevice]{Items: entities, Total: total}, nil
}

func (r *deviceRepository) GetByAccessTokenHash(
	ctx context.Context,
	hash string,
) (*capture.CaptureDevice, error) {
	return r.getByTokenColumn(ctx, buncolgen.CaptureDeviceColumns.AccessTokenHash, hash)
}

func (r *deviceRepository) GetByRefreshTokenHash(
	ctx context.Context,
	hash string,
) (*capture.CaptureDevice, error) {
	return r.getByTokenColumn(ctx, buncolgen.CaptureDeviceColumns.RefreshTokenHash, hash)
}

func (r *deviceRepository) GetByPreviousRefreshHash(
	ctx context.Context,
	hash string,
) (*capture.CaptureDevice, error) {
	return r.getByTokenColumn(ctx, buncolgen.CaptureDeviceColumns.PreviousRefreshHash, hash)
}

func (r *deviceRepository) getByTokenColumn(
	ctx context.Context,
	column buncolgen.Column,
	hash string,
) (*capture.CaptureDevice, error) {
	entity := new(capture.CaptureDevice)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(column.Eq(), hash).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Device")
	}

	return entity, nil
}

func (r *deviceRepository) Touch(
	ctx context.Context,
	req repositories.TouchCaptureDeviceRequest,
) error {
	cols := buncolgen.CaptureDeviceColumns

	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*capture.CaptureDevice)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CaptureDeviceScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Set(cols.LastSeenAt.Set(), req.SeenAt).
		Set(cols.LastIP.Set(), req.IP).
		Exec(ctx)

	return err
}
