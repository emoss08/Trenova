package ptopolicyrepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/worker"
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

func New(p Params) repositories.PTOPolicyRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.pto-policy-repository"),
	}
}

func orderRules(sq *bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.PTOPolicyRuleColumns
	return sq.Order(cols.SortOrder.OrderAsc()).Order(cols.PTOType.OrderAsc())
}

func (r *repository) applyListFilters(
	q *bun.SelectQuery,
	req *repositories.ListPTOPoliciesRequest,
) *bun.SelectQuery {
	cols := buncolgen.PTOPolicyColumns
	if req.Status != "" {
		q = q.Where(cols.Status.Eq(), req.Status)
	}
	if req.IncludeRules {
		q = q.Relation(buncolgen.PTOPolicyRelations.Rules, orderRules)
	}
	return q
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListPTOPoliciesRequest,
) (*pagination.CursorListResult[*worker.PTOPolicy], error) {
	log := r.l.With(zap.String("operation", "List"))

	dba := r.db.DBForContext(ctx)
	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*worker.PTOPolicy)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.PTOPolicyTable.Alias,
					req.Filter,
					(*worker.PTOPolicy)(nil),
				)
				cols := buncolgen.PTOPolicyColumns
				if req.Status != "" {
					sq = sq.Where(cols.Status.Eq(), req.Status)
				}
				return sq
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count PTO policies", zap.Error(err))
			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*worker.PTOPolicy]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: totalCount,
		Query: func(items *[]*worker.PTOPolicy) *bun.SelectQuery {
			return dba.NewSelect().Model(items).ColumnExpr(buncolgen.PTOPolicyTable.All())
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			sq, applyErr := querybuilder.ApplyCursorFilters(
				sq,
				buncolgen.PTOPolicyTable.Alias,
				req.Filter,
				req.Cursor,
				(*worker.PTOPolicy)(nil),
			)
			if applyErr != nil {
				return sq, applyErr
			}
			return r.applyListFilters(sq, req), nil
		},
	})
	if err != nil {
		log.Error("failed to list PTO policies", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.PTOPolicySelectOptionsRequest,
) (*pagination.ListResult[*worker.PTOPolicy], error) {
	cols := buncolgen.PTOPolicyColumns
	return dbhelper.SelectOptions[*worker.PTOPolicy](
		ctx,
		r.db.DBForContext(ctx),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				cols.ID,
				cols.Code,
				cols.Name,
				cols.Description,
				cols.IsDefault,
				cols.YearBasis,
				cols.RequiresApproval,
				cols.CreatedAt,
			},
			OrgColumnRef: &cols.OrganizationID,
			BuColumnRef:  &cols.BusinessUnitID,
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where(cols.Status.Eq(), worker.PTOPolicyStatusActive).
					Order(cols.IsDefault.OrderDesc()).
					Order(cols.Name.OrderAsc())
			},
			EntityName: "PTOPolicy",
			SearchColumnRefs: []buncolgen.Column{
				cols.Code,
				cols.Name,
				cols.Description,
			},
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetPTOPolicyByIDRequest,
) (*worker.PTOPolicy, error) {
	log := r.l.With(zap.String("operation", "GetByID"), zap.String("id", req.ID.String()))

	entity := new(worker.PTOPolicy)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PTOPolicyScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.PTOPolicyColumns.ID.Eq(), req.ID)
		})
	if req.IncludeRules {
		q = q.Relation(buncolgen.PTOPolicyRelations.Rules, orderRules)
	}

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to get PTO policy", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "PTOPolicy")
	}

	return entity, nil
}

func (r *repository) GetDefault(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*worker.PTOPolicy, error) {
	log := r.l.With(zap.String("operation", "GetDefault"))

	cols := buncolgen.PTOPolicyColumns
	entity := new(worker.PTOPolicy)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.PTOPolicyRelations.Rules, orderRules).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PTOPolicyScopeTenant(sq, tenantInfo).
				Where(cols.IsDefault.Eq(), true).
				Where(cols.Status.Eq(), worker.PTOPolicyStatusActive)
		}).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, dberror.HandleNotFoundError(err, "PTOPolicy")
		}
		log.Error("failed to get default PTO policy", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) CodeExists(
	ctx context.Context,
	req *repositories.PTOPolicyCodeExistsRequest,
) (bool, error) {
	cols := buncolgen.PTOPolicyColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.PTOPolicy)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.PTOPolicyScopeTenant(sq, req.TenantInfo).
				Where("LOWER("+cols.Code.String()+") = ?", strings.ToLower(req.Code))
		})
	if !req.ExcludeID.IsNil() {
		q = q.Where(cols.ID.Ne(), req.ExcludeID)
	}

	exists, err := q.Exists(ctx)
	if err != nil {
		r.l.Error("failed to check PTO policy code", zap.Error(err))
		return false, err
	}

	return exists, nil
}

