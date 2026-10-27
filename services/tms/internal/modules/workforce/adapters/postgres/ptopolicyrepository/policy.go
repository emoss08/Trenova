package ptopolicyrepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const policyBusyMessage = "PTO policy is busy. Retry the request."

var policyErrors = dberror.Mapping{Entity: "PTOPolicy"}

type policyCount struct {
	PolicyID pulid.ID
	Count    int64
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) workerports.PTOPolicyRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.pto-policy-repository"),
	}
}

func orderedRules() keel.Loader {
	rl := workerdb.PTOPolicies.Rules()
	return rl.Order(rl.SortOrder.Asc(), rl.PTOType.Asc())
}

func statusFilter(status string) keel.Pred {
	if status == "" {
		return nil
	}

	return workerdb.PTOPolicies.Status.Eq(worker.PTOPolicyStatus(status))
}

func (r *repository) List(
	ctx context.Context,
	req *workerports.ListPTOPoliciesRequest,
) (*pagination.CursorListResult[*worker.PTOPolicy], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*worker.PTOPolicy], error) {
			p := workerdb.PTOPolicies
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				p.Where(statusFilter(req.Status)),
				&listquery.Request[worker.PTOPolicy]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(q *keel.Query[*worker.PTOPolicy]) *keel.Query[*worker.PTOPolicy] {
						return q.Load(listquery.When(req.IncludeRules, orderedRules())...)
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list PTO policies", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *workerports.PTOPolicySelectOptionsRequest,
) (*pagination.ListResult[*worker.PTOPolicy], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*worker.PTOPolicy], error) {
			p := workerdb.PTOPolicies
			q := p.Pick(
				p.ID,
				p.Code,
				p.Name,
				p.Description,
				p.IsDefault,
				p.YearBasis,
				p.RequiresApproval,
				p.CreatedAt,
			).
				Where(p.Status.Eq(worker.PTOPolicyStatusActive)).
				Load(orderedRules()).
				Order(p.IsDefault.Desc(), p.Name.Asc())

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(
						p.Code.ContainsFold(s),
						p.Name.ContainsFold(s),
						p.Description.ContainsFold(s),
					)
				},
			)
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *workerports.GetPTOPolicyByIDRequest,
) (*worker.PTOPolicy, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.PTOPolicy, error) {
		p := workerdb.PTOPolicies
		entity, err := p.Where(p.ID.Eq(req.ID)).
			Load(listquery.When(req.IncludeRules, orderedRules())...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, policyErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetDefault(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*worker.PTOPolicy, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.PTOPolicy, error) {
		p := workerdb.PTOPolicies
		entity, err := p.Where(p.IsDefault.IsTrue(), p.Status.Eq(worker.PTOPolicyStatusActive)).
			Load(orderedRules()).
			Order(p.CreatedAt.Asc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			if !dberror.IsNotFoundError(err) {
				r.l.Error("failed to get default PTO policy", zap.Error(err))
			}
			return nil, policyErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *repository) CodeExists(
	ctx context.Context,
	req *workerports.PTOPolicyCodeExistsRequest,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		p := workerdb.PTOPolicies
		exists, err := p.Where(
			p.Code.Lower().Eq(strings.ToLower(req.Code)),
			keel.When(req.ExcludeID.IsNotNil(), p.ID.Ne(req.ExcludeID)),
		).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to check PTO policy code", zap.Error(err))
			return false, err
		}

		return exists, nil
	})
}

func stampRules(entity *worker.PTOPolicy, resetIDs bool) []*worker.PTOPolicyRule {
	rules := make([]*worker.PTOPolicyRule, 0, len(entity.Rules))
	for i, rule := range entity.Rules {
		if rule == nil {
			continue
		}

		if resetIDs {
			rule.ID = pulid.Nil
		}
		rule.PTOPolicyID = entity.ID
		rule.OrganizationID = entity.OrganizationID
		rule.BusinessUnitID = entity.BusinessUnitID
		rule.SortOrder = int32(i) //nolint:gosec // rule counts are tiny
		rules = append(rules, rule)
	}

	return rules
}

