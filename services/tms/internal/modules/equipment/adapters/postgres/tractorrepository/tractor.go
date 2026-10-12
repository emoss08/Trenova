package tractorrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/equipmentmanufacturercol"
	"github.com/emoss08/trenova/pkg/buncolgen/equipmenttypecol"
	"github.com/emoss08/trenova/pkg/buncolgen/fleetcodecol"
	"github.com/emoss08/trenova/pkg/buncolgen/tenantcol"
	"github.com/emoss08/trenova/pkg/buncolgen/tractorcol"
	"github.com/emoss08/trenova/pkg/buncolgen/workercol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/quotatx"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
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
	Quota  platformplanports.QuotaGuard `optional:"true"`
}

type repository struct {
	db    *postgres.Connection
	l     *zap.Logger
	quota platformplanports.QuotaGuard
}

func New(p Params) tractorports.TractorRepository {
	return &repository{
		db:    p.DB,
		l:     p.Logger.Named("postgres.tractor-repository"),
		quota: quotaservice.OrUnlimited(p.Quota),
	}
}

func (r *repository) applyListFilters(
	q *bun.SelectQuery,
	req *tractorports.ListTractorsRequest,
) (*bun.SelectQuery, error) {
	q = applyTractorRelations(q, req.TractorRelationIncludes)

	q, err := querybuilder.ApplyCursorFilters(
		q,
		tractorcol.TractorTable.Alias,
		req.Filter,
		req.Cursor,
		(*tractor.Tractor)(nil),
	)
	if err != nil {
		return q, err
	}

	q = r.applyStatusFilter(q, req.Status)

	return q, nil
}

func (r *repository) applyListCountFilters(
	q *bun.SelectQuery,
	req *tractorports.ListTractorsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFiltersWithoutSort(
		q,
		tractorcol.TractorTable.Alias,
		req.Filter,
		(*tractor.Tractor)(nil),
	)

	return r.applyStatusFilter(q, req.Status)
}

func (r *repository) applyStatusFilter(q *bun.SelectQuery, value string) *bun.SelectQuery {
	if value == "" {
		return q
	}

	status, err := domaintypes.EquipmentStatusFromString(value)
	if err != nil {
		r.l.Error("failed to parse equipment status", zap.Error(err))
		return q
	}

	return q.Where(tractorcol.TractorColumns.Status.Eq(), status)
}

