package detentionrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const policyBusyMessage = "Detention policy is busy. Retry the request."

type PolicyParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type policyRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPolicyRepository(p PolicyParams) detentionports.DetentionPolicyRepository {
	return &policyRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.detention-policy-repository"),
	}
}

func orderedTiers() keel.Loader {
	t := detentiondb.DetentionPolicies.Tiers()
	return t.Order(t.FromMinute.Asc(), t.SortOrder.Asc())
}

func (r *policyRepository) List(
	ctx context.Context,
	req *detentionports.ListDetentionPoliciesRequest,
) (*pagination.ListResult[*detention.DetentionPolicy], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*detention.DetentionPolicy], error) {
			p := detentiondb.DetentionPolicies
			q := p.Where(
				keel.When(req.Status != "", p.Status.Eq(detention.PolicyStatus(req.Status))),
				optionalIDEq(p.CustomerID, req.CustomerID),
				optionalIDEq(p.LocationID, req.LocationID),
			)

			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				q,
				&listquery.Request[detention.DetentionPolicy]{
					Filter: req.Filter,
					Shape: func(
						q *keel.Query[detention.DetentionPolicy],
					) *keel.Query[detention.DetentionPolicy] {
						return q.Load(orderedTiers())
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list detention policies", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *policyRepository) ListConnection(
	ctx context.Context,
	req *detentionports.ListDetentionPolicyConnectionRequest,
) (*pagination.CursorListResult[*detention.DetentionPolicy], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*detention.DetentionPolicy], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				detentiondb.DetentionPolicies.Query(),
				&listquery.Request[detention.DetentionPolicy]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.DetentionPolicyColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list detention policy connection", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *policyRepository) GetByID(
	ctx context.Context,
	req *detentionports.GetDetentionPolicyByIDRequest,
) (*detention.DetentionPolicy, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*detention.DetentionPolicy, error) {
		p := detentiondb.DetentionPolicies
		q := p.Where(p.ID.Eq(req.DetentionPolicyID))
		if req.IncludeTiers {
			q = q.Load(orderedTiers())
		}

		entity, err := q.One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to get detention policy",
				zap.String("id", req.DetentionPolicyID.String()),
				zap.Error(err),
			)
			return nil, policyErrors.Map(err, req.DetentionPolicyID.String())
		}

		return entity, nil
	})
}

// GetResolutionCandidates returns every active policy that could apply to a
// stop: the ones scoped to this customer or facility, plus every unscoped
// policy. Narrowing further in SQL would hide the near-misses the resolution
// explainer needs to show.
func (r *policyRepository) GetResolutionCandidates(
	ctx context.Context,
	req *detentionports.GetResolutionCandidatesRequest,
) ([]*detention.DetentionPolicy, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*detention.DetentionPolicy, error) {
		p := detentiondb.DetentionPolicies
		entities, err := p.Where(
			p.Status.Eq(detention.PolicyStatusActive),
			keel.Or(p.CustomerID.IsNull(), p.CustomerID.Eq(req.CustomerID)),
			keel.Or(p.LocationID.IsNull(), p.LocationID.Eq(req.LocationID)),
		).
			Load(orderedTiers()).
			Order(p.Priority.Desc(), p.SpecificityScore.Desc(), p.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to load detention policy candidates", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func stampTiers(entity *detention.DetentionPolicy, resetIDs bool) []*detention.DetentionPolicyTier {
	tiers := make([]*detention.DetentionPolicyTier, 0, len(entity.Tiers))
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
		tiers = append(tiers, tier)
	}

	return tiers
}

func (r *policyRepository) Create(
	ctx context.Context,
	entity *detention.DetentionPolicy,
) (*detention.DetentionPolicy, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*detention.DetentionPolicy, error) {
		db := r.db.Keel(ctx)
		err := detentiondb.DetentionPolicies.Create(ctx, db, entity)
		if err == nil {
			err = detentiondb.DetentionPolicyTiers.CreateMany(ctx, db, stampTiers(entity, false))
		}
		if err != nil {
			r.l.Error("failed to create detention policy", zap.Error(err))
			return nil, dberror.MapRetryableTransactionError(
				policyErrors.Map(err, entity.ID.String()),
				policyBusyMessage,
			)
		}

		return entity, nil
	})
}

