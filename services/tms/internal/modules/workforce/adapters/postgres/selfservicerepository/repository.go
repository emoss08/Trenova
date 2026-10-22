package selfservicerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/buncolgen/workercol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerteam"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultPolicyPageSize     = 200
	defaultAckPageSize        = 500
	defaultChangePageSize     = 200
	defaultCompliancePageSize = 1000
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

func New(p Params) workerports.SelfServiceRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.self-service-repository"),
	}
}

func limitOr(requested, fallback int) int {
	if requested > 0 && requested <= fallback {
		return requested
	}
	return fallback
}

func (r *repository) ListPolicies(
	ctx context.Context,
	req *workerports.ListWorkerPoliciesRequest,
) ([]*worker.WorkerPolicy, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.WorkerPolicy, error) {
		cols := workercol.WorkerPolicyColumns
		entities := make([]*worker.WorkerPolicy, 0, 8)

		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = workercol.WorkerPolicyScopeTenant(sq, req.TenantInfo)
				if req.ActiveOnly {
					sq = sq.Where(cols.Status.Eq(), domaintypes.StatusActive)
				}
				if req.AsOf > 0 {
					sq = sq.Where(cols.EffectiveFrom.Lte(), req.AsOf)
				}
				return sq
			}).
			Order(cols.Title.OrderAsc()).
			Limit(int64(limitOr(req.Limit, defaultPolicyPageSize))).
			Scan(ctx); err != nil {
			r.l.Error("failed to list worker policies", zap.Error(err))
			return nil, fmt.Errorf("list worker policies: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) GetPolicyByID(
	ctx context.Context,
	req *workerports.GetWorkerPolicyByIDRequest,
) (*worker.WorkerPolicy, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.WorkerPolicy, error) {
		entity := new(worker.WorkerPolicy)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return workercol.WorkerPolicyScopeTenant(sq, req.TenantInfo).
					Where(workercol.WorkerPolicyColumns.ID.Eq(), req.ID)
			}).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, "WorkerPolicy")
		}

		return entity, nil
	})
}

func (r *repository) CreatePolicy(
	ctx context.Context,
	entity *worker.WorkerPolicy,
) (*worker.WorkerPolicy, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerPolicy, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			r.l.Error("failed to create worker policy", zap.Error(err))
			return nil, fmt.Errorf("create worker policy: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) UpdatePolicy(
	ctx context.Context,
	entity *worker.WorkerPolicy,
) (*worker.WorkerPolicy, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerPolicy, error) {
		ov := entity.Version
		entity.Version++

		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(workercol.WorkerPolicyColumns.Version.Eq(), ov).
			Returning("*").
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to update worker policy", zap.Error(err))
			return nil, fmt.Errorf("update worker policy: %w", err)
		}
		if err = dberror.CheckRowsAffected(
			results,
			"WorkerPolicy",
			entity.ID.String(),
		); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) ListAcknowledgements(
	ctx context.Context,
	req *workerports.ListPolicyAcknowledgementsRequest,
) ([]*worker.WorkerPolicyAcknowledgement, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*worker.WorkerPolicyAcknowledgement, error) {
			cols := workercol.WorkerPolicyAcknowledgementColumns
			entities := make([]*worker.WorkerPolicyAcknowledgement, 0, 8)

			q := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					sq = workercol.WorkerPolicyAcknowledgementScopeTenant(sq, req.TenantInfo)
					if !req.PolicyID.IsNil() {
						sq = sq.Where(cols.PolicyID.Eq(), req.PolicyID)
					}
					if !req.WorkerID.IsNil() {
						sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
					}
					if req.VersionLabel != "" {
						sq = sq.Where(cols.VersionLabel.Eq(), req.VersionLabel)
					}
					return sq
				}).
				Order(cols.AcknowledgedAt.OrderDesc()).
				Limit(int64(limitOr(req.Limit, defaultAckPageSize)))

			if req.IncludeWorker {
				q = q.Relation(workercol.WorkerPolicyAcknowledgementRelations.Worker)
			}

			if err := q.Scan(ctx); err != nil {
				r.l.Error("failed to list policy acknowledgements", zap.Error(err))
				return nil, fmt.Errorf("list policy acknowledgements: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) CreateAcknowledgement(
	ctx context.Context,
	entity *worker.WorkerPolicyAcknowledgement,
) (*worker.WorkerPolicyAcknowledgement, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*worker.WorkerPolicyAcknowledgement, error) {
			if _, err := r.db.DBForContext(ctx).
				NewInsert().
				Model(entity).
				Returning("*").
				Exec(ctx); err != nil {
				r.l.Error("failed to record policy acknowledgement", zap.Error(err))
				return nil, fmt.Errorf("record policy acknowledgement: %w", err)
			}

			return entity, nil
		},
	)
}

