package tractorrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
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

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.TractorRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.tractor-repository"),
	}
}

func (r *repository) applyListFilters(
	q *bun.SelectQuery,
	req *repositories.ListTractorsRequest,
) *bun.SelectQuery {
	log := r.l.With(
		zap.String("operation", "applyListFilters"),
		zap.Any("request", req),
	)

	q = applyTractorRelations(q, req.TractorRelationIncludes)

	q = querybuilder.ApplyFilters(
		q,
		buncolgen.TractorTable.Alias,
		req.Filter,
		(*tractor.Tractor)(nil),
	)

	if req.Status != "" {
		status, err := domaintypes.EquipmentStatusFromString(req.Status)
		if err != nil {
			log.Error("failed to parse equipment status", zap.Error(err))
			return q
		}

		q = q.Where(buncolgen.TractorColumns.Status.Eq(), status)
	}

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListTractorsRequest,
) (*pagination.ListResult[*tractor.Tractor], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*tractor.Tractor, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTractorColumns(sq, req.TractorRelationIncludes)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTractorLastKnownLocationJoin(sq, req.TractorRelationIncludes)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.applyListFilters(sq, req).
				Limit(req.Filter.Pagination.SafeLimit()).
				Offset(req.Filter.Pagination.SafeOffset())
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count tractors", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*tractor.Tractor]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) ListCursor(
	ctx context.Context,
	req *repositories.ListTractorsCursorRequest,
) (*pagination.CursorListResult[*tractor.Tractor], error) {
	log := r.l.With(
		zap.String("operation", "ListCursor"),
		zap.Any("request", req),
	)

	filter := *req.Filter
	filter.Sort = nil
	offsetReq := &repositories.ListTractorsRequest{
		Filter:                  &filter,
		TractorRelationIncludes: req.TractorRelationIncludes,
		Status:                  req.Status,
	}
	limit := req.Cursor.Limit
	entities := make([]*tractor.Tractor, 0, limit+1)
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTractorColumns(sq, req.TractorRelationIncludes)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTractorLastKnownLocationJoin(sq, req.TractorRelationIncludes)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			q := r.applyListFilters(sq, offsetReq)
			if req.Cursor.After != "" {
				q = q.WhereGroup(" AND ", func(cq *bun.SelectQuery) *bun.SelectQuery {
					return cq.
						Where(buncolgen.TractorColumns.CreatedAt.Lt(), req.Cursor.Cursor.CreatedAt).
						WhereOr(
							buncolgen.TractorColumns.CreatedAt.Eq()+
								" AND "+buncolgen.TractorColumns.ID.Lt(),
							req.Cursor.Cursor.CreatedAt,
							req.Cursor.Cursor.ID,
						)
				})
			}
			return q.
				Order(buncolgen.TractorColumns.CreatedAt.OrderDesc()).
				Order(buncolgen.TractorColumns.ID.OrderDesc()).
				Limit(limit + 1)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to scan cursor tractors", zap.Error(err))
		return nil, err
	}

	hasNextPage := len(entities) > limit
	if hasNextPage {
		entities = entities[:limit]
	}

	return &pagination.CursorListResult[*tractor.Tractor]{
		Items:       entities,
		HasNextPage: hasNextPage,
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *tractor.Tractor,
) (*tractor.Tractor, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("code", entity.Code),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create tractor", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *tractor.Tractor,
) (*tractor.Tractor, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.TractorColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update tractor", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "Tractor", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetTractorByIDRequest,
) (*tractor.Tractor, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(tractor.Tractor)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTractorColumns(sq, req.TractorRelationIncludes)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTractorLastKnownLocationJoin(sq, req.TractorRelationIncludes)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTractorRelations(sq, req.TractorRelationIncludes)
		}).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TractorScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.TractorColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get tractor", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Tractor")
	}

	return entity, nil
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetTractorsByIDsRequest,
) ([]*tractor.Tractor, error) {
	log := r.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Any("request", req),
	)

	entities := make([]*tractor.Tractor, 0, len(req.TractorIDs))
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTractorColumns(sq, req.TractorRelationIncludes)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTractorLastKnownLocationJoin(sq, req.TractorRelationIncludes)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTractorRelations(sq, req.TractorRelationIncludes)
		}).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TractorScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.TractorColumns.ID.In(), bun.List(req.TractorIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get tractors", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Tractor")
	}

	return entities, nil
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateTractorStatusRequest,
) ([]*tractor.Tractor, error) {
	log := r.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	entities := make([]*tractor.Tractor, 0, len(req.TractorIDs))
	results, err := r.db.DB().
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.TractorScopeTenantUpdate(uq, req.TenantInfo).
				Where(buncolgen.TractorColumns.ID.In(), bun.List(req.TractorIDs))
		}).
		Set(buncolgen.TractorColumns.Status.Set(), req.Status).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update tractor status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(results, "Tractor", req.TractorIDs); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.TractorSelectOptionsRequest,
) (*pagination.ListResult[*tractor.Tractor], error) {
	cols := buncolgen.TractorColumns
	rel := buncolgen.TractorRelations

	return dbhelper.SelectOptions[*tractor.Tractor](
		ctx,
		r.db.DB(),
		req.SelectOptionsRequest,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				cols.ID,
				cols.CreatedAt,
				cols.Status,
				cols.Code,
				cols.PrimaryWorkerID,
				cols.SecondaryWorkerID,
			},
			OrgColumnRef:     &cols.OrganizationID,
			BuColumnRef:      &cols.BusinessUnitID,
			SearchColumnRefs: []buncolgen.Column{cols.Code},
			EntityName:       "Tractor",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where(
					cols.Status.Eq(),
					domaintypes.EquipmentStatusAvailable,
				).
					Relation(rel.PrimaryWorker).
					Relation(rel.SecondaryWorker)
			},
		},
	)
}

