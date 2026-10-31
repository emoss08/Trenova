package billingqueuerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/billingqueueprojection"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const recommendationIssueAlias = "rec_iss"

type recommendationSpec struct {
	kind   billingqueueports.RecommendationKind
	source billingqueueports.RecommendationSource
	where  keel.Pred
}

type recommendationTotals struct {
	Count      int
	TotalMinor *int64
}

func (r *repository) ListRecommendations(
	ctx context.Context,
	req *billingqueueports.ListRecommendationsRequest,
) ([]*billingqueueports.Recommendation, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*billingqueueports.Recommendation, error) {
			db := r.db.Keel(ctx)
			specs := recommendationSpecs(req)
			out := make([]*billingqueueports.Recommendation, 0, len(specs))
			for _, spec := range specs {
				rec, err := recommendationFor(ctx, db, spec)
				if err != nil {
					return nil, err
				}
				if rec != nil {
					out = append(out, rec)
				}
			}

			return out, nil
		},
	)
}

func recommendationFor(
	ctx context.Context,
	db *keel.DB,
	spec recommendationSpec,
) (*billingqueueports.Recommendation, error) {
	bq := billingqueuedb.BillingQueueItems
	totals, err := keel.IntoOne[recommendationTotals](
		ctx,
		db,
		bq.Pick(keel.Count(), bq.AllocatedTotalAmountMinor.Sum()).Where(spec.where),
	)
	if err != nil {
		return nil, err
	}
	if totals.Count == 0 {
		return nil, nil
	}

	ids, err := keel.Into[pulid.ID](
		ctx,
		db,
		bq.Pick(bq.ID).
			Where(spec.where).
			Order(bq.CreatedAt.Asc(), bq.ID.Asc()).
			Limit(billingqueueports.MaxRecommendationItems),
	)
	if err != nil {
		return nil, err
	}

	total := decimal.Zero
	if totals.TotalMinor != nil {
		total = money.DecimalFromMinor(*totals.TotalMinor)
	}

	return &billingqueueports.Recommendation{
		Kind:    spec.kind,
		Source:  spec.source,
		Count:   totals.Count,
		Total:   total,
		ItemIDs: ids,
	}, nil
}

func recommendationSpecs(req *billingqueueports.ListRecommendationsRequest) []recommendationSpec {
	bq := billingqueuedb.BillingQueueItems
	rules := billingqueueports.RecommendationSourceRules
	agent := billingqueueports.RecommendationSourceAgent
	reviewing := bq.Lane.In(billingqueue.ReviewLanes()...)

	specs := []recommendationSpec{
		{
			kind:   billingqueueports.RecommendApproveAgentAgreed,
			source: agent,
			where: keel.And(
				bq.Lane.Eq(billingqueue.LaneReady),
				bq.AgentVerdict.Eq(billingqueue.AgentVerdictAgrees),
			),
		},
		{
			kind:   billingqueueports.RecommendReviewAgentConcerns,
			source: agent,
			where: keel.And(
				bq.Lane.Eq(billingqueue.LaneReady),
				bq.AgentVerdict.Eq(billingqueue.AgentVerdictConcern),
			),
		},
		{
			kind:   billingqueueports.RecommendApproveReady,
			source: rules,
			where:  bq.Lane.Eq(billingqueue.LaneReady),
		},
		{
			kind:   billingqueueports.RecommendRequestPaperwork,
			source: rules,
			where: keel.And(reviewing, openIssueWith(
				true,
				billingqueue.IssuePODMissing,
				billingqueue.IssuePODUnsigned,
			)),
		},
		{
			kind:   billingqueueports.RecommendReviewUnbilled,
			source: rules,
			where: keel.And(
				reviewing,
				openIssueWith(false, billingqueue.IssueUnbilledAccessorial),
			),
		},
		{
			kind:   billingqueueports.RecommendHoldDuplicates,
			source: rules,
			where:  keel.And(reviewing, openIssueWith(false, billingqueue.IssuePossibleDuplicate)),
		},
	}
	if req.CanAssign && req.UserID.IsNotNil() {
		specs = append(specs, recommendationSpec{
			kind:   billingqueueports.RecommendAssignToMe,
			source: rules,
			where: keel.And(
				bq.Lane.Eq(billingqueue.LaneNeedsYou),
				bq.AssignedBillerID.IsNull(),
				keel.Not(billingqueueprojection.HasActionableIssue(bq)),
			),
		})
	}
	if req.CanPost {
		specs = append(specs, recommendationSpec{
			kind:   billingqueueports.RecommendPostApproved,
			source: rules,
			where:  bq.Lane.Eq(billingqueue.LaneApproved),
		})
	}

	return specs
}

func openIssueWith(unrequested bool, codes ...billingqueue.IssueCode) keel.Pred {
	bq := billingqueuedb.BillingQueueItems
	iss := billingqueuedb.Issues.As(recommendationIssueAlias)

	return keel.Exists(iss.Pick(iss.ID).Where(
		iss.ItemID.EqCol(bq.ID),
		iss.OrganizationID.EqCol(bq.OrganizationID),
		iss.BusinessUnitID.EqCol(bq.BusinessUnitID),
		iss.ResolutionKey.IsNull(),
		iss.Code.In(codes...),
		keel.When(unrequested, iss.RequestedAt.IsNull()),
	))
}
