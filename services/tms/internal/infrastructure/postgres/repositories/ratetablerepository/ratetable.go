package ratetablerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ratetable"
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

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.RateTableRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.rate-table-repository"),
	}
}

func orderEntries(sq *bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.RateTableEntryColumns
	return sq.Order(cols.SortOrder.OrderAsc()).
		Order(cols.RangeMin.OrderAsc()).
		Order(cols.MatchKey.OrderAsc())
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListRateTablesRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.RateTableTable.Alias,
		req.Filter,
		(*ratetable.RateTable)(nil),
	)

	cols := buncolgen.RateTableColumns
	if req.LookupType != "" {
		q = q.Where(cols.LookupType.Eq(), req.LookupType)
	}

	if req.Active != nil {
		q = q.Where(cols.Active.Eq(), *req.Active)
	}

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListRateTablesRequest,
) (*pagination.ListResult[*ratetable.RateTable], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*ratetable.RateTable, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count rate tables", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*ratetable.RateTable]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListRateTableConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		buncolgen.RateTableTable.Alias,
		req.Filter,
		req.Cursor,
		(*ratetable.RateTable)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListRateTableConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.RateTableTable.Alias,
		req.Filter,
		(*ratetable.RateTable)(nil),
	)
}

func applyRateTableColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.RateTableTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListRateTableConnectionRequest,
) (*pagination.CursorListResult[*ratetable.RateTable], error) {
	log := r.l.With(
		zap.String("operation", "ListConnection"),
		zap.Any("request", req),
	)

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*ratetable.RateTable)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.applyTotalCountFilters(sq, req)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count rate tables", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*ratetable.RateTable]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*ratetable.RateTable) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return applyRateTableColumns(sq, req.RateTableColumns)
					})
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return r.applyCursorPageFilters(sq, req)
			},
		})
	if err != nil {
		log.Error("failed to scan rate tables", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetRateTableByIDRequest,
) (*ratetable.RateTable, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.RateTableID.String()),
	)

	cols := buncolgen.RateTableColumns
	entity := new(ratetable.RateTable)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.RateTableRelations.Entries, orderEntries).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateTableScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.RateTableID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get rate table", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "RateTable")
	}

	return entity, nil
}

func (r *repository) GetByKeys(
	ctx context.Context,
	req *repositories.GetRateTablesByKeysRequest,
) ([]*ratetable.RateTable, error) {
	if len(req.Keys) == 0 {
		return []*ratetable.RateTable{}, nil
	}

	log := r.l.With(
		zap.String("operation", "GetByKeys"),
		zap.Strings("keys", req.Keys),
	)

	cols := buncolgen.RateTableColumns
	entities := make([]*ratetable.RateTable, 0, len(req.Keys))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateTableScopeTenant(sq, req.TenantInfo).
				Where(cols.Key.In(), bun.List(req.Keys))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get rate tables by keys", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetLookupData(
	ctx context.Context,
	req *repositories.GetRateTableLookupDataRequest,
) ([]*ratetable.RateTable, error) {
	log := r.l.With(
		zap.String("operation", "GetLookupData"),
	)

	cols := buncolgen.RateTableColumns
	entities := make([]*ratetable.RateTable, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.RateTableRelations.Entries, orderEntries).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateTableScopeTenant(sq, req.TenantInfo).
				Where(cols.Active.Eq(), true)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get rate table lookup data", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func stampEntries(entity *ratetable.RateTable, resetIDs bool) {
	for _, entry := range entity.Entries {
		if entry == nil {
			continue
		}

		if resetIDs {
			entry.ID = pulid.Nil
		}

		entry.RateTableID = entity.ID
		entry.OrganizationID = entity.OrganizationID
		entry.BusinessUnitID = entity.BusinessUnitID
	}
}

func (r *repository) insertEntries(ctx context.Context, entity *ratetable.RateTable) error {
	if len(entity.Entries) == 0 {
		return nil
	}

	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&entity.Entries).
		Returning("*").
		Exec(ctx)

	return err
}

func (r *repository) Create(
	ctx context.Context,
	entity *ratetable.RateTable,
) (*ratetable.RateTable, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
	)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); iErr != nil {
			return iErr
		}

		stampEntries(entity, false)

		return r.insertEntries(c, entity)
	})
	if err != nil {
		log.Error("failed to create rate table", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Rate table is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *ratetable.RateTable,
) (*ratetable.RateTable, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	cols := buncolgen.RateTableEntryColumns
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

		if uErr = dberror.CheckRowsAffected(results, "RateTable", entity.ID.String()); uErr != nil {
			return uErr
		}

		if _, dErr := r.db.DBForContext(c).
			NewDelete().
			Model((*ratetable.RateTableEntry)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.RateTableEntryScopeTenantDelete(dq, pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				}).Where(cols.RateTableID.Eq(), entity.ID)
			}).
			Exec(c); dErr != nil {
			return dErr
		}

		stampEntries(entity, true)

		return r.insertEntries(c, entity)
	})
	if err != nil {
		log.Error("failed to update rate table", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Rate table is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req *repositories.GetRateTableByIDRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.RateTableID.String()),
	)

	cols := buncolgen.RateTableColumns
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*ratetable.RateTable)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.RateTableScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.ID.Eq(), req.RateTableID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete rate table", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(results, "RateTable", req.RateTableID.String())
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.RateTableSelectOptionsRequest,
) (*pagination.ListResult[*ratetable.RateTable], error) {
	cols := buncolgen.RateTableColumns
	return dbhelper.SelectOptions[*ratetable.RateTable](
		ctx,
		r.db.DBForContext(ctx),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				cols.ID,
				cols.Name,
				cols.Key,
				cols.Description,
				cols.LookupType,
				cols.Active,
			},
			OrgColumnRef: &cols.OrganizationID,
			BuColumnRef:  &cols.BusinessUnitID,
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where(cols.Active.Eq(), true).
					Order(cols.Name.OrderAsc())
			},
			EntityName: "RateTable",
			SearchColumnRefs: []buncolgen.Column{
				cols.Name,
				cols.Key,
				cols.Description,
			},
		},
	)
}
