//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package capturerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capturerequest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capturerequest/capturerequestdb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/zap"
)

const maxRequestsForTarget = 50

var requestErrors = dberror.Mapping{Entity: "Capture request"}

type requestRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewRequestRepository(p Params) captureports.CaptureRequestRepository {
	return &requestRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.capture-request-repository"),
	}
}

var (
	openRequestStatuses = []capturerequest.RequestStatus{
		capturerequest.RequestPending,
		capturerequest.RequestDelivered,
		capturerequest.RequestInProgress,
	}
	waitingRequestStatuses = []capturerequest.RequestStatus{
		capturerequest.RequestPending,
		capturerequest.RequestDelivered,
	}
)

func (r *requestRepository) Create(
	ctx context.Context,
	entity *capturerequest.CaptureRequest,
) (*capturerequest.CaptureRequest, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*capturerequest.CaptureRequest, error) {
		if err := capturerequestdb.CaptureRequests.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *requestRepository) Update(
	ctx context.Context,
	entity *capturerequest.CaptureRequest,
) (*capturerequest.CaptureRequest, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*capturerequest.CaptureRequest, error) {
		if err := capturerequestdb.CaptureRequests.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, requestErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *requestRepository) GetByID(
	ctx context.Context,
	req captureports.GetCaptureRequestByIDRequest,
) (*capturerequest.CaptureRequest, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*capturerequest.CaptureRequest, error) {
		cr := capturerequestdb.CaptureRequests
		entity, err := cr.Where(cr.ID.Eq(req.ID)).Load(cr.CaptureProfile()).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, requestErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *requestRepository) ListOpen(
	ctx context.Context,
	req captureports.ListOpenCaptureRequestsRequest,
) ([]*capturerequest.CaptureRequest, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*capturerequest.CaptureRequest, error) {
			cr := capturerequestdb.CaptureRequests
			return orEmptyRequests(cr.Where(
				cr.DeviceID.Eq(req.DeviceID),
				cr.Status.In(openRequestStatuses...),
			).
				Load(cr.CaptureProfile()).
				Order(cr.CreatedAt.Asc()).
				All(ctx, r.db.Keel(ctx)))
		},
	)
}

func (r *requestRepository) ListForTarget(
	ctx context.Context,
	req captureports.ListCaptureRequestsForTargetRequest,
) ([]*capturerequest.CaptureRequest, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*capturerequest.CaptureRequest, error) {
			limit := req.Limit
			if limit <= 0 || limit > maxRequestsForTarget {
				limit = maxRequestsForTarget
			}

			cr := capturerequestdb.CaptureRequests
			return orEmptyRequests(cr.Where(
				cr.TargetType.Eq(req.TargetType),
				cr.TargetID.Eq(req.TargetID),
				keel.When(req.UserID.IsNotNil(), cr.UserID.Eq(req.UserID)),
			).
				Order(cr.CreatedAt.Desc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx)))
		},
	)
}

func (r *requestRepository) ListExpired(
	ctx context.Context,
	now int64,
	limit int,
) ([]*capturerequest.CaptureRequest, error) {
	ctx = dbscope.WithSystem(ctx, "list expired capture requests across every organization")
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*capturerequest.CaptureRequest, error) {
			cr := capturerequestdb.CaptureRequests
			return orEmptyRequests(cr.AllTenants().
				Where(cr.Status.In(waitingRequestStatuses...), cr.ExpiresAt.Lte(now)).
				Order(cr.ExpiresAt.Asc()).
				Limit(boundedLimit(limit)).
				List(ctx, r.db.Keel(ctx)))
		},
	)
}

func orEmptyRequests(
	entities []*capturerequest.CaptureRequest,
	err error,
) ([]*capturerequest.CaptureRequest, error) {
	if err != nil {
		return nil, err
	}
	if entities == nil {
		return []*capturerequest.CaptureRequest{}, nil
	}

	return entities, nil
}
