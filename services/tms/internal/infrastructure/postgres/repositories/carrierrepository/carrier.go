package carrierrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
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

func New(p Params) repositories.CarrierRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-repository"),
	}
}

func (r *repository) addOptions(
	q *bun.SelectQuery,
	opts repositories.CarrierFilterOptions,
) *bun.SelectQuery {
	if opts.IncludeState {
		q = q.Relation("State")
		q = q.Relation("RemitState")
	}

	if opts.IncludeContacts {
		q = q.Relation("Contacts")
	}

	if opts.IncludeInsurancePolicies {
		q = q.Relation("InsurancePolicies")
	}

	return q
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListCarrierRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.CarrierTable.Alias,
		req.Filter,
		(*carrier.Carrier)(nil),
	)

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, req.CarrierFilterOptions)
	})

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListCarrierRequest,
) (*pagination.ListResult[*carrier.Carrier], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*carrier.Carrier, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count carriers", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*carrier.Carrier]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListCarrierConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		buncolgen.CarrierTable.Alias,
		req.Filter,
		req.Cursor,
		(*carrier.Carrier)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListCarrierConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.CarrierTable.Alias,
		req.Filter,
		(*carrier.Carrier)(nil),
	)
}

func applyCarrierColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.CarrierTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListCarrierConnectionRequest,
) (*pagination.CursorListResult[*carrier.Carrier], error) {
	log := r.l.With(
		zap.String("operation", "ListConnection"),
		zap.Any("request", req),
	)

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*carrier.Carrier)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.applyTotalCountFilters(sq, req)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count carriers", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*carrier.Carrier]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*carrier.Carrier) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return applyCarrierColumns(sq, req.CarrierColumns)
					}).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return r.addOptions(sq, req.CarrierFilterOptions)
					})
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return r.applyCursorPageFilters(sq, req)
			},
		})
	if err != nil {
		log.Error("failed to scan carriers", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetCarrierByIDRequest,
) (*carrier.Carrier, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	cols := buncolgen.CarrierColumns
	entity := new(carrier.Carrier)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.CarrierFilterOptions)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get carrier", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Carrier")
	}

	return entity, nil
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetCarriersByIDsRequest,
) ([]*carrier.Carrier, error) {
	log := r.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Any("request", req),
	)

	cols := buncolgen.CarrierColumns
	entities := make([]*carrier.Carrier, 0, len(req.CarrierIDs))
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.In(), bun.List(req.CarrierIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get carriers", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Carrier")
	}

	return entities, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.CarrierSelectOptionsRequest,
) (*pagination.ListResult[*carrier.Carrier], error) {
	return dbhelper.SelectOptions[*carrier.Carrier](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				"id",
				"code",
				"name",
			},
			OrgColumn: "carr.organization_id",
			BuColumn:  "carr.business_unit_id",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where("carr.status = ?", carrier.StatusActive)
			},
			EntityName: "Carrier",
			SearchColumns: []string{
				"carr.code",
				"carr.name",
			},
		},
	)
}

func (r *repository) stampChildren(entity *carrier.Carrier) {
	for _, contact := range entity.Contacts {
		contact.ID = ""
		contact.CarrierID = entity.ID
		contact.OrganizationID = entity.OrganizationID
		contact.BusinessUnitID = entity.BusinessUnitID
	}

	for _, policy := range entity.InsurancePolicies {
		policy.ID = ""
		policy.CarrierID = entity.ID
		policy.OrganizationID = entity.OrganizationID
		policy.BusinessUnitID = entity.BusinessUnitID
	}
}

func (r *repository) insertChildren(ctx context.Context, entity *carrier.Carrier) error {
	if len(entity.Contacts) > 0 {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&entity.Contacts).
			Returning("*").
			Exec(ctx); err != nil {
			return err
		}
	}

	if len(entity.InsurancePolicies) > 0 {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&entity.InsurancePolicies).
			Returning("*").
			Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) deleteChildren(ctx context.Context, entity *carrier.Carrier) error {
	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	if _, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*carrier.CarrierContact)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.CarrierContactScopeTenantDelete(dq, tenantInfo).
				Where(buncolgen.CarrierContactColumns.CarrierID.Eq(), entity.ID)
		}).
		Exec(ctx); err != nil {
		return err
	}

	if _, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*carrier.CarrierInsurancePolicy)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.CarrierInsurancePolicyScopeTenantDelete(dq, tenantInfo).
				Where(buncolgen.CarrierInsurancePolicyColumns.CarrierID.Eq(), entity.ID)
		}).
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *carrier.Carrier,
) (*carrier.Carrier, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("code", entity.Code),
	)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); iErr != nil {
			return iErr
		}

		r.stampChildren(entity)

		return r.insertChildren(c, entity)
	})
	if err != nil {
		log.Error("failed to create carrier", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *carrier.Carrier,
) (*carrier.Carrier, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		ov := entity.Version
		entity.Version++

		results, uErr := r.db.DBForContext(c).NewUpdate().
			Model(entity).
			WherePK().
			Where("version = ?", ov).
			Returning("*").
			Exec(c)
		if uErr != nil {
			return uErr
		}

		if uErr = dberror.CheckRowsAffected(results, "Carrier", entity.ID.String()); uErr != nil {
			return uErr
		}

		if dErr := r.deleteChildren(c, entity); dErr != nil {
			return dErr
		}

		r.stampChildren(entity)

		return r.insertChildren(c, entity)
	})
	if err != nil {
		log.Error("failed to update carrier", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Carrier is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateCarrierStatusRequest,
) ([]*carrier.Carrier, error) {
	log := r.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	cols := buncolgen.CarrierColumns
	entities := make([]*carrier.Carrier, 0, len(req.CarrierIDs))
	results, err := r.db.DB().
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CarrierScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.In(), bun.List(req.CarrierIDs))
		}).
		Set("status = ?", req.Status).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update carrier status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(results, "Carrier", req.CarrierIDs); err != nil {
		return nil, err
	}

	return entities, nil
}
