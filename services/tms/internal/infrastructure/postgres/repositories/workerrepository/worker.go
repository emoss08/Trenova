package workerrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports"
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

func New(p Params) repositories.WorkerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListWorkersRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.WorkerTable.Alias,
		req.Filter,
		(*worker.Worker)(nil),
	)

	q = q.Order(buncolgen.WorkerColumns.CreatedAt.OrderDesc())

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.WorkerSelectOptionsRequest,
) (*pagination.ListResult[*worker.Worker], error) {
	return dbhelper.SelectOptions[*worker.Worker](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				buncolgen.WorkerColumns.ID,
				buncolgen.WorkerColumns.FirstName,
				buncolgen.WorkerColumns.LastName,
				buncolgen.WorkerColumns.Status,
			},
			OrgColumnRef: &buncolgen.WorkerColumns.OrganizationID,
			BuColumnRef:  &buncolgen.WorkerColumns.BusinessUnitID,
			SearchColumnRefs: []buncolgen.Column{
				buncolgen.WorkerColumns.FirstName,
				buncolgen.WorkerColumns.LastName,
			},
			EntityName: "Worker",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where(buncolgen.WorkerColumns.Status.Eq(), domaintypes.StatusActive)
			},
		},
	)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListWorkersRequest,
) (*pagination.ListResult[*worker.Worker], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*worker.Worker, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.WorkerRelations.State).
		Relation(buncolgen.WorkerRelations.Profile).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count workers", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*worker.Worker]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(worker.Worker)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerColumns.ID.Eq(), req.ID)
		})

	if req.IncludeProfile {
		q.Relation(buncolgen.WorkerRelations.Profile)
	}
	if req.IncludeState {
		q.Relation(buncolgen.WorkerRelations.State)
	}

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to get worker", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Worker")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *worker.Worker,
) (*worker.Worker, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("firstName", entity.FirstName),
		zap.String("lastName", entity.LastName),
	)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		if _, insertErr := r.db.
			DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); insertErr != nil {
			log.Error("failed to create worker", zap.Error(insertErr))
			return insertErr
		}

		if entity.Profile != nil {
			entity.Profile.WorkerID = entity.ID
			entity.Profile.OrganizationID = entity.OrganizationID
			entity.Profile.BusinessUnitID = entity.BusinessUnitID

			if _, profileErr := r.db.DBForContext(c).NewInsert().
				Model(entity.Profile).
				Returning("*").
				Exec(c); profileErr != nil {
				log.Error("failed to create worker profile", zap.Error(profileErr))
				return profileErr
			}
		}

		return nil
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(err, "Worker is busy. Retry the request.")
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *worker.Worker,
) (*worker.Worker, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		results, updateErr := r.db.DBForContext(c).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(buncolgen.WorkerColumns.Version.Eq(), ov).
			Returning("*").
			Exec(c)
		if updateErr != nil {
			log.Error("failed to update worker", zap.Error(updateErr))
			return updateErr
		}

		if checkErr := dberror.CheckRowsAffected(
			results,
			"Worker",
			entity.ID.String(),
		); checkErr != nil {
			return checkErr
		}

		if entity.Profile != nil {
			entity.Profile.WorkerID = entity.ID
			entity.Profile.OrganizationID = entity.OrganizationID
			entity.Profile.BusinessUnitID = entity.BusinessUnitID

			pv := entity.Profile.Version
			entity.Profile.Version++

			profileResults, profileErr := r.db.DBForContext(c).
				NewUpdate().
				Model(entity.Profile).
				WherePK().
				Where(buncolgen.WorkerProfileColumns.Version.Eq(), pv).
				Returning("*").
				Exec(c)
			if profileErr != nil {
				log.Error("failed to update worker profile", zap.Error(profileErr))
				return profileErr
			}

			if checkErr := dberror.CheckRowsAffected(
				profileResults,
				"WorkerProfile",
				entity.Profile.ID.String(),
			); checkErr != nil {
				return checkErr
			}
		}

		return nil
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(err, "Worker is busy. Retry the request.")
	}

	return entity, nil
}

func (r *repository) GetWorkerSyncReadinessCounts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*repositories.WorkerSyncReadinessCounts, error) {
	counts := new(repositories.WorkerSyncReadinessCounts)

	wrk := buncolgen.WorkerColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.Worker)(nil)).
		ColumnExpr(buncolgen.Count("total_workers")).
		ColumnExpr(buncolgen.CountFilter("active_workers", wrk.Status.Eq()),
			domaintypes.StatusActive,
		).
		ColumnExpr(
			buncolgen.CountFilter("synced_active_workers",
				wrk.Status.Eq(),
				wrk.ExternalID.Expr("NULLIF(BTRIM({}), '') IS NOT NULL"),
			),
			domaintypes.StatusActive,
		).
		Apply(buncolgen.WorkerApplyTenant(tenantInfo)).
		Scan(ctx, counts)
	if err != nil {
		return nil, err
	}

	return counts, nil
}

func (r *repository) ReplaceWorkerSyncDrifts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	drifts []repositories.WorkerSyncDriftRecord,
) error {
	return dberror.MapRetryableTransactionError(
		r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
			if _, err := r.db.DBForContext(c).NewDelete().
				Model((*worker.WorkerSyncDrift)(nil)).
				WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
					return buncolgen.WorkerSyncDriftScopeTenantDelete(dq, tenantInfo).
						Where(buncolgen.WorkerSyncDriftColumns.WorkerID.In(), bun.List(drifts))
				}).
				Exec(c); err != nil {
				return err
			}

			if len(drifts) == 0 {
				return nil
			}

			models := make([]*worker.WorkerSyncDrift, 0, len(drifts))
			for idx := range drifts {
				drift := drifts[idx]
				models = append(models, &worker.WorkerSyncDrift{
					OrganizationID:  tenantInfo.OrgID.String(),
					BusinessUnitID:  tenantInfo.BuID.String(),
					WorkerID:        drift.WorkerID,
					DriftType:       drift.DriftType,
					WorkerName:      drift.WorkerName,
					Message:         drift.Message,
					LocalExternalID: drift.LocalExternalID,
					RemoteDriverID:  drift.RemoteDriverID,
					DetectedAt:      drift.DetectedAt,
				})
			}

			_, err := r.db.DBForContext(c).NewInsert().Model(&models).Exec(c)
			return err
		}),
		"Worker sync state is busy. Retry the request.",
	)
}

func (r *repository) ListWorkerSyncDrifts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]repositories.WorkerSyncDriftRecord, error) {
	models := make([]*worker.WorkerSyncDrift, 0)

	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&models).Apply(buncolgen.WorkerSyncDriftApplyTenant(tenantInfo)).
		Order(buncolgen.WorkerSyncDriftColumns.DetectedAt.OrderDesc()).
		Order(buncolgen.WorkerSyncDriftColumns.WorkerName.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, err
	}

	records := make([]repositories.WorkerSyncDriftRecord, 0, len(models))
	for idx := range models {
		model := models[idx]
		records = append(records, repositories.WorkerSyncDriftRecord{
			WorkerID:        model.WorkerID,
			WorkerName:      model.WorkerName,
			DriftType:       model.DriftType,
			Message:         model.Message,
			LocalExternalID: model.LocalExternalID,
			RemoteDriverID:  model.RemoteDriverID,
			DetectedAt:      model.DetectedAt,
		})
	}

	return records, nil
}
