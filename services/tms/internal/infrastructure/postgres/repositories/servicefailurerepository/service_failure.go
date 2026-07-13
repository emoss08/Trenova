package servicefailurerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) repositories.ServiceFailureRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.service-failure-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListServiceFailuresRequest,
) (*pagination.ListResult[*servicefailure.ServiceFailure], error) {
	req.EnsureFilter()
	if req.Filter.Pagination.Limit <= 0 {
		req.Filter.Pagination.Limit = 50
	}

	entities := make([]*servicefailure.ServiceFailure, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			q = querybuilder.ApplyFilters(q, "sf", req.Filter, (*servicefailure.ServiceFailure)(nil))
			if req.ShipmentID.IsNotNil() {
				q = q.Where("sf.shipment_id = ?", req.ShipmentID)
			}
			return q.Limit(req.Filter.Pagination.SafeLimit()).
				Offset(req.Filter.Pagination.SafeOffset())
		}).
		Relation("ReasonCode").
		Relation("Shipment").
		Relation("Stop").
		Relation("Stop.Location").
		Relation("Stop.Location.State").
		Order("sf.created_at DESC").
		ScanAndCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("list service failures: %w", err)
	}

	return &pagination.ListResult[*servicefailure.ServiceFailure]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListServiceFailureConnectionRequest,
) (*bun.SelectQuery, error) {
	q, err := querybuilder.ApplyCursorFilters(
		q,
		buncolgen.ServiceFailureTable.Alias,
		req.Filter,
		req.Cursor,
		(*servicefailure.ServiceFailure)(nil),
	)
	if err != nil {
		return q, err
	}

	return applyShipmentFilter(q, req.ShipmentID), nil
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListServiceFailureConnectionRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.ServiceFailureTable.Alias,
		req.Filter,
		(*servicefailure.ServiceFailure)(nil),
	)

	return applyShipmentFilter(q, req.ShipmentID)
}

func applyShipmentFilter(q *bun.SelectQuery, shipmentID *pulid.ID) *bun.SelectQuery {
	if shipmentID == nil {
		return q
	}

	return q.Where(buncolgen.ServiceFailureColumns.ShipmentID.Eq(), *shipmentID)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListServiceFailureConnectionRequest,
) (*pagination.CursorListResult[*servicefailure.ServiceFailure], error) {
	log := r.l.With(
		zap.String("operation", "ListConnection"),
		zap.Any("request", req),
	)

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*servicefailure.ServiceFailure)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.applyTotalCountFilters(sq, req)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count service failures", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*servicefailure.ServiceFailure]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*servicefailure.ServiceFailure) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					ColumnExpr(buncolgen.ServiceFailureTable.All()).
					Relation("Shipment").
					Relation("Stop").
					Relation("Stop.Location").
					Relation("Stop.Location.State").
					Relation("ReasonCode")
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return r.applyCursorPageFilters(sq, req)
			},
		})
	if err != nil {
		log.Error("failed to scan service failures", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetServiceFailureByIDRequest,
) (*servicefailure.ServiceFailure, error) {
	entity := new(servicefailure.ServiceFailure)
	err := r.baseGetQuery(ctx, entity).
		Where("sf.id = ?", req.ID).
		Where("sf.organization_id = ?", req.TenantInfo.OrgID).
		Where("sf.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Service failure")
	}

	return entity, nil
}

func (r *repository) GetByShipment(
	ctx context.Context,
	req *repositories.GetServiceFailureByShipmentRequest,
) (*servicefailure.ServiceFailure, error) {
	entity := new(servicefailure.ServiceFailure)
	err := r.baseGetQuery(ctx, entity).
		Where("sf.id = ?", req.ID).
		Where("sf.shipment_id = ?", req.ShipmentID).
		Where("sf.organization_id = ?", req.TenantInfo.OrgID).
		Where("sf.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Service failure")
	}

	return entity, nil
}

func (r *repository) baseGetQuery(
	ctx context.Context,
	entity *servicefailure.ServiceFailure,
) *bun.SelectQuery {
	return r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("ReasonCode").
		Relation("Shipment").
		Relation("ShipmentMove").
		Relation("Stop").
		Relation("Stop.Location").
		Relation("Stop.Location.State").
		Relation("CreatedBy").
		Relation("ReviewedBy").
		Relation("ResolvedBy").
		Relation("VoidedBy")
}

func (r *repository) Create(
	ctx context.Context,
	entity *servicefailure.ServiceFailure,
) (*servicefailure.ServiceFailure, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, mapServiceFailureConstraint(err, entity)
	}

	return r.GetByID(ctx, &repositories.GetServiceFailureByIDRequest{
		ID:         entity.ID,
		TenantInfo: serviceFailureTenantInfo(entity),
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *servicefailure.ServiceFailure,
) (*servicefailure.ServiceFailure, error) {
	now := timeutils.NowUnix()
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*servicefailure.ServiceFailure)(nil)).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("reason_code_id = ?", entity.ReasonCodeID).
		Set("status = ?", entity.Status).
		Set("notes = ?", entity.Notes).
		Set("internal_notes = ?", entity.InternalNotes).
		Set("x12_status_code_override = ?", entity.X12StatusCodeOverride).
		Set("x12_reason_code_override = ?", entity.X12ReasonCodeOverride).
		Set("x12_exception_code = ?", entity.X12ExceptionCode).
		Set("reviewed_at = ?", entity.ReviewedAt).
		Set("reviewed_by_id = ?", entity.ReviewedByID).
		Set("resolved_at = ?", entity.ResolvedAt).
		Set("resolved_by_id = ?", entity.ResolvedByID).
		Set("voided_at = ?", entity.VoidedAt).
		Set("voided_by_id = ?", entity.VoidedByID).
		Set("void_reason = ?", entity.VoidReason).
		Set("version = version + 1").
		Set("updated_at = ?", now).
		Exec(ctx)
	if err != nil {
		return nil, mapServiceFailureConstraint(err, entity)
	}
	if err = dberror.CheckRowsAffected(result, "Service failure", entity.ID.String()); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, &repositories.GetServiceFailureByIDRequest{
		ID:         entity.ID,
		TenantInfo: serviceFailureTenantInfo(entity),
	})
}

