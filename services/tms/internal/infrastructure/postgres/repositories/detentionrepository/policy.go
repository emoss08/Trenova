package detentionrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/detention"
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

type PolicyParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type policyRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPolicyRepository(p PolicyParams) repositories.DetentionPolicyRepository {
	return &policyRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.detention-policy-repository"),
	}
}

func orderTiers(sq *bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.DetentionPolicyTierColumns
	return sq.Order(cols.FromMinute.OrderAsc()).Order(cols.SortOrder.OrderAsc())
}

func (r *policyRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListDetentionPoliciesRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.DetentionPolicyTable.Alias,
		req.Filter,
		(*detention.DetentionPolicy)(nil),
	)

	cols := buncolgen.DetentionPolicyColumns

	if req.Status != "" {
		q = q.Where(cols.Status.Eq(), req.Status)
	}

	if req.CustomerID != nil && !req.CustomerID.IsNil() {
		q = q.Where(cols.CustomerID.Eq(), *req.CustomerID)
	}

	if req.LocationID != nil && !req.LocationID.IsNil() {
		q = q.Where(cols.LocationID.Eq(), *req.LocationID)
	}

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *policyRepository) List(
	ctx context.Context,
	req *repositories.ListDetentionPoliciesRequest,
) (*pagination.ListResult[*detention.DetentionPolicy], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*detention.DetentionPolicy, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.DetentionPolicyRelations.Tiers, orderTiers).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count detention policies", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*detention.DetentionPolicy]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *policyRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListDetentionPolicyConnectionRequest,
) (*pagination.CursorListResult[*detention.DetentionPolicy], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*detention.DetentionPolicy)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				buncolgen.DetentionPolicyTable.Alias,
				req.Filter,
				(*detention.DetentionPolicy)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count detention policies", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*detention.DetentionPolicy]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*detention.DetentionPolicy) *bun.SelectQuery {
				q := dba.NewSelect().Model(entities)
				if len(req.DetentionPolicyColumns) == 0 {
					q = q.ColumnExpr(buncolgen.DetentionPolicyTable.All())
				} else {
					q = q.Column(req.DetentionPolicyColumns...)
				}
				return q
			},
			Apply: func(q *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					q,
					buncolgen.DetentionPolicyTable.Alias,
					req.Filter,
					req.Cursor,
					(*detention.DetentionPolicy)(nil),
				)
			},
		},
	)
	if err != nil {
		log.Error("failed to cursor list detention policies", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *policyRepository) GetByID(
	ctx context.Context,
	req *repositories.GetDetentionPolicyByIDRequest,
) (*detention.DetentionPolicy, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.DetentionPolicyID.String()),
	)

	cols := buncolgen.DetentionPolicyColumns
	entity := new(detention.DetentionPolicy)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DetentionPolicyScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.DetentionPolicyID)
		})

	if req.IncludeTiers {
		q = q.Relation(buncolgen.DetentionPolicyRelations.Tiers, orderTiers)
	}

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to get detention policy", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// GetResolutionCandidates returns every active policy that could apply to a
// stop: the ones scoped to this customer or facility, plus every unscoped
// policy. Narrowing further in SQL would hide the near-misses the resolution
// explainer needs to show.
func (r *policyRepository) GetResolutionCandidates(
	ctx context.Context,
	req *repositories.GetResolutionCandidatesRequest,
) ([]*detention.DetentionPolicy, error) {
	log := r.l.With(zap.String("operation", "GetResolutionCandidates"))

	cols := buncolgen.DetentionPolicyColumns
	entities := make([]*detention.DetentionPolicy, 0, 8)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.DetentionPolicyRelations.Tiers, orderTiers).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DetentionPolicyScopeTenant(sq, req.TenantInfo).
				Where(cols.Status.Eq(), detention.PolicyStatusActive)
		}).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				WhereOr(cols.CustomerID.IsNull()).
				WhereOr(cols.CustomerID.Eq(), req.CustomerID)
		}).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				WhereOr(cols.LocationID.IsNull()).
				WhereOr(cols.LocationID.Eq(), req.LocationID)
		}).
		Order(cols.Priority.OrderDesc()).
		Order(cols.SpecificityScore.OrderDesc()).
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		log.Error("failed to load detention policy candidates", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func stampTiers(entity *detention.DetentionPolicy, resetIDs bool) {
	for i, tier := range entity.Tiers {
		if tier == nil {
			continue
		}

		if resetIDs {
			tier.ID = pulid.Nil
		}

		tier.DetentionPolicyID = entity.ID
		tier.OrganizationID = entity.OrganizationID
		tier.BusinessUnitID = entity.BusinessUnitID
		tier.SortOrder = int32(i)
	}
}

func (r *policyRepository) insertTiers(
	ctx context.Context,
	entity *detention.DetentionPolicy,
) error {
	if len(entity.Tiers) == 0 {
		return nil
	}

	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&entity.Tiers).
		Returning("*").
		Exec(ctx)

	return err
}

