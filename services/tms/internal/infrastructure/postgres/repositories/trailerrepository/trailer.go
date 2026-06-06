package trailerrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/trailer"
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

func New(p Params) repositories.TrailerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.trailer-repository"),
	}
}

func (r *repository) applyListFilters(
	q *bun.SelectQuery,
	req *repositories.ListTrailersRequest,
) *bun.SelectQuery {
	log := r.l.With(
		zap.String("operation", "applyListFilters"),
		zap.Any("request", req),
	)

	q = applyTrailerRelations(q, req.TrailerRelationIncludes)

	q = querybuilder.ApplyFilters(
		q,
		buncolgen.TrailerTable.Alias,
		req.Filter,
		(*trailer.Trailer)(nil),
	)

	if req.Status != "" {
		status, err := domaintypes.EquipmentStatusFromString(req.Status)
		if err != nil {
			log.Error("failed to parse equipment status", zap.Error(err))
			return q
		}

		q = q.Where(buncolgen.TrailerColumns.Status.Eq(), status)
	}

	return q
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListTrailersRequest,
) (*pagination.ListResult[*trailer.Trailer], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*trailer.Trailer, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTrailerColumns(sq, req.TrailerRelationIncludes)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyLastKnownLocationJoin(sq, req.TrailerRelationIncludes)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.applyListFilters(sq, req).
				Limit(req.Filter.Pagination.SafeLimit()).
				Offset(req.Filter.Pagination.SafeOffset())
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count trailers", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*trailer.Trailer]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) ListCursor(
	ctx context.Context,
	req *repositories.ListTrailersCursorRequest,
) (*pagination.CursorListResult[*trailer.Trailer], error) {
	log := r.l.With(
		zap.String("operation", "ListCursor"),
		zap.Any("request", req),
	)

	filter := *req.Filter
	filter.Sort = nil
	offsetReq := &repositories.ListTrailersRequest{
		Filter:                  &filter,
		TrailerRelationIncludes: req.TrailerRelationIncludes,
		Status:                  req.Status,
	}
	limit := req.Cursor.Limit
	entities := make([]*trailer.Trailer, 0, limit+1)
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTrailerColumns(sq, req.TrailerRelationIncludes)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyLastKnownLocationJoin(sq, req.TrailerRelationIncludes)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			q := r.applyListFilters(sq, offsetReq)
			if req.Cursor.After != "" {
				q = q.WhereGroup(" AND ", func(cq *bun.SelectQuery) *bun.SelectQuery {
					return cq.
						Where(buncolgen.TrailerColumns.CreatedAt.Lt(), req.Cursor.Cursor.CreatedAt).
						WhereOr(
							buncolgen.TrailerColumns.CreatedAt.Eq()+
								" AND "+buncolgen.TrailerColumns.ID.Lt(),
							req.Cursor.Cursor.CreatedAt,
							req.Cursor.Cursor.ID,
						)
				})
			}
			return q.
				Order(buncolgen.TrailerColumns.CreatedAt.OrderDesc()).
				Order(buncolgen.TrailerColumns.ID.OrderDesc()).
				Limit(limit + 1)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to scan cursor trailers", zap.Error(err))
		return nil, err
	}

	hasNextPage := len(entities) > limit
	if hasNextPage {
		entities = entities[:limit]
	}

	return &pagination.CursorListResult[*trailer.Trailer]{
		Items:       entities,
		HasNextPage: hasNextPage,
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *trailer.Trailer,
) (*trailer.Trailer, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("code", entity.Code),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create trailer", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *trailer.Trailer,
) (*trailer.Trailer, error) {
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
		Where(buncolgen.TrailerColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update trailer", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "Trailer", entity.ID.String()); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, repositories.GetTrailerByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetTrailerByIDRequest,
) (*trailer.Trailer, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(trailer.Trailer)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTrailerColumns(sq, req.TrailerRelationIncludes)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyLastKnownLocationJoin(sq, req.TrailerRelationIncludes)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTrailerRelations(sq, req.TrailerRelationIncludes)
		}).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TrailerScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.TrailerColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get trailer", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Trailer")
	}

	return entity, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*trailer.Trailer], error) {
	cols := buncolgen.TrailerColumns

	return dbhelper.SelectOptions[*trailer.Trailer](
		ctx,
		r.db.DB(),
		req,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				cols.ID,
				cols.CreatedAt,
				cols.Status,
				cols.Code,
			},
			OrgColumnRef: &cols.OrganizationID,
			BuColumnRef:  &cols.BusinessUnitID,
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where(cols.Status.Eq(), domaintypes.EquipmentStatusAvailable)
			},
			EntityName:       "Trailer",
			SearchColumnRefs: []buncolgen.Column{cols.Code},
		},
	)
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetTrailersByIDsRequest,
) ([]*trailer.Trailer, error) {
	log := r.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Any("request", req),
	)

	entities := make([]*trailer.Trailer, 0, len(req.TrailerIDs))
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTrailerColumns(sq, req.TrailerRelationIncludes)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyLastKnownLocationJoin(sq, req.TrailerRelationIncludes)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyTrailerRelations(sq, req.TrailerRelationIncludes)
		}).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TrailerScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.TrailerColumns.ID.In(), bun.List(req.TrailerIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get trailers", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Trailer")
	}

	return entities, nil
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateTrailerStatusRequest,
) ([]*trailer.Trailer, error) {
	log := r.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	entities := make([]*trailer.Trailer, 0, len(req.TrailerIDs))
	results, err := r.db.DB().
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.TrailerScopeTenantUpdate(uq, req.TenantInfo).
				Where(buncolgen.TrailerColumns.ID.In(), bun.List(req.TrailerIDs))
		}).
		Set(buncolgen.TrailerColumns.Status.Set(), req.Status).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update trailer status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(results, "Trailer", req.TrailerIDs); err != nil {
		return nil, err
	}

	return entities, nil
}

