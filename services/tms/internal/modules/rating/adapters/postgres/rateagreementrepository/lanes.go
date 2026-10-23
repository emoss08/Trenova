package rateagreementrepository

import (
	"context"
	"database/sql"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/formulatemplatecol"
	"github.com/emoss08/trenova/pkg/buncolgen/rateagreementcol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// ListLanes pages an agreement's live lanes — active rules an amendment has
// not closed out, most specific first — and counts the live and retired rules
// in one aggregate pass, so an agreement with tens of thousands of lanes is
// never read whole.
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

			if err := r.requireAgreement(ctx, req); err != nil {
				log.Error("failed to find rate agreement", zap.Error(err))
				return nil, err
			}

			total, live, err := r.countLanes(ctx, req, now)
			if err != nil {
				log.Error("failed to count rate agreement lanes", zap.Error(err))
				return nil, err
			}

			pageSize := min(req.Limit, max(live-req.Offset, 0))
			page := &rateagreementports.RateAgreementLanePage{
				Lanes:   make([]*rateagreement.RateAgreementRule, 0, pageSize),
				Count:   live,
				Retired: total - live,
				Limit:   req.Limit,
				Offset:  req.Offset,
			}

			if req.Offset >= live {
				return page, nil
			}

			cols := rateagreementcol.RateAgreementRuleColumns
			if err = r.db.DBForContext(ctx).
				NewSelect().
				Model(&page.Lanes).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return applyLiveLanes(scopeAgreementRules(sq, req), now)
				}).
				Order(cols.SpecificityScore.OrderDesc()).
				Order(cols.ID.OrderAsc()).
				Limit(int64(req.Limit)).
				Offset(int64(req.Offset)).
				Scan(ctx); err != nil {
				log.Error("failed to list rate agreement lanes", zap.Error(err))
				return nil, err
			}

			return page, nil
		},
	)
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

			cols := rateagreementcol.RateAgreementRuleColumns
			ftCols := formulatemplatecol.FormulaTemplateColumns
			groups := make([]*rateagreementports.LanePricingGroup, 0, 8)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model((*rateagreement.RateAgreementRule)(nil)).
				ColumnExpr(cols.FormulaTemplateID.As("formula_template_id")).
				ColumnExpr("COALESCE(?, '') AS formula_template_name", bun.Safe(ftCols.Name.Qualified())).
				ColumnExpr("COALESCE(?, '') AS currency", bun.Safe(cols.Currency.Qualified())).
				ColumnExpr(buncolgen.Count("lanes")).
				ColumnExpr("MIN(?) AS min_rate", bun.Safe(rateInputExpr(cols.Inputs.Qualified()))).
				ColumnExpr("MAX(?) AS max_rate", bun.Safe(rateInputExpr(cols.Inputs.Qualified()))).
				Join(
					"LEFT JOIN ? AS ? ON ?",
					bun.Safe(formulatemplatecol.FormulaTemplateTable.Name),
					bun.Safe(formulatemplatecol.FormulaTemplateTable.Alias),
					bun.Safe(ftCols.ID.EqColumn(cols.FormulaTemplateID)+
						" AND "+ftCols.OrganizationID.EqColumn(cols.OrganizationID)+
						" AND "+ftCols.BusinessUnitID.EqColumn(cols.BusinessUnitID)),
				).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return applyLiveLanes(scopeAgreementRules(sq, req), timeutils.NowUnix())
				}).
				GroupExpr(cols.FormulaTemplateID.Qualified()).
				GroupExpr(ftCols.Name.Qualified()).
				GroupExpr(cols.Currency.Qualified()).
				OrderExpr("lanes DESC").
				Scan(ctx, &groups)
			if err != nil {
				log.Error("failed to summarize rate agreement lanes", zap.Error(err))
				return nil, err
			}

			summary := &rateagreementports.RateAgreementLaneSummary{Groups: groups}
			for _, group := range groups {
				summary.Count += group.Lanes
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
	cols := rateagreementcol.RateAgreementColumns

	exists, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*rateagreement.RateAgreement)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return rateagreementcol.RateAgreementScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.RateAgreementID)
		}).
		Exists(ctx)
	if err != nil {
		return err
	}

	if !exists {
		return dberror.HandleNotFoundError(sql.ErrNoRows, "RateAgreement")
	}

	return nil
}

func (r *repository) countLanes(
	ctx context.Context,
	req *rateagreementports.ListRateAgreementLanesRequest,
	now int64,
) (total, live int, err error) {
	cols := rateagreementcol.RateAgreementRuleColumns

	err = r.db.DBForContext(ctx).
		NewSelect().
		Model((*rateagreement.RateAgreementRule)(nil)).
		ColumnExpr(buncolgen.Count("total")).
		ColumnExpr(
			buncolgen.CountFilter("live", cols.Status.Eq(), openRuleWindowCondition()),
			rateagreement.RuleStatusActive,
			now,
		).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return scopeAgreementRules(sq, req)
		}).
		Scan(ctx, &total, &live)

	return total, live, err
}

func scopeAgreementRules(
	q *bun.SelectQuery,
	req *rateagreementports.ListRateAgreementLanesRequest,
) *bun.SelectQuery {
	cols := rateagreementcol.RateAgreementRuleColumns

	return rateagreementcol.RateAgreementRuleScopeTenant(q, req.TenantInfo).
		Where(cols.RateAgreementID.Eq(), req.RateAgreementID)
}

// applyLiveLanes keeps the rules that price today or will: active, and not
// closed out by an amendment. countLanes states the same predicate as an
// aggregate filter, so the page and its count can never disagree.
func applyLiveLanes(q *bun.SelectQuery, now int64) *bun.SelectQuery {
	cols := rateagreementcol.RateAgreementRuleColumns

	return applyOpenRuleWindow(q.Where(cols.Status.Eq(), rateagreement.RuleStatusActive), now)
}

// openRuleWindowCondition is applyOpenRuleWindow as a single predicate, for an
// aggregate FILTER clause. It binds the moment once.
func openRuleWindowCondition() string {
	cols := rateagreementcol.RateAgreementRuleColumns

	return "(" + cols.EffectiveTo.IsNull() + " OR " + cols.EffectiveTo.Gt() + ")"
}

func rateInputExpr(column string) string {
	return "CASE WHEN jsonb_typeof(" + column + " -> 'rate') = 'object' THEN (" + column +
		" -> 'rate' ->> '$q')::numeric END"
}
