//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package capturerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture/capturedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var deviceErrors = dberror.Mapping{Entity: "Device"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type deviceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewDeviceRepository(p Params) captureports.CaptureDeviceRepository {
	return &deviceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.capture-device-repository"),
	}
}

func (r *deviceRepository) Create(
	ctx context.Context,
	entity *capture.CaptureDevice,
) (*capture.CaptureDevice, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*capture.CaptureDevice, error) {
		if err := capturedb.CaptureDevices.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *deviceRepository) Update(
	ctx context.Context,
	entity *capture.CaptureDevice,
) (*capture.CaptureDevice, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*capture.CaptureDevice, error) {
		if err := capturedb.CaptureDevices.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, deviceErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *deviceRepository) GetByID(
	ctx context.Context,
	req captureports.GetCaptureDeviceByIDRequest,
) (*capture.CaptureDevice, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*capture.CaptureDevice, error) {
		entity, err := capturedb.CaptureDevices.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, deviceErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *deviceRepository) List(
	ctx context.Context,
	req *captureports.ListCaptureDevicesRequest,
) (*pagination.ListResult[*capture.CaptureDevice], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*capture.CaptureDevice], error) {
			d := capturedb.CaptureDevices
			return listquery.List(
				ctx,
				r.db.Keel(ctx),
				d.Where(
					keel.When(req.UserID.IsNotNil(), d.UserID.Eq(req.UserID)),
					keel.When(req.Status != "", d.Status.Eq(req.Status)),
				),
				&listquery.Request[capture.CaptureDevice]{
					Filter: req.Filter,
					Then:   []keel.Order{d.CreatedAt.Desc()},
				},
			)
		},
	)
}

func (r *deviceRepository) GetByAccessTokenHash(
	ctx context.Context,
	hash string,
) (*capture.CaptureDevice, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"resolve a capture device access token before its tenant is known",
	)
	return r.getByToken(ctx, capturedb.CaptureDevices.AccessTokenHash.Eq(hash))
}

func (r *deviceRepository) GetByRefreshTokenHash(
	ctx context.Context,
	hash string,
) (*capture.CaptureDevice, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"resolve a capture device refresh token before its tenant is known",
	)
	return r.getByToken(ctx, capturedb.CaptureDevices.RefreshTokenHash.Eq(hash))
}

func (r *deviceRepository) GetByPreviousRefreshHash(
	ctx context.Context,
	hash string,
) (*capture.CaptureDevice, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"detect reuse of a rotated capture device refresh token before its tenant is known",
	)
	return r.getByToken(ctx, capturedb.CaptureDevices.PreviousRefreshHash.Eq(hash))
}

func (r *deviceRepository) getByToken(
	ctx context.Context,
	match keel.Pred,
) (*capture.CaptureDevice, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*capture.CaptureDevice, error) {
		entity, err := capturedb.CaptureDevices.AllTenants().Where(match).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, deviceErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *deviceRepository) Touch(
	ctx context.Context,
	req captureports.TouchCaptureDeviceRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		d := capturedb.CaptureDevices
		_, err := d.Where(d.ID.Eq(req.ID)).Update(
			ctx,
			r.db.Keel(ctx),
			d.LastSeenAt.To(req.SeenAt),
			d.LastIP.To(req.IP),
			d.Version.ToExpr(d.Version),
			d.UpdatedAt.ToExpr(d.UpdatedAt),
		)

		return err
	})
}
