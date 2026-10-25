package servicefailurerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure/servicefailuredb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/keelkinds"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const serviceFailureEntity = "Service failure"

var serviceFailureErrors = dberror.Mapping{Entity: serviceFailureEntity}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) servicefailureports.ServiceFailureRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.service-failure-repository"),
	}
}

func listLoads() []keel.Loader {
	sf := servicefailuredb.ServiceFailures
	return []keel.Loader{sf.ReasonCode(), sf.Shipment(), sf.Stop().Location().State()}
}

func detailQuery() *keel.Query[servicefailure.ServiceFailure] {
	sf := servicefailuredb.ServiceFailures
	return sf.Load(
		sf.ReasonCode(),
		sf.Shipment(),
		sf.ShipmentMove(),
		sf.Stop().Location().State(),
		sf.CreatedBy(),
		sf.ReviewedBy(),
		sf.ResolvedBy(),
		sf.VoidedBy(),
	)
}

func optionalEq(col keelkinds.PulidIDCol, id *pulid.ID) keel.Pred {
	if id == nil {
		return nil
	}

	return col.Eq(*id)
}

func unresolved() keel.Pred {
	return servicefailuredb.ServiceFailures.Status.In(servicefailure.UnresolvedStatuses()...)
}

func (r *repository) List(
	ctx context.Context,
	req *servicefailureports.ListServiceFailuresRequest,
) (*pagination.ListResult[*servicefailure.ServiceFailure], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*servicefailure.ServiceFailure], error) {
			req.EnsureFilter()
			if req.Filter.Pagination.Limit <= 0 {
				req.Filter.Pagination.Limit = 50
			}

			sf := servicefailuredb.ServiceFailures
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				sf.Where(keel.When(req.ShipmentID.IsNotNil(), sf.ShipmentID.Eq(req.ShipmentID))),
				&listquery.Request[servicefailure.ServiceFailure]{
					Filter: req.Filter,
					Shape: func(
						q *keel.Query[servicefailure.ServiceFailure],
					) *keel.Query[servicefailure.ServiceFailure] {
						return q.Load(listLoads()...)
					},
					Then: []keel.Order{sf.CreatedAt.Desc()},
				},
			)
			if err != nil {
				return nil, fmt.Errorf("list service failures: %w", err)
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *servicefailureports.ListServiceFailureConnectionRequest,
) (*pagination.CursorListResult[*servicefailure.ServiceFailure], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*servicefailure.ServiceFailure], error) {
			sf := servicefailuredb.ServiceFailures
			q := sf.Where(
				optionalEq(sf.ShipmentID, req.ShipmentID),
				optionalEq(sf.Stop().LocationID, req.LocationID),
			)

			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				q,
				&listquery.Request[servicefailure.ServiceFailure]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.ServiceFailureColumns,
					Shape: func(
						q *keel.Query[servicefailure.ServiceFailure],
					) *keel.Query[servicefailure.ServiceFailure] {
						return q.Load(listLoads()...)
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list service failures", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *servicefailureports.GetServiceFailureByIDRequest,
) (*servicefailure.ServiceFailure, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*servicefailure.ServiceFailure, error) {
		return r.get(ctx, req.ID)
	})
}

func (r *repository) get(
	ctx context.Context,
	id pulid.ID,
) (*servicefailure.ServiceFailure, error) {
	sf := servicefailuredb.ServiceFailures
	entity, err := detailQuery().Where(sf.ID.Eq(id)).One(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, serviceFailureErrors.Map(err, id.String())
	}

	return entity, nil
}

func (r *repository) GetByShipment(
	ctx context.Context,
	req *servicefailureports.GetServiceFailureByShipmentRequest,
) (*servicefailure.ServiceFailure, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*servicefailure.ServiceFailure, error) {
		sf := servicefailuredb.ServiceFailures
		entity, err := detailQuery().
			Where(sf.ID.Eq(req.ID), sf.ShipmentID.Eq(req.ShipmentID)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, serviceFailureErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *servicefailure.ServiceFailure,
) (*servicefailure.ServiceFailure, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*servicefailure.ServiceFailure, error) {
		err := servicefailuredb.ServiceFailures.Create(ctx, r.db.Keel(ctx), entity)
		if err != nil {
			return nil, mapServiceFailureError(err, entity)
		}

		return r.get(ctx, entity.ID)
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *servicefailure.ServiceFailure,
) (*servicefailure.ServiceFailure, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*servicefailure.ServiceFailure, error) {
		sf := servicefailuredb.ServiceFailures
		if err := sf.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			sf.ReasonCodeID,
			sf.Status,
			sf.Notes,
			sf.InternalNotes,
			sf.X12StatusCodeOverride,
			sf.X12ReasonCodeOverride,
			sf.X12ExceptionCode,
			sf.ReviewedAt,
			sf.ReviewedByID,
			sf.ResolvedAt,
			sf.ResolvedByID,
			sf.VoidedAt,
			sf.VoidedByID,
			sf.VoidReason,
		); err != nil {
			return nil, mapServiceFailureError(err, entity)
		}

		return r.get(ctx, entity.ID)
	})
}