func (r *repository) insertRules(ctx context.Context, rules []*worker.PTOPolicyRule) error {
	if len(rules) == 0 {
		return nil
	}

	return workerdb.PTOPolicyRules.CreateMany(ctx, r.db.Keel(ctx), rules)
}

func (r *repository) Create(
	ctx context.Context,
	entity *worker.PTOPolicy,
) (*worker.PTOPolicy, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.PTOPolicy, error) {
		if err := r.create(ctx, entity); err != nil {
			r.l.Error("failed to create PTO policy", zap.Error(err))
			return nil, dberror.MapRetryableTransactionError(
				policyErrors.Map(err, entity.ID.String()),
				policyBusyMessage,
			)
		}

		return entity, nil
	})
}

func (r *repository) create(ctx context.Context, entity *worker.PTOPolicy) error {
	if entity.IsDefault {
		if err := r.ClearDefault(ctx, &workerports.ClearDefaultPTOPolicyRequest{}); err != nil {
			return err
		}
	}

	if err := workerdb.PTOPolicies.Create(ctx, r.db.Keel(ctx), entity); err != nil {
		return err
	}

	return r.insertRules(ctx, stampRules(entity, false))
}

func (r *repository) Update(
	ctx context.Context,
	entity *worker.PTOPolicy,
) (*worker.PTOPolicy, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.PTOPolicy, error) {
		if err := r.update(ctx, entity); err != nil {
			r.l.Error(
				"failed to update PTO policy",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, dberror.MapRetryableTransactionError(
				policyErrors.Map(err, entity.ID.String()),
				policyBusyMessage,
			)
		}

		return entity, nil
	})
}

func (r *repository) update(ctx context.Context, entity *worker.PTOPolicy) error {
	if entity.IsDefault {
		err := r.ClearDefault(ctx, &workerports.ClearDefaultPTOPolicyRequest{ExceptID: entity.ID})
		if err != nil {
			return err
		}
	}

	db := r.db.Keel(ctx)
	if err := workerdb.PTOPolicies.Save(ctx, db, entity); err != nil {
		return err
	}

	rl := workerdb.PTOPolicyRules
	if _, err := rl.Where(rl.PTOPolicyID.Eq(entity.ID)).Delete(ctx, db); err != nil {
		return err
	}

	return r.insertRules(ctx, stampRules(entity, true))
}

func (r *repository) ClearDefault(
	ctx context.Context,
	req *workerports.ClearDefaultPTOPolicyRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := workerdb.PTOPolicies
		_, err := p.Where(
			p.IsDefault.IsTrue(),
			keel.When(req.ExceptID.IsNotNil(), p.ID.Ne(req.ExceptID)),
		).Update(ctx, r.db.Keel(ctx), p.IsDefault.To(false))
		if err != nil {
			r.l.Error("failed to clear default PTO policy", zap.Error(err))
			return err
		}

		return nil
	})
}

func (r *repository) CountOpenAssignments(
	ctx context.Context,
	req *workerports.CountOpenPTOAssignmentsRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		a := workerdb.WorkerPTOPolicyAssignments
		count, err := a.Where(a.PTOPolicyID.Eq(req.PolicyID), a.EffectiveTo.IsNull()).
			Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count open PTO assignments", zap.Error(err))
			return 0, err
		}

		return int(count), nil
	})
}

func (r *repository) CountOpenAssignmentsByIDs(
	ctx context.Context,
	req *workerports.CountOpenPTOAssignmentsByIDsRequest,
) (map[pulid.ID]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]int, error) {
		if len(req.PolicyIDs) == 0 {
			return map[pulid.ID]int{}, nil
		}

		a := workerdb.WorkerPTOPolicyAssignments
		rows, err := keel.Into[policyCount](ctx, r.db.Keel(ctx), a.
			Group(a.PTOPolicyID).
			Pick(a.PTOPolicyID, keel.Count()).
			Where(a.PTOPolicyID.In(req.PolicyIDs...), a.EffectiveTo.IsNull()))
		if err != nil {
			r.l.Error("failed to count open PTO assignments by policy", zap.Error(err))
			return nil, err
		}

		counts := make(map[pulid.ID]int, len(rows))
		for _, row := range rows {
			counts[row.PolicyID] = int(row.Count)
		}

		return counts, nil
	})
}