func (r *repository) CountAcknowledgements(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	policyID pulid.ID,
	versionLabel string,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := workercol.WorkerPolicyAcknowledgementColumns
		total, err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*worker.WorkerPolicyAcknowledgement)(nil)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return workercol.WorkerPolicyAcknowledgementScopeTenant(sq, tenantInfo).
					Where(cols.PolicyID.Eq(), policyID).
					Where(cols.VersionLabel.Eq(), versionLabel)
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count policy acknowledgements", zap.Error(err))
			return 0, fmt.Errorf("count policy acknowledgements: %w", err)
		}

		return int(total), nil
	})
}

// PolicyCompliance is every active worker the policy binds, left-joined to
// their signature on the version in force. One query rather than a walk of
// the roster, and the audience filter is applied in SQL so a policy for
// employees never lists a contractor as outstanding.
func (r *repository) PolicyCompliance(
	ctx context.Context,
	req *workerports.PolicyComplianceRequest,
) ([]workerports.PolicyComplianceRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]workerports.PolicyComplianceRow, error) {
			wrk := workercol.WorkerColumns
			ack := workercol.WorkerPolicyAcknowledgementColumns
			rows := make([]workerports.PolicyComplianceRow, 0, 64)

			q := workercol.WorkerScopeTenant(
				r.db.DBForContext(ctx).NewSelect().Model((*worker.Worker)(nil)),
				req.TenantInfo,
			).
				Where(wrk.Status.Eq(), domaintypes.StatusActive).
				Join("LEFT JOIN worker_policy_acknowledgements AS wpak").
				JoinOn(ack.WorkerID.EqColumn(wrk.ID)).
				JoinOn(ack.OrganizationID.EqColumn(wrk.OrganizationID)).
				JoinOn(ack.BusinessUnitID.EqColumn(wrk.BusinessUnitID)).
				JoinOn(ack.PolicyID.Eq(), req.PolicyID).
				JoinOn(ack.VersionLabel.Eq(), req.VersionLabel).
				ColumnExpr(wrk.ID.As("worker_id")).
				ColumnExpr(wrk.FirstName.As("first_name")).
				ColumnExpr(wrk.LastName.As("last_name")).
				ColumnExpr(wrk.Type.As("worker_type")).
				ColumnExpr("COALESCE(" + ack.AcknowledgedAt.Qualified() + ", 0) AS acknowledged_at").
				ColumnExpr("COALESCE(" + ack.SignatureName.Qualified() + ", '') AS signature_name").
				OrderExpr(wrk.LastName.Qualified() + ", " + wrk.FirstName.Qualified()).
				Limit(int64(limitOr(req.Limit, defaultCompliancePageSize)))

			switch req.AppliesTo {
			case worker.PolicyAudienceEmployees:
				q = q.Where(wrk.Type.Eq(), worker.WorkerTypeEmployee)
			case worker.PolicyAudienceContractors:
				q = q.Where(wrk.Type.Eq(), worker.WorkerTypeContractor)
			case worker.PolicyAudienceAll:
			}

			if err := q.Scan(ctx, &rows); err != nil {
				r.l.Error("failed to read policy compliance", zap.Error(err))
				return nil, fmt.Errorf("read policy compliance: %w", err)
			}

			return rows, nil
		},
	)
}