func (r *policyRepository) Update(
	ctx context.Context,
	entity *detention.DetentionPolicy,
) (*detention.DetentionPolicy, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*detention.DetentionPolicy, error) {
		db := r.db.Keel(ctx)
		t := detentiondb.DetentionPolicyTiers
		err := detentiondb.DetentionPolicies.Save(ctx, db, entity)
		if err == nil {
			_, err = t.Where(t.DetentionPolicyID.Eq(entity.ID)).Delete(ctx, db)
		}
		if err == nil {
			err = t.CreateMany(ctx, db, stampTiers(entity, true))
		}
		if err != nil {
			r.l.Error(
				"failed to update detention policy",
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

func (r *policyRepository) Delete(
	ctx context.Context,
	req *detentionports.GetDetentionPolicyByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := detentiondb.DetentionPolicies.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&detention.DetentionPolicy{ID: req.DetentionPolicyID},
		)
		if err != nil {
			r.l.Error(
				"failed to delete detention policy",
				zap.String("id", req.DetentionPolicyID.String()),
				zap.Error(err),
			)
			return policyErrors.Map(err, req.DetentionPolicyID.String())
		}

		return nil
	})
}

func (r *policyRepository) CodeExists(
	ctx context.Context,
	req *detentionports.DetentionPolicyCodeExistsRequest,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		p := detentiondb.DetentionPolicies
		exists, err := p.Where(
			keel.Raw("lower(?) = lower(?)", p.Code, req.Code),
			excludeID(p.ID, req.ExcludeID),
		).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to check detention policy code", zap.Error(err))
			return false, err
		}

		return exists, nil
	})
}

func excludeID(col interface{ Ne(pulid.ID) keel.Pred }, id *pulid.ID) keel.Pred {
	if id == nil || id.IsNil() {
		return nil
	}

	return col.Ne(*id)
}

func scopeEq(
	col interface {
		Eq(pulid.ID) keel.Pred
		IsNull() keel.Pred
	},
	id *pulid.ID,
) keel.Pred {
	if id == nil || id.IsNil() {
		return col.IsNull()
	}

	return col.Eq(*id)
}

// FindOverlapping locates active policies that pin the same scope and whose
// effective windows intersect. Two of those make resolution a coin flip.
func (r *policyRepository) FindOverlapping(
	ctx context.Context,
	req *detentionports.FindOverlappingDetentionPoliciesRequest,
) ([]*detention.DetentionPolicy, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*detention.DetentionPolicy, error) {
		p := detentiondb.DetentionPolicies
		preds := []keel.Pred{
			p.Status.Eq(detention.PolicyStatusActive),
			p.IsOrgDefault.Eq(req.IsOrgDefault),
			excludeID(p.ID, req.ExcludeID),
			scopeEq(p.CustomerID, req.CustomerID),
			scopeEq(p.LocationID, req.LocationID),
		}
		if req.EffectiveStartDate != nil {
			preds = append(preds, keel.Or(
				p.EffectiveEndDate.IsNull(),
				p.EffectiveEndDate.Gte(*req.EffectiveStartDate),
			))
		}
		if req.EffectiveEndDate != nil {
			preds = append(preds, keel.Or(
				p.EffectiveStartDate.IsNull(),
				p.EffectiveStartDate.Lte(*req.EffectiveEndDate),
			))
		}

		entities, err := p.Where(preds...).All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to find overlapping detention policies", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *policyRepository) SelectOptions(
	ctx context.Context,
	req *detentionports.DetentionPolicySelectOptionsRequest,
) (*pagination.ListResult[*detention.DetentionPolicy], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*detention.DetentionPolicy], error) {
			p := detentiondb.DetentionPolicies
			q := p.Pick(p.ID, p.Name, p.Code, p.Description, p.Status, p.BillingFreeMinutes).
				Where(p.Status.Eq(detention.PolicyStatusActive)).
				Order(p.Name.Asc())

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(
						p.Name.ContainsFold(s),
						p.Code.ContainsFold(s),
						p.Description.ContainsFold(s),
					)
				},
			)
		},
	)
}
