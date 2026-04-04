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

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListTractorsRequest,
) *bun.SelectQuery {
	log := r.l.With(
		zap.String("operation", "filterQuery"),
		zap.Any("request", req),
	)

	rel := buncolgen.TractorRelations
	if req.IncludeEquipmentDetails {
		q = q.Relation(rel.EquipmentType).
			Relation(rel.EquipmentManufacturer)
	}

	if req.IncludeFleetDetails {
		q = q.Relation(rel.FleetCode)
	}

	if req.IncludeWorkerDetails {
		q = q.Relation(rel.PrimaryWorker).
			Relation(rel.SecondaryWorker)
	}

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
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count tractors", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*tractor.Tractor]{
		Items: entities,
		Total: total,
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
			Columns: []string{
				cols.ID.String(),
				cols.Status.String(),
				cols.Code.String(),
				cols.PrimaryWorkerID.String(),
				cols.SecondaryWorkerID.String(),
			},
			OrgColumn: cols.OrganizationID.Qualified(),
			BuColumn:  cols.BusinessUnitID.Qualified(),
			SearchColumns: []string{
				cols.Code.Qualified(),
			},
			EntityName: "Tractor",
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