func (r *repository) ListChangeRequests(
	ctx context.Context,
	req *workerports.ListProfileChangeRequestsRequest,
) ([]*worker.WorkerProfileChangeRequest, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*worker.WorkerProfileChangeRequest, error) {
			cols := workercol.WorkerProfileChangeRequestColumns
			entities := make([]*worker.WorkerProfileChangeRequest, 0, 8)

			q := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					sq = workercol.WorkerProfileChangeRequestScopeTenant(sq, req.TenantInfo)
					if !req.WorkerID.IsNil() {
						sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
					}
					if len(req.Statuses) > 0 {
						sq = sq.Where(cols.Status.In(), bun.List(req.Statuses))
					}
					return workerteam.Where(r.db.DBForContext(ctx), workerteam.Scope{
						WorkerID:       cols.WorkerID,
						OrganizationID: cols.OrganizationID,
						BusinessUnitID: cols.BusinessUnitID,
						ManagerIDs:     req.ManagerIDs,
					})(sq)
				}).
				Order(cols.SubmittedAt.OrderDesc()).
				Limit(int64(limitOr(req.Limit, defaultChangePageSize)))

			if req.IncludeWorker {
				q = q.Relation(workercol.WorkerProfileChangeRequestRelations.Worker)
			}

			if err := q.Scan(ctx); err != nil {
				r.l.Error("failed to list profile change requests", zap.Error(err))
				return nil, fmt.Errorf("list profile change requests: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) GetChangeRequestByID(
	ctx context.Context,
	req *workerports.GetProfileChangeRequestByIDRequest,
) (*worker.WorkerProfileChangeRequest, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*worker.WorkerProfileChangeRequest, error) {
			entity := new(worker.WorkerProfileChangeRequest)

			q := r.db.DBForContext(ctx).
				NewSelect().
				Model(entity).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return workercol.WorkerProfileChangeRequestScopeTenant(sq, req.TenantInfo).
						Where(workercol.WorkerProfileChangeRequestColumns.ID.Eq(), req.ID)
				})
			if req.IncludeWorker {
				q = q.Relation(workercol.WorkerProfileChangeRequestRelations.Worker)
			}

			if err := q.Scan(ctx); err != nil {
				return nil, dberror.HandleNotFoundError(err, "WorkerProfileChangeRequest")
			}

			return entity, nil
		},
	)
}

func (r *repository) CreateChangeRequest(
	ctx context.Context,
	entity *worker.WorkerProfileChangeRequest,
) (*worker.WorkerProfileChangeRequest, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*worker.WorkerProfileChangeRequest, error) {
			if _, err := r.db.DBForContext(ctx).
				NewInsert().
				Model(entity).
				Returning("*").
				Exec(ctx); err != nil {
				r.l.Error("failed to create profile change request", zap.Error(err))
				return nil, fmt.Errorf("create profile change request: %w", err)
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateChangeRequest(
	ctx context.Context,
	entity *worker.WorkerProfileChangeRequest,
) (*worker.WorkerProfileChangeRequest, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*worker.WorkerProfileChangeRequest, error) {
			ov := entity.Version
			entity.Version++

			results, err := r.db.DBForContext(ctx).
				NewUpdate().
				Model(entity).
				WherePK().
				Where(workercol.WorkerProfileChangeRequestColumns.Version.Eq(), ov).
				Returning("*").
				Exec(ctx)
			if err != nil {
				r.l.Error("failed to update profile change request", zap.Error(err))
				return nil, fmt.Errorf("update profile change request: %w", err)
			}
			if err = dberror.CheckRowsAffected(
				results,
				"WorkerProfileChangeRequest",
				entity.ID.String(),
			); err != nil {
				return nil, err
			}

			return entity, nil
		},
	)
}