func stampRules(entity *worker.PTOPolicy, resetIDs bool) {
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
	}
}

func (r *repository) insertRules(ctx context.Context, entity *worker.PTOPolicy) error {
	if len(entity.Rules) == 0 {
		return nil
	}

	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&entity.Rules).
		Returning("*").
		Exec(ctx)

	return err
}

func (r *repository) Create(
	ctx context.Context,
	entity *worker.PTOPolicy,
) (*worker.PTOPolicy, error) {
	log := r.l.With(zap.String("operation", "Create"))

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if entity.IsDefault {
			if cErr := r.ClearDefault(c, &repositories.ClearDefaultPTOPolicyRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				},
			}); cErr != nil {
				return cErr
			}
		}

		if _, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); iErr != nil {
			return iErr
		}

		stampRules(entity, false)
		return r.insertRules(c, entity)
	})
	if err != nil {
		log.Error("failed to create PTO policy", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"PTO policy is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *worker.PTOPolicy,
) (*worker.PTOPolicy, error) {
	log := r.l.With(zap.String("operation", "Update"), zap.String("id", entity.ID.String()))

	ov := entity.Version
	entity.Version++
	ruleCols := buncolgen.PTOPolicyRuleColumns
	tenant := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if entity.IsDefault {
			if cErr := r.ClearDefault(c, &repositories.ClearDefaultPTOPolicyRequest{
				TenantInfo: tenant,
				ExceptID:   entity.ID,
			}); cErr != nil {
				return cErr
			}
		}

		results, uErr := r.db.DBForContext(c).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(buncolgen.PTOPolicyColumns.Version.Eq(), ov).
			Returning("*").
			Exec(c)
		if uErr != nil {
			return uErr
		}
		if uErr = dberror.CheckRowsAffected(results, "PTOPolicy", entity.ID.String()); uErr != nil {
			return uErr
		}

		if _, dErr := r.db.DBForContext(c).
			NewDelete().
			Model((*worker.PTOPolicyRule)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.PTOPolicyRuleScopeTenantDelete(dq, tenant).
					Where(ruleCols.PTOPolicyID.Eq(), entity.ID)
			}).
			Exec(c); dErr != nil {
			return dErr
		}

		stampRules(entity, true)
		return r.insertRules(c, entity)
	})
	if err != nil {
		log.Error("failed to update PTO policy", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"PTO policy is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) ClearDefault(
	ctx context.Context,
	req *repositories.ClearDefaultPTOPolicyRequest,
) error {
	cols := buncolgen.PTOPolicyColumns
	q := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*worker.PTOPolicy)(nil)).
		Set(cols.IsDefault.Set(), false).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.PTOPolicyScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.IsDefault.Eq(), true)
		})
	if !req.ExceptID.IsNil() {
		q = q.Where(cols.ID.Ne(), req.ExceptID)
	}

	if _, err := q.Exec(ctx); err != nil {
		r.l.Error("failed to clear default PTO policy", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) CountOpenAssignments(
	ctx context.Context,
	req *repositories.CountOpenPTOAssignmentsRequest,
) (int, error) {
	cols := buncolgen.WorkerPTOPolicyAssignmentColumns
	count, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerPTOPolicyAssignment)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerPTOPolicyAssignmentScopeTenant(sq, req.TenantInfo).
				Where(cols.PTOPolicyID.Eq(), req.PolicyID).
				Where(cols.EffectiveTo.IsNull())
		}).
		Count(ctx)
	if err != nil {
		r.l.Error("failed to count open PTO assignments", zap.Error(err))
		return 0, err
	}

	return count, nil
}

func (r *repository) CountOpenAssignmentsByIDs(
	ctx context.Context,
	req *repositories.CountOpenPTOAssignmentsByIDsRequest,
) (map[pulid.ID]int, error) {
	if len(req.PolicyIDs) == 0 {
		return map[pulid.ID]int{}, nil
	}

	cols := buncolgen.WorkerPTOPolicyAssignmentColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerPTOPolicyAssignment)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerPTOPolicyAssignmentScopeTenant(sq, req.TenantInfo).
				Where(cols.PTOPolicyID.In(), bun.In(req.PolicyIDs)).
				Where(cols.EffectiveTo.IsNull())
		})

	counts, err := dbhelper.CountByID(ctx, q, cols.PTOPolicyID, len(req.PolicyIDs))
	if err != nil {
		r.l.Error("failed to count open PTO assignments by policy", zap.Error(err))
		return nil, err
	}

	return counts, nil
}
