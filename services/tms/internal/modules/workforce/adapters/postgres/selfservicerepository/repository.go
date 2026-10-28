package selfservicerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerteam"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/profilechange"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/profilechange/profilechangedb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerpolicy"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerpolicy/workerpolicydb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultPolicyPageSize     = 200
	defaultAckPageSize        = 500
	defaultChangePageSize     = 200
	defaultCompliancePageSize = 1000
)

var policyErrors = dberror.Mapping{Entity: "WorkerPolicy"}

var acknowledgementErrors = dberror.Mapping{Entity: "WorkerPolicyAcknowledgement"}

var changeRequestErrors = dberror.Mapping{Entity: "WorkerProfileChangeRequest"}

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
) ([]*workerpolicy.WorkerPolicy, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*workerpolicy.WorkerPolicy, error) {
		p := workerpolicydb.WorkerPolicies
		entities, err := p.Where(
			keel.When(req.ActiveOnly, p.Status.Eq(domaintypes.StatusActive)),
			keel.When(req.AsOf > 0, p.EffectiveFrom.Lte(req.AsOf)),
		).
			Order(p.Title.Asc()).
			Limit(limitOr(req.Limit, defaultPolicyPageSize)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list worker policies", zap.Error(err))
			return nil, fmt.Errorf("list worker policies: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) GetPolicyByID(
	ctx context.Context,
	req *workerports.GetWorkerPolicyByIDRequest,
) (*workerpolicy.WorkerPolicy, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*workerpolicy.WorkerPolicy, error) {
		entity, err := workerpolicydb.WorkerPolicies.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, policyErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreatePolicy(
	ctx context.Context,
	entity *workerpolicy.WorkerPolicy,
) (*workerpolicy.WorkerPolicy, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*workerpolicy.WorkerPolicy, error) {
		if err := workerpolicydb.WorkerPolicies.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create worker policy", zap.Error(err))
			return nil, policyErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdatePolicy(
	ctx context.Context,
	entity *workerpolicy.WorkerPolicy,
) (*workerpolicy.WorkerPolicy, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*workerpolicy.WorkerPolicy, error) {
		if err := workerpolicydb.WorkerPolicies.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update worker policy", zap.Error(err))
			return nil, policyErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func acknowledgementsQuery(
	req *workerports.ListPolicyAcknowledgementsRequest,
) *keel.Query[*workerpolicy.WorkerPolicyAcknowledgement] {
	a := workerpolicydb.WorkerPolicyAcknowledgements
	q := a.Where(
		keel.When(req.PolicyID.IsNotNil(), a.PolicyID.Eq(req.PolicyID)),
		keel.When(req.WorkerID.IsNotNil(), a.WorkerID.Eq(req.WorkerID)),
		keel.When(req.VersionLabel != "", a.VersionLabel.Eq(req.VersionLabel)),
	).
		Order(a.AcknowledgedAt.Desc()).
		Limit(limitOr(req.Limit, defaultAckPageSize))
	if req.IncludeWorker {
		q = q.Load(a.Worker())
	}

	return q
}

func (r *repository) ListAcknowledgements(
	ctx context.Context,
	req *workerports.ListPolicyAcknowledgementsRequest,
) ([]*workerpolicy.WorkerPolicyAcknowledgement, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*workerpolicy.WorkerPolicyAcknowledgement, error) {
			entities, err := acknowledgementsQuery(req).List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list policy acknowledgements", zap.Error(err))
				return nil, fmt.Errorf("list policy acknowledgements: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) CreateAcknowledgement(
	ctx context.Context,
	entity *workerpolicy.WorkerPolicyAcknowledgement,
) (*workerpolicy.WorkerPolicyAcknowledgement, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*workerpolicy.WorkerPolicyAcknowledgement, error) {
			err := workerpolicydb.WorkerPolicyAcknowledgements.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to record policy acknowledgement", zap.Error(err))
				return nil, acknowledgementErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) CountAcknowledgements(
	ctx context.Context,
	_ pagination.TenantInfo,
	policyID pulid.ID,
	versionLabel string,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		a := workerpolicydb.WorkerPolicyAcknowledgements
		total, err := a.Where(a.PolicyID.Eq(policyID), a.VersionLabel.Eq(versionLabel)).
			Count(ctx, r.db.Keel(ctx))
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
			rows, err := keel.Into[workerports.PolicyComplianceRow](
				ctx,
				r.db.Keel(ctx),
				policyComplianceQuery(req),
			)
			if err != nil {
				r.l.Error("failed to read policy compliance", zap.Error(err))
				return nil, fmt.Errorf("read policy compliance: %w", err)
			}

			return rows, nil
		},
	)
}

func audienceFilter(audience workerpolicy.PolicyAudience) keel.Pred {
	w := workerdb.Workers
	switch audience {
	case workerpolicy.PolicyAudienceEmployees:
		return w.Type.Eq(worker.WorkerTypeEmployee)
	case workerpolicy.PolicyAudienceContractors:
		return w.Type.Eq(worker.WorkerTypeContractor)
	case workerpolicy.PolicyAudienceAll:
		return nil
	}

	return nil
}

func policyComplianceQuery(req *workerports.PolicyComplianceRequest) *keel.Query[*worker.Worker] {
	w := workerdb.Workers
	a := workerpolicydb.WorkerPolicyAcknowledgements

	return w.LeftJoin(a,
		a.WorkerID.EqCol(w.ID),
		a.PolicyID.Eq(req.PolicyID),
		a.VersionLabel.Eq(req.VersionLabel),
	).
		Where(w.Status.Eq(domaintypes.StatusActive), audienceFilter(req.AppliesTo)).
		Pick(
			w.ID,
			w.FirstName,
			w.LastName,
			w.Type,
			a.AcknowledgedAt.Coalesce(int64(0)),
			a.SignatureName.Coalesce(""),
		).
		Order(w.LastName.Asc(), w.FirstName.Asc()).
		Limit(limitOr(req.Limit, defaultCompliancePageSize))
}

func changeRequestsQuery(
	req *workerports.ListProfileChangeRequestsRequest,
) *keel.Query[*profilechange.WorkerProfileChangeRequest] {
	c := profilechangedb.WorkerProfileChangeRequests
	q := c.Where(
		keel.When(req.WorkerID.IsNotNil(), c.WorkerID.Eq(req.WorkerID)),
		keel.When(len(req.Statuses) > 0, c.Status.In(req.Statuses...)),
		workerteam.Where(c.WorkerID.Col, req.ManagerIDs),
	).
		Order(c.SubmittedAt.Desc()).
		Limit(limitOr(req.Limit, defaultChangePageSize))
	if req.IncludeWorker {
		q = q.Load(c.Worker())
	}

	return q
}

func (r *repository) ListChangeRequests(
	ctx context.Context,
	req *workerports.ListProfileChangeRequestsRequest,
) ([]*profilechange.WorkerProfileChangeRequest, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*profilechange.WorkerProfileChangeRequest, error) {
			entities, err := changeRequestsQuery(req).List(ctx, r.db.Keel(ctx))
			if err != nil {
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
) (*profilechange.WorkerProfileChangeRequest, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*profilechange.WorkerProfileChangeRequest, error) {
			c := profilechangedb.WorkerProfileChangeRequests
			q := c.Where(c.ID.Eq(req.ID))
			if req.IncludeWorker {
				q = q.Load(c.Worker())
			}

			entity, err := q.One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, changeRequestErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) CreateChangeRequest(
	ctx context.Context,
	entity *profilechange.WorkerProfileChangeRequest,
) (*profilechange.WorkerProfileChangeRequest, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*profilechange.WorkerProfileChangeRequest, error) {
			err := profilechangedb.WorkerProfileChangeRequests.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to create profile change request", zap.Error(err))
				return nil, changeRequestErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateChangeRequest(
	ctx context.Context,
	entity *profilechange.WorkerProfileChangeRequest,
) (*profilechange.WorkerProfileChangeRequest, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*profilechange.WorkerProfileChangeRequest, error) {
			err := profilechangedb.WorkerProfileChangeRequests.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to update profile change request", zap.Error(err))
				return nil, changeRequestErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}
