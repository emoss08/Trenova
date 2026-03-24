package assignmentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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

func New(p Params) repositories.AssignmentRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.assignment-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListAssignmentsRequest,
) (*pagination.ListResult[*shipment.Assignment], error) {
	entities := make([]*shipment.Assignment, 0, req.Filter.Pagination.Limit)

	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where("a.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("a.business_unit_id = ?", req.Filter.TenantInfo.BuID).
		Where("a.archived_at IS NULL").
		Relation("ShipmentMove").
		Relation("Tractor").
		Relation("Trailer").
		Relation("PrimaryWorker").
		Relation("SecondaryWorker").
		Order("a.created_at DESC").
		Limit(req.Filter.Pagination.Limit).
		Offset(req.Filter.Pagination.Offset).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*shipment.Assignment]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetAssignmentByIDRequest,
) (*shipment.Assignment, error) {
	entity := new(shipment.Assignment)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("a.id = ?", req.AssignmentID).
		Where("a.organization_id = ?", req.TenantInfo.OrgID).
		Where("a.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("ShipmentMove").
		Relation("Tractor").
		Relation("Trailer").
		Relation("PrimaryWorker").
		Relation("SecondaryWorker").
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Assignment")
	}

	return entity, nil
}

func (r *repository) GetByMoveID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (*shipment.Assignment, error) {
	return r.getAssignmentByMoveID(ctx, tenantInfo, moveID)
}

func (r *repository) GetMoveByID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (*shipment.ShipmentMove, error) {
	move := new(shipment.ShipmentMove)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(move).
		Where("sm.id = ?", moveID).
		Where("sm.organization_id = ?", tenantInfo.OrgID).
		Where("sm.business_unit_id = ?", tenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Shipment move")
	}

	return move, nil
}

func (r *repository) FindInProgressByTractorID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tractorID pulid.ID,
	excludeMoveID pulid.ID,
) (*shipment.Assignment, error) {
	return r.findInProgressAssignment(ctx, tenantInfo, "a.tractor_id", tractorID, excludeMoveID)
}

func (r *repository) FindInProgressByTrailerID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	trailerID pulid.ID,
	excludeMoveID pulid.ID,
) (*shipment.Assignment, error) {
	return r.findInProgressAssignment(ctx, tenantInfo, "a.trailer_id", trailerID, excludeMoveID)
}

func (r *repository) FindNearestActualEventByTractorID(
	ctx context.Context,
	req repositories.FindNearestActualTimelineEventRequest,
	tractorID pulid.ID,
) (*repositories.ActualTimelineEvent, error) {
	return r.findNearestActualEvent(ctx, req, "a.tractor_id", tractorID)
}

func (r *repository) FindNearestActualEventByPrimaryWorkerID(
	ctx context.Context,
	req repositories.FindNearestActualTimelineEventRequest,
	workerID pulid.ID,
) (*repositories.ActualTimelineEvent, error) {
	return r.findNearestActualEvent(ctx, req, "a.primary_worker_id", workerID)
}

func (r *repository) FindOverlappingActualWindowByTractorID(
	ctx context.Context,
	req repositories.FindOverlappingActualTimelineWindowRequest,
	tractorID pulid.ID,
) (*repositories.ActualTimelineWindow, error) {
	return r.findOverlappingActualWindow(ctx, req, "a.tractor_id", tractorID)
}

func (r *repository) FindOverlappingActualWindowByPrimaryWorkerID(
	ctx context.Context,
	req repositories.FindOverlappingActualTimelineWindowRequest,
	workerID pulid.ID,
) (*repositories.ActualTimelineWindow, error) {
	return r.findOverlappingActualWindow(ctx, req, "a.primary_worker_id", workerID)
}

func (r *repository) getAssignmentByMoveID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) (*shipment.Assignment, error) {
	entity := new(shipment.Assignment)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("a.shipment_move_id = ?", moveID).
		Where("a.organization_id = ?", tenantInfo.OrgID).
		Where("a.business_unit_id = ?", tenantInfo.BuID).
		Where("a.archived_at IS NULL").
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}

	return entity, nil
}

