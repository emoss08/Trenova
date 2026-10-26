package rateagreementrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement/rateagreementdb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// ListLanes pages an agreement's live lanes — active rules an amendment has
// not closed out, most specific first — and counts the live and retired rules
// in the same round trip, exactly up to pagination.ExactTotalUpTo and by the
// planner's estimate above, so an agreement with tens of thousands of lanes is
// never read or counted whole.
func (r *repository) ListLanes(
	ctx context.Context,
	req *rateagreementports.ListRateAgreementLanesRequest,
) (*rateagreementports.RateAgreementLanePage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*rateagreementports.RateAgreementLanePage, error) {
			log := r.l.With(
				zap.String("operation", "ListLanes"),
				zap.String("agreementId", req.RateAgreementID.String()),
			)

			req.Normalize()
			now := timeutils.NowUnix()
			ra := rateagreementdb.RateAgreements

			var (
				exists  bool
				lanes   []*rateagreement.RateAgreementRule
				live    keel.RowCount
				retired keel.RowCount
			)
			if err := keel.Batch(ctx, r.db.Keel(ctx),
				keel.ExistsInto(ra.Where(ra.ID.Eq(req.RateAgreementID)), &exists),
				keel.ListInto(lanePageQuery(req, now), &lanes),
				keel.CountEstimateInto(liveLaneQuery(req, now), pagination.ExactTotalUpTo, &live),
				keel.CountEstimateInto(
					retiredLaneQuery(req, now),
					pagination.ExactTotalUpTo,
					&retired,
				),
			); err != nil {
				log.Error("failed to list rate agreement lanes", zap.Error(err))
				return nil, err
			}
			if !exists {
				return nil, agreementErrors.Map(keel.ErrNotFound, req.RateAgreementID.String())
			}
			if lanes == nil {
				lanes = []*rateagreement.RateAgreementRule{}
			}

			return &rateagreementports.RateAgreementLanePage{
				Lanes:        lanes,
				Count:        int(live.N),
				CountExact:   live.Exact,
				Retired:      int(retired.N),
				RetiredExact: retired.Exact,
				Limit:        req.Limit,
				Offset:       req.Offset,
			}, nil
		},
	)
}

func agreementRules(
	req *rateagreementports.ListRateAgreementLanesRequest,
) *keel.Query[*rateagreement.RateAgreementRule] {
	rr := rateagreementdb.RateAgreementRules

	return rr.Where(rr.RateAgreementID.Eq(req.RateAgreementID))
}

func liveLaneQuery(
	req *rateagreementports.ListRateAgreementLanesRequest,
	now int64,
) *keel.Query[*rateagreement.RateAgreementRule] {
	rr := rateagreementdb.RateAgreementRules

	return agreementRules(req).Where(liveLanes(rr, now))
}

func retiredLaneQuery(
	req *rateagreementports.ListRateAgreementLanesRequest,
	now int64,
) *keel.Query[*rateagreement.RateAgreementRule] {
	rr := rateagreementdb.RateAgreementRules

	return agreementRules(req).Where(keel.Not(liveLanes(rr, now)))
}

func lanePageQuery(
	req *rateagreementports.ListRateAgreementLanesRequest,
	now int64,
) *keel.Query[*rateagreement.RateAgreementRule] {
	rr := rateagreementdb.RateAgreementRules

	return agreementRules(req).
		Where(liveLanes(rr, now)).
		Order(rr.SpecificityScore.Desc(), rr.ID.Asc()).
		Limit(req.Limit).
		Offset(req.Offset)
}

func laneSummaryQuery(
	req *rateagreementports.ListRateAgreementLanesRequest,
	now int64,
) *keel.Query[*rateagreement.RateAgreementRule] {
	rr := rateagreementdb.RateAgreementRules
	ft := formulatemplatedb.FormulaTemplates
	rate := keel.CaseNumeric().
		When(
			keel.JSONBOf(rr.Inputs.Path("rate")).TypeOf().Eq("object"),
			keel.CastNumeric(rr.Inputs.Text("rate", "$q")),
		).
		End()

	return agreementRules(req).
		LeftJoin(ft, ft.ID.EqCol(rr.FormulaTemplateID)).
		Where(liveLanes(rr, now)).
		Group(rr.FormulaTemplateID, ft.Name, rr.Currency).
		Pick(
			rr.FormulaTemplateID,
			ft.Name.Coalesce(""),
			rr.Currency.Coalesce(""),
			keel.Count(),
			keel.Min(rate),
			keel.Max(rate),
		).
		Order(keel.Count().Desc())
}

func (r *repository) SummarizeLanes(
	ctx context.Context,
	req *rateagreementports.ListRateAgreementLanesRequest,
) (*rateagreementports.RateAgreementLaneSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*rateagreementports.RateAgreementLaneSummary, error) {
			log := r.l.With(
				zap.String("operation", "SummarizeLanes"),
				zap.String("agreementId", req.RateAgreementID.String()),
			)

			if err := r.requireAgreement(ctx, req); err != nil {
				log.Error("failed to find rate agreement", zap.Error(err))
				return nil, err
			}

			groups, err := keel.Into[rateagreementports.LanePricingGroup](
				ctx,
				r.db.Keel(ctx),
				laneSummaryQuery(req, timeutils.NowUnix()),
			)
			if err != nil {
				log.Error("failed to summarize rate agreement lanes", zap.Error(err))
				return nil, err
			}

			summary := &rateagreementports.RateAgreementLaneSummary{
				Groups: make([]*rateagreementports.LanePricingGroup, 0, len(groups)),
			}
			for i := range groups {
				summary.Groups = append(summary.Groups, &groups[i])
				summary.Count += groups[i].Lanes
			}

			return summary, nil
		},
	)
}

// requireAgreement answers not found, exactly as GetByID would, when the
// agreement is not in the caller's tenant. Without it an unknown agreement
// would read as one with no lanes.
func (r *repository) requireAgreement(
	ctx context.Context,
	req *rateagreementports.ListRateAgreementLanesRequest,
) error {
	ra := rateagreementdb.RateAgreements
	exists, err := ra.Where(ra.ID.Eq(req.RateAgreementID)).Exists(ctx, r.db.Keel(ctx))
	if err != nil {
		return err
	}

	if !exists {
		return agreementErrors.Map(keel.ErrNotFound, req.RateAgreementID.String())
	}

	return nil
}