func (r *policyRepository) Create(
	ctx context.Context,
	entity *detention.DetentionPolicy,
) (*detention.DetentionPolicy, error) {
	log := r.l.With(zap.String("operation", "Create"))

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); iErr != nil {
			return iErr
		}

		stampTiers(entity, false)

		return r.insertTiers(c, entity)
	})
	if err != nil {
		log.Error("failed to create detention policy", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Detention policy is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *policyRepository) Update(
	ctx context.Context,
	entity *detention.DetentionPolicy,
) (*detention.DetentionPolicy, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	tierCols := buncolgen.DetentionPolicyTierColumns
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		results, uErr := r.db.DBForContext(c).
			NewUpdate().
			Model(entity).
			WherePK().
			Where("version = ?", ov).
			OmitZero().
			Returning("*").
			Exec(c)
		if uErr != nil {
			return uErr
		}

		if uErr = dberror.CheckRowsAffected(
			results, "DetentionPolicy", entity.ID.String(),
		); uErr != nil {
			return uErr
		}

		if _, dErr := r.db.DBForContext(c).
			NewDelete().
			Model((*detention.DetentionPolicyTier)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.DetentionPolicyTierScopeTenantDelete(dq, pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				}).Where(tierCols.DetentionPolicyID.Eq(), entity.ID)
			}).
			Exec(c); dErr != nil {
			return dErr
		}

		stampTiers(entity, true)

		return r.insertTiers(c, entity)
	})
	if err != nil {
		log.Error("failed to update detention policy", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Detention policy is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *policyRepository) Delete(
	ctx context.Context,
	req *repositories.GetDetentionPolicyByIDRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.DetentionPolicyID.String()),
	)

	cols := buncolgen.DetentionPolicyColumns
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*detention.DetentionPolicy)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.DetentionPolicyScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.ID.Eq(), req.DetentionPolicyID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete detention policy", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(
		results, "DetentionPolicy", req.DetentionPolicyID.String(),
	)
}

func (r *policyRepository) CodeExists(
	ctx context.Context,
	req *repositories.DetentionPolicyCodeExistsRequest,
) (bool, error) {
	cols := buncolgen.DetentionPolicyColumns

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*detention.DetentionPolicy)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DetentionPolicyScopeTenant(sq, req.TenantInfo).
				Where(cols.Code.Expr("LOWER({}) = LOWER(?)"), req.Code)
		})

	if req.ExcludeID != nil && !req.ExcludeID.IsNil() {
		q = q.Where(cols.ID.Ne(), *req.ExcludeID)
	}

	exists, err := q.Exists(ctx)
	if err != nil {
		r.l.Error("failed to check detention policy code", zap.Error(err))
		return false, err
	}

	return exists, nil
}

// FindOverlapping locates active policies that pin the same scope and whose
// effective windows intersect. Two of those make resolution a coin flip.
func (r *policyRepository) FindOverlapping(
	ctx context.Context,
	req *repositories.FindOverlappingDetentionPoliciesRequest,
) ([]*detention.DetentionPolicy, error) {
	cols := buncolgen.DetentionPolicyColumns
	entities := make([]*detention.DetentionPolicy, 0, 4)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DetentionPolicyScopeTenant(sq, req.TenantInfo).
				Where(cols.Status.Eq(), detention.PolicyStatusActive).
				Where(cols.IsOrgDefault.Eq(), req.IsOrgDefault)
		})

	if req.ExcludeID != nil && !req.ExcludeID.IsNil() {
		q = q.Where(cols.ID.Ne(), *req.ExcludeID)
	}

	if req.CustomerID != nil && !req.CustomerID.IsNil() {
		q = q.Where(cols.CustomerID.Eq(), *req.CustomerID)
	} else {
		q = q.Where(cols.CustomerID.IsNull())
	}

	if req.LocationID != nil && !req.LocationID.IsNil() {
		q = q.Where(cols.LocationID.Eq(), *req.LocationID)
	} else {
		q = q.Where(cols.LocationID.IsNull())
	}

	if req.EffectiveStartDate != nil {
		q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				WhereOr(cols.EffectiveEndDate.IsNull()).
				WhereOr(cols.EffectiveEndDate.Gte(), *req.EffectiveStartDate)
		})
	}

	if req.EffectiveEndDate != nil {
		q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				WhereOr(cols.EffectiveStartDate.IsNull()).
				WhereOr(cols.EffectiveStartDate.Lte(), *req.EffectiveEndDate)
		})
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to find overlapping detention policies", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *policyRepository) SelectOptions(
	ctx context.Context,
	req *repositories.DetentionPolicySelectOptionsRequest,
) (*pagination.ListResult[*detention.DetentionPolicy], error) {
	cols := buncolgen.DetentionPolicyColumns
	return dbhelper.SelectOptions[*detention.DetentionPolicy](
		ctx,
		r.db.DBForContext(ctx),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				cols.ID,
				cols.Name,
				cols.Code,
				cols.Description,
				cols.Status,
				cols.BillingFreeMinutes,
			},
			OrgColumnRef: &cols.OrganizationID,
			BuColumnRef:  &cols.BusinessUnitID,
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where(cols.Status.Eq(), detention.PolicyStatusActive).
					Order(cols.Name.OrderAsc())
			},
			EntityName: "DetentionPolicy",
			SearchColumnRefs: []buncolgen.Column{
				cols.Name,
				cols.Code,
				cols.Description,
			},
		},
	)
}