func (r *repository) findInProgressAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	column string,
	equipmentID pulid.ID,
	excludeMoveID pulid.ID,
) (*shipment.Assignment, error) {
	entity := new(shipment.Assignment)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Join("JOIN shipment_moves AS sm ON sm.id = a.shipment_move_id").
		Where("a.organization_id = ?", tenantInfo.OrgID).
		Where("a.business_unit_id = ?", tenantInfo.BuID).
		Where("a.archived_at IS NULL").
		Where(column+" = ?", equipmentID).
		Where("sm.status = ?", shipment.MoveStatusInTransit)

	if !excludeMoveID.IsNil() {
		query = query.Where("a.shipment_move_id != ?", excludeMoveID)
	}

	if err := query.Scan(ctx); err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}

	return entity, nil
}

func (r *repository) findNearestActualEvent(
	ctx context.Context,
	req repositories.FindNearestActualTimelineEventRequest,
	column string,
	resourceID pulid.ID,
) (*repositories.ActualTimelineEvent, error) {
	event := new(repositories.ActualTimelineEvent)
	comparison := "<="
	order := "event.timestamp DESC"
	if req.Direction == repositories.ActualTimelineDirectionNext {
		comparison = ">="
		order = "event.timestamp ASC"
	}

	actualEvent := r.buildAssignedActualTimelineEventQuery(
		ctx,
		req.TenantInfo,
		column,
		resourceID,
	)

	query := r.db.DBForContext(ctx).
		NewSelect().
		With("actual_event", actualEvent).
		TableExpr("actual_event AS event").
		Column("event.timestamp", "event.event_type", "event.stop_id", "event.shipment_move_id", "event.shipment_id", "event.location_name").
		Where("event.timestamp "+comparison+" ?", req.Timestamp)

	if !req.ExcludeShipmentID.IsNil() {
		query = query.Where("event.shipment_id != ?", req.ExcludeShipmentID)
	}

	if err := query.OrderExpr(order).Limit(1).Scan(ctx, event); err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}

	return event, nil
}

func (r *repository) findOverlappingActualWindow(
	ctx context.Context,
	req repositories.FindOverlappingActualTimelineWindowRequest,
	column string,
	resourceID pulid.ID,
) (*repositories.ActualTimelineWindow, error) {
	window := new(repositories.ActualTimelineWindow)
	actualEvent := r.buildAssignedActualTimelineEventQuery(
		ctx,
		req.TenantInfo,
		column,
		resourceID,
	)

	query := r.db.DBForContext(ctx).
		NewSelect().
		With("actual_event", actualEvent).
		TableExpr("actual_event AS event").
		ColumnExpr("MIN(event.timestamp) AS start_timestamp").
		ColumnExpr("MAX(event.timestamp) AS end_timestamp").
		ColumnExpr("event.shipment_move_id").
		ColumnExpr("event.shipment_id").
		GroupExpr("event.shipment_move_id, event.shipment_id").
		Having("MIN(event.timestamp) <= ?", req.Timestamp).
		Having("MAX(event.timestamp) >= ?", req.Timestamp)

	if !req.ExcludeShipmentID.IsNil() {
		query = query.Where("event.shipment_id != ?", req.ExcludeShipmentID)
	}

	if err := query.OrderExpr("MIN(event.timestamp) DESC").Limit(1).Scan(ctx, window); err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}

	return window, nil
}