func (r *repository) UpdateDetectionSnapshot(
	ctx context.Context,
	entity *servicefailure.ServiceFailure,
) (*servicefailure.ServiceFailure, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*servicefailure.ServiceFailure, error) {
		sf := servicefailuredb.ServiceFailures
		reason := sf.ReasonCodeID.ToNull()
		if entity.ReasonCodeID != nil {
			reason = sf.ReasonCodeID.To(*entity.ReasonCodeID)
		}

		n, err := sf.Where(sf.ID.Eq(entity.ID), unresolved()).Update(
			ctx,
			r.db.Keel(ctx),
			sf.ScheduledCutoff.To(entity.ScheduledCutoff),
			sf.ActualArrival.To(entity.ActualArrival),
			sf.GracePeriodMinutes.To(int64(entity.GracePeriodMinutes)),
			sf.LateMinutes.To(entity.LateMinutes),
			reason,
			sf.Notes.To(entity.Notes),
		)
		if err != nil {
			return nil, mapServiceFailureError(err, entity)
		}
		if err = dberror.CheckAffected(n, serviceFailureEntity, entity.ID.String()); err != nil {
			return nil, err
		}

		return r.get(ctx, entity.ID)
	})
}

func (r *repository) FindUnresolvedByStop(
	ctx context.Context,
	req *servicefailureports.ServiceFailureActiveStopRequest,
) (*servicefailure.ServiceFailure, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*servicefailure.ServiceFailure, error) {
		sf := servicefailuredb.ServiceFailures
		entity, err := detailQuery().
			Where(
				sf.ShipmentID.Eq(req.ShipmentID),
				sf.ShipmentMoveID.Eq(req.ShipmentMoveID),
				sf.StopID.Eq(req.StopID),
				sf.Type.Eq(req.Type),
				unresolved(),
			).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, serviceFailureErrors.Map(err, req.StopID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListUnresolvedByShipment(
	ctx context.Context,
	req *servicefailureports.ServiceFailuresByShipmentRequest,
) ([]*servicefailure.ServiceFailure, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*servicefailure.ServiceFailure, error) {
			sf := servicefailuredb.ServiceFailures
			entities, err := sf.Where(sf.ShipmentID.Eq(req.ShipmentID), unresolved()).
				Load(sf.ReasonCode()).
				Order(sf.CreatedAt.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list unresolved service failures: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) ListUnresolvedByShipmentIDs(
	ctx context.Context,
	req *servicefailureports.ServiceFailuresByShipmentIDsRequest,
) ([]*servicefailure.ServiceFailure, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*servicefailure.ServiceFailure, error) {
			if len(req.ShipmentIDs) == 0 {
				return []*servicefailure.ServiceFailure{}, nil
			}

			sf := servicefailuredb.ServiceFailures
			entities, err := sf.Where(sf.ShipmentID.In(req.ShipmentIDs...), unresolved()).
				Load(sf.ReasonCode()).
				Order(sf.CreatedAt.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list unresolved service failures by shipments: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) CountUnresolvedByShipment(
	ctx context.Context,
	req *servicefailureports.ServiceFailuresByShipmentRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		sf := servicefailuredb.ServiceFailures
		count, err := sf.Where(sf.ShipmentID.Eq(req.ShipmentID), unresolved()).
			Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count unresolved service failures: %w", err)
		}

		return int(count), nil
	})
}

func (r *repository) CountUnresolved(
	ctx context.Context,
	_ *servicefailureports.CountUnresolvedServiceFailuresRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		count, err := servicefailuredb.ServiceFailures.Where(unresolved()).
			Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count unresolved service failures for tenant: %w", err)
		}

		return int(count), nil
	})
}

func mapServiceFailureError(err error, entity *servicefailure.ServiceFailure) error {
	if dberror.IsUniqueConstraintViolation(err) {
		switch dberror.ExtractConstraintName(err) {
		case "ux_sf_active_stop_type":
			return errortypes.NewBusinessError(
				"An unresolved service failure already exists for this stop",
			).
				WithParam("shipmentId", entity.ShipmentID.String()).
				WithParam("stopId", entity.StopID.String())
		case "ux_sf_tenant_number":
			return errortypes.NewBusinessError("Service failure number already exists").
				WithParam("number", entity.Number)
		}
	}

	return serviceFailureErrors.Map(err, entity.ID.String())
}
