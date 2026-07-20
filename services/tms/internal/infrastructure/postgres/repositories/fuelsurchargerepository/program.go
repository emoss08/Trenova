package fuelsurchargerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ProgramParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type programRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewProgramRepository(p ProgramParams) repositories.FuelSurchargeProgramRepository {
	return &programRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.fuel-surcharge-program-repository"),
	}
}

func orderTableRows(sq *bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.FuelSurchargeTableRowColumns
	return sq.Order(cols.SortOrder.OrderAsc()).
		Order(cols.PriceMin.OrderAsc())
}

func applyProgramColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.FuelSurchargeProgramTable.All())
	}

	return q.Column(columns...)
}

func (r *programRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListFuelSurchargeProgramConnectionRequest,
) (*pagination.CursorListResult[*fuelsurcharge.FuelSurchargeProgram], error) {
	log := r.l.With(
		zap.String("operation", "ListConnection"),
		zap.Any("request", req),
	)

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*fuelsurcharge.FuelSurchargeProgram)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				buncolgen.FuelSurchargeProgramTable.Alias,
				req.Filter,
				(*fuelsurcharge.FuelSurchargeProgram)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count fuel surcharge programs", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*fuelsurcharge.FuelSurchargeProgram]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*fuelsurcharge.FuelSurchargeProgram) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return applyProgramColumns(sq, req.FuelSurchargeProgramColumns)
					})
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.FuelSurchargeProgramTable.Alias,
					req.Filter,
					req.Cursor,
					(*fuelsurcharge.FuelSurchargeProgram)(nil),
				)
			},
		})
	if err != nil {
		log.Error("failed to scan fuel surcharge programs", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *programRepository) ListActive(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*fuelsurcharge.FuelSurchargeProgram, error) {
	cols := buncolgen.FuelSurchargeProgramColumns
	entities := make([]*fuelsurcharge.FuelSurchargeProgram, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.FuelSurchargeProgramRelations.FuelIndex).
		Relation(buncolgen.FuelSurchargeProgramRelations.TableRows, orderTableRows).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FuelSurchargeProgramScopeTenant(sq, tenantInfo).
				Where(cols.Status.Eq(), fuelsurcharge.ProgramStatusActive)
		}).
		Order(cols.Name.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list active fuel surcharge programs", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *programRepository) GetByID(
	ctx context.Context,
	req *repositories.GetFuelSurchargeProgramByIDRequest,
) (*fuelsurcharge.FuelSurchargeProgram, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ProgramID.String()),
	)

	cols := buncolgen.FuelSurchargeProgramColumns
	entity := new(fuelsurcharge.FuelSurchargeProgram)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FuelSurchargeProgramScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ProgramID)
		})

	if req.IncludeRows {
		q = q.Relation(buncolgen.FuelSurchargeProgramRelations.TableRows, orderTableRows)
	}
	if req.IncludeIndex {
		q = q.Relation(buncolgen.FuelSurchargeProgramRelations.FuelIndex)
	}
	if req.IncludeCharge {
		q = q.Relation(buncolgen.FuelSurchargeProgramRelations.AccessorialCharge)
	}

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to get fuel surcharge program", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "FuelSurchargeProgram")
	}

	return entity, nil
}

func stampTableRows(entity *fuelsurcharge.FuelSurchargeProgram, resetIDs bool) {
	for _, row := range entity.TableRows {
		if row == nil {
			continue
		}

		if resetIDs {
			row.ID = pulid.Nil
		}

		row.FuelSurchargeProgramID = entity.ID
		row.OrganizationID = entity.OrganizationID
		row.BusinessUnitID = entity.BusinessUnitID
	}
}

func (r *programRepository) insertTableRows(
	ctx context.Context,
	entity *fuelsurcharge.FuelSurchargeProgram,
) error {
	if len(entity.TableRows) == 0 {
		return nil
	}

	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&entity.TableRows).
		Returning("*").
		Exec(ctx)

	return err
}

