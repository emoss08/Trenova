package orgstructurerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/approvaldelegation"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/approvaldelegation/approvaldelegationdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/orgstructureports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultPositionPageSize   = 200
	defaultDelegationPageSize = 200
)

var (
	jobPositionErrors        = dberror.Mapping{Entity: "JobPosition"}
	approvalDelegationErrors = dberror.Mapping{Entity: "ApprovalDelegation"}
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

func New(p Params) orgstructureports.OrgStructureRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.org-structure-repository"),
	}
}

func limitOr(requested, fallback int) int {
	if requested > 0 && requested <= fallback {
		return requested
	}
	return fallback
}

func (r *repository) ListPositions(
	ctx context.Context,
	req *orgstructureports.ListJobPositionsRequest,
) ([]*worker.JobPosition, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.JobPosition, error) {
		jp := workerdb.JobPositions
		q := jp.Where(
			keel.When(req.ActiveOnly, jp.Status.Eq(domaintypes.StatusActive)),
			keel.When(req.DrivingOnly, jp.IsDrivingPosition.IsTrue()),
			keel.When(req.Department != "", jp.Department.Eq(req.Department)),
		).
			Order(jp.Department.Asc(), jp.Title.Asc()).
			Limit(limitOr(req.Limit, defaultPositionPageSize))
		if req.IncludeReportsTo {
			q = q.Load(jp.ReportsTo())
		}

		entities, err := q.List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list job positions", zap.Error(err))
			return nil, fmt.Errorf("list job positions: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) GetPositionsByIDs(
	ctx context.Context,
	req *orgstructureports.GetJobPositionsByIDsRequest,
) ([]*worker.JobPosition, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.JobPosition, error) {
		if len(req.PositionIDs) == 0 {
			return []*worker.JobPosition{}, nil
		}

		jp := workerdb.JobPositions
		entities, err := jp.Where(jp.ID.In(req.PositionIDs...)).
			Limit(len(req.PositionIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get job positions", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) GetPositionByID(
	ctx context.Context,
	req *orgstructureports.GetJobPositionByIDRequest,
) (*worker.JobPosition, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.JobPosition, error) {
		jp := workerdb.JobPositions
		q := jp.Where(jp.ID.Eq(req.ID))
		if req.IncludeReportsTo {
			q = q.Load(jp.ReportsTo())
		}

		entity, err := q.One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, jobPositionErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreatePosition(
	ctx context.Context,
	entity *worker.JobPosition,
) (*worker.JobPosition, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.JobPosition, error) {
		if err := workerdb.JobPositions.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create job position", zap.Error(err))
			return nil, jobPositionErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdatePosition(
	ctx context.Context,
	entity *worker.JobPosition,
) (*worker.JobPosition, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.JobPosition, error) {
		if err := workerdb.JobPositions.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update job position", zap.Error(err))
			return nil, jobPositionErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CountPositionHolders(
	ctx context.Context,
	_ pagination.TenantInfo,
	positionID pulid.ID,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		w := workerdb.Workers
		m := tenantdb.OrganizationMemberships
		var workers, staff int64
		if err := keel.Batch(
			ctx,
			r.db.Keel(ctx),
			keel.CountInto(w.Where(w.PositionID.Eq(positionID)), &workers),
			keel.CountInto(m.Where(m.PositionID.Eq(positionID)), &staff),
		); err != nil {
			r.l.Error("failed to count position holders", zap.Error(err))
			return 0, fmt.Errorf("count position holders: %w", err)
		}

		return int(workers + staff), nil
	})
}

func (r *repository) ListDelegations(
	ctx context.Context,
	req *orgstructureports.ListApprovalDelegationsRequest,
) ([]*approvaldelegation.ApprovalDelegation, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*approvaldelegation.ApprovalDelegation, error) {
			ad := approvaldelegationdb.ApprovalDelegations
			var active keel.Pred
			if req.ActiveAt > 0 {
				active = keel.And(
					ad.StartsAt.Lte(req.ActiveAt),
					keel.Or(ad.EndsAt.IsNull(), ad.EndsAt.Gte(req.ActiveAt)),
					keel.Or(ad.RevokedAt.IsNull(), ad.RevokedAt.Gt(req.ActiveAt)),
				)
			}

			q := ad.Where(
				keel.When(!req.DelegatorID.IsNil(), ad.DelegatorID.Eq(req.DelegatorID)),
				keel.When(!req.DelegateID.IsNil(), ad.DelegateID.Eq(req.DelegateID)),
				active,
			).
				Order(ad.StartsAt.Desc()).
				Limit(limitOr(req.Limit, defaultDelegationPageSize))
			if req.IncludeUsers {
				q = q.Load(ad.Delegator(), ad.Delegate())
			}

			entities, err := q.List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list approval delegations", zap.Error(err))
				return nil, fmt.Errorf("list approval delegations: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) GetDelegationByID(
	ctx context.Context,
	req *orgstructureports.GetApprovalDelegationByIDRequest,
) (*approvaldelegation.ApprovalDelegation, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*approvaldelegation.ApprovalDelegation, error) {
			entity, err := approvaldelegationdb.ApprovalDelegations.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, approvalDelegationErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) CreateDelegation(
	ctx context.Context,
	entity *approvaldelegation.ApprovalDelegation,
) (*approvaldelegation.ApprovalDelegation, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*approvaldelegation.ApprovalDelegation, error) {
			if err := approvaldelegationdb.ApprovalDelegations.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error("failed to create approval delegation", zap.Error(err))
				return nil, approvalDelegationErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateDelegation(
	ctx context.Context,
	entity *approvaldelegation.ApprovalDelegation,
) (*approvaldelegation.ApprovalDelegation, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*approvaldelegation.ApprovalDelegation, error) {
			if err := approvaldelegationdb.ApprovalDelegations.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error("failed to update approval delegation", zap.Error(err))
				return nil, approvalDelegationErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}