func applyTractorRelations(
	q *bun.SelectQuery,
	includes repositories.TractorRelationIncludes,
) *bun.SelectQuery {
	rel := buncolgen.TractorRelations
	equipmentTypeRel := buncolgen.EquipmentTypeRelations
	equipmentManufacturerRel := buncolgen.EquipmentManufacturerRelations
	fleetCodeRel := buncolgen.FleetCodeRelations
	organizationRel := buncolgen.OrganizationRelations
	workerRel := buncolgen.WorkerRelations
	includeBusinessUnit := includes.IncludeTenantDetails || includes.IncludeBusinessUnit
	includeOrganization := includes.IncludeTenantDetails || includes.IncludeOrganization
	includeEquipmentType := includes.IncludeEquipmentDetails || includes.IncludeEquipmentType
	includeEquipmentManufacturer := includes.IncludeEquipmentDetails ||
		includes.IncludeEquipmentManufacturer
	includeFleetCode := includes.IncludeFleetDetails || includes.IncludeFleetCode
	includePrimaryWorker := includes.IncludeWorkerDetails || includes.IncludePrimaryWorker
	includeSecondaryWorker := includes.IncludeWorkerDetails || includes.IncludeSecondaryWorker

	if includeBusinessUnit {
		q = q.Relation(rel.BusinessUnit)
	}
	if includeOrganization {
		q = q.Relation(rel.Organization)
	}
	if includes.IncludeTenantDetails {
		q = q.Relation(buncolgen.Rel(rel.Organization, organizationRel.BusinessUnit)).
			Relation(buncolgen.Rel(rel.Organization, organizationRel.State))
	}
	if includes.IncludeState {
		q = q.Relation(rel.State)
	}
	if includeEquipmentType {
		q = q.Relation(rel.EquipmentType, dbhelper.RelationColumns(includes.EquipmentTypeColumns))
	}
	if includeEquipmentManufacturer {
		q = q.Relation(
			rel.EquipmentManufacturer,
			dbhelper.RelationColumns(includes.EquipmentManufacturerColumns),
		)
	}
	if includeEquipmentType && includes.IncludeTenantDetails {
		q = q.Relation(buncolgen.Rel(rel.EquipmentType, equipmentTypeRel.BusinessUnit)).
			Relation(buncolgen.Rel(rel.EquipmentType, equipmentTypeRel.Organization))
	}
	if includeEquipmentManufacturer && includes.IncludeTenantDetails {
		q = q.Relation(buncolgen.Rel(
			rel.EquipmentManufacturer,
			equipmentManufacturerRel.BusinessUnit,
		)).
			Relation(buncolgen.Rel(
				rel.EquipmentManufacturer,
				equipmentManufacturerRel.Organization,
			))
	}
	if includeFleetCode {
		q = q.Relation(rel.FleetCode, dbhelper.RelationColumns(includes.FleetCodeColumns))
	}
	if includeFleetCode && includes.IncludeTenantDetails {
		q = q.Relation(buncolgen.Rel(rel.FleetCode, fleetCodeRel.BusinessUnit)).
			Relation(buncolgen.Rel(rel.FleetCode, fleetCodeRel.Organization))
	}
	if includePrimaryWorker {
		q = q.Relation(rel.PrimaryWorker, dbhelper.RelationColumns(includes.PrimaryWorkerColumns))
	}
	if includeSecondaryWorker {
		q = q.Relation(
			rel.SecondaryWorker,
			dbhelper.RelationColumns(includes.SecondaryWorkerColumns),
		)
	}
	if includePrimaryWorker && includes.IncludeTenantDetails {
		q = q.Relation(buncolgen.Rel(rel.PrimaryWorker, workerRel.BusinessUnit)).
			Relation(buncolgen.Rel(rel.PrimaryWorker, workerRel.Organization)).
			Relation(buncolgen.Rel(rel.PrimaryWorker, workerRel.State))
	}
	if includePrimaryWorker && includes.IncludePrimaryWorkerState {
		q = q.Relation(buncolgen.Rel(rel.PrimaryWorker, workerRel.State))
	}
	if includePrimaryWorker && includes.IncludePrimaryWorkerFleet {
		q = q.Relation(buncolgen.Rel(rel.PrimaryWorker, workerRel.FleetCode))
	}
	if includePrimaryWorker && includes.IncludePrimaryWorkerManager {
		q = q.Relation(buncolgen.Rel(rel.PrimaryWorker, workerRel.Manager))
	}
	if includeSecondaryWorker && includes.IncludeTenantDetails {
		q = q.Relation(buncolgen.Rel(rel.SecondaryWorker, workerRel.BusinessUnit)).
			Relation(buncolgen.Rel(rel.SecondaryWorker, workerRel.Organization)).
			Relation(buncolgen.Rel(rel.SecondaryWorker, workerRel.State))
	}
	if includeSecondaryWorker && includes.IncludeSecondaryWorkerState {
		q = q.Relation(buncolgen.Rel(rel.SecondaryWorker, workerRel.State))
	}
	if includeSecondaryWorker && includes.IncludeSecondaryWorkerFleet {
		q = q.Relation(buncolgen.Rel(rel.SecondaryWorker, workerRel.FleetCode))
	}
	if includeSecondaryWorker && includes.IncludeSecondaryWorkerManager {
		q = q.Relation(buncolgen.Rel(rel.SecondaryWorker, workerRel.Manager))
	}

	return q
}

func applyTractorColumns(
	q *bun.SelectQuery,
	includes repositories.TractorRelationIncludes,
) *bun.SelectQuery {
	if len(includes.TractorColumns) == 0 {
		return q
	}

	return q.Column(includes.TractorColumns...)
}

func applyTractorLastKnownLocationJoin(
	q *bun.SelectQuery,
	includes repositories.TractorRelationIncludes,
) *bun.SelectQuery {
	if !includes.IncludeLastKnownLocation {
		return q
	}

	if len(includes.TractorColumns) == 0 {
		q = q.Column("trac.*")
	}

	return q.ColumnExpr("ec.current_location_id AS last_known_location_id").
		ColumnExpr("COALESCE(lkl.name, '') AS last_known_location_name").
		Join("LEFT JOIN equipment_continuity AS ec ON ec.equipment_id = trac.id AND ec.equipment_type = ? AND ec.organization_id = trac.organization_id AND ec.business_unit_id = trac.business_unit_id AND ec.is_current = TRUE", "Tractor").
		Join("LEFT JOIN locations AS lkl ON lkl.id = ec.current_location_id AND lkl.organization_id = ec.organization_id AND lkl.business_unit_id = ec.business_unit_id")
}