func (r *programRepository) Create(
	ctx context.Context,
	entity *fuelsurcharge.FuelSurchargeProgram,
) (*fuelsurcharge.FuelSurchargeProgram, error) {
	log := r.l.With(zap.String("operation", "Create"))

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); iErr != nil {
			return iErr
		}

		stampTableRows(entity, false)

		return r.insertTableRows(c, entity)
	})
	if err != nil {
		log.Error("failed to create fuel surcharge program", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Fuel surcharge program is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *programRepository) Update(
	ctx context.Context,
	entity *fuelsurcharge.FuelSurchargeProgram,
) (*fuelsurcharge.FuelSurchargeProgram, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	cols := buncolgen.FuelSurchargeTableRowColumns
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		results, uErr := r.db.DBForContext(c).
			NewUpdate().
			Model(entity).
			WherePK().
			Where("version = ?", ov).
			OmitZero().
			Returning("*").
			Exec(c)
		if uErr != nil {
			return uErr
		}

		if uErr = dberror.CheckRowsAffected(results, "FuelSurchargeProgram", entity.ID.String()); uErr != nil {
			return uErr
		}

		if _, dErr := r.db.DBForContext(c).
			NewDelete().
			Model((*fuelsurcharge.FuelSurchargeTableRow)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.FuelSurchargeTableRowScopeTenantDelete(dq, pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				}).Where(cols.FuelSurchargeProgramID.Eq(), entity.ID)
			}).
			Exec(c); dErr != nil {
			return dErr
		}

		stampTableRows(entity, true)

		return r.insertTableRows(c, entity)
	})
	if err != nil {
		log.Error("failed to update fuel surcharge program", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Fuel surcharge program is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *programRepository) Delete(
	ctx context.Context,
	req *repositories.GetFuelSurchargeProgramByIDRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.ProgramID.String()),
	)

	cols := buncolgen.FuelSurchargeProgramColumns
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*fuelsurcharge.FuelSurchargeProgram)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.FuelSurchargeProgramScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ProgramID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete fuel surcharge program", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(results, "FuelSurchargeProgram", req.ProgramID.String())
}

func (r *programRepository) ListFallbackShipmentIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	limit int,
) ([]pulid.ID, error) {
	if limit <= 0 {
		limit = 200
	}

	ids := make([]pulid.ID, 0, limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*shipment.AdditionalCharge)(nil)).
		ColumnExpr("DISTINCT ac.shipment_id").
		Join("JOIN shipments AS sp ON sp.id = ac.shipment_id AND sp.organization_id = ac.organization_id AND sp.business_unit_id = ac.business_unit_id").
		Where("ac.organization_id = ?", tenantInfo.OrgID).
		Where("ac.business_unit_id = ?", tenantInfo.BuID).
		Where("ac.is_system_generated = TRUE").
		Where("ac.fuel_surcharge_program_id IS NOT NULL").
		Where("ac.fuel_surcharge_detail->>'usedFallback' = 'true'").
		Where("sp.status NOT IN (?)", bun.In([]shipment.Status{
			shipment.StatusInvoiced,
			shipment.StatusCanceled,
		})).
		Limit(limit).
		Scan(ctx, &ids)
	if err != nil {
		r.l.Error("failed to list fuel surcharge fallback shipments", zap.Error(err))
		return nil, err
	}

	return ids, nil
}

func (r *programRepository) SelectOptions(
	ctx context.Context,
	req *repositories.FuelSurchargeProgramSelectOptionsRequest,
) (*pagination.ListResult[*fuelsurcharge.FuelSurchargeProgram], error) {
	cols := buncolgen.FuelSurchargeProgramColumns
	return dbhelper.SelectOptions[*fuelsurcharge.FuelSurchargeProgram](
		ctx,
		r.db.DBForContext(ctx),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				cols.ID,
				cols.Name,
				cols.Code,
				cols.Description,
				cols.Method,
				cols.Status,
			},
			OrgColumnRef: &cols.OrganizationID,
			BuColumnRef:  &cols.BusinessUnitID,
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where(cols.Status.Eq(), fuelsurcharge.ProgramStatusActive).
					Order(cols.Name.OrderAsc())
			},
			EntityName: "FuelSurchargeProgram",
			SearchColumnRefs: []buncolgen.Column{
				cols.Name,
				cols.Code,
				cols.Description,
			},
		},
	)
}