func (r *repository) buildAssignedActualTimelineEventQuery(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	column string,
	resourceID pulid.ID,
) *bun.SelectQuery {
	db := r.db.DBForContext(ctx)

	arrivalQuery := db.NewSelect().
		TableExpr("assignments AS a").
		ColumnExpr("s.actual_arrival AS timestamp").
		ColumnExpr("? AS event_type", repositories.ActualTimelineEventTypeArrival).
		ColumnExpr("s.id AS stop_id").
		ColumnExpr("s.shipment_move_id").
		ColumnExpr("sm.shipment_id").
		ColumnExpr("COALESCE(loc.name, '') AS location_name").
		Join("JOIN shipment_moves AS sm ON sm.id = a.shipment_move_id AND sm.organization_id = a.organization_id AND sm.business_unit_id = a.business_unit_id").
		Join("JOIN stops AS s ON s.shipment_move_id = sm.id AND s.organization_id = sm.organization_id AND s.business_unit_id = sm.business_unit_id").
		Join("LEFT JOIN locations AS loc ON loc.id = s.location_id AND loc.organization_id = s.organization_id AND loc.business_unit_id = s.business_unit_id").
		Where("a.organization_id = ?", tenantInfo.OrgID).
		Where("a.business_unit_id = ?", tenantInfo.BuID).
		Where("a.archived_at IS NULL").
		Where(column+" = ?", resourceID).
		Where("sm.status != ?", shipment.MoveStatusCanceled).
		Where("s.status != ?", shipment.StopStatusCanceled).
		Where("s.actual_arrival IS NOT NULL")

	departureQuery := db.NewSelect().
		TableExpr("assignments AS a").
		ColumnExpr("s.actual_departure AS timestamp").
		ColumnExpr("? AS event_type", repositories.ActualTimelineEventTypeDeparture).
		ColumnExpr("s.id AS stop_id").
		ColumnExpr("s.shipment_move_id").
		ColumnExpr("sm.shipment_id").
		ColumnExpr("COALESCE(loc.name, '') AS location_name").
		Join("JOIN shipment_moves AS sm ON sm.id = a.shipment_move_id AND sm.organization_id = a.organization_id AND sm.business_unit_id = a.business_unit_id").
		Join("JOIN stops AS s ON s.shipment_move_id = sm.id AND s.organization_id = sm.organization_id AND s.business_unit_id = sm.business_unit_id").
		Join("LEFT JOIN locations AS loc ON loc.id = s.location_id AND loc.organization_id = s.organization_id AND loc.business_unit_id = s.business_unit_id").
		Where("a.organization_id = ?", tenantInfo.OrgID).
		Where("a.business_unit_id = ?", tenantInfo.BuID).
		Where("a.archived_at IS NULL").
		Where(column+" = ?", resourceID).
		Where("sm.status != ?", shipment.MoveStatusCanceled).
		Where("s.status != ?", shipment.StopStatusCanceled).
		Where("s.actual_departure IS NOT NULL")

	return arrivalQuery.UnionAll(departureQuery)
}

func (r *repository) Create(
	ctx context.Context,
	entity *shipment.Assignment,
) (*shipment.Assignment, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, &repositories.GetAssignmentByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		AssignmentID: entity.ID,
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *shipment.Assignment,
) (*shipment.Assignment, error) {
	ov := entity.Version
	entity.Version++

	result, err := r.db.DBForContext(ctx).NewUpdate().
		Model(entity).
		Column(
			"primary_worker_id",
			"tractor_id",
			"trailer_id",
			"secondary_worker_id",
			"status",
			"version",
			"updated_at",
		).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", ov).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update assignment: %w", err)
	}

	if err = dberror.CheckRowsAffected(result, "Assignment", entity.ID.String()); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, &repositories.GetAssignmentByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		AssignmentID: entity.ID,
	})
}

func (r *repository) Unassign(
	ctx context.Context,
	entity *shipment.Assignment,
) (*shipment.Assignment, error) {
	ov := entity.Version
	entity.Version++

	now := timeutils.NowUnix()
	entity.Status = shipment.AssignmentStatusCanceled
	entity.ArchivedAt = &now

	result, err := r.db.DBForContext(ctx).NewUpdate().
		Model(entity).
		Column("status", "archived_at", "version", "updated_at").
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", ov).
		Where("archived_at IS NULL").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("unassign assignment: %w", err)
	}

	if err = dberror.CheckRowsAffected(result, "Assignment", entity.ID.String()); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, &repositories.GetAssignmentByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		AssignmentID: entity.ID,
	})
}
