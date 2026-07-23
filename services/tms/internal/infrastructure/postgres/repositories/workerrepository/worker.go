package workerrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
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

func New(p Params) repositories.WorkerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-repository"),
	}
}

func (r *repository) cursorFilterQuery(
	q *bun.SelectQuery,
	req *repositories.ListWorkersRequest,
) (*bun.SelectQuery, error) {
	q, err := querybuilder.ApplyCursorFilters(
		q,
		buncolgen.WorkerTable.Alias,
		req.Filter,
		req.Cursor,
		(*worker.Worker)(nil),
	)
	if err != nil {
		return q, err
	}

	return q, nil
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
				buncolgen.WorkerColumns.CreatedAt,
				buncolgen.WorkerColumns.FirstName,
				buncolgen.WorkerColumns.LastName,
				buncolgen.WorkerColumns.FleetCodeID,
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
				q = q.
					Where(buncolgen.WorkerColumns.Status.Eq(), domaintypes.StatusActive).
					Relation(buncolgen.WorkerRelations.FleetCode)
				if req.OwnerOperatorsOnly {
					now := timeutils.NowUnix()
					q = q.Where(
						`(wrk.type = ? OR EXISTS (
							SELECT 1
							FROM worker_pay_assignments wpa
							JOIN driver_pay_profiles dpp
								ON dpp.id = wpa.pay_profile_id
								AND dpp.organization_id = wpa.organization_id
								AND dpp.business_unit_id = wpa.business_unit_id
							WHERE wpa.worker_id = wrk.id
								AND wpa.organization_id = wrk.organization_id
								AND wpa.business_unit_id = wrk.business_unit_id
								AND wpa.effective_from <= ?
								AND (wpa.effective_to IS NULL OR wpa.effective_to > ?)
								AND dpp.classification = ?
						))`,
						worker.WorkerTypeContractor, now, now,
						driverpay.PayeeClassificationOwnerOperator,
					)
				}
				return q
			},
		},
	)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListWorkersRequest,
) (*pagination.CursorListResult[*worker.Worker], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.Worker)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				buncolgen.WorkerTable.Alias,
				req.Filter,
				(*worker.Worker)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count workers", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*worker.Worker]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: &total,
		Query: func(items *[]*worker.Worker) *bun.SelectQuery {
			return r.db.DBForContext(ctx).
				NewSelect().
				Model(items).
				ColumnExpr(buncolgen.WorkerTable.All()).
				Relation(buncolgen.WorkerRelations.FleetCode).
				Relation(buncolgen.WorkerRelations.State).
				Relation(buncolgen.WorkerRelations.Profile)
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			return r.cursorFilterQuery(sq, req)
		},
	})
	if err != nil {
		log.Error("failed to scan workers", zap.Error(err))
		return nil, err
	}

	return result, nil
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

func (r *repository) ListWorkersWithExpiringCredentials(
	ctx context.Context,
	req repositories.ListExpiringCredentialsRequest,
) ([]*worker.Worker, error) {
	now := timeutils.NowUnix()
	horizon := now + int64(req.HorizonDays)*86400
	grace := now - int64(req.GraceDays)*86400

	wcols := buncolgen.WorkerColumns
	profileAlias := "profile"
	expiryColumns := []buncolgen.Column{
		buncolgen.WorkerProfileColumns.LicenseExpiry.WithAlias(profileAlias),
		buncolgen.WorkerProfileColumns.HazmatExpiry.WithAlias(profileAlias),
		buncolgen.WorkerProfileColumns.MedicalCardExpiry.WithAlias(profileAlias),
		buncolgen.WorkerProfileColumns.PhysicalDueDate.WithAlias(profileAlias),
		buncolgen.WorkerProfileColumns.MVRDueDate.WithAlias(profileAlias),
		buncolgen.WorkerProfileColumns.TWICExpiry.WithAlias(profileAlias),
	}

	items := make([]*worker.Worker, 0, 32)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Relation(buncolgen.WorkerRelations.Profile).
		Where(wcols.Status.Eq(), domaintypes.StatusActive).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			for i := range expiryColumns {
				predicate := expiryColumns[i].Between()
				if i == 0 {
					sq = sq.Where(predicate, grace, horizon)
				} else {
					sq = sq.WhereOr(predicate, grace, horizon)
				}
			}
			return sq
		})
	if err := query.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list workers with expiring credentials: %w", err)
	}
	return items, nil
}