func (r *repository) UpdateDetectionSnapshot(
	ctx context.Context,
	entity *servicefailure.ServiceFailure,
) (*servicefailure.ServiceFailure, error) {
	now := timeutils.NowUnix()
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*servicefailure.ServiceFailure)(nil)).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("status IN (?)", bun.In([]servicefailure.Status{
			servicefailure.StatusOpen,
			servicefailure.StatusReviewed,
		})).
		Set("scheduled_cutoff = ?", entity.ScheduledCutoff).
		Set("actual_arrival = ?", entity.ActualArrival).
		Set("grace_period_minutes = ?", entity.GracePeriodMinutes).
		Set("late_minutes = ?", entity.LateMinutes).
		Set("reason_code_id = ?", entity.ReasonCodeID).
		Set("notes = ?", entity.Notes).
		Set("version = version + 1").
		Set("updated_at = ?", now).
		Exec(ctx)
	if err != nil {
		return nil, mapServiceFailureConstraint(err, entity)
	}
	if err = dberror.CheckRowsAffected(result, "Service failure", entity.ID.String()); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, &repositories.GetServiceFailureByIDRequest{
		ID:         entity.ID,
		TenantInfo: serviceFailureTenantInfo(entity),
	})
}

func (r *repository) FindUnresolvedByStop(
	ctx context.Context,
	req *repositories.ServiceFailureActiveStopRequest,
) (*servicefailure.ServiceFailure, error) {
	entity := new(servicefailure.ServiceFailure)
	err := r.baseGetQuery(ctx, entity).
		Where("sf.organization_id = ?", req.TenantInfo.OrgID).
		Where("sf.business_unit_id = ?", req.TenantInfo.BuID).
		Where("sf.shipment_id = ?", req.ShipmentID).
		Where("sf.shipment_move_id = ?", req.ShipmentMoveID).
		Where("sf.stop_id = ?", req.StopID).
		Where("sf.type = ?", req.Type).
		Where("sf.status IN (?)", bun.In([]servicefailure.Status{
			servicefailure.StatusOpen,
			servicefailure.StatusReviewed,
		})).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Service failure")
	}

	return entity, nil
}

func (r *repository) ListUnresolvedByShipment(
	ctx context.Context,
	req *repositories.ServiceFailuresByShipmentRequest,
) ([]*servicefailure.ServiceFailure, error) {
	entities := make([]*servicefailure.ServiceFailure, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where("sf.organization_id = ?", req.TenantInfo.OrgID).
		Where("sf.business_unit_id = ?", req.TenantInfo.BuID).
		Where("sf.shipment_id = ?", req.ShipmentID).
		Where("sf.status IN (?)", bun.In([]servicefailure.Status{
			servicefailure.StatusOpen,
			servicefailure.StatusReviewed,
		})).
		Relation("ReasonCode").
		Order("sf.created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list unresolved service failures: %w", err)
	}

	return entities, nil
}

func (r *repository) CountUnresolvedByShipment(
	ctx context.Context,
	req *repositories.ServiceFailuresByShipmentRequest,
) (int, error) {
	count, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*servicefailure.ServiceFailure)(nil)).
		Where("sf.organization_id = ?", req.TenantInfo.OrgID).
		Where("sf.business_unit_id = ?", req.TenantInfo.BuID).
		Where("sf.shipment_id = ?", req.ShipmentID).
		Where("sf.status IN (?)", bun.In([]servicefailure.Status{
			servicefailure.StatusOpen,
			servicefailure.StatusReviewed,
		})).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count unresolved service failures: %w", err)
	}

	return count, nil
}

func mapServiceFailureConstraint(err error, entity *servicefailure.ServiceFailure) error {
	if dberror.IsUniqueConstraintViolation(err) &&
		dberror.ExtractConstraintName(err) == "ux_sf_active_stop_type" {
		return errortypes.NewBusinessError("An unresolved service failure already exists for this stop").
			WithParam("shipmentId", entity.ShipmentID.String()).
			WithParam("stopId", entity.StopID.String())
	}
	if dberror.IsUniqueConstraintViolation(err) &&
		dberror.ExtractConstraintName(err) == "ux_sf_tenant_number" {
		return errortypes.NewBusinessError("Service failure number already exists").
			WithParam("number", entity.Number)
	}
	return err
}

func serviceFailureTenantInfo(entity *servicefailure.ServiceFailure) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
}