func (r *repository) List(
	ctx context.Context,
	req *tractorports.ListTractorsRequest,
) (*pagination.CursorListResult[*tractor.Tractor], error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*tractor.Tractor], error) {
			log := r.l.With(
				zap.String("operation", "List"),
				zap.Any("request", req),
			)

			dba := r.db.DBForContext(ctx)
			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.
					NewSelect().
					Model((*tractor.Tractor)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return r.applyListCountFilters(sq, req)
					}).
					Count(ctx)
				if err != nil {
					log.Error("failed to count tractors", zap.Error(err))
					return nil, err
				}
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*tractor.Tractor]{
				Filter:     req.Filter,
				Cursor:     req.Cursor,
				TotalCount: totalCount,
				Query: func(items *[]*tractor.Tractor) *bun.SelectQuery {
					return dba.
						NewSelect().
						Model(items).
						Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
							return applyTractorColumns(sq, req.TractorRelationIncludes)
						}).
						Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
							return applyTractorLastKnownLocationJoin(
								sq,
								req.TractorRelationIncludes,
							)
						})
				},
				Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
					return r.applyListFilters(sq, req)
				},
			})
			if err != nil {
				log.Error("failed to scan tractors", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *tractor.Tractor,
) (*tractor.Tractor, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tractor.Tractor, error) {
		log := r.l.With(
			zap.String("operation", "Create"),
			zap.String("code", entity.Code),
		)

		err := quotatx.Run(ctx, r.db, r.quota, func(c context.Context) error {
			if _, insertErr := r.db.DBForContext(c).
				NewInsert().
				Model(entity).
				Returning("*").
				Exec(c); insertErr != nil {
				log.Error("failed to create tractor", zap.Error(insertErr))
				return insertErr
			}

			return nil
		}, platformplanports.QuotaRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			Meter:    platformcatalog.MeterTractorsTotal,
			Quantity: 1,
		})
		if err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *tractor.Tractor,
) (*tractor.Tractor, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tractor.Tractor, error) {
		log := r.l.With(
			zap.String("operation", "Update"),
			zap.String("id", entity.ID.String()),
		)

		ov := entity.Version
		entity.Version++

		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(tractorcol.TractorColumns.Version.Eq(), ov).
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
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req tractorports.GetTractorByIDRequest,
) (*tractor.Tractor, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tractor.Tractor, error) {
		log := r.l.With(
			zap.String("operation", "GetByID"),
			zap.String("id", req.ID.String()),
		)

		entity := new(tractor.Tractor)
		err := r.db.DBForContext(ctx).
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
				return tractorcol.TractorScopeTenant(sq, req.TenantInfo).
					Where(tractorcol.TractorColumns.ID.Eq(), req.ID)
			}).
			Scan(ctx)
		if err != nil {
			log.Error("failed to get tractor", zap.Error(err))
			return nil, dberror.HandleNotFoundError(err, "Tractor")
		}

		return entity, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req tractorports.GetTractorsByIDsRequest,
) ([]*tractor.Tractor, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*tractor.Tractor, error) {
		log := r.l.With(
			zap.String("operation", "GetByIDs"),
			zap.Any("request", req),
		)

		entities := make([]*tractor.Tractor, 0, len(req.TractorIDs))
		err := r.db.DBForContext(ctx).
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
				return tractorcol.TractorScopeTenant(sq, req.TenantInfo).
					Where(tractorcol.TractorColumns.ID.In(), bun.List(req.TractorIDs))
			}).
			Scan(ctx)
		if err != nil {
			log.Error("failed to get tractors", zap.Error(err))
			return nil, dberror.HandleNotFoundError(err, "Tractor")
		}

		return entities, nil
	})
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *tractorports.BulkUpdateTractorStatusRequest,
) ([]*tractor.Tractor, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*tractor.Tractor, error) {
		log := r.l.With(
			zap.String("operation", "BulkUpdateStatus"),
			zap.Any("request", req),
		)

		entities := make([]*tractor.Tractor, 0, len(req.TractorIDs))
		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(&entities).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return tractorcol.TractorScopeTenantUpdate(uq, req.TenantInfo).
					Where(tractorcol.TractorColumns.ID.In(), bun.List(req.TractorIDs))
			}).
			Set(tractorcol.TractorColumns.Status.Set(), req.Status).
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
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *tractorports.TractorSelectOptionsRequest,
) (*pagination.ListResult[*tractor.Tractor], error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*tractor.Tractor], error) {
			cols := tractorcol.TractorColumns
			rel := tractorcol.TractorRelations

			return dbhelper.SelectOptions[*tractor.Tractor](
				ctx,
				r.db.DBForContext(ctx),
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
		},
	)
}

func applyTractorRelations(
	q *bun.SelectQuery,
	includes tractorports.TractorRelationIncludes,
) *bun.SelectQuery {
	rel := tractorcol.TractorRelations
	equipmentTypeRel := equipmenttypecol.EquipmentTypeRelations
	equipmentManufacturerRel := equipmentmanufacturercol.EquipmentManufacturerRelations
	fleetCodeRel := fleetcodecol.FleetCodeRelations
	organizationRel := tenantcol.OrganizationRelations
	workerRel := workercol.WorkerRelations
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
	if includes.IncludeOwnerWorker {
		q = q.Relation(rel.OwnerWorker, dbhelper.RelationColumns(includes.OwnerWorkerColumns))
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
	includes tractorports.TractorRelationIncludes,
) *bun.SelectQuery {
	if len(includes.TractorColumns) == 0 {
		return q.ColumnExpr(tractorcol.TractorTable.All())
	}

	return q.Column(includes.TractorColumns...)
}

func applyTractorLastKnownLocationJoin(
	q *bun.SelectQuery,
	includes tractorports.TractorRelationIncludes,
) *bun.SelectQuery {
	if !includes.IncludeLastKnownLocation {
		return q
	}

	if len(includes.TractorColumns) == 0 {
		q = q.Column("trac.*")
	}

	return q.ColumnExpr("ec.current_location_id AS last_known_location_id").
		ColumnExpr("COALESCE(lkl.name, '') AS last_known_location_name").
		ColumnExpr("lkl.latitude AS last_known_location_latitude").
		ColumnExpr("lkl.longitude AS last_known_location_longitude").
		Join("LEFT JOIN equipment_continuity AS ec ON ec.equipment_id = trac.id AND ec.equipment_type = ? AND ec.organization_id = trac.organization_id AND ec.business_unit_id = trac.business_unit_id AND ec.is_current = TRUE", "Tractor").
		Join("LEFT JOIN locations AS lkl ON lkl.id = ec.current_location_id AND lkl.organization_id = ec.organization_id AND lkl.business_unit_id = ec.business_unit_id")
}
