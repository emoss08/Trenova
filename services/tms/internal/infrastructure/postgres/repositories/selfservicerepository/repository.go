package selfservicerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
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

func New(p Params) repositories.SelfServiceRepository {
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
	req *repositories.ListWorkerPoliciesRequest,
) ([]*worker.WorkerPolicy, error) {
	cols := buncolgen.WorkerPolicyColumns
	entities := make([]*worker.WorkerPolicy, 0, 8)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerPolicyScopeTenant(sq, req.TenantInfo)
			if req.ActiveOnly {
				sq = sq.Where(cols.Status.Eq(), domaintypes.StatusActive)
			}
			if req.AsOf > 0 {
				sq = sq.Where(cols.EffectiveFrom.Lte(), req.AsOf)
			}
			return sq
		}).
		Order(cols.Title.OrderAsc()).
		Limit(limitOr(req.Limit, defaultPolicyPageSize)).
		Scan(ctx); err != nil {
		r.l.Error("failed to list worker policies", zap.Error(err))
		return nil, fmt.Errorf("list worker policies: %w", err)
	}

	return entities, nil
}

func (r *repository) GetPolicyByID(
	ctx context.Context,
	req *repositories.GetWorkerPolicyByIDRequest,
) (*worker.WorkerPolicy, error) {
	entity := new(worker.WorkerPolicy)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerPolicyScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerPolicyColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerPolicy")
	}

	return entity, nil
}

func (r *repository) CreatePolicy(
	ctx context.Context,
	entity *worker.WorkerPolicy,
) (*worker.WorkerPolicy, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create worker policy", zap.Error(err))
		return nil, fmt.Errorf("create worker policy: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdatePolicy(
	ctx context.Context,
	entity *worker.WorkerPolicy,
) (*worker.WorkerPolicy, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerPolicyColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update worker policy", zap.Error(err))
		return nil, fmt.Errorf("update worker policy: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "WorkerPolicy", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListAcknowledgements(
	ctx context.Context,
	req *repositories.ListPolicyAcknowledgementsRequest,
) ([]*worker.WorkerPolicyAcknowledgement, error) {
	cols := buncolgen.WorkerPolicyAcknowledgementColumns
	entities := make([]*worker.WorkerPolicyAcknowledgement, 0, 8)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerPolicyAcknowledgementScopeTenant(sq, req.TenantInfo)
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
		Limit(limitOr(req.Limit, defaultAckPageSize))

	if req.IncludeWorker {
		q = q.Relation(buncolgen.WorkerPolicyAcknowledgementRelations.Worker)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list policy acknowledgements", zap.Error(err))
		return nil, fmt.Errorf("list policy acknowledgements: %w", err)
	}

	return entities, nil
}

func (r *repository) CreateAcknowledgement(
	ctx context.Context,
	entity *worker.WorkerPolicyAcknowledgement,
) (*worker.WorkerPolicyAcknowledgement, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to record policy acknowledgement", zap.Error(err))
		return nil, fmt.Errorf("record policy acknowledgement: %w", err)
	}

	return entity, nil
}

func (r *repository) CountAcknowledgements(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	policyID pulid.ID,
	versionLabel string,
) (int, error) {
	cols := buncolgen.WorkerPolicyAcknowledgementColumns
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerPolicyAcknowledgement)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerPolicyAcknowledgementScopeTenant(sq, tenantInfo).
				Where(cols.PolicyID.Eq(), policyID).
				Where(cols.VersionLabel.Eq(), versionLabel)
		}).
		Count(ctx)
	if err != nil {
		r.l.Error("failed to count policy acknowledgements", zap.Error(err))
		return 0, fmt.Errorf("count policy acknowledgements: %w", err)
	}

	return total, nil
}

// PolicyCompliance is every active worker the policy binds, left-joined to
// their signature on the version in force. One query rather than a walk of
// the roster, and the audience filter is applied in SQL so a policy for
// employees never lists a contractor as outstanding.
func (r *repository) PolicyCompliance(
	ctx context.Context,
	req *repositories.PolicyComplianceRequest,
) ([]repositories.PolicyComplianceRow, error) {
	wrk := buncolgen.WorkerColumns
	ack := buncolgen.WorkerPolicyAcknowledgementColumns
	rows := make([]repositories.PolicyComplianceRow, 0, 64)

	q := buncolgen.WorkerScopeTenant(
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
		Limit(limitOr(req.Limit, defaultCompliancePageSize))

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
}

func (r *repository) ListChangeRequests(
	ctx context.Context,
	req *repositories.ListProfileChangeRequestsRequest,
) ([]*worker.WorkerProfileChangeRequest, error) {
	cols := buncolgen.WorkerProfileChangeRequestColumns
	entities := make([]*worker.WorkerProfileChangeRequest, 0, 8)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerProfileChangeRequestScopeTenant(sq, req.TenantInfo)
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			if len(req.Statuses) > 0 {
				sq = sq.Where(cols.Status.In(), bun.List(req.Statuses))
			}
			if len(req.ManagerIDs) > 0 {
				wrk := buncolgen.WorkerColumns
				sq = sq.Where(
					cols.WorkerID.Qualified()+" IN (?)",
					r.db.DBForContext(ctx).
						NewSelect().
						Model((*worker.Worker)(nil)).
						Column(wrk.ID.Bare()).
						Where(wrk.OrganizationID.EqColumn(cols.OrganizationID)).
						Where(wrk.BusinessUnitID.EqColumn(cols.BusinessUnitID)).
						Where(wrk.ManagerID.In(), bun.List(req.ManagerIDs)),
				)
			}
			return sq
		}).
		Order(cols.SubmittedAt.OrderDesc()).
		Limit(limitOr(req.Limit, defaultChangePageSize))

	if req.IncludeWorker {
		q = q.Relation(buncolgen.WorkerProfileChangeRequestRelations.Worker)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list profile change requests", zap.Error(err))
		return nil, fmt.Errorf("list profile change requests: %w", err)
	}

	return entities, nil
}

func (r *repository) GetChangeRequestByID(
	ctx context.Context,
	req *repositories.GetProfileChangeRequestByIDRequest,
) (*worker.WorkerProfileChangeRequest, error) {
	entity := new(worker.WorkerProfileChangeRequest)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerProfileChangeRequestScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerProfileChangeRequestColumns.ID.Eq(), req.ID)
		})
	if req.IncludeWorker {
		q = q.Relation(buncolgen.WorkerProfileChangeRequestRelations.Worker)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerProfileChangeRequest")
	}

	return entity, nil
}

func (r *repository) CreateChangeRequest(
	ctx context.Context,
	entity *worker.WorkerProfileChangeRequest,
) (*worker.WorkerProfileChangeRequest, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create profile change request", zap.Error(err))
		return nil, fmt.Errorf("create profile change request: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateChangeRequest(
	ctx context.Context,
	entity *worker.WorkerProfileChangeRequest,
) (*worker.WorkerProfileChangeRequest, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerProfileChangeRequestColumns.Version.Eq(), ov).
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
}