func applyTrailerRelations(
	q *bun.SelectQuery,
	includes repositories.TrailerRelationIncludes,
) *bun.SelectQuery {
	rel := buncolgen.TrailerRelations
	equipmentTypeRel := buncolgen.EquipmentTypeRelations
	equipmentManufacturerRel := buncolgen.EquipmentManufacturerRelations
	fleetCodeRel := buncolgen.FleetCodeRelations
	organizationRel := buncolgen.OrganizationRelations
	includeBusinessUnit := includes.IncludeTenantDetails || includes.IncludeBusinessUnit
	includeOrganization := includes.IncludeTenantDetails || includes.IncludeOrganization
	includeRegistrationState := includes.IncludeRegistrationDetails ||
		includes.IncludeRegistrationState
	includeEquipmentType := includes.IncludeEquipmentDetails || includes.IncludeEquipmentType
	includeEquipmentManufacturer := includes.IncludeEquipmentDetails ||
		includes.IncludeEquipmentManufacturer
	includeFleetCode := includes.IncludeFleetDetails || includes.IncludeFleetCode

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
	if includeRegistrationState {
		q = q.Relation(rel.RegistrationState)
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
	if includes.IncludeFleetManager {
		q = q.Relation(buncolgen.Rel(rel.FleetCode, fleetCodeRel.Manager))
	}
	if includeFleetCode && includes.IncludeTenantDetails {
		q = q.Relation(buncolgen.Rel(rel.FleetCode, fleetCodeRel.BusinessUnit)).
			Relation(buncolgen.Rel(rel.FleetCode, fleetCodeRel.Organization))
	}

	return q
}

func applyTrailerColumns(
	q *bun.SelectQuery,
	includes repositories.TrailerRelationIncludes,
) *bun.SelectQuery {
	if len(includes.TrailerColumns) == 0 {
		return q
	}

	return q.Column(includes.TrailerColumns...)
}

func applyLastKnownLocationJoin(
	q *bun.SelectQuery,
	includes repositories.TrailerRelationIncludes,
) *bun.SelectQuery {
	if !includes.IncludeLastKnownLocation {
		return q
	}

	if len(includes.TrailerColumns) == 0 {
		q = q.Column("tr.*")
	}

	return q.ColumnExpr("ec.current_location_id AS last_known_location_id").
		ColumnExpr("COALESCE(lkl.name, '') AS last_known_location_name").
		Join("LEFT JOIN equipment_continuity AS ec ON ec.equipment_id = tr.id AND ec.equipment_type = ? AND ec.organization_id = tr.organization_id AND ec.business_unit_id = tr.business_unit_id AND ec.is_current = TRUE", "Trailer").
		Join("LEFT JOIN locations AS lkl ON lkl.id = ec.current_location_id AND lkl.organization_id = ec.organization_id AND lkl.business_unit_id = ec.business_unit_id")
}
